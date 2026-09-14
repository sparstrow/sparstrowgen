package main

import "testing"

func withRelease(t *testing.T, api, ws string) {
	t.Helper()
	oldAPI, oldWS := releaseAPI, releaseWS
	releaseAPI, releaseWS = api, ws
	t.Cleanup(func() { releaseAPI, releaseWS = oldAPI, oldWS })
}

func TestDevelopmentReadsTheServerFromTheEnvironment(t *testing.T) {
	withRelease(t, "", "")
	t.Setenv("SERVER_WS", "wss://api.example.test/daemon")
	t.Setenv("SERVER_API", "")
	if got := serverWS(); got != "wss://api.example.test/daemon" {
		t.Errorf("serverWS = %q", got)
	}
	if got := serverAPI(); got != "https://api.example.test" {
		t.Errorf("serverAPI derived = %q", got)
	}
}

func TestAReleaseIgnoresLeftoverEnvironmentVariables(t *testing.T) {
	withRelease(t, "https://api.sparstrow.com", "wss://api.sparstrow.com/daemon")
	t.Setenv("SERVER_WS", "ws://localhost:8080/daemon")
	t.Setenv("SERVER_API", "http://localhost:8080")
	t.Setenv("DAEMON_TOKEN", "a-leftover-shared-token-from-running-by-hand")
	t.Setenv("SPARSTROWGEN_HOME", t.TempDir())

	if got := serverWS(); got != "wss://api.sparstrow.com/daemon" {
		t.Errorf("serverWS = %q", got)
	}
	if got := serverAPI(); got != "https://api.sparstrow.com" {
		t.Errorf("serverAPI = %q", got)
	}
	if token, _ := machineToken(); token != "" {
		t.Errorf("an unpaired release fell back to DAEMON_TOKEN")
	}
}
