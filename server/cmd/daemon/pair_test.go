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
	"sync"
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

// isolatedHome points the credentials at a temporary directory for one test.
func isolatedHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPARSTROWGEN_HOME", dir)
	return dir
}

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestClaimSavesTheNewCredentialAsPending(t *testing.T) {
	isolatedHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/daemon/pair" || body["request"] != "req-1" || body["name"] != "DESKTOP-RIVER" || body["current"] != "" {
			http.Error(w, `{"error":"unexpected"}`, http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"machineId":"m1","credential":"earned-credential"}`))
	}))
	defer srv.Close()

	already, err := claim(context.Background(), srv.URL, "req-1", "DESKTOP-RIVER")
	if err != nil || already {
		t.Fatalf("claim: already=%v err=%v", already, err)
	}
	if got, kind := machineToken(); got != "earned-credential" || kind != pendingCredential {
		t.Fatalf("machineToken = %q, kind %v; want the pending credential", got, kind)
	}
	if c := readCredential(credentialName); c != "" {
		t.Errorf("an unapproved credential became the computer's credential: %q", c)
	}
	if !recentlyPaired() {
		t.Error("a credential saved just now is not treated as awaiting approval")
	}
}

func TestClaimOnAnAlreadyConnectedComputerChangesNothing(t *testing.T) {
	isolatedHome(t)
	if err := writeCredential(credentialName, "working-credential"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["current"] != "working-credential" {
			http.Error(w, `{"error":"the current credential was not sent"}`, http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"machineId":"m1","alreadyPaired":true}`))
	}))
	defer srv.Close()

	already, err := claim(context.Background(), srv.URL, "req-2", "DESKTOP-RIVER")
	if err != nil || !already {
		t.Fatalf("claim: already=%v err=%v", already, err)
	}
	if c := readCredential(pendingName); c != "" {
		t.Errorf("an already-connected computer saved a pending credential: %q", c)
	}
	if c := readCredential(credentialName); c != "working-credential" {
		t.Errorf("the working credential changed: %q", c)
	}
}

func TestARefusedClaimSaysWhyAndSavesNothing(t *testing.T) {
	isolatedHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"that pairing request has expired or was already used"}`))
	}))
	defer srv.Close()

	_, err := claim(context.Background(), srv.URL, "req-1", "PC")
	if err == nil || !strings.Contains(err.Error(), "expired or was already used") {
		t.Fatalf("err = %v", err)
	}
	if c := readCredential(pendingName); c != "" {
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
		d := &daemon{log: quietLog(), turns: newRunningTurns()}
		err := d.run(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/daemon", "token")
		srv.Close()
		if !errors.Is(err, want) {
			t.Errorf("status %d: err = %v, want %v", status, err, want)
		}
	}
}

func TestAnApprovedPendingCredentialReplacesTheOldOne(t *testing.T) {
	isolatedHome(t)
	if err := writeCredential(credentialName, "old"); err != nil {
		t.Fatal(err)
	}
	if err := writeCredential(pendingName, "new"); err != nil {
		t.Fatal(err)
	}
	if err := promotePending("some-other-pairing"); err != nil {
		t.Fatal(err)
	}
	if readCredential(credentialName) != "old" || readCredential(pendingName) != "new" {
		t.Fatal("promoting a credential that is not the pending one changed something")
	}
	if err := promotePending("new"); err != nil {
		t.Fatal(err)
	}
	if got := readCredential(credentialName); got != "new" {
		t.Errorf("credential after approval = %q, want new", got)
	}
	if got := readCredential(pendingName); got != "" {
		t.Errorf("pending credential still present after approval: %q", got)
	}
}

// A new pairing can land between an old copy's dial and its 403. The 403 is
// about the credential that copy dialled with, not whatever is on disk now.
func TestForgettingARevokedCredentialKeepsANewerOne(t *testing.T) {
	isolatedHome(t)
	if err := writeCredential(credentialName, "just-paired"); err != nil {
		t.Fatal(err)
	}
	if err := forgetCredential(credentialName, "revoked-earlier"); err != nil {
		t.Fatal(err)
	}
	if got := readCredential(credentialName); got != "just-paired" {
		t.Fatalf("a newer credential was deleted: %q", got)
	}
	if err := forgetCredential(credentialName, "just-paired"); err != nil {
		t.Fatal(err)
	}
	if got := readCredential(credentialName); got != "" {
		t.Fatalf("the refused credential was kept: %q", got)
	}
}

// Declining a new pairing ("Not now") must not unpair a computer that was
// already connected: it goes back to dialling with its earlier credential.
func TestADeclinedPairingFallsBackToTheEarlierCredential(t *testing.T) {
	isolatedHome(t)
	if err := writeCredential(credentialName, "earlier"); err != nil {
		t.Fatal(err)
	}
	if err := writeCredential(pendingName, "declined"); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		mu.Lock()
		seen = append(seen, token)
		mu.Unlock()
		if token == "declined" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// Refuse the earlier one too, as "waiting", so the test never reaches a
		// real connection (which would detect the agent CLIs on PATH).
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	t.Setenv("SERVER_WS", "ws"+strings.TrimPrefix(srv.URL, "http")+"/daemon")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { serve(ctx, quietLog()); close(done) }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		fellBack := len(seen) >= 2 && seen[0] == "declined" && seen[len(seen)-1] == "earlier"
		mu.Unlock()
		if fellBack {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("never dialled with the earlier credential; dialled %v", seen)
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-done

	if got := readCredential(pendingName); got != "" {
		t.Errorf("the declined credential is still pending: %q", got)
	}
	if got := readCredential(credentialName); got != "earlier" {
		t.Errorf("the earlier credential was lost: %q", got)
	}
}

func TestADisconnectedComputerForgetsItsCredentialAndStops(t *testing.T) {
	isolatedHome(t)
	if err := writeCredential(credentialName, "revoked-credential"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	t.Setenv("SERVER_WS", "ws"+strings.TrimPrefix(srv.URL, "http")+"/daemon")

	done := make(chan bool, 1)
	go func() { done <- serve(context.Background(), quietLog()) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a disconnected computer kept retrying")
	}
	if c := readCredential(credentialName); c != "" {
		t.Fatalf("the revoked credential is still on disk: %q", c)
	}
	p, _ := credentialPath(credentialName)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("credential file still exists: %v", err)
	}
}

// B-27: reinstalling on a paired computer must not send the person back to
// Add computer.
func TestTheInstallNoticeOnlyAsksToPairAnUnpairedComputer(t *testing.T) {
	isolatedHome(t)
	if got := installedNotice(); !strings.Contains(got, "Add computer") {
		t.Errorf("a first install must say how to pair: %q", got)
	}
	if err := writeCredential(credentialName, "working-credential"); err != nil {
		t.Fatal(err)
	}
	if got := installedNotice(); strings.Contains(got, "Add computer") || !strings.Contains(got, "already paired") {
		t.Errorf("a reinstall on a paired computer: %q", got)
	}
}
