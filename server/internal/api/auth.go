package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* The front door.

Everything below this file eventually causes a coding agent to run on somebody's
machine, in a folder the request names, with whatever tools that agent can
reach. So the rule here is not the usual web-app rule. It is: nothing gets
through that has not proved which account it is, and the small number of
exceptions are written out one by one rather than being whatever happens to be
outside a route group.

Two credentials, and only two:

  a session cookie   a person, after an email and a password
  the daemon token   the owner's machine, dialling in, until computers are paired

Reachable without either, and each earns it:

  /api/health                   liveness only, so a deploy can check it
  /api/auth/session             "are YOU signed in", and as whom
  /api/auth/login               the door itself
  /api/auth/logout              must work with a session that is already dead
  /api/auth/register            takes an address; an invited one gets an email,
                                any other becomes a request to the owner
  /api/auth/register/resend     the same email again, spaced out per address
  /api/auth/register/complete   spends a confirmation link; the link is the proof
  /api/auth/links/check         what a link is still good for, without spending it
  /api/auth/password/forgot     takes an address; answers identically either way
  /api/auth/password/reset      spends a reset link

There is no configuration that removes the two credentials. The server refuses
to start without its secrets and mail settings (cmd/server/main.go), which means
"deployed with authentication accidentally switched off" is not a state this
program can be in. */

var (
	errNotLoggedIn   = errors.New("not signed in")
	errNoEmail       = errors.New("that does not look like an email address")
	errPasswordShort = errors.New("a password needs to be at least 12 characters")
)

// minPassword is a floor on length and nothing else.
//
// No character-class rule: "one capital, one digit, one symbol" mostly produces
// passwords people cannot remember and therefore reuse, and length is what
// actually resists guessing. Twelve because this one credential stands in front
// of a shell on somebody's laptop.
const minPassword = 12

// requireSession rejects anything without a live session cookie.
func (a *API) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.CookieName(a.cfg.SecureCookie))
		if err != nil {
			a.fail(w, errNotLoggedIn, http.StatusUnauthorized)
			return
		}
		user, ok, err := a.store.SessionUser(r.Context(), cookie.Value)
		if err != nil {
			// A database that cannot answer is not permission to proceed.
			a.log.Error("could not check a session", "err", err)
			a.fail(w, errors.New("could not verify your session"), http.StatusServiceUnavailable)
			return
		}
		if !ok {
			// Clear the cookie on the way out. Otherwise the browser keeps
			// presenting a token that will never work again, and every request
			// pays a database lookup to be told so.
			http.SetCookie(w, auth.ClearCookie(a.cfg.SecureCookie))
			a.fail(w, errNotLoggedIn, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user, cookie.Value)))
	})
}

// maxLoginBody bounds a sign-in request. A password is a password; a megabyte
// of JSON is somebody making the server allocate. The limit is applied BEFORE
// decoding, because a check afterwards has already paid for the allocation it
// was meant to prevent.
const maxLoginBody = 4 << 10

// hashSlots caps how many argon2 hashes run at once.
//
// Each one is 19 MiB of working memory by design — that is what makes it a good
// password hash — which also makes it a good way to exhaust a small VPS. Anything
// beyond it is turned away immediately rather than queued, since a queue is just
// the same exhaustion with extra steps.
const hashSlots = 4

// acquireHash takes a hashing slot, or answers "busy" and reports false.
func (a *API) acquireHash(w http.ResponseWriter) (release func(), ok bool) {
	select {
	case a.hashing <- struct{}{}:
		return func() { <-a.hashing }, true
	default:
		w.Header().Set("Retry-After", "1")
		a.fail(w, errors.New("the server is busy — try again in a moment"), http.StatusServiceUnavailable)
		return nil, false
	}
}

// ---------------------------------------------------------------------------
// signing in and out
// ---------------------------------------------------------------------------

