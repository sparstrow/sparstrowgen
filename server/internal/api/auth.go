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
	errNotLoggedIn   = errors.New("not signed in")
	errNoPassword    = errors.New("a password is required")
	errNoEmail       = errors.New("that does not look like an email address")
	errAlreadySetUp  = errors.New("this app already has an account")
	errBadSetupCode  = errors.New("that setup code is not right — check the server's startup log")
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
// password hash — which also makes it a good way to exhaust a small VPS. Four
// at a time is more than a single-user app will ever legitimately need, and it
// bounds the damage at about 76 MiB. Anything beyond it is turned away
// immediately rather than queued, since a queue is just the same exhaustion
// with extra steps.
const hashSlots = 4

// ---------------------------------------------------------------------------
// claiming the app
// ---------------------------------------------------------------------------

// signUp creates the one account, and is reachable only while there isn't one.
func (a *API) signUp(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	client := clientIP(r)

	// Throttled like a sign-in, because guessing a setup code is guessing.
	wait, admitted := a.throttle.Begin(client)
	if !admitted {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		a.fail(w, errors.New("too many attempts — try again in "+wait.Round(time.Second).String()),
			http.StatusTooManyRequests)
		return
	}

	claimed, err := a.store.Claimed(r.Context())
	if err != nil {
		a.fail(w, errors.New("could not reach the database"), http.StatusServiceUnavailable)
		return
	}
	if claimed {
		a.fail(w, errAlreadySetUp, http.StatusConflict)
		return
	}

	var body struct {
		SetupCode string `json:"setupCode"`
		Email     string `json:"email"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, errors.New("that request could not be read"), http.StatusBadRequest)
		return
	}

	// Constant-time: a fixed secret compared on every attempt is exactly the
	// shape a timing attack likes.
	if subtle.ConstantTimeCompare(
		[]byte(strings.TrimSpace(body.SetupCode)), []byte(a.setupCode)) != 1 {
		a.throttle.Failed(client)
		a.log.Warn("wrong setup code", "ip", client)
		a.fail(w, errBadSetupCode, http.StatusUnauthorized)
		return
	}
	if err := checkCredentials(body.Email, body.Password); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}

	select {
	case a.hashing <- struct{}{}:
		defer func() { <-a.hashing }()
	default:
		a.fail(w, errors.New("the server is busy — try again in a moment"), http.StatusServiceUnavailable)
		return
	}

	user, err := a.store.CreateUser(r.Context(), body.Email, body.Password)
	if errors.Is(err, store.ErrEmailTaken) {
		// Two sign-ups at once: both passed the check above and the database
		// decided between them. This is the loser, and the app is claimed.
		a.fail(w, errAlreadySetUp, http.StatusConflict)
		return
	}
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}

	a.throttle.Succeeded(client)
	a.log.Info("account created — sign-up is now closed", "email", user.Email)
	a.startSession(w, r, user)
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
	// hash sitting in the gap, and a burst of simultaneous requests all ask
	// before any of them has counted (docs/Bugs.md B-12).
	wait, admitted := a.throttle.Begin(client)
	if !admitted {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		a.fail(w, errors.New("too many attempts — try again in "+wait.Round(time.Second).String()),
			http.StatusTooManyRequests)
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

	user, err := a.store.Authenticate(r.Context(), body.Email, body.Password)
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
	a.startSession(w, r, user)
}

// startSession is the last step of signing up, signing in, and changing a
// password — every path that should leave this browser holding a NEW session.
func (a *API) startSession(w http.ResponseWriter, r *http.Request, user store.User) {
	token, expires, err := a.store.StartSession(r.Context(), user.ID, r.UserAgent(), clientIP(r))
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, auth.Cookie(token, a.cfg.SecureCookie, expires))
	writeJSON(w, map[string]any{"ok": true, "email": user.Email})
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
			if user, ok, uerr := a.store.SessionUser(r.Context(), cookie.Value); uerr == nil && ok {
				n, derr := a.store.EndAllSessions(r.Context(), user.ID)
				if derr != nil {
					a.fail(w, derr, http.StatusInternalServerError)
					return
				}
				a.log.Info("signed out everywhere", "sessions", n)
			}
		} else if derr := a.store.EndSession(r.Context(), cookie.Value); derr != nil {
			a.fail(w, derr, http.StatusInternalServerError)
			return
		}
	}
	http.SetCookie(w, auth.ClearCookie(a.cfg.SecureCookie))
	writeJSON(w, map[string]any{"ok": true})
}

// session tells the app which screen to show: whether anybody has claimed this
// instance yet, and whether this browser is signed in.
func (a *API) session(w http.ResponseWriter, r *http.Request) {
	claimed, err := a.store.Claimed(r.Context())
	if err != nil {
		a.fail(w, errors.New("could not reach the database"), http.StatusServiceUnavailable)
		return
	}
	out := map[string]any{"claimed": claimed, "signedIn": false}

	if cookie, cerr := r.Cookie(auth.CookieName(a.cfg.SecureCookie)); cerr == nil {
		user, ok, uerr := a.store.SessionUser(r.Context(), cookie.Value)
		if uerr != nil {
			a.fail(w, errors.New("could not verify your session"), http.StatusServiceUnavailable)
			return
		}
		if ok {
			out["signedIn"] = true
			out["email"] = user.Email
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

	select {
	case a.hashing <- struct{}{}:
		defer func() { <-a.hashing }()
	default:
		a.fail(w, errors.New("the server is busy — try again in a moment"), http.StatusServiceUnavailable)
		return
	}

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

	// Every session just died, this browser's included. Issuing a fresh one
	// here keeps the owner signed in without keeping the old token alive — a
	// copy of it, taken by whoever prompted the change, is now dead too.
	a.log.Info("password changed", "sessions_ended", ended)
	a.startSession(w, r, user)
}

// ---------------------------------------------------------------------------

// checkCredentials rejects what is obviously not an email and a password that
// is too short to be one.
//
// Deliberately not a regular expression. The only address that matters is one
// the owner can actually receive mail at, and no pattern decides that.
func checkCredentials(email, password string) error {
	trimmed := strings.TrimSpace(email)
	at := strings.Index(trimmed, "@")
	if at <= 0 || at == len(trimmed)-1 || strings.ContainsAny(trimmed, " \t") {
		return errNoEmail
	}
	if password == "" {
		return errNoPassword
	}
	if len(password) < minPassword {
		return errPasswordShort
	}
	return nil
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
