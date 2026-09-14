package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// B-26: installing over a running copy asks it to stop and waits. A copy that
// was connected sat in its read with nothing arriving and never noticed, so the
// installer timed out. A quiet server between turns is the ordinary case.
func TestAConnectedComputerStopsWhenAsked(t *testing.T) {
	hello := make(chan struct{})
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		close(hello)
		// Say nothing more, and wait for the daemon to go.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	d := &daemon{log: quietLog(), turns: newRunningTurns(),
		detect: func(context.Context) []protocol.Provider { return nil }}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- d.run(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), "credential") }()

	select {
	case <-hello:
	case err := <-done:
		t.Fatalf("never connected: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("never connected")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a connected computer ignored the request to stop")
	}
}
