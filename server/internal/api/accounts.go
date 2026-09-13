package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/mail"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* Account access (docs/specs/2026-09-12-first-usable-release.md, US1).

Invited people create their own accounts: an address, a link to prove it, then a
password. Anybody else who tries becomes a request the owner is emailed about.
A forgotten password is reset through a link to the registered address.

Two properties run through every handler here, and the tests hold them:

  - A response never reveals whether an address has an account. Registering an
    address that already has one sends an email saying so, and answers exactly
    like a new registration; asking for a reset answers identically either way.
  - A link is spent once, and only the newest one for an address works. Both are
    enforced in SQL (queries/accounts.sql), not here. */

const (
	// linkLifetime is how long an emailed link works.
	linkLifetime = 30 * time.Minute
	// defaultSendGap is the least time between two emails of the same kind to
	// the same address. The countdown in the browser only mirrors it.
	defaultSendGap = 30 * time.Second
	// mailTimeout bounds a conversation with the mail server.
	mailTimeout = 20 * time.Second
)

var (
	errMailFailed   = errors.New("the email could not be sent just now — try again in a few minutes")
	errNotInvited   = errors.New("this address is no longer invited — ask the person who invited you")
	errSendTooSoon  = errors.New("wait a moment before sending another email to that address")
	errUnknownKind  = errors.New("that is not a kind of link this app sends")
	errUnreadable   = errors.New("that request could not be read")
	errBadLinkInput = errors.New("that link is missing its token")
)

func (a *API) isInvited(email string) bool {
	return a.invited[store.NormaliseEmail(email)]
}

// decode reads a small JSON body, bounded before decoding.
func decode(w http.ResponseWriter, r *http.Request, into any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	return json.NewDecoder(r.Body).Decode(into)
}

// ---------------------------------------------------------------------------
// registering
// ---------------------------------------------------------------------------

// register takes an address. What happens next is the server's decision, and
// the answer only reports it:
//
//	invited, no account   a confirmation link is emailed     → check-email
//	invited, has account  "you already have one" is emailed   → check-email
//	not invited           the request is recorded             → request-sent
//
// The first two answer identically on purpose.
func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decode(w, r, &body); err != nil {
		a.fail(w, errUnreadable, http.StatusBadRequest)
		return
	}
	if err := checkEmail(body.Email); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	email := store.NormaliseEmail(body.Email)

	if !a.isInvited(email) {
		a.requestAccess(w, r, email)
		return
	}
	if !a.sendConfirmation(w, r, email) {
		return
	}
	writeJSON(w, map[string]any{"outcome": "check-email", "email": email})
}

// requestAccess records an uninvited address and tells the owner — once per
// address, however often it asks.
func (a *API) requestAccess(w http.ResponseWriter, r *http.Request, email string) {
	// Counted per client, separately from sign-in, so flooding requests slows
	// the flooder and never the owner's own sign-in.
	if wait, admitted := a.requests.Begin(clientIP(r)); !admitted {
		tooMany(w, wait, a)
		return
	}
	first, err := a.store.RecordAccessRequest(r.Context(), email)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	if first {
		a.log.Info("access requested", "email", email)
		// The request is recorded whether or not this email arrives, and the
		// person asking has done everything they can. A failure is the owner's
		// problem to see in the log, not theirs to be shown.
		a.sendLater(requestEmail(a.cfg.OwnerEmail, email))
	}
	writeJSON(w, map[string]any{"outcome": "request-sent", "email": email})
}

// resendConfirmation sends the confirmation again. For an address that is not
// invited it does nothing and says it did: the page already said the request
// was sent, and this must not become a way to test the list.
func (a *API) resendConfirmation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decode(w, r, &body); err != nil {
		a.fail(w, errUnreadable, http.StatusBadRequest)
		return
	}
	if err := checkEmail(body.Email); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	email := store.NormaliseEmail(body.Email)
	if a.isInvited(email) && !a.sendConfirmation(w, r, email) {
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// sendConfirmation emails an invited address either its link or the note that
// it already has an account, and reports whether the handler may answer
// success. It writes the failure response itself when it may not.
//
// Sent synchronously, both branches: a send that fails is something the person
// can retry, and doing the same work either way keeps the two indistinguishable
// by timing as well as by response.
func (a *API) sendConfirmation(w http.ResponseWriter, r *http.Request, email string) bool {
	key := "confirm:" + email
	if wait := a.mailGate.wait(key, a.sendGap, time.Now()); wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		a.fail(w, errSendTooSoon, http.StatusTooManyRequests)
		return false
	}

	_, exists, err := a.store.UserByEmail(r.Context(), email)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return false
	}
	var message mail.Message
	if exists {
		message = alreadyRegisteredEmail(email, a.cfg.Origin)
	} else {
		token, err := a.store.IssueLink(r.Context(), store.LinkVerify, email, linkLifetime)
		if err != nil {
			a.fail(w, err, http.StatusInternalServerError)
			return false
		}
		message = confirmEmail(email, linkURL(a.cfg.Origin, "verify", token))
	}

	ctx, cancel := context.WithTimeout(r.Context(), mailTimeout)
	defer cancel()
	if err := a.cfg.Mailer.Send(ctx, message); err != nil {
		a.log.Error("could not send a confirmation email", "err", err)
		a.fail(w, errMailFailed, http.StatusBadGateway)
		return false
	}
	a.mailGate.mark(key, time.Now())
	return true
}

