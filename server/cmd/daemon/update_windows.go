//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// updatesSupported is true for a release running from its installed location.
// A copy opened from Downloads installs itself first, and a development build
// never updates.
func updatesSupported() bool {
	if !released() || updateSource() == "" {
		return false
	}
	self, err := os.Executable()
	if err != nil {
		return false
	}
	dir, err := installDir()
	if err != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(self), filepath.Clean(filepath.Join(dir, "sparstrowgen.exe")))
}

// startUpdate copies this executable out of the way and starts it as the
// updater, so the code doing the swap is the version known to work.
func startUpdate(newExe, from, to string) error {
	dir, err := updatesDir()
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	helper := filepath.Join(dir, updaterExe)
	if err := copyFile(self, helper); err != nil {
		return fmt.Errorf("prepare the updater: %w", err)
	}
	cmd := exec.Command(helper, "apply-update", "-new", newExe, "-target", self, "-from", from, "-to", to)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start the updater: %w", err)
	}
	return cmd.Process.Release()
}

// applyUpdateCommand is the updater process: `sparstrowgen-updater.exe apply-update`.
func applyUpdateCommand(args []string) int {
	fs := flag.NewFlagSet("apply-update", flag.ContinueOnError)
	newExe := fs.String("new", "", "verified installer to put in place")
	target := fs.String("target", "", "installed executable to replace")
	from := fs.String("from", "", "version being replaced")
	to := fs.String("to", "", "version being installed")
	if err := fs.Parse(args); err != nil || *newExe == "" || *target == "" || *from == "" || *to == "" {
		return 2
	}
	log, closeLog := fileLogger()
	defer closeLog()
	log = log.With("process", "updater")
	dir, err := updatesDir()
	if err != nil {
		log.Error("no data directory", "err", err)
		return 1
	}
	job := updateJob{Target: *target, New: *newExe, From: *from, To: *to, Dir: dir}
	if err := runUpdate(log, windowsUpdateOps(), job, healthWait); err != nil {
		return 1
	}
	return 0
}

func windowsUpdateOps() updateOps {
	return updateOps{
		stopRunning: stopRunning,
		start: func(exe string) (<-chan struct{}, func(), error) {
			cmd := exec.Command(exe, "run")
			cmd.SysProcAttr = &syscall.SysProcAttr{
				CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
				HideWindow:    true,
			}
			if err := cmd.Start(); err != nil {
				return nil, nil, err
			}
			exited := make(chan struct{})
			go func() { _ = cmd.Wait(); close(exited) }()
			return exited, func() { _ = cmd.Process.Kill() }, nil
		},
		sleep: time.Sleep,
		now:   time.Now,
	}
}
