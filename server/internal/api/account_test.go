package api

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* The properties that make one account one account.

Every test here was written for a defect found by review rather than by use, and
each one fails on the shape the code had before the fix. That is the bar: a test
that passes either way proves nothing, and a security test that proves nothing
is worse than none, because it reads like coverage.

The sign-in / password-change race is NOT here, and deliberately. Firing
concurrent requests at it passes just as happily with the fix removed — the
window is real but too narrow to hit from out here. It is tested at the
mechanism instead, in store/users_test.go, where it is deterministic.
*/

// Sign-up checked "does an account exist?" and then inserted, which is
// check-then-act across a ~50ms password hash. Two requests with DIFFERENT
// emails both passed the check and both inserted, and the deployment
// permanently had two owners — each able to run agents on the machine. The
// UNIQUE constraint on email did not help: the emails differ.
//
// Fails without the users_only_one index (migrations/00008).
func TestTwoSimultaneousSignUpsCannotBothGetAnAccount(t *testing.T) {
	r := newRig(t)

	// Back to unclaimed, so sign-up is open to the race.
	if _, err := r.store.DeleteEveryUser(context.Background()); err != nil {
		t.Fatalf("clear users: %v", err)
	}

	const racers = 8
	code := r.api.SetupCode()

	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	codes := make([]int, racers)

	for i := 0; i < racers; i++ {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait() // release them together
			res := r.anon("POST", "/api/auth/signup", map[string]any{
				"setupCode": code,
				// A DIFFERENT email each, which is the case the email
				// constraint cannot arbitrate.
				"email":    "owner" + string(rune('a'+i)) + "@sparstrow.test",
				"password": "a-long-enough-passphrase",
			})
			codes[i] = res.StatusCode
		}(i)
	}
	start.Done()
	done.Wait()

	won := 0
	for i, c := range codes {
		switch c {
		case http.StatusOK:
			won++
		case http.StatusConflict, http.StatusTooManyRequests, http.StatusServiceUnavailable:
			// Refused, which is the point.
		default:
			t.Errorf("racer %d got an unexpected %d", i, c)
		}
	}
	if won != 1 {
		t.Errorf("%d of %d sign-ups succeeded; exactly one may", won, racers)
	}

	// The real assertion: one ROW, whatever the HTTP statuses said. A test that
	// only counted 200s would pass against a server that returned one 200 and
	// wrote two rows.
	n, err := r.store.UserCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("the users table holds %d rows; this app has one account", n)
	}
}

// Sign-up used to consume a throttle attempt BEFORE checking whether the app
// was already claimed. The throttle is keyed by client IP, which behind a
// reverse proxy is one value for the whole internet — so any stranger could
// hold the owner's SIGN-IN in back-off by hammering an endpoint that does
// nothing at all.
func TestHammeringAClosedSignUpDoesNotLockTheOwnerOut(t *testing.T) {
	r := newRig(t) // already claimed

	for i := 0; i < 40; i++ {
		res := r.anon("POST", "/api/auth/signup", map[string]any{
			"setupCode": "not-the-code", "email": "x@y.test", "password": "aaaaaaaaaaaa",
		})
		if res.StatusCode != http.StatusConflict {
			t.Fatalf("attempt %d: got %d, want 409 — a claimed app refuses for free", i, res.StatusCode)
		}
	}

	// The owner must still be able to sign in, immediately.
	res := r.post("/api/auth/login", map[string]any{
		"email": testEmail, "password": testPassword,
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("the owner was locked out by strangers knocking: %d", res.StatusCode)
	}
}

// Changing a password ends every session. A socket, though, is authenticated
// once at the handshake and then lives as long as the tab — so the rows went
// and the connection stayed, still receiving every conversation event. The
// thing "sign out everywhere" exists to stop was exactly what kept happening.
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