// ---------------------------------------------------------------------------
// links
// ---------------------------------------------------------------------------

// checkLink says what an emailed link is still good for, without spending it,
// so its page can explain a dead link before anybody types a password into it.
func (a *API) checkLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind  string `json:"kind"`
		Token string `json:"token"`
	}
	if err := decode(w, r, &body); err != nil {
		a.fail(w, errUnreadable, http.StatusBadRequest)
		return
	}
	kind, ok := linkKind(body.Kind)
	if !ok {
		a.fail(w, errUnknownKind, http.StatusBadRequest)
		return
	}
	state, err := a.store.InspectLink(r.Context(), kind, body.Token)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	if state.Usable {
		writeJSON(w, map[string]any{"usable": true, "email": state.Email})
		return
	}
	writeJSON(w, map[string]any{
		"usable": false, "reason": state.Reason, "accountReady": state.AccountReady,
	})
}

// completeRegistration spends a confirmation link and creates the account with
// the password chosen now, then signs this browser in.
func (a *API) completeRegistration(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &body); err != nil {
		a.fail(w, errUnreadable, http.StatusBadRequest)
		return
	}
	if body.Token == "" {
		a.fail(w, errBadLinkInput, http.StatusBadRequest)
		return
	}
	if len(body.Password) < minPassword {
		a.fail(w, errPasswordShort, http.StatusBadRequest)
		return
	}

	// An address can be taken off the list after its link was sent. The link
	// proves the address; it does not outlive the invitation.
	state, err := a.store.InspectLink(r.Context(), store.LinkVerify, body.Token)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	if state.Usable && !a.isInvited(state.Email) {
		a.fail(w, errNotInvited, http.StatusForbidden)
		return
	}

	release, ok := a.acquireHash(w)
	if !ok {
		return
	}
	defer release()

	user, token, expires, err := a.store.CompleteRegistration(
		r.Context(), body.Token, body.Password, r.UserAgent(), clientIP(r))
	switch {
	case errors.Is(err, store.ErrLinkUnusable):
		a.fail(w, err, http.StatusGone)
		return
	case errors.Is(err, store.ErrEmailTaken):
		a.fail(w, err, http.StatusConflict)
		return
	case err != nil:
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.log.Info("account created", "email", user.Email)
	a.setSession(w, user, token, expires)
}

// ---------------------------------------------------------------------------
// forgotten passwords
// ---------------------------------------------------------------------------

// forgotPassword takes an address and answers the same way whether or not it
// has an account. The email, when there is one to send, goes in the background:
// sending only for a real account would otherwise make this endpoint's response
// time the oracle its response text refuses to be.
func (a *API) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decode(w, r, &body); err != nil {
		a.fail(w, errUnreadable, http.StatusBadRequest)
		return
	}
	if err := checkEmail(body.Email); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	email := store.NormaliseEmail(body.Email)

	// Spaced per address typed, whether or not it has an account, so the wait
	// itself reveals nothing either.
	key := "reset:" + email
	if wait := a.mailGate.wait(key, a.sendGap, time.Now()); wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		a.fail(w, errSendTooSoon, http.StatusTooManyRequests)
		return
	}
	a.mailGate.mark(key, time.Now())

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), mailTimeout)
		defer cancel()
		_, exists, err := a.store.UserByEmail(ctx, email)
		if err != nil {
			a.log.Error("could not look up an address for a reset", "err", err)
			return
		}
		if !exists {
			return
		}
		token, err := a.store.IssueLink(ctx, store.LinkReset, email, linkLifetime)
		if err != nil {
			a.log.Error("could not issue a reset link", "err", err)
			return
		}
		if err := a.cfg.Mailer.Send(ctx, resetEmail(email, linkURL(a.cfg.Origin, "reset", token))); err != nil {
			a.log.Error("could not send a reset email", "err", err)
		}
	}()

	writeJSON(w, map[string]any{"ok": true, "email": email})
}

