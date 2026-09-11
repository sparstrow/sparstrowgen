package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
)

/* The front door, tested as an attacker would.

Everything behind these routes eventually runs a coding agent on the owner's
machine, so "it seems to work when I'm logged in" is not the question. The
question is whether it works when you are NOT, and every test here is written to
fail if the answer ever becomes yes. */

// anon is a client with no cookie and no token: a stranger who found the URL.
func (r *rig) anon(method, path string, body any) *http.Response {
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
	res, err := (&http.Client{}).Do(req) // no jar: nothing is carried
	if err != nil {
		r.t.Fatalf("%s %s: %v", method, path, err)
	}
	r.t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

// The whole point, stated once: no route that touches a conversation, a folder
// or a turn may be reachable without signing in. Listed one by one rather than
// derived from the router, so that adding a route to the protected group is a
// deliberate act and adding one OUTSIDE it fails this test.
func TestNothingIsReachableWithoutSigningIn(t *testing.T) {
	r := newRig(t)
	c := r.conversation("claude")

	protected := []struct{ method, path string }{
		{http.MethodGet, "/api/providers"},
		{http.MethodGet, "/api/conversations"},
		{http.MethodPost, "/api/conversations"},
		{http.MethodGet, "/api/conversations/" + c.ID},
		{http.MethodPatch, "/api/conversations/" + c.ID},
		{http.MethodDelete, "/api/conversations/" + c.ID},
		{http.MethodGet, "/api/conversations/" + c.ID + "/switch-cost"},
		{http.MethodGet, "/api/directories?path=D:%5C"},
		{http.MethodGet, "/api/folders/recent"},
		{http.MethodPost, "/api/conversations/" + c.ID + "/messages"},
		{http.MethodPost, "/api/turns/whatever/stop"},
	}
	for _, route := range protected {
		res := r.anon(route.method, route.path, map[string]any{"text": "hello"})
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s answered %s to a stranger, want 401",
				route.method, route.path, res.Status)
		}
	}
}

// Liveness has to be reachable — a deployment checks it — but it must not
// report on the owner's machine. "Is he at his desk" is not a question an
// unauthenticated endpoint should answer.
func TestHealthIsPublicButSaysNothingAboutTheMachine(t *testing.T) {
	r := newRig(t)
	res := r.anon(http.MethodGet, "/api/health", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health: %s, want it reachable", res.Status)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, leaked := body["daemon"]; leaked {
		t.Error("health told an anonymous caller whether the owner's machine is connected")
	}
}

func TestTheWrongPasswordDoesNotGetIn(t *testing.T) {
	r := newRig(t)

	res := r.anon(http.MethodPost, "/api/auth/login", map[string]any{"password": "not it"})
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password: %s, want 401", res.Status)
	}
	for _, c := range res.Cookies() {
		if c.Name == auth.CookieName(false) && c.Value != "" {
			t.Fatal("a failed sign-in handed out a session cookie")
		}
	}

	// And an empty password is refused as a bad request rather than being
	// hashed and compared, which would make it just another guess.
	if res := r.anon(http.MethodPost, "/api/auth/login", map[string]any{"password": ""}); res.StatusCode != http.StatusBadRequest {
		t.Errorf("empty password: %s, want 400", res.Status)
	}
}

// The property that DB-backed sessions exist for: a session can be taken away,
// and it stops working at once rather than when it happens to expire. This is
// what a signed token cannot do, and it is the reason for the extra table.
func TestSigningOutStopsTheSessionImmediately(t *testing.T) {
	r := newRig(t)

	// Signed in by the harness, so this works.
	if res := r.post("/api/conversations", nil); res.StatusCode != http.StatusOK {
		t.Fatalf("while signed in: %s", res.Status)
	}
	if res := r.post("/api/auth/logout", nil); res.StatusCode != http.StatusOK {
		t.Fatalf("logout: %s", res.Status)
	}
	// Same client, same cookie jar — the cookie is simply no longer good.
	if res := r.post("/api/conversations", nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("after signing out: %s, want 401", res.Status)
	}
}

// "Sign out everywhere", which is the thing the owner would want in a hurry.
func TestSigningOutEverywhereEndsOtherDevicesToo(t *testing.T) {
	r := newRig(t)

	// A second device: its own jar, its own sign-in, same password.
	other := newClientOn(t, r)
	if res := other.post("/api/conversations", nil); res.StatusCode != http.StatusOK {
		t.Fatalf("the second device could not use its own session: %s", res.Status)
	}

	if res := r.post("/api/auth/logout", map[string]any{"everywhere": true}); res.StatusCode != http.StatusOK {
		t.Fatalf("logout everywhere: %s", res.Status)
	}
	if res := other.post("/api/conversations", nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("the other device still worked after signing out everywhere: %s", res.Status)
	}
}

