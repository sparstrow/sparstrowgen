package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPairingRequestOnlyAcceptsOneOpaquePairRequest(t *testing.T) {
	good, err := pairingRequest("sparstrowgen://pair?request=AbCd-123_")
	if err != nil || good != "AbCd-123_" {
		t.Fatalf("good pairing link: request=%q err=%v", good, err)
	}
	// Browsers append a trailing slash to custom-scheme links on some versions.
	if got, err := pairingRequest("sparstrowgen://pair/?request=AbCd"); err != nil || got != "AbCd" {
		t.Errorf("trailing slash: request=%q err=%v", got, err)
	}
	for _, raw := range []string{
		"https://pair?request=AbCd", "sparstrowgen://other?request=AbCd",
		"sparstrowgen://pair", "sparstrowgen://pair?request=AbCd&extra=1",
		"sparstrowgen://pair?request=has%20space", "sparstrowgen://pair/x?request=AbCd",
	} {
		if _, err := pairingRequest(raw); err == nil {
			t.Errorf("accepted unsafe link %q", raw)
		}
	}
}

// isolatedHome points the credential at a temporary directory for one test.
func isolatedHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPARSTROWGEN_HOME", dir)
	return dir
}

func TestClaimSavesTheCredentialItEarns(t *testing.T) {
	isolatedHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/daemon/pair" || body["request"] != "req-1" || body["name"] != "DESKTOP-RIVER" {
			http.Error(w, `{"error":"unexpected"}`, http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"machineId":"m1","credential":"earned-credential"}`))
	}))
	defer srv.Close()

	if err := claim(context.Background(), srv.URL, "req-1", "DESKTOP-RIVER"); err != nil {
		t.Fatal(err)
	}
	if got, paired := machineToken(); got != "earned-credential" || !paired {
		t.Fatalf("machineToken = %q, paired %v", got, paired)
	}
	if !recentlyPaired() {
		t.Error("a credential saved just now is not treated as awaiting approval")
	}
}

func TestARefusedClaimSaysWhyAndSavesNothing(t *testing.T) {
	isolatedHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"that pairing request has expired or was already used"}`))
	}))
	defer srv.Close()

	err := claim(context.Background(), srv.URL, "req-1", "PC")
	if err == nil || !strings.Contains(err.Error(), "expired or was already used") {
		t.Fatalf("err = %v", err)
	}
	if c := readCredential(); c != "" {
		t.Fatalf("a refused claim saved a credential: %q", c)
	}
}

func TestRunTellsApprovalApartFromDisconnection(t *testing.T) {
	for status, want := range map[int]error{
		http.StatusUnauthorized: errAwaitingApproval,
		http.StatusForbidden:    errDisconnected,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		d := &daemon{log: slog.New(slog.NewTextHandler(io.Discard, nil)), turns: newRunningTurns()}
		err := d.run(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/daemon", "token")
		srv.Close()
		if !errors.Is(err, want) {
			t.Errorf("status %d: err = %v, want %v", status, err, want)
		}
	}
}

func TestADisconnectedComputerForgetsItsCredentialAndStops(t *testing.T) {
	isolatedHome(t)
	if err := saveCredential("revoked-credential"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	t.Setenv("SERVER_WS", "ws"+strings.TrimPrefix(srv.URL, "http")+"/daemon")

	done := make(chan bool, 1)
	go func() {
		done <- serve(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a disconnected computer kept retrying")
	}
	if c := readCredential(); c != "" {
		t.Fatalf("the revoked credential is still on disk: %q", c)
	}
	if _, err := os.Stat(func() string { p, _ := credentialFile(); return p }()); !os.IsNotExist(err) {
		t.Errorf("credential file still exists: %v", err)
	}
}
