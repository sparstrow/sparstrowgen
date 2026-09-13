package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* Account access, tested end to end: an address goes in, an email comes out,
the link in it is spent, and a session results.

Each test states a property the spec or the design depends on, and is written to
fail on the version of the code that lacked it — in particular the two that
matter most to a stranger probing the server: which addresses have accounts, and
whether a link can be used twice. */

type registered struct {
	Outcome string `json:"outcome"`
	Email   string `json:"email"`
}

type linkState struct {
	Usable       bool   `json:"usable"`
	Email        string `json:"email"`
	Reason       string `json:"reason"`
	AccountReady bool   `json:"accountReady"`
}

func (r *rig) checkLink(kind, token string) linkState {
	r.t.Helper()
	res := r.post("/api/auth/links/check", map[string]any{"kind": kind, "token": token})
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("check link: %s", res.Status)
	}
	var s linkState
	decodeInto(r.t, res, &s)
	return s
}

func (r *rig) userCount() int64 {
	r.t.Helper()
	n, err := r.store.UserCount(context.Background())
	if err != nil {
		r.t.Fatal(err)
	}
	return n
}

// The whole journey for an invited person: address, email, link, password,
// signed in — to an account with nothing in it.
func TestAnInvitedPersonCreatesTheirAccountFromTheEmail(t *testing.T) {
	r := newRig(t)
	s := r.stranger()

	res := s.post("/api/auth/register", map[string]any{"email": "  Invited@Sparstrow.TEST "})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("register: %s", res.Status)
	}
	var reg registered
	decodeInto(t, res, &reg)
	if reg.Outcome != "check-email" || reg.Email != testInvited {
		t.Fatalf("register answered %+v, want check-email for the normalised address", reg)
	}

	m := r.mail.await(t, testInvited, "Finish creating")
	token := linkToken(t, m.Body)
	if !strings.HasPrefix(tokenInLink.FindString(m.Body), "/verify?token=") ||
		!strings.Contains(m.Body, testOrigin+"/verify?token=") {
		t.Errorf("the link does not open the web app's /verify page:\n%s", m.Body)
	}

	// The page can say the link works before anybody types a password into it.
	if got := s.checkLink("verify", token); !got.Usable || got.Email != testInvited {
		t.Fatalf("a fresh link checked as %+v", got)
	}

	// A password that is too short is refused WITHOUT spending the link.
	if res := s.post("/api/auth/register/complete", map[string]any{
		"token": token, "password": "short",
	}); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("a short password: %s, want 400", res.Status)
	}
	if !s.checkLink("verify", token).Usable {
		t.Fatal("refusing a short password spent the link")
	}

	res = s.post("/api/auth/register/complete", map[string]any{
		"token": token, "password": "a-long-enough-passphrase",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("complete registration: %s", res.Status)
	}
	if len(s.client.Jar.Cookies(mustURL(t, s.http.URL))) == 0 {
		t.Fatal("creating the account did not sign this browser in")
	}

	// Signed in as the new account, which owns nothing — the owner's work is
	// not part of it.
	r.conversation("claude")
	list := s.get("/api/conversations")
	if list.StatusCode != http.StatusOK {
		t.Fatalf("list as the new account: %s", list.Status)
	}
	var convs []protocol.Conversation
	decodeInto(t, list, &convs)
	if len(convs) != 0 {
		t.Errorf("a brand-new account can see %d conversations", len(convs))
	}

	// Spent. The page says the account is set up, and it cannot be spent again.
	if got := s.checkLink("verify", token); got.Usable || got.Reason != "used" || !got.AccountReady {
		t.Errorf("a used link checked as %+v, want used with the account ready", got)
	}
	if res := r.stranger().post("/api/auth/register/complete", map[string]any{
		"token": token, "password": "another-long-passphrase",
	}); res.StatusCode != http.StatusGone {
		t.Errorf("spending the link twice: %s, want 410", res.Status)
	}
	if n := r.userCount(); n != 2 {
		t.Errorf("%d accounts exist, want the owner and the new one", n)
	}
}

