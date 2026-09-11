package auth

import (
	"sync"
	"time"
)

/* Slowing down guessing.

argon2id already costs about 19 MiB and tens of milliseconds per attempt, which
is itself a rate limit — an attacker cannot try passwords faster than the server
can hash them. This adds the second half: a client that keeps getting it wrong
is made to wait, so a patient attacker is not merely slow but stopped.

Deliberately NOT a global lockout. With one password and one user, "too many
failures, nobody may log in" is a button any stranger can press to lock the owner
out of his own machine. The counter is per client, and the owner coming from a
different address is unaffected by an attacker hammering from theirs.

In memory, not in Postgres. A restart forgetting the counters is an acceptable
loss — restarts are rare and an attacker cannot cause one — and the alternative
is a database write on every failed login, which is a much better denial-of-
service target than the thing it protects. */

const (
	// Failures allowed before waiting starts. The fifth wrong password is the
	// one that begins the delays. Enough for a genuine mistyping and a retry,
	// not enough to be a strategy.
	freeAttempts = 5
	// How long failures are remembered. Short enough that a locked-out owner
	// can simply wait, long enough that an attacker gains nothing by pausing.
	failureWindow = 15 * time.Minute
	// The wait after the free attempts are gone, doubling each time up to the
	// cap. The cap exists so the answer is always "wait", never "give up".
	baseDelay = 2 * time.Second
	maxDelay  = 5 * time.Minute
	// maxShift is the largest doubling worth computing: 2s << 8 is already
	// past the cap, and anything beyond it only risks overflow.
	maxShift = 8
)

type attempts struct {
	failures int
	last     time.Time
}

// Throttle records failed logins per client and says how long that client must
// wait before the next one is worth trying.
type Throttle struct {
	mu  sync.Mutex
	by  map[string]*attempts
	now func() time.Time // swappable so the tests do not sleep
}

func NewThrottle() *Throttle {
	return &Throttle{by: map[string]*attempts{}, now: time.Now}
}

// Begin asks permission to make an attempt, and reserves it in the same breath.
//
// Checking and recording have to happen under ONE lock. The obvious shape — ask
// Wait, hash the password, then record a failure — leaves the whole argon2 hash
// sitting in the gap between the two, and a hundred requests arriving together
// all ask before any of them has recorded anything. Every one is admitted. The
// allowance stops meaning five attempts and starts meaning five per burst, and
// a hundred simultaneous hashes is about 1.9 GiB of working memory.
//
// So the attempt is counted the moment it is admitted, and Succeeded takes it
// back. An attempt that is admitted and then abandoned leaves its count behind
// until the window passes, which is the safe direction to be wrong in.
//
// Returns how long to wait, and whether the attempt may proceed now.
func (t *Throttle) Begin(client string) (time.Duration, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if wait := t.waitLocked(client); wait > 0 {
		return wait, false
	}
	t.recordLocked(client)
	return 0, true
}

// Wait reports how long this client must wait, without reserving anything.
func (t *Throttle) Wait(client string) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.waitLocked(client)
}

func (t *Throttle) waitLocked(client string) time.Duration {
	a := t.by[client]
	if a == nil {
		return 0
	}
	now := t.now()
	if now.Sub(a.last) > failureWindow {
		delete(t.by, client)
		return 0
	}
	if a.failures < freeAttempts {
		return 0
	}
	// Excess starts at zero on the failure that spends the allowance, so the
	// first wait is baseDelay and the shift is never negative. Capping the
	// exponent before shifting rather than the result after it: a large enough
	// shift overflows to a nonsense duration, and a guard on the result is then
	// reading a value that has already lost its meaning.
	excess := a.failures - freeAttempts
	if excess > maxShift {
		excess = maxShift
	}
	delay := baseDelay << excess
	if delay > maxDelay {
		delay = maxDelay
	}
	if waited := now.Sub(a.last); waited < delay {
		return delay - waited
	}
	return 0
}

// Failed records a wrong password.
//
// The attempt itself was already counted by Begin, so this only refreshes the
// clock — counting again here would charge a single wrong password twice.
func (t *Throttle) Failed(client string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if a := t.by[client]; a != nil {
		a.last = t.now()
	}
}

// recordLocked counts one attempt against a client. Called under the lock.
func (t *Throttle) recordLocked(client string) {
	now := t.now()
	a := t.by[client]
	if a == nil || now.Sub(a.last) > failureWindow {
		a = &attempts{}
		t.by[client] = a
	}
	a.failures++
	a.last = now
	t.sweep(now)
}

// Succeeded forgets a client's failures. Getting in proves the attempts were the
// owner mistyping, not somebody guessing.
func (t *Throttle) Succeeded(client string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.by, client)
}

// sweep drops entries nobody has touched inside the window, so a long-running
// server does not accumulate a map entry per address that ever guessed wrong.
// Called under the lock, on write, which is the only time the map grows.
func (t *Throttle) sweep(now time.Time) {
	if len(t.by) < 1024 {
		return
	}
	for client, a := range t.by {
		if now.Sub(a.last) > failureWindow {
			delete(t.by, client)
		}
	}
}
