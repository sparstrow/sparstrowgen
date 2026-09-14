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

const (
	credentialName = "machine-credential"
	// pendingName holds a credential that is waiting for approval. The computer
	// keeps its working credential until the new one is approved, so declining
	// a new pairing never unpairs a computer that was already connected.
	pendingName = "machine-credential.pending"
)

type credentialKind int

const (
	noCredential credentialKind = iota
	pendingCredential
	pairedCredential
	sharedToken
)

func credentialPath(name string) (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func readCredential(name string) string {
	p, err := credentialPath(name)
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func writeCredential(name, credential string) error {
	p, err := credentialPath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(credential+"\n"), 0o600)
}

// promotePending makes an approved pending credential this computer's
// credential, replacing any earlier one. It does nothing if the pending file
// has since been replaced by a newer pairing.
func promotePending(credential string) error {
	if readCredential(pendingName) != credential {
		return nil
	}
	from, err := credentialPath(pendingName)
	if err != nil {
		return err
	}
	to, err := credentialPath(credentialName)
	if err != nil {
		return err
	}
	return os.Rename(from, to)
}

// forgetCredential removes a saved credential only if it is still the one that
// was refused. A new pairing can land between an old copy's dial and its 403,
// and deleting that would unpair the computer the person just paired.
func forgetCredential(name, refused string) error {
	if readCredential(name) != refused {
		return nil
	}
	p, err := credentialPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// machineToken is what the daemon dials with next. A pairing waiting for
// approval goes first; then the computer's approved credential. Only a
// development build falls back to the shared DAEMON_TOKEN route.
func machineToken() (string, credentialKind) {
	if c := readCredential(pendingName); c != "" {
		return c, pendingCredential
	}
	if c := readCredential(credentialName); c != "" {
		return c, pairedCredential
	}
	if released() {
		return "", noCredential
	}
	if t := os.Getenv("DAEMON_TOKEN"); t != "" {
		return t, sharedToken
	}
	return "", noCredential
}