// Registering an address that already has an account must look exactly like
// registering one that does not. The email is where the difference lives.
func TestRegisteringAnExistingAccountRevealsNothing(t *testing.T) {
	r := newRig(t)
	s := r.stranger()

	fresh := s.post("/api/auth/register", map[string]any{"email": testInvited})
	existing := s.post("/api/auth/register", map[string]any{"email": testEmail})
	if fresh.StatusCode != http.StatusOK || existing.StatusCode != http.StatusOK {
		t.Fatalf("register: %s and %s, want both 200", fresh.Status, existing.Status)
	}
	var a, b registered
	decodeInto(t, fresh, &a)
	decodeInto(t, existing, &b)
	if a.Outcome != b.Outcome {
		t.Errorf("a new address answered %q and an existing one %q — that difference is an oracle",
			a.Outcome, b.Outcome)
	}

	r.mail.await(t, testInvited, "Finish creating")
	note := r.mail.await(t, testEmail, "already have")
	if strings.Contains(note.Body, "token=") {
		t.Error("the already-registered email carries a link that could create a second account")
	}
}

// Somebody not on the list is not turned away: they become one request, and the
// owner hears about it once however often they ask.
func TestAnUninvitedAddressBecomesOneRequestToTheOwner(t *testing.T) {
	r := newRig(t)
	s := r.stranger()
	const requester = "jordan.reyes@contoso.test"

	for i := 0; i < 3; i++ {
		res := s.post("/api/auth/register", map[string]any{"email": requester})
		if res.StatusCode != http.StatusOK {
			t.Fatalf("request %d: %s", i, res.Status)
		}
		var reg registered
		decodeInto(t, res, &reg)
		if reg.Outcome != "request-sent" {
			t.Fatalf("request %d answered %q, want request-sent", i, reg.Outcome)
		}
	}

	m := r.mail.await(t, testEmail, "Access request from "+requester)
	if !strings.Contains(m.Body, "ALLOWED_EMAILS") {
		t.Errorf("the owner is not told how to approve it:\n%s", m.Body)
	}
	r.mail.quiet(t, testEmail, 300*time.Millisecond)
	r.mail.quiet(t, requester, 100*time.Millisecond)

	// Resending for an uninvited address does nothing, and says it did.
	if res := s.post("/api/auth/register/resend", map[string]any{"email": requester}); res.StatusCode != http.StatusOK {
		t.Errorf("resend for a request: %s, want 200", res.Status)
	}
	r.mail.quiet(t, requester, 200*time.Millisecond)

	if n := r.userCount(); n != 1 {
		t.Errorf("a request created an account: %d exist", n)
	}
}

// Only the newest link for an address works, so an old email sitting in an
// inbox cannot be used after a new one was asked for.
func TestOnlyTheNewestLinkWorks(t *testing.T) {
	r := newRig(t)
	s := r.stranger()

	s.post("/api/auth/register", map[string]any{"email": testInvited})
	older := linkToken(t, r.mail.await(t, testInvited, "Finish creating").Body)
	if res := s.post("/api/auth/register/resend", map[string]any{"email": testInvited}); res.StatusCode != http.StatusOK {
		t.Fatalf("resend: %s", res.Status)
	}
	newer := linkToken(t, r.mail.await(t, testInvited, "Finish creating").Body)

	if got := s.checkLink("verify", older); got.Usable || got.Reason != "superseded" {
		t.Errorf("the older link checked as %+v, want superseded", got)
	}
	if res := s.post("/api/auth/register/complete", map[string]any{
		"token": older, "password": "a-long-enough-passphrase",
	}); res.StatusCode != http.StatusGone {
		t.Errorf("the older link: %s, want 410", res.Status)
	}
	if res := s.post("/api/auth/register/complete", map[string]any{
		"token": newer, "password": "a-long-enough-passphrase",
	}); res.StatusCode != http.StatusOK {
		t.Errorf("the newest link: %s, want 200", res.Status)
	}
}

