package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Set by the release build (scripts/package-windows.ps1) with
// -ldflags "-X main.releaseAPI=... -X main.releaseWS=...".
//
// A release knows exactly one server and ignores SERVER_WS and SERVER_API, so a
// variable left behind from running the daemon by hand can never point an
// installed copy somewhere else.
var (
	releaseAPI string
	releaseWS  string
)

func released() bool { return releaseWS != "" }

func serverWS() string {
	if released() {
		return releaseWS
	}
	return env("SERVER_WS", "ws://localhost:8080/daemon")
}

func serverAPI() string {
	if released() && releaseAPI != "" {
		return strings.TrimSuffix(releaseAPI, "/")
	}
	if v := os.Getenv("SERVER_API"); v != "" && !released() {
		return strings.TrimSuffix(v, "/")
	}
	base := strings.TrimSuffix(serverWS(), "/daemon")
	switch {
	case strings.HasPrefix(base, "wss://"):
		return "https://" + strings.TrimPrefix(base, "wss://")
	case strings.HasPrefix(base, "ws://"):
		return "http://" + strings.TrimPrefix(base, "ws://")
	}
	return base
}

// dataDir holds this computer's credential and logs.
//
// Local rather than roaming on Windows: the credential identifies one computer,
// and a roaming profile would carry it to another. A development build keeps
// its own directory so `go run` never borrows an installed copy's credential.
func dataDir() (string, error) {
	if v := os.Getenv("SPARSTROWGEN_HOME"); v != "" {
		return v, nil
	}
	var base string
	var err error
	if runtime.GOOS == "windows" {
		base, err = os.UserCacheDir() // %LOCALAPPDATA%
	} else {
		base, err = os.UserConfigDir()
	}
	if err != nil {
		return "", err
	}
	name := "sparstrowgen"
	if !released() {
		name = "sparstrowgen-dev"
	}
	return filepath.Join(base, name), nil
}

func credentialFile() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "machine-credential"), nil
}

func readCredential() string {
	p, err := credentialFile()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func saveCredential(credential string) error {
	p, err := credentialFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(credential+"\n"), 0o600)
}

func forgetCredential() error {
	p, err := credentialFile()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// machineToken is what the daemon dials with. A paired credential always wins.
// Only a development build falls back to the shared DAEMON_TOKEN route.
func machineToken() (token string, paired bool) {
	if c := readCredential(); c != "" {
		return c, true
	}
	if released() {
		return "", false
	}
	return os.Getenv("DAEMON_TOKEN"), false
}