// login exchanges an email and password for a session.
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	client := clientIP(r)

	// Admission and the record of the attempt happen under one lock. Asking
	// "may I?" and only counting the attempt after the hash leaves the whole
	// hash sitting in the gap (docs/Bugs.md B-12).
	wait, admitted := a.throttle.Begin(client)
	if !admitted {
		tooMany(w, wait, a)
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, errors.New("that request could not be read"), http.StatusBadRequest)
		return
	}
	if body.Email == "" || body.Password == "" {
		a.fail(w, store.ErrNoAccount, http.StatusUnauthorized)
		return
	}

	release, ok := a.acquireHash(w)
	if !ok {
		return
	}
	defer release()

	// Checking the password and starting the session are one transaction in the
	// store, not two calls from here (see store.SignIn).
	user, token, expires, err := a.store.SignIn(
		r.Context(), body.Email, body.Password, r.UserAgent(), client)
	if errors.Is(err, store.ErrNoAccount) {
		a.throttle.Failed(client)
		a.log.Warn("failed sign-in", "ip", client)
		// One message for a wrong password and for an email with no account.
		// Telling them apart tells a stranger which addresses exist here.
		a.fail(w, store.ErrNoAccount, http.StatusUnauthorized)
		return
	}
	if err != nil {
		a.log.Error("sign-in failed unexpectedly", "err", err)
		a.fail(w, errors.New("could not sign you in"), http.StatusInternalServerError)
		return
	}

	a.throttle.Succeeded(client)
	a.log.Info("signed in", "ip", client)
	a.setSession(w, user, token, expires)
}

// setSession hands the browser its session cookie and says who it now is.
//
// The email comes back from the SERVER rather than being echoed from the form,
// because the server normalised it and the account menu should show the address
// that exists, not the one that was typed.
func (a *API) setSession(w http.ResponseWriter, user store.User, token string, expires time.Time) {
	http.SetCookie(w, auth.Cookie(token, a.cfg.SecureCookie, expires))
	writeJSON(w, map[string]any{"ok": true, "email": user.Email})
}

// startSession is the last step of changing a password: every session was just
// deleted, and this browser gets a fresh one.
func (a *API) startSession(w http.ResponseWriter, r *http.Request, user store.User) {
	token, expires, err := a.store.StartSession(r.Context(), user.ID, r.UserAgent(), clientIP(r))
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.setSession(w, user, token, expires)
}

