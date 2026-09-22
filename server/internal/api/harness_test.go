package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/mail"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
	"github.com/sparstrow/sparstrowgen/server/internal/testdb"
)

/* A real server, a real database, a websocket pretending to be the daemon, and
a mailbox that keeps what it is sent.

docs/Later.md L-12 recorded that `postMessage` had no test at all, because
exercising it needs a database, a hub and something acting as a daemon. It needs
all three because that is genuinely what the handler touches — it writes rows,
broadcasts events and sends work to a machine — and any two of them without the
third tests a mock.

So the harness is the real thing rather than a stand-in: chi routing through
httptest, pgx against the TEST database, and a gorilla client dialling `/daemon`
exactly as the daemon binary does. Nothing here is a fake except the agent and
the mail server. The agent must never run in a test — resolving `claude` from
PATH would spend real quota (docs/KnownGaps.md G-12) — and a test must never
send real email.

These SKIP without a database, matching the store tests. */

// testPassword is the password the harness signs in with. A fixed string in a
// test file is not a secret — the hash is generated fresh in newRig, so nothing
// here is a credential that works anywhere but inside this process.
const testPassword = "correct-horse-battery-staple"

// testEmail is the owner account every rig signs in as.
const testEmail = "owner@sparstrow.test"

// testInvited is on the invitation list and has no account until a test makes
// one.
const testInvited = "invited@sparstrow.test"

// testSecond is also invited, for a second account.
const testSecond = "second@sparstrow.test"

// testDaemonToken must clear the 32-character floor the server enforces.
const testDaemonToken = "test-daemon-token-0123456789abcdefgh"

// testOrigin is what the harness claims to be. It matters: the API answers
// CORS for exactly one origin, refuses websockets from any other, and builds
// the links in emails from it.
const testOrigin = "http://sparstrowgen.test"

type rig struct {
	t     *testing.T
	api   *API
	store *store.Store
	http  *httptest.Server
	// client carries the session cookie, so these tests exercise the same path
	// a browser does rather than quietly bypassing the login.
	client *http.Client
	// userID is the account this client is signed in as.
	userID string
	// workspaceID is that account's workspace. Every conversation lives in one
	// (D-050), so every test that makes one uses this.
	workspaceID string
	mail        *mailbox
}

func newRig(t *testing.T) *rig {
	t.Helper()
	pool := testdb.Pool(t)

	// Discard the log: these tests deliberately provoke disconnections and
	// failures, and the warnings they produce are the expected outcome.
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := store.New(pool)
	h := hub.New(quiet)
	box := newMailbox()

	a := New(s, h, quiet, Config{
		DaemonToken: testDaemonToken,
		Origin:      testOrigin,
		// The harness speaks http://127.0.0.1, and a Secure cookie would never
		// be stored by the jar.
		SecureCookie: false,
		OwnerEmail:   testEmail,
		Invited:      []string{testInvited, testSecond},
		Mailer:       box,
	})
	// Tests send several emails to one address in quick succession on purpose.
	// The one test about spacing sets this back.
	a.sendGap = 0

	srv := httptest.NewServer(a.Routes())
	t.Cleanup(srv.Close)

	r := &rig{t: t, api: a, store: s, http: srv, client: newJarClient(t), mail: box}
	r.claim()
	return r
}

// claim empties the test database and gives it its owner, signed in.
//
// Safe because this is the TEST database, which internal/testdb creates and
// owns. The account is made directly rather than through registration: every
// test needs one, and registration has tests of its own.
func (r *rig) claim() {
	r.t.Helper()
	ctx := context.Background()
	if _, err := r.store.DeleteEveryUser(ctx); err != nil {
		r.t.Fatalf("clear users: %v", err)
	}
	owner, err := r.store.CreateUser(ctx, testEmail, testPassword)
	if err != nil {
		r.t.Fatalf("create the owner: %v", err)
	}
	r.userID = owner.ID
	workspace, err := r.store.CreateWorkspace(ctx, owner.ID, "Personal")
	if err != nil {
		r.t.Fatalf("create the workspace: %v", err)
	}
	r.workspaceID = workspace.ID
	r.signIn()
}

// signIn is an ordinary sign-in as the owner.
func (r *rig) signIn() {
	r.t.Helper()
	res := r.post("/api/auth/login", map[string]any{
		"email": testEmail, "password": testPassword,
	})
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("sign in: %s", res.Status)
	}
	if len(r.client.Jar.Cookies(mustURL(r.t, r.http.URL))) == 0 {
		r.t.Fatal("signing in set no cookie")
	}
}

// stranger is a browser on the same server with no session: somebody arriving
// at the registration page.
func (r *rig) stranger() *rig {
	r.t.Helper()
	return &rig{t: r.t, api: r.api, store: r.store, http: r.http, client: newJarClient(r.t), mail: r.mail}
}

// newJarClient is a fresh browser: its own cookie jar, nothing carried over.
func newJarClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// cookieHeader is what a websocket dial needs, since the gorilla dialler does
// not share the http.Client's jar.
func (r *rig) cookieHeader() http.Header {
	r.t.Helper()
	var pairs []string
	for _, c := range r.client.Jar.Cookies(mustURL(r.t, r.http.URL)) {
		pairs = append(pairs, c.Name+"="+c.Value)
	}
	return http.Header{
		"Cookie": []string{strings.Join(pairs, "; ")},
		"Origin": []string{testOrigin},
	}
}

func (r *rig) ws(path string) string {
	return "ws" + strings.TrimPrefix(r.http.URL, "http") + path
}

