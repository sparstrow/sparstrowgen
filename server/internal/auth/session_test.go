package auth

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// The bug this exists for: the cookie was named `__Host-sgsession` always, while
// Secure was off in development. A browser refuses a `__Host-` cookie that is
// not Secure, so signing in appeared to work and every request after it was
// "not signed in". Every Go test passed, because the harness and the server
// agreed with each other — only actually running it showed the browser
// disagreeing with both.
func TestTheCookieNameIsOneABrowserWillAccept(t *testing.T) {
	secure := Cookie("token", true, time.Now().Add(time.Hour))
	if !strings.HasPrefix(secure.Name, "__Host-") {
		t.Errorf("a secure cookie is named %q, want the __Host- prefix that earns its protection", secure.Name)
	}
	if !secure.Secure {
		t.Error("a __Host- cookie without Secure is refused by every browser")
	}
	if secure.Path != "/" {
		t.Errorf("path = %q; __Host- requires exactly /", secure.Path)
	}
	if secure.Domain != "" {
		t.Errorf("domain = %q; __Host- requires none", secure.Domain)
	}

	plain := Cookie("token", false, time.Now().Add(time.Hour))
	if strings.HasPrefix(plain.Name, "__Host-") {
		t.Error("a non-secure cookie wears the __Host- prefix, so the browser will silently drop it")
	}
	if plain.Secure {
		t.Error("the cookie is Secure despite being configured not to be")
	}
}

// Whatever the name, the cookie must never be readable by scripts and must
// refuse to ride along on a cross-site POST.
func TestTheSessionCookieIsNotReachableFromJavaScriptOrCrossSite(t *testing.T) {
	for _, secure := range []bool{true, false} {
		c := Cookie("token", secure, time.Now().Add(time.Hour))
		if !c.HttpOnly {
			t.Errorf("secure=%v: not HttpOnly, so an XSS bug could steal the session", secure)
		}
		if c.SameSite != http.SameSiteLaxMode {
			t.Errorf("secure=%v: SameSite is %v, want Lax", secure, c.SameSite)
		}
	}
}

// Clearing has to match the cookie it is replacing, or the browser keeps the
// one it already has and "sign out" does nothing visible.
func TestClearingTheCookieMatchesTheOneItReplaces(t *testing.T) {
	for _, secure := range []bool{true, false} {
		live := Cookie("token", secure, time.Now().Add(time.Hour))
		dead := ClearCookie(secure)
		if dead.Name != live.Name {
			t.Errorf("secure=%v: clearing %q but the session is %q", secure, dead.Name, live.Name)
		}
		if dead.Path != live.Path || dead.Secure != live.Secure {
			t.Errorf("secure=%v: attributes differ, so the browser keeps the live cookie", secure)
		}
		if dead.Value != "" || dead.MaxAge >= 0 {
			t.Errorf("secure=%v: value=%q maxage=%d, want an emptied and expired cookie",
				secure, dead.Value, dead.MaxAge)
		}
	}
}

// A token must not be guessable, and two must never collide.
func TestTokensAreRandomAndTheirHashesAreNotTheToken(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		token, hash, err := NewToken()
		if err != nil {
			t.Fatal(err)
		}
		if seen[token] {
			t.Fatal("the same token was generated twice")
		}
		seen[token] = true

		if string(hash) == token {
			t.Fatal("the stored hash IS the token — a database dump would be a list of working logins")
		}
		if strings.Contains(token, string(hash)) {
			t.Fatal("the token contains its own hash")
		}
		if got := HashToken(token); string(got) != string(hash) {
			t.Fatal("hashing the token again gave a different answer, so no session would ever validate")
		}
	}
}
