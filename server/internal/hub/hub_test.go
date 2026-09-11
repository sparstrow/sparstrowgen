package hub

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* The daemon-identity contract, tested here rather than through the API.

An equivalent API-level test was written first and thrown away: it passed with
the guard removed, because the ordering it needs — a superseded socket's
teardown running WHILE the replacement has a turn in flight — is decided by when
a background read loop happens to notice a closed connection, and a test cannot
pin that down. A test that cannot fail is worse than no test, since it reads
like coverage.

This is the same question asked where the answer is deterministic: given two
sockets, does ClearDaemon know which one is current? */

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
// machine going away — which would tell every browser the machine is
// unreachable moments after it came back, and would let the caller abandon the
// new connection's turns (docs/Bugs.md B-8).
func TestClearingASupersededSocketIsNotTheMachineGoingAway(t *testing.T) {
	h := testHub(t)
	first, second := dial(t), dial(t)

	h.SetDaemon(first)
	h.SetDaemon(second) // the reconnect

	if h.ClearDaemon(first) {
		t.Error("the superseded socket reported itself as the current daemon")
	}
	if !h.DaemonOnline() {
		t.Error("the machine was reported offline while its new socket was connected")
	}

	if !h.ClearDaemon(second) {
		t.Error("the current socket did not report itself as current")
	}
	if h.DaemonOnline() {
		t.Error("the machine is still online after its only socket went")
	}
}

// Providers are forgotten when the machine goes, because availability is not
// knowledge we still have — but only when it has actually gone.
func TestASupersededSocketDoesNotForgetWhatIsInstalled(t *testing.T) {
	h := testHub(t)
	first, second := dial(t), dial(t)

	h.SetDaemon(first)
	h.SetProviders([]protocol.Provider{{ID: "claude", Label: "claude"}})
	h.SetDaemon(second)

	h.ClearDaemon(first)
	if len(h.Providers()) != 1 {
		t.Errorf("providers = %d, want them kept: the machine is still connected", len(h.Providers()))
	}

	h.ClearDaemon(second)
	if len(h.Providers()) != 0 {
		t.Error("providers survived the machine going, which claims it can still answer")
	}
}