// Emails to one address are spaced out on the server. The browser's countdown
// only saves a pointless click; this is what stops an inbox being flooded.
func TestEmailsToOneAddressAreSpacedOut(t *testing.T) {
	r := newRig(t)
	r.api.sendGap = time.Hour
	s := r.stranger()

	if res := s.post("/api/auth/register", map[string]any{"email": testInvited}); res.StatusCode != http.StatusOK {
		t.Fatalf("first: %s", res.Status)
	}
	for _, path := range []string{"/api/auth/register", "/api/auth/register/resend"} {
		if res := s.post(path, map[string]any{"email": testInvited}); res.StatusCode != http.StatusTooManyRequests {
			t.Errorf("%s straight after: %s, want 429", path, res.Status)
		}
	}
	// A different address is its own allowance.
	if res := s.post("/api/auth/register", map[string]any{"email": testSecond}); res.StatusCode != http.StatusOK {
		t.Errorf("another address: %s, want 200", res.Status)
	}
}

// A mail server that is down is reported, and does not use up the spacing:
// the person can try again as soon as it is back.
func TestAMailFailureIsReportedAndCanBeRetried(t *testing.T) {
	r := newRig(t)
	r.api.sendGap = time.Hour
	s := r.stranger()

	r.mail.fail(errors.New("connection refused"))
	res := s.post("/api/auth/register", map[string]any{"email": testInvited})
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("with the mail server down: %s, want 502", res.Status)
	}
	var body map[string]string
	decodeInto(t, res, &body)
	if !strings.Contains(body["error"], "could not be sent") {
		t.Errorf("error = %q, want it to say the email could not be sent", body["error"])
	}

	r.mail.fail(nil)
	if res := s.post("/api/auth/register", map[string]any{"email": testInvited}); res.StatusCode != http.StatusOK {
		t.Errorf("retry once it is back: %s, want 200", res.Status)
	}
}

