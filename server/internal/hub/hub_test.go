package hub

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* The machine-identity contract, tested here rather than through the API.

An equivalent API-level test was written first and thrown away: it passed with
the guard removed, because the ordering it needs — a superseded socket's
teardown running WHILE the replacement has a turn in flight — is decided by when
a background read loop happens to notice a closed connection. This is the same
question asked where the answer is deterministic. */

const (
	alice = "account-alice"
	bob   = "account-bob"
)

func testHub(t *testing.T) *Hub {
	t.Helper()
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// dial gives a real *websocket.Conn. The hub compares connections by pointer,
// so two must be genuinely distinct objects for the test to mean anything.
func dial(t *testing.T) *websocket.Conn {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// A daemon that reconnects replaces its own earlier socket, and the old one's
// read loop only notices afterwards. Its teardown must not be mistaken for the
// machine going away (docs/Bugs.md B-8).
func TestClearingASupersededSocketIsNotTheMachineGoingAway(t *testing.T) {
	h := testHub(t)
	first, second := dial(t), dial(t)

	h.SetDaemon(alice, first)
	h.SetDaemon(alice, second) // the reconnect

	if h.ClearDaemon(alice, first) {
		t.Error("the superseded socket reported itself as the current daemon")
	}
	if !h.DaemonOnline(alice) {
		t.Error("the machine was reported offline while its new socket was connected")
	}

	if !h.ClearDaemon(alice, second) {
		t.Error("the current socket did not report itself as current")
	}
	if h.DaemonOnline(alice) {
		t.Error("the machine is still online after its only socket went")
	}
}

// Providers are forgotten when the machine goes, because availability is not
// knowledge we still have — but only when it has actually gone.
func TestASupersededSocketDoesNotForgetWhatIsInstalled(t *testing.T) {
	h := testHub(t)
	first, second := dial(t), dial(t)

	h.SetDaemon(alice, first)
	h.SetDaemon(alice, second)
	h.SetProviders(alice, []protocol.Provider{{ID: "claude", Label: "claude"}})

	h.ClearDaemon(alice, first)
	if len(h.Providers(alice)) != 1 {
		t.Errorf("providers = %d, want them kept: the machine is still connected", len(h.Providers(alice)))
	}

	h.ClearDaemon(alice, second)
	if len(h.Providers(alice)) != 0 {
		t.Error("providers survived the machine going, which claims it can still answer")
	}
}

// One account's machine is no use to, and invisible to, another account.
func TestOneAccountsMachineIsNotAnothers(t *testing.T) {
	h := testHub(t)
	h.SetDaemon(alice, dial(t))
	h.SetProviders(alice, []protocol.Provider{{ID: "claude", Label: "claude"}})

	if h.DaemonOnline(bob) {
		t.Error("bob is told alice's machine is online for him")
	}
	if len(h.Providers(bob)) != 0 {
		t.Error("bob sees alice's providers")
	}
	if h.SendToDaemon(bob, protocol.ServerMessage{Type: protocol.ServerRunTurn}) {
		t.Error("work for bob was sent to alice's machine")
	}

	// A hello arriving for an account with no machine does not invent one.
	h.SetProviders(bob, []protocol.Provider{{ID: "codex", Label: "codex"}})
	if h.DaemonOnline(bob) || len(h.Providers(bob)) != 0 {
		t.Error("providers for an account with no machine created a machine")
	}
}

// A reply is only accepted from a machine of the account that asked. Request
// ids are a counter; the account is what makes an answer belong.
func TestOnlyTheAskingAccountsMachineCanAnswer(t *testing.T) {
	h := testHub(t)
	h.SetDaemon(alice, dial(t))

	answered := make(chan protocol.DaemonMessage, 1)
	go func() {
		m, err := h.Ask(context.Background(), alice, protocol.ServerMessage{Type: protocol.ServerListDir, Path: "C:\\"})
		if err == nil {
			answered <- m
		}
	}()

	// Wait for the request to be registered before answering it.
	deadline := time.Now().Add(2 * time.Second)
	var id string
	for id == "" && time.Now().Before(deadline) {
		h.mu.RLock()
		for k := range h.pending {
			id = k
		}
		h.mu.RUnlock()
		time.Sleep(5 * time.Millisecond)
	}
	if id == "" {
		t.Fatal("the request was never registered")
	}

	forged := protocol.DaemonMessage{RequestID: id, Listing: &protocol.DirListing{Path: "forged"}}
	h.Deliver(bob, forged)
	select {
	case m := <-answered:
		t.Fatalf("another account's machine answered alice's request: %+v", m.Listing)
	case <-time.After(100 * time.Millisecond):
	}

	h.Deliver(alice, protocol.DaemonMessage{RequestID: id, Listing: &protocol.DirListing{Path: "C:\\"}})
	select {
	case m := <-answered:
		if m.Listing.Path != "C:\\" {
			t.Errorf("alice received %q", m.Listing.Path)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("alice's own machine could not answer")
	}
}