// conversation makes one for this rig's account and removes it afterwards.
func (r *rig) conversation(provider string) protocol.Conversation {
	r.t.Helper()
	c, err := r.store.Create(context.Background(), r.userID, r.workspaceID, "D:\\test", provider,
		protocol.Model{ID: "m1", Label: "M One"})
	if err != nil {
		r.t.Fatalf("create conversation: %v", err)
	}
	r.t.Cleanup(func() { _ = r.store.Delete(context.Background(), r.userID, c.ID) })
	return c
}

// createConversation makes one through the API and removes it afterwards.
//
// Rows left behind once white-screened the real sidebar (docs/Bugs.md B-13),
// which is why every conversation a test makes is cleaned up.
func (r *rig) createConversation() protocol.Conversation {
	r.t.Helper()
	res := r.post("/api/conversations", map[string]any{"workspaceId": r.workspaceID})
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("create conversation: %s", res.Status)
	}
	var c protocol.Conversation
	if err := json.NewDecoder(res.Body).Decode(&c); err != nil {
		r.t.Fatal(err)
	}
	r.t.Cleanup(func() { _ = r.store.Delete(context.Background(), r.userID, c.ID) })
	return c
}

func (r *rig) post(path string, body any) *http.Response {
	r.t.Helper()
	return r.do(http.MethodPost, path, body)
}

func (r *rig) get(path string) *http.Response {
	r.t.Helper()
	return r.do(http.MethodGet, path, nil)
}

func (r *rig) do(method, path string, body any) *http.Response {
	r.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			r.t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, r.http.URL+path, &buf)
	if err != nil {
		r.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", testOrigin)
	res, err := r.client.Do(req)
	if err != nil {
		r.t.Fatalf("%s %s: %v", method, path, err)
	}
	r.t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

// decodeInto reads a JSON response body.
func decodeInto(t *testing.T, res *http.Response, into any) {
	t.Helper()
	if err := json.NewDecoder(res.Body).Decode(into); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// entry re-reads one entry from the database, so an assertion is about what was
// actually stored rather than about what a handler returned.
func (r *rig) entry(conversationID, entryID string) protocol.Entry {
	r.t.Helper()
	c, err := r.store.Get(context.Background(), r.userID, conversationID)
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
// a mailbox
// ---------------------------------------------------------------------------

// mailbox is the mail server as far as the tests can see: it keeps every
// message, and can be told to fail.
type mailbox struct {
	mu      sync.Mutex
	failing error
	arrived chan mail.Message
}

func newMailbox() *mailbox {
	return &mailbox{arrived: make(chan mail.Message, 64)}
}

func (b *mailbox) Send(_ context.Context, m mail.Message) error {
	b.mu.Lock()
	failing := b.failing
	b.mu.Unlock()
	if failing != nil {
		return failing
	}
	b.arrived <- m
	return nil
}

func (b *mailbox) fail(err error) {
	b.mu.Lock()
	b.failing = err
	b.mu.Unlock()
}

// await returns the next message to `to` whose subject contains `subject`,
// failing if none arrives. Messages to anyone else are discarded on the way.
func (b *mailbox) await(t *testing.T, to, subject string) mail.Message {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case m := <-b.arrived:
			if m.To == to && strings.Contains(m.Subject, subject) {
				return m
			}
		case <-deadline:
			t.Fatalf("no email to %s about %q arrived", to, subject)
			return mail.Message{}
		}
	}
}

// quiet asserts that nothing arrives for `to` within d.
func (b *mailbox) quiet(t *testing.T, to string, d time.Duration) {
	t.Helper()
	deadline := time.After(d)
	for {
		select {
		case m := <-b.arrived:
			if m.To == to {
				t.Fatalf("an email to %s arrived that should not have: %q", to, m.Subject)
			}
		case <-deadline:
			return
		}
	}
}

var tokenInLink = regexp.MustCompile(`/(verify|reset)\?token=([^\s]+)`)

// linkToken takes the token out of the link in an email body.
func linkToken(t *testing.T, body string) string {
	t.Helper()
	m := tokenInLink.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no link in the email:\n%s", body)
	}
	token, err := url.QueryUnescape(m[2])
	if err != nil {
		t.Fatal(err)
	}
	return token
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
	conn, _, err := websocket.DefaultDialer.Dial(r.ws("/daemon"), http.Header{
		"Authorization": []string{"Bearer " + testDaemonToken},
	})
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
	// The shared-token daemon works for the owner account. Wait for the hub to
	// register it rather than sleeping and hoping.
	deadline := time.Now().Add(3 * time.Second)
	for !r.api.hub.DaemonOnline(r.userID) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !r.api.hub.DaemonOnline(r.userID) {
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
	// gone is closed when the read loop ends, which is how a test learns the
	// SERVER hung up. It has to come from that one goroutine: a websocket.Conn
	// allows a single concurrent reader.
	gone chan struct{}
}

func (r *rig) watch() *browser {
	r.t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(r.ws("/ws"), r.cookieHeader())
	if err != nil {
		r.t.Fatalf("dial /ws: %v", err)
	}
	b := &browser{
		t:      r.t,
		conn:   conn,
		events: make(chan protocol.ClientEvent, 64),
		gone:   make(chan struct{}),
	}
	r.t.Cleanup(func() { _ = conn.Close() })
	go func() {
		defer close(b.gone)
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

// closed waits for the server to hang up, and reports whether it did within d.
func (b *browser) closed(d time.Duration) bool {
	b.t.Helper()
	select {
	case <-b.gone:
		return true
	case <-time.After(d):
		return false
	}
}

// await returns the first event matching want, ignoring the others. A single
// action produces several, and a test should say which one it is about.
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