// An expired link says so, and creates nothing.
func TestAnExpiredLinkCannotCreateAnAccount(t *testing.T) {
	r := newRig(t)
	s := r.stranger()
	token, err := r.store.IssueLink(context.Background(), store.LinkVerify, testInvited, -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.checkLink("verify", token); got.Usable || got.Reason != "expired" {
		t.Errorf("an expired link checked as %+v", got)
	}
	if res := s.post("/api/auth/register/complete", map[string]any{
		"token": token, "password": "a-long-enough-passphrase",
	}); res.StatusCode != http.StatusGone {
		t.Errorf("an expired link: %s, want 410", res.Status)
	}
	// A link that never existed looks the same as an expired one.
	if got := s.checkLink("verify", "not-a-real-token"); got.Usable || got.Reason != "expired" {
		t.Errorf("an invented token checked as %+v", got)
	}
	if n := r.userCount(); n != 1 {
		t.Errorf("%d accounts exist, want only the owner", n)
	}
}

// Taking an address off the list stops its outstanding link from creating an
// account. The link proves the address, not that it is still welcome.
func TestALinkDoesNotOutliveItsInvitation(t *testing.T) {
	r := newRig(t)
	token, err := r.store.IssueLink(context.Background(), store.LinkVerify, "removed@sparstrow.test", linkLifetime)
	if err != nil {
		t.Fatal(err)
	}
	if res := r.stranger().post("/api/auth/register/complete", map[string]any{
		"token": token, "password": "a-long-enough-passphrase",
	}); res.StatusCode != http.StatusForbidden {
		t.Errorf("a link for an address no longer invited: %s, want 403", res.Status)
	}
	if n := r.userCount(); n != 1 {
		t.Errorf("%d accounts exist, want only the owner", n)
	}
}

// A forgotten password is reset through the registered address: every other
// device is signed out, the old password stops working, and the link is spent.
func TestResettingAForgottenPassword(t *testing.T) {
	r := newRig(t)
	otherDevice := newClientOn(t, r)
	otherDevice.createConversation() // its session works
	socket := r.watch()
	socket.await("greeting", func(protocol.ClientEvent) bool { return true })
	s := r.stranger()

	known := s.post("/api/auth/password/forgot", map[string]any{"email": testEmail})
	unknown := s.post("/api/auth/password/forgot", map[string]any{"email": "nobody@sparstrow.test"})
	if known.StatusCode != http.StatusOK || unknown.StatusCode != http.StatusOK {
		t.Fatalf("forgot: %s and %s, want both 200", known.Status, unknown.Status)
	}
	var a, b map[string]any
	decodeInto(t, known, &a)
	decodeInto(t, unknown, &b)
	if len(a) != len(b) || a["ok"] != b["ok"] {
		t.Errorf("an account answered %v and no account answered %v — that difference is an oracle", a, b)
	}

	token := linkToken(t, r.mail.await(t, testEmail, "Reset your").Body)
	r.mail.quiet(t, "nobody@sparstrow.test", 300*time.Millisecond)

	if res := s.post("/api/auth/password/reset", map[string]any{
		"token": token, "password": "short",
	}); res.StatusCode != http.StatusBadRequest {
		t.Errorf("a short new password: %s, want 400", res.Status)
	}

	const next = "a-brand-new-passphrase"
	res := s.post("/api/auth/password/reset", map[string]any{"token": token, "password": next})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reset: %s", res.Status)
	}
	if len(s.client.Jar.Cookies(mustURL(t, s.http.URL))) == 0 {
		t.Error("the browser that reset the password was not signed in")
	}

	if res := otherDevice.post("/api/conversations", nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("another device still worked after the reset: %s", res.Status)
	}
	if !socket.closed(3 * time.Second) {
		t.Error("a socket opened before the reset is still connected")
	}
	r.mail.await(t, testEmail, "was changed")

	old := r.stranger().post("/api/auth/login", map[string]any{"email": testEmail, "password": testPassword})
	if old.StatusCode != http.StatusUnauthorized {
		t.Errorf("the old password still signs in: %s", old.Status)
	}
	fresh := r.stranger().post("/api/auth/login", map[string]any{"email": testEmail, "password": next})
	if fresh.StatusCode != http.StatusOK {
		t.Errorf("the new password does not sign in: %s", fresh.Status)
	}
	if res := r.stranger().post("/api/auth/password/reset", map[string]any{
		"token": token, "password": "yet-another-passphrase",
	}); res.StatusCode != http.StatusGone {
		t.Errorf("spending the reset link twice: %s, want 410", res.Status)
	}
}

// A link token is only ever stored hashed, like a session token.
func TestLinkTokensAreStoredHashed(t *testing.T) {
	r := newRig(t)
	token, err := r.store.IssueLink(context.Background(), store.LinkVerify, testInvited, linkLifetime)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.stranger().checkLink("verify", string(auth.HashToken(token))); got.Usable {
		t.Error("presenting the stored hash worked as though it were the token")
	}
}

// ---------------------------------------------------------------------------
// sockets outliving their sessions (D-030)
// ---------------------------------------------------------------------------

// Changing a password ends every session. A socket, though, is authenticated
// once at the handshake and then lives as long as the tab — so the rows went
// and the connection stayed, still receiving every conversation event.
func TestChangingThePasswordClosesOpenSockets(t *testing.T) {
	r := newRig(t)
	b := r.watch()
	// The socket is live: the hub greets a new client with the daemon state.
	b.await("greeting", func(ev protocol.ClientEvent) bool { return true })

	res := r.post("/api/auth/password", map[string]any{
		"current": testPassword, "next": "a-completely-different-passphrase",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("change password: %d", res.StatusCode)
	}

	if !b.closed(3 * time.Second) {
		t.Error("the socket opened under the old password is still connected")
	}
}

// The same for the panic button itself.
func TestSigningOutEverywhereClosesOpenSockets(t *testing.T) {
	r := newRig(t)
	b := r.watch()
	b.await("greeting", func(ev protocol.ClientEvent) bool { return true })

	res := r.post("/api/auth/logout", map[string]any{"everywhere": true})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("sign out everywhere: %d", res.StatusCode)
	}

	if !b.closed(3 * time.Second) {
		t.Error("a revoked session's socket is still connected")
	}
}
