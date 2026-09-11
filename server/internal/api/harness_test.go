package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* A real server, a real database, and a websocket pretending to be the daemon.

docs/Later.md L-12 recorded that `postMessage` had no test at all, because
exercising it needs a database, a hub and something acting as a daemon. It needs
all three because that is genuinely what the handler touches — it writes rows,
broadcasts events and sends work to a machine — and any two of them without the
third tests a mock. B-7 lived in that handler and was caught by hand; B-8 and
B-9 both live in the same path.

So the harness is the real thing rather than a stand-in: chi routing through
httptest, pgx against the development database, and a gorilla client dialling
`/daemon` exactly as the daemon binary does. Nothing here is a fake except the
agent, and the agent is the one part that must never run in a test — resolving
`claude` from PATH would spend the owner's quota on every `go test ./...`
(docs/KnownGaps.md G-12).

These SKIP without a database, matching the store tests, so the suite stays
useful on a machine with nothing running. Start one with `make db && make
migrate`. */

type rig struct {
	t     *testing.T
	api   *API
	store *store.Store
	http  *httptest.Server
}

func newRig(t *testing.T) *rig {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://sparstrowgen:sparstrowgen@localhost:5433/sparstrowgen?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("no database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skip("no database reachable — run `make db && make migrate`")
	}
	t.Cleanup(pool.Close)

	// Discard the log: these tests deliberately provoke disconnections and
	// failures, and the warnings they produce are the expected outcome rather
	// than something worth printing.
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := store.New(pool)
	h := hub.New(quiet)
	a := New(s, h, quiet)

	srv := httptest.NewServer(a.Routes())
	t.Cleanup(srv.Close)

	return &rig{t: t, api: a, store: s, http: srv}
}

func (r *rig) ws(path string) string {
	return "ws" + strings.TrimPrefix(r.http.URL, "http") + path
}

// conversation makes one and removes it afterwards: this runs against the
// development database, and leaving rows behind would make the sidebar a
// graveyard of test runs.
func (r *rig) conversation(provider string) protocol.Conversation {
	r.t.Helper()
	c, err := r.store.Create(context.Background(), "D:\\test", provider,
		protocol.Model{ID: "m1", Label: "M One"})
	if err != nil {
		r.t.Fatalf("create conversation: %v", err)
	}
	r.t.Cleanup(func() { _ = r.store.Delete(context.Background(), c.ID) })
	return c
}

func (r *rig) post(path string, body any) *http.Response {
	r.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			r.t.Fatal(err)
		}
	}
	res, err := http.Post(r.http.URL+path, "application/json", &buf)
	if err != nil {
		r.t.Fatalf("POST %s: %v", path, err)
	}
	r.t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

// entry re-reads one entry from the database, so an assertion is about what was
// actually stored rather than about what a handler returned.
func (r *rig) entry(conversationID, entryID string) protocol.Entry {
	r.t.Helper()
	c, err := r.store.Get(context.Background(), conversationID)
	if err != nil {
		r.t.Fatalf("get conversation: %v", err)
	}
	for _, e := range c.Entries {
		if e.ID == entryID {
			return e
		}
	}
	r.t.Fatalf("entry %s is not in the transcript", entryID)
	return protocol.Entry{}
}

// awaitEntry polls for an entry to reach a settled state. The turn's ending
// travels server→daemon→server over real sockets, so it lands a moment after
// the call that caused it.
func (r *rig) awaitEntry(conversationID, entryID string, done func(protocol.Entry) bool) protocol.Entry {
	r.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last protocol.Entry
	for time.Now().Before(deadline) {
		last = r.entry(conversationID, entryID)
		if done(last) {
			return last
		}
		time.Sleep(20 * time.Millisecond)
	}
	r.t.Fatalf("entry never settled: text=%q failure=%q stopped=%v",
		last.Text, last.Failure, last.Stopped)
	return last
}

// ---------------------------------------------------------------------------
// a machine that is not there
// ---------------------------------------------------------------------------

// daemon is a websocket client behaving as the daemon binary does: it says
// hello, receives work, and reports on it.
type daemon struct {
	t    *testing.T
	conn *websocket.Conn
	work chan protocol.ServerMessage
}

func (r *rig) connectDaemon() *daemon {
	r.t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(r.ws("/daemon"), nil)
	if err != nil {
		r.t.Fatalf("dial /daemon: %v", err)
	}
	d := &daemon{t: r.t, conn: conn, work: make(chan protocol.ServerMessage, 8)}
	go func() {
		for {
			var msg protocol.ServerMessage
			if err := conn.ReadJSON(&msg); err != nil {
				close(d.work)
				return
			}
			d.work <- msg
		}
	}()

	d.send(protocol.DaemonMessage{
		Type: protocol.DaemonHello, Machine: "test",
		Providers: []protocol.Provider{
			{ID: "claude", Label: "claude", Availability: protocol.Available},
			{ID: "codex", Label: "codex", Availability: protocol.Available},
		},
	})
	// The hub marks the daemon online when the socket is accepted, but
	// postMessage is only allowed to proceed once it has; wait for that rather
	// than sleeping and hoping.
	deadline := time.Now().Add(3 * time.Second)
	for !r.api.hub.DaemonOnline() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !r.api.hub.DaemonOnline() {
		r.t.Fatal("the server never registered the daemon")
	}
	return d
}

func (d *daemon) send(msg protocol.DaemonMessage) {
	d.t.Helper()
	if err := d.conn.WriteJSON(msg); err != nil {
		d.t.Fatalf("daemon send: %v", err)
	}
}

// nextTurn returns the next run_turn the server hands over.
func (d *daemon) nextTurn() protocol.RunTurn {
	d.t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case msg, ok := <-d.work:
			if !ok {
				d.t.Fatal("the daemon socket closed before any work arrived")
			}
			if msg.Type == protocol.ServerRunTurn && msg.Turn != nil {
				return *msg.Turn
			}
		case <-deadline:
			d.t.Fatal("no run_turn arrived")
		}
	}
}

// vanish drops the socket the way a closing laptop does — no goodbye frame,
// just a connection that stops existing.
func (d *daemon) vanish() { _ = d.conn.Close() }

// ---------------------------------------------------------------------------
// a browser watching
// ---------------------------------------------------------------------------

type browser struct {
	t      *testing.T
	conn   *websocket.Conn
	events chan protocol.ClientEvent
}

func (r *rig) watch() *browser {
	r.t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(r.ws("/ws"), nil)
	if err != nil {
		r.t.Fatalf("dial /ws: %v", err)
	}
	b := &browser{t: r.t, conn: conn, events: make(chan protocol.ClientEvent, 64)}
	r.t.Cleanup(func() { _ = conn.Close() })
	go func() {
		for {
			var ev protocol.ClientEvent
			if err := conn.ReadJSON(&ev); err != nil {
				return
			}
			b.events <- ev
		}
	}()
	return b
}

// await returns the first event matching want, ignoring the others. A single
// action produces several — a message adds an entry, changes a conversation and
// starts a turn — and a test should say which one it is about.
func (b *browser) await(what string, want func(protocol.ClientEvent) bool) protocol.ClientEvent {
	b.t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev := <-b.events:
			if want(ev) {
				return ev
			}
		case <-deadline:
			b.t.Fatalf("no %s event arrived", what)
		}
	}
}
