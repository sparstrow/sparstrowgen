// Command launcher is the small Windows-facing part of a sparstrowgen
// installation. It owns URI activation only; the daemon remains the process
// that talks to the server and runs agents.
package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const scheme = "sparstrowgen"

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: sparstrowgen-launcher <install|sparstrowgen://pair?request=...>")
	}
	if os.Args[1] == "install" {
		if err := register(os.Args[0]); err != nil {
			log.Fatal(err)
		}
		return
	}
	request, err := pairingRequest(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	if err := claimAndStart(request); err != nil {
		log.Fatal(err)
	}
}

func pairingRequest(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("read pairing link: %w", err)
	}
	if u.Scheme != scheme || u.Host != "pair" || u.Path != "" {
		return "", fmt.Errorf("this is not a sparstrowgen pairing link")
	}
	q := u.Query()
	if len(q) != 1 || len(q["request"]) != 1 || q.Get("request") == "" {
		return "", fmt.Errorf("the pairing link is incomplete")
	}
	request := q.Get("request")
	if len(request) > 128 || strings.ContainsAny(request, " /\\\r\n") {
		return "", fmt.Errorf("the pairing link is not valid")
	}
	return request, nil
}

func claimAndStart(request string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	daemon := filepath.Join(filepath.Dir(exe), "sparstrowgen-daemon.exe")
	if _, err := os.Stat(daemon); err != nil {
		return fmt.Errorf("the sparstrowgen daemon is missing: %w", err)
	}
	cmd := exec.Command(daemon, "pair", "-request", request)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claim pairing request: %w", err)
	}
	if err := registerStartup(daemon); err != nil {
		return err
	}
	// The server holds the machine at the approval boundary. Starting before
	// approval is safe: it receives 401 until the browser approves, then its
	// existing reconnect loop establishes the paired websocket.
	return exec.Command(daemon).Start()
}

func register(executable string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\sparstrowgen`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("register pairing link: %w", err)
	}
	defer key.Close()
	if err := key.SetStringValue("", "URL:sparstrowgen Protocol"); err != nil {
		return err
	}
	if err := key.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}
	command, _, err := registry.CreateKey(key, `shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer command.Close()
	return command.SetStringValue("", fmt.Sprintf("\"%s\" \"%%1\"", executable))
}

func registerStartup(daemon string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("register start at sign-in: %w", err)
	}
	defer key.Close()
	return key.SetStringValue("sparstrowgen", fmt.Sprintf("\"%s\"", daemon))
}