// logout ends this session, and optionally every other one too.
//
// Not behind requireSession on purpose: signing out has to work even when the
// session is already dead, or a browser holding a stale cookie has no way to be
// rid of it.
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Everywhere bool `json:"everywhere"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // an empty body means "just me"

	if cookie, err := r.Cookie(auth.CookieName(a.cfg.SecureCookie)); err == nil {
		if body.Everywhere {
			// Every failure here is reported. "Sign out everywhere" is the
			// button somebody presses because they think a device is in the
			// wrong hands, and answering {"ok":true} while the sessions are
			// still live is the worst thing this endpoint could do.
			user, ok, uerr := a.store.SessionUser(r.Context(), cookie.Value)
			if uerr != nil {
				a.fail(w, errors.New("could not reach the database, so nothing was signed out"),
					http.StatusServiceUnavailable)
				return
			}
			if !ok {
				// This session is already dead, so there is nobody to sign out
				// everywhere FOR. Clearing the cookie below is the whole job.
				a.log.Info("sign out everywhere on a session that had already ended")
			} else {
				n, derr := a.store.EndAllSessions(r.Context(), user.ID)
				if derr != nil {
					a.fail(w, errors.New("could not sign every device out"),
						http.StatusInternalServerError)
					return
				}
				a.hub.DisconnectUser(user.ID)
				a.log.Info("signed out everywhere", "sessions", n)
			}
		} else {
			if derr := a.store.EndSession(r.Context(), cookie.Value); derr != nil {
				a.fail(w, errors.New("could not sign you out"), http.StatusInternalServerError)
				return
			}
			a.hub.DisconnectSession(string(auth.HashToken(cookie.Value)))
		}
	}
	http.SetCookie(w, auth.ClearCookie(a.cfg.SecureCookie))
	writeJSON(w, map[string]any{"ok": true})
}

// session tells the app whether this browser is signed in, and as whom.
func (a *API) session(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"signedIn": false}

	if cookie, cerr := r.Cookie(auth.CookieName(a.cfg.SecureCookie)); cerr == nil {
		user, ok, uerr := a.store.SessionUser(r.Context(), cookie.Value)
		if uerr != nil {
			a.fail(w, errors.New("could not verify your session"), http.StatusServiceUnavailable)
			return
		}
		if ok {
			out["signedIn"] = true
			out["email"] = user.Email
			// Carried here so the app can paint this person's own theme as soon
			// as it knows who they are, rather than after a second request.
			out["appearance"] = user.Appearance
		}
	}
	writeJSON(w, out)
}

// ---------------------------------------------------------------------------
// the account
// ---------------------------------------------------------------------------

// changePassword replaces it, having first proved the current one, and leaves
// this browser holding a brand new session.
func (a *API) changePassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	user, _ := userFrom(r.Context())

	var body struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, errors.New("that request could not be read"), http.StatusBadRequest)
		return
	}
	if len(body.Next) < minPassword {
		a.fail(w, errPasswordShort, http.StatusBadRequest)
		return
	}

	release, ok := a.acquireHash(w)
	if !ok {
		return
	}
	defer release()

	// The current password is required even though the session already proves
	// who this is. The session is what somebody would have if they walked up to
	// an unlocked laptop, which is exactly the case this check is for.
	ended, err := a.store.ChangePassword(r.Context(), user.ID, body.Current, body.Next)
	if errors.Is(err, store.ErrNoAccount) {
		a.log.Warn("wrong current password on a change attempt", "ip", clientIP(r))
		a.fail(w, errors.New("that is not your current password"), http.StatusUnauthorized)
		return
	}
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}

	// Every session just died, this browser's included, but a socket opened
	// under one is still connected and still receiving every conversation
	// event. Closing them is the difference between "signed out" and "signed
	// out" (docs/Decisions.md D-030).
	a.hub.DisconnectUser(user.ID)

	a.log.Info("password changed", "sessions_ended", ended)
	a.startSession(w, r, user)
}

// ---------------------------------------------------------------------------

// checkEmail rejects what is obviously not an email address.
//
// Deliberately not a regular expression. The only address that matters is one
// the person can actually receive mail at, and no pattern decides that — the
// confirmation email does. Line breaks and control characters are refused
// outright: the address ends up in an email header.
func checkEmail(email string) error {
	trimmed := strings.TrimSpace(email)
	at := strings.LastIndex(trimmed, "@")
	if at <= 0 || at == len(trimmed)-1 || len(trimmed) > 254 {
		return errNoEmail
	}
	for _, r := range trimmed {
		if r <= ' ' || r == 0x7f {
			return errNoEmail
		}
	}
	return nil
}

// tooMany answers a throttled request with how long to wait.
func tooMany(w http.ResponseWriter, wait time.Duration, a *API) {
	w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
	a.fail(w, errors.New("too many attempts — try again in "+wait.Round(time.Second).String()),
		http.StatusTooManyRequests)
}

// ---------------------------------------------------------------------------
// the daemon's way in
// ---------------------------------------------------------------------------

// daemonAuthorised checks the shared secret the daemon presents.
//
// A separate credential from anybody's password, because they are separate
// things: a password is a person proving who they are, this is one machine
// proving it is the one that was installed (docs/Decisions.md D-028).
//
// DEVELOPMENT ONLY since D-038. A deployed server has no daemon token at all,
// and this answers false to everything — installed computers authenticate with
// the credential they were paired with, and never present this. An empty token
// must therefore refuse rather than compare: it is not a secret anybody has to
// guess, it is the absence of a way in.
//
// Compared in constant time. This is a fixed secret compared on every reconnect,
// which is exactly the shape a timing attack likes.
func (a *API) daemonAuthorised(r *http.Request) bool {
	if a.cfg.DaemonToken == "" {
		return false
	}
	presented := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(presented) <= len(prefix) || presented[:len(prefix)] != prefix {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented[len(prefix):]), []byte(a.cfg.DaemonToken)) == 1
}

// clientIP is the address a failed attempt is counted against.
//
// r.RemoteAddr, NOT X-Forwarded-For. Behind the proxy this is deployed behind,
// that means every client counts as one — which makes the throttle coarser than
// it should be, and is recorded as a gap rather than fixed by trusting a header
// any client can set.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