// resetPassword spends a reset link, sets the new password, signs out every
// other browser, and signs this one in.
func (a *API) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &body); err != nil {
		a.fail(w, errUnreadable, http.StatusBadRequest)
		return
	}
	if body.Token == "" {
		a.fail(w, errBadLinkInput, http.StatusBadRequest)
		return
	}
	if len(body.Password) < minPassword {
		a.fail(w, errPasswordShort, http.StatusBadRequest)
		return
	}

	release, ok := a.acquireHash(w)
	if !ok {
		return
	}
	defer release()

	user, token, expires, err := a.store.ResetPassword(
		r.Context(), body.Token, body.Password, r.UserAgent(), clientIP(r))
	if errors.Is(err, store.ErrLinkUnusable) {
		a.fail(w, err, http.StatusGone)
		return
	}
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}

	// Every other session was deleted in the same transaction; their sockets
	// are still open until closed here (D-030). The new session's socket does
	// not exist yet, so closing all of this account's is exactly right.
	a.hub.DisconnectUser(user.ID)
	a.log.Info("password reset", "email", user.Email)
	a.sendLater(passwordChangedEmail(user.Email, a.cfg.Origin))
	a.setSession(w, user, token, expires)
}

// sendLater sends an email that nothing waits on, and logs a failure. Only for
// messages whose loss the person can recover from without being told.
func (a *API) sendLater(m mail.Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), mailTimeout)
		defer cancel()
		if err := a.cfg.Mailer.Send(ctx, m); err != nil {
			a.log.Error("could not send an email", "subject", m.Subject, "err", err)
		}
	}()
}

// ---------------------------------------------------------------------------
// spacing out emails
// ---------------------------------------------------------------------------

// sendGate remembers when each kind of email last went to each address.
//
// In memory, like the sign-in throttle: a restart forgetting it lets one extra
// email through, which is harmless, and a table would turn every send into a
// write for no real protection.
type sendGate struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func newSendGate() *sendGate {
	return &sendGate{last: map[string]time.Time{}}
}

// wait reports how long until key may be sent to again.
func (g *sendGate) wait(key string, gap time.Duration, now time.Time) time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	if last, ok := g.last[key]; ok {
		if elapsed := now.Sub(last); elapsed < gap {
			return gap - elapsed
		}
	}
	return 0
}

func (g *sendGate) mark(key string, now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.last[key] = now
	if len(g.last) > 4096 {
		for k, t := range g.last {
			if now.Sub(t) > time.Hour {
				delete(g.last, k)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// the emails
// ---------------------------------------------------------------------------

func linkKind(s string) (store.LinkKind, bool) {
	switch store.LinkKind(s) {
	case store.LinkVerify:
		return store.LinkVerify, true
	case store.LinkReset:
		return store.LinkReset, true
	default:
		return "", false
	}
}

// linkURL is the page a link opens: /verify or /reset on the web app. The token
// is the only thing in the query string.
func linkURL(origin, page, token string) string {
	return strings.TrimRight(origin, "/") + "/" + page + "?token=" + url.QueryEscape(token)
}

func confirmEmail(to, link string) mail.Message {
	return mail.Message{
		To:      to,
		Subject: "Finish creating your sparstrowgen account",
		Body: fmt.Sprintf(`Someone, hopefully you, started creating a sparstrowgen account for %s.

Finish creating your account:
%s

This link works once, for the next 30 minutes. If this wasn't you, ignore this email and nothing will be created.
`, to, link),
	}
}

func alreadyRegisteredEmail(to, origin string) mail.Message {
	base := strings.TrimRight(origin, "/")
	return mail.Message{
		To:      to,
		Subject: "You already have a sparstrowgen account",
		Body: fmt.Sprintf(`Someone tried to create an account for %s, but that address already has one.

Sign in:
%s/

Forgotten your password? Reset it here:
%s/forgot

If this wasn't you, you don't need to do anything.
`, to, base, base),
	}
}

func resetEmail(to, link string) mail.Message {
	return mail.Message{
		To:      to,
		Subject: "Reset your sparstrowgen password",
		Body: fmt.Sprintf(`Someone asked to reset the password for %s.

Choose a new password:
%s

This link works once, for the next 30 minutes. Changing your password signs out every other browser signed in to this account.

If this wasn't you, ignore this email and your password stays as it is.
`, to, link),
	}
}

func passwordChangedEmail(to, origin string) mail.Message {
	return mail.Message{
		To:      to,
		Subject: "Your sparstrowgen password was changed",
		Body: fmt.Sprintf(`The password for %s was just changed, and every other browser signed in to this account was signed out.

If this wasn't you, reset your password now:
%s/forgot
`, to, strings.TrimRight(origin, "/")),
	}
}

func requestEmail(owner, requester string) mail.Message {
	return mail.Message{
		To:      owner,
		Subject: "Access request from " + requester,
		Body: fmt.Sprintf(`%s asked to use sparstrowgen. That address has not been invited, so no account was created.

To approve, add the address to ALLOWED_EMAILS in the server's configuration and restart it. They can then create their account with it.

Further requests from the same address will not be emailed again.
`, requester),
	}
}
