package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
)

/* The front door.

Everything below this file eventually causes a coding agent to run on the
owner's machine, in a folder the request names, with whatever tools that agent
can reach. So the rule here is not the usual web-app rule. It is: nothing gets
through that has not proved it is the owner, and the small number of exceptions
are written out one by one below rather than being whatever happens to be
outside a route group.

Three ways in, and only three:

  a session cookie   the browser, after a password
  the daemon token   the owner's own machine, dialling in
  /api/health        nothing but liveness, so a deploy can check it

There is no fourth, and there is no configuration that removes the first two.
The server refuses to start without both secrets set (cmd/server/main.go), which
means "deployed with authentication accidentally switched off" is not a state
this program can be in. */

var (
	errNotLoggedIn = errors.New("not signed in")
	errBadLogin    = errors.New("that password is not right")
	errNoPassword  = errors.New("a password is required")
)

// requireSession rejects anything without a live session cookie.
func (a *API) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.CookieName(a.cfg.SecureCookie))
		if err != nil {
			a.fail(w, errNotLoggedIn, http.StatusUnauthorized)
			return
		}
		ok, err := a.store.ValidSession(r.Context(), cookie.Value)
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
		next.ServeHTTP(w, r)
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
// password hash — which also makes it a good way to exhaust a small VPS. Four
// at a time is more than a single-user app will ever legitimately need, and it
// bounds the damage at about 76 MiB. Anything beyond it is turned away
// immediately rather than queued, since a queue is just the same exhaustion
// with extra steps.
const hashSlots = 4

// login exchanges the owner's password for a session.
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	client := clientIP(r)

	// Admission and the record of the attempt happen together, under one lock.
	// Asking "may I?" and only counting the attempt after the hash leaves the
	// whole hash sitting in the gap, and a burst of simultaneous requests all
	// ask before any of them has counted (docs/Bugs.md B-12).
	wait, admitted := a.throttle.Begin(client)
	if !admitted {
		// Say how long, rather than refusing blankly. The owner who mistyped
		// three times needs to know this ends; an attacker learns only that the
		// door is timed, which they could measure anyway.
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		a.fail(w, errors.New("too many attempts — try again in "+wait.Round(time.Second).String()),
			http.StatusTooManyRequests)
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// The attempt was already counted by Begin, and it stays counted: a
		// malformed body is not a free retry.
		a.fail(w, errors.New("that request could not be read"), http.StatusBadRequest)
		return
	}
	if body.Password == "" {
		a.fail(w, errNoPassword, http.StatusBadRequest)
		return
	}

	// Turned away rather than queued when the server is already hashing as much
	// as it is prepared to.
	select {
	case a.hashing <- struct{}{}:
		defer func() { <-a.hashing }()
	default:
		w.Header().Set("Retry-After", "1")
		a.fail(w, errors.New("the server is busy — try again in a moment"), http.StatusServiceUnavailable)
		return
	}

	ok, err := auth.VerifyPassword(a.cfg.PasswordHash, body.Password)
	if err != nil {
		// The hash itself is unreadable — a deployment fault, not a wrong
		// password, and worth a loud log because nobody can sign in until it is
		// fixed.
		a.log.Error("OWNER_PASSWORD_HASH cannot be read", "err", err)
		a.fail(w, errors.New("this server's password is misconfigured"), http.StatusInternalServerError)
		return
	}
	if !ok {
		a.throttle.Failed(client)
		a.log.Warn("failed sign-in", "ip", client)
		a.fail(w, errBadLogin, http.StatusUnauthorized)
		return
	}

	token, expires, err := a.store.StartSession(r.Context(), r.UserAgent(), client)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.throttle.Succeeded(client)
	http.SetCookie(w, auth.Cookie(token, a.cfg.SecureCookie, expires))
	a.log.Info("signed in", "ip", client)
	writeJSON(w, map[string]any{"ok": true})
}

// logout ends this session, and optionally every other one too.
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Everywhere bool `json:"everywhere"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // an empty body means "just me"

	if body.Everywhere {
		n, err := a.store.EndAllSessions(r.Context())
		if err != nil {
			a.fail(w, err, http.StatusInternalServerError)
			return
		}
		a.log.Info("signed out everywhere", "sessions", n)
	} else if cookie, err := r.Cookie(auth.CookieName(a.cfg.SecureCookie)); err == nil {
		if err := a.store.EndSession(r.Context(), cookie.Value); err != nil {
			a.fail(w, err, http.StatusInternalServerError)
			return
		}
	}
	http.SetCookie(w, auth.ClearCookie(a.cfg.SecureCookie))
	writeJSON(w, map[string]any{"ok": true})
}

// session answers "am I signed in", so the web app can decide between the login
// screen and the chat without first firing a request that 401s.
func (a *API) session(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.CookieName(a.cfg.SecureCookie))
	if err != nil {
		writeJSON(w, map[string]any{"signedIn": false})
		return
	}
	ok, err := a.store.ValidSession(r.Context(), cookie.Value)
	if err != nil {
		a.fail(w, errors.New("could not verify your session"), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]any{"signedIn": ok})
}

// ---------------------------------------------------------------------------
// the daemon's way in
// ---------------------------------------------------------------------------

// daemonAuthorised checks the shared secret the daemon presents.
//
// A separate credential from the owner's password, because they are separate
// things: the password is a person proving who they are, this is one machine
// proving it is the one that was installed. Giving the daemon the password would
// mean the password lives in plain text in a service configuration on a laptop,
// and rotating it would sign the owner out of everything.
//
// Compared in constant time. This is a fixed secret compared on every reconnect,
// which is exactly the shape a timing attack likes.
func (a *API) daemonAuthorised(r *http.Request) bool {
	presented := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(presented) <= len(prefix) || presented[:len(prefix)] != prefix {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented[len(prefix):]), []byte(a.cfg.DaemonToken)) == 1
}

// clientIP is the address a failed login is counted against.
//
// r.RemoteAddr, NOT X-Forwarded-For. Behind the proxy this will be deployed
// behind, that means every client counts as one — which makes the throttle
// coarser than it should be, and is recorded as a gap rather than fixed by
// trusting a header any client can set. A spoofable X-Forwarded-For would let an
// attacker get unlimited attempts by changing one string, which is worse than
// coarse.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
