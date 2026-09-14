//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	instanceObject = `Local\sparstrowgen-daemon`
	exitObject     = `Local\sparstrowgen-daemon-exit`
	stopWait       = 15 * time.Second
)

func objectName(base string) *uint16 {
	if !released() {
		base += "-dev"
	}
	p, _ := windows.UTF16PtrFromString(base)
	return p
}

// installDir is the per-user program location, so installing needs no
// administrator and touches no other Windows account.
func installDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Programs", "sparstrowgen"), nil
}

// install copies this executable into place, registers the pairing link and
// start-at-sign-in, and starts the background copy.
func install() error {
	if !released() {
		return errors.New("installing needs a release build: run scripts/package-windows.ps1")
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	dir, err := installDir()
	if err != nil {
		return err
	}
	target := filepath.Join(dir, "sparstrowgen.exe")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	if !strings.EqualFold(filepath.Clean(self), filepath.Clean(target)) {
		if err := stopRunning(); err != nil {
			return err
		}
		removeOldCopies(dir)
		if _, err := os.Stat(target); err == nil {
			// Windows will not overwrite a running executable, but it will rename one.
			if err := os.Rename(target, fmt.Sprintf("%s.%d.old", target, time.Now().UnixNano())); err != nil {
				return fmt.Errorf("replace the installed copy: %w", err)
			}
		}
		if err := copyFile(self, target); err != nil {
			return fmt.Errorf("install to %s: %w", dir, err)
		}
	}
	if err := registerLink(target); err != nil {
		return err
	}
	if err := registerStartup(target); err != nil {
		return err
	}
	return startBackground(target)
}

// activate handles a sparstrowgen://pair link opened by the browser.
func activate(link string) error {
	if !released() {
		return errors.New("pairing links need a release build: use `pair -request` in development")
	}
	request, err := pairingRequest(link)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	already, err := claim(ctx, serverAPI(), request, computerName())
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if err := registerStartup(self); err != nil {
		return err
	}
	if already {
		// Nothing changed, so nothing running is interrupted. A second
		// background copy exits by itself if one is already running.
		return startBackground(self)
	}
	// A copy already running is dialling with its old credential, or none.
	// Restart it so it waits on the pending one.
	if err := stopRunning(); err != nil {
		return err
	}
	return startBackground(self)
}

// singleInstance holds a named mutex for the life of the background daemon, so
// the same computer is never connected twice. It must be called on the thread
// that later releases it: mutex ownership on Windows belongs to a thread.
func singleInstance() (release func(), ok bool) {
	runtime.LockOSThread()
	h, err := windows.CreateMutex(nil, true, objectName(instanceObject))
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(h)
		runtime.UnlockOSThread()
		return nil, false
	}
	if err != nil {
		runtime.UnlockOSThread()
		return func() {}, true
	}
	return func() {
		_ = windows.ReleaseMutex(h)
		_ = windows.CloseHandle(h)
		runtime.UnlockOSThread()
	}, true
}

// watchForExit cancels when another copy asks this one to stop. Auto-reset, so
// a copy started afterwards never inherits a stale request.
func watchForExit(cancel context.CancelFunc) {
	h, err := windows.CreateEvent(nil, 0, 0, objectName(exitObject))
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return
	}
	go func() {
		if s, _ := windows.WaitForSingleObject(h, windows.INFINITE); s == windows.WAIT_OBJECT_0 {
			cancel()
		}
	}()
}

// stopRunning asks a running background daemon to exit and waits until it has.
// Active turns end, and the server records that the computer disconnected.
func stopRunning() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	h, err := windows.CreateMutex(nil, false, objectName(instanceObject))
	if err == nil {
		// We created it, so nothing was running.
		_ = windows.CloseHandle(h)
		return nil
	}
	if !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return nil
	}
	defer windows.CloseHandle(h)
	if ev, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, objectName(exitObject)); err == nil {
		_ = windows.SetEvent(ev)
		_ = windows.CloseHandle(ev)
	}
	s, err := windows.WaitForSingleObject(h, uint32(stopWait.Milliseconds()))
	switch s {
	case windows.WAIT_OBJECT_0, windows.WAIT_ABANDONED:
		_ = windows.ReleaseMutex(h)
		return nil
	}
	if err != nil {
		return fmt.Errorf("wait for the running sparstrowgen to stop: %w", err)
	}
	return errors.New("the running sparstrowgen did not stop in time; sign out of Windows and try again")
}

func startBackground(exe string) error {
	cmd := exec.Command(exe, "run")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start sparstrowgen: %w", err)
	}
	return cmd.Process.Release()
}

func registerLink(exe string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\sparstrowgen`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("register the pairing link: %w", err)
	}
	defer key.Close()
	if err := key.SetStringValue("", "URL:sparstrowgen"); err != nil {
		return err
	}
	if err := key.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}
	command, _, err := registry.CreateKey(key, `shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("register the pairing link: %w", err)
	}
	defer command.Close()
	return command.SetStringValue("", fmt.Sprintf(`"%s" "%%1"`, exe))
}

func registerStartup(exe string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("register start at sign-in: %w", err)
	}
	defer key.Close()
	return key.SetStringValue("sparstrowgen", fmt.Sprintf(`"%s" run`, exe))
}

func removeOldCopies(dir string) {
	matches, _ := filepath.Glob(filepath.Join(dir, "sparstrowgen.exe.*.old"))
	for _, m := range matches {
		_ = os.Remove(m)
	}
}

// ensureHiddenConsole gives a windowless daemon a hidden console before any
// agent starts, so the CLIs it launches inherit it instead of each flashing a
// window. Multica's daemon does the same (internal/util/proc_windows.go).
func ensureHiddenConsole() {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getConsole := kernel32.NewProc("GetConsoleWindow")
	if hwnd, _, _ := getConsole.Call(); hwnd != 0 {
		return
	}
	if r, _, _ := kernel32.NewProc("AllocConsole").Call(); r == 0 {
		return
	}
	if hwnd, _, _ := getConsole.Call(); hwnd != 0 {
		windows.NewLazySystemDLL("user32.dll").NewProc("ShowWindow").Call(hwnd, 0)
	}
}

// notify is the only way a windowless executable can speak to the person who
// opened it.
func notify(text string, failed bool) {
	if !released() {
		fmt.Fprintln(os.Stderr, text)
		return
	}
	var flags uint32 = windows.MB_OK | windows.MB_ICONINFORMATION
	if failed {
		flags = windows.MB_OK | windows.MB_ICONERROR
	}
	body, _ := windows.UTF16PtrFromString(text)
	title, _ := windows.UTF16PtrFromString("sparstrowgen")
	_, _ = windows.MessageBox(0, body, title, flags)
}