// A token that was never issued must not work, however well-formed it looks.
func TestAnInventedSessionTokenIsNotASession(t *testing.T) {
	r := newRig(t)

	token, _, err := auth.NewToken() // correctly shaped, simply never stored
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodGet, r.http.URL+"/api/conversations", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName(false), Value: token})
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("a token nobody issued answered %s, want 401", res.Status)
	}
}

// ---------------------------------------------------------------------------
// the two sockets
// ---------------------------------------------------------------------------

// The daemon socket is the other way in, and it is the more dangerous one: what
// connects here is handed turns to run. It must refuse anything without the
// shared secret, and it must refuse BEFORE upgrading, so nothing is registered
// in the hub on the strength of a connection about to be rejected.
func TestTheDaemonSocketRefusesAnythingWithoutTheToken(t *testing.T) {
	r := newRig(t)

	for _, header := range []http.Header{
		nil,
		{"Authorization": []string{"Bearer wrong-token-entirely"}},
		{"Authorization": []string{testDaemonToken}},                                      // no Bearer prefix
		{"Authorization": []string{"Bearer " + testDaemonToken[:len(testDaemonToken)-1]}}, // one char short
	} {
		conn, res, err := websocket.DefaultDialer.Dial(r.ws("/daemon"), header)
		if err == nil {
			conn.Close()
			t.Fatalf("a daemon connection was accepted with %v", header)
		}
		if res == nil || res.StatusCode != http.StatusUnauthorized {
			t.Errorf("header %v was refused with %v, want a 401", header, res)
		}
	}
	if r.api.hub.DaemonOnline() {
		t.Error("a refused connection still registered as the owner's machine")
	}
}

// A websocket is not covered by the same-origin policy, so without an explicit
// check any page the owner visited could open one, be authenticated by his own
// cookie, and read every conversation. The cookie here is genuine — the origin
// is what must be refused.
func TestTheBrowserSocketRefusesAForeignOrigin(t *testing.T) {
	r := newRig(t)

	header := r.cookieHeader()
	header.Set("Origin", "https://somewhere-else.example")
	conn, res, err := websocket.DefaultDialer.Dial(r.ws("/ws"), header)
	if err == nil {
		conn.Close()
		t.Fatal("a socket from a foreign origin was accepted, carrying the owner's own cookie")
	}
	if res == nil || res.StatusCode != http.StatusForbidden {
		t.Errorf("refused with %v, want 403", res)
	}

	// And the configured origin still works, or the check is useless in the
	// other direction.
	good, _, err := websocket.DefaultDialer.Dial(r.ws("/ws"), r.cookieHeader())
	if err != nil {
		t.Fatalf("the real origin was refused: %v", err)
	}
	good.Close()
}

// The browser socket carries conversations, so it needs the cookie too — an
// origin check alone would let anything non-browser listen in.
func TestTheBrowserSocketNeedsASession(t *testing.T) {
	r := newRig(t)

	conn, res, err := websocket.DefaultDialer.Dial(r.ws("/ws"), http.Header{
		"Origin": []string{testOrigin},
	})
	if err == nil {
		conn.Close()
		t.Fatal("a socket with no session was accepted")
	}
	if res == nil || res.StatusCode != http.StatusUnauthorized {
		t.Errorf("refused with %v, want 401", res)
	}
}

// ---------------------------------------------------------------------------
// CORS
// ---------------------------------------------------------------------------

// "*" cannot be combined with credentials, and more to the point the list of
// origins allowed to act as the owner should be one, named in the deployment.
func TestCORSAnswersOnlyTheConfiguredOrigin(t *testing.T) {
	r := newRig(t)

	req, err := http.NewRequest(http.MethodOptions, r.http.URL+"/api/conversations", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://not-the-app.example")
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("a foreign origin was allowed: %q", got)
	}

	req.Header.Set("Origin", testOrigin)
	res2, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if got := res2.Header.Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Errorf("the configured origin was answered %q, want %q", got, testOrigin)
	}
	if got := res2.Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Error("credentials are not allowed, so the browser will not send the session cookie")
	}
	if !strings.Contains(res2.Header.Get("Vary"), "Origin") {
		t.Error("no Vary: Origin — a cache could hand one origin's permission to another")
	}
}

// ---------------------------------------------------------------------------

// newClientOn is a second signed-in client against the same server, standing in
// for another device.
func newClientOn(t *testing.T, r *rig) *rig {
	t.Helper()
	second := &rig{t: t, api: r.api, store: r.store, http: r.http, client: newJarClient(t)}
	second.signIn()
	return second
}
