package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

// CookieName is the session cookie's name, which depends on whether it is
// Secure — and that is not a stylistic choice.
//
// The `__Host-` prefix is enforced by the browser rather than by us: it REFUSES
// the cookie outright unless it is Secure, has Path=/ and carries no Domain. In
// production that is free protection, and it means no subdomain can overwrite
// the session. Over plain http://localhost it is the opposite of protection:
// the cookie cannot be Secure, so the browser drops it, and the owner watches a
// successful sign-in be followed by "not signed in" on the very next request.
//
// So the prefix is worn only when it can be honoured. Anything that reads the
// cookie must ask for the name the same way.
func CookieName(secure bool) string {
	if secure {
		return "__Host-sgsession"
	}
	return "sgsession"
}

// Lifetime is how long a session lasts before it must be earned again.
//
// A week, because the balance here is not the usual one. Behind this cookie is
// something that runs code on the owner's machine, so a stolen laptop should not
// be a standing invitation — but the owner is also the only user, on his own
// devices, and an app that logs him out every day is an app he will be tempted
// to leave open on a machine he should not.
const Lifetime = 7 * 24 * time.Hour

// IdleLifetime expires a session nobody has used, separately from one that is
// merely old. An abandoned browser tab is a different risk from a daily habit.
const IdleLifetime = 48 * time.Hour

// TokenBytes is 256 bits of randomness. Session tokens are not guessed, they are
// stolen — but only if there are few enough of them to enumerate, and there are
// not.
const TokenBytes = 32

// NewToken mints a session token and returns it with its storage hash.
//
// The token goes to the browser and is never written down here; the hash goes to
// the database and is never enough to log in with. Losing either one alone
// gives an attacker nothing.
func NewToken() (token string, hash []byte, err error) {
	raw := make([]byte, TokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generating a session token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, HashToken(token), nil
}

// HashToken is the one-way mapping from a presented token to the stored row.
// SHA-256 rather than a password hash on purpose: the input is already 256 bits
// of randomness, so there is nothing to slow an attacker down to, and this runs
// on every single request.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// Cookie builds the session cookie.
//
// secure is a parameter rather than a constant because `Secure` makes a cookie
// unusable over plain HTTP, which is exactly right in production and exactly
// wrong on http://localhost during development. It is the deployment's job to
// set it, and main.go defaults it ON so that forgetting is the safe mistake.
func Cookie(token string, secure bool, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:  CookieName(secure),
		Value: token,
		Path:  "/",
		// Unreadable from JavaScript, so an XSS bug in the app cannot walk off
		// with the session.
		HttpOnly: true,
		Secure:   secure,
		// Lax, not Strict: Strict would mean following a link to the app from
		// anywhere else lands you logged out, which reads as a broken app. Lax
		// still refuses to send the cookie on a cross-site POST, which is the
		// case that matters.
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	}
}

// ClearCookie is the same cookie, emptied and already expired. The attributes
// must match the original or the browser keeps the one it has.
func ClearCookie(secure bool) *http.Cookie {
	c := Cookie("", secure, time.Unix(0, 0))
	c.MaxAge = -1
	return c
}
