package auth

import (
	"testing"
	"time"
)

// wrongPassword is one complete failed attempt, in the order login performs it:
// admission first, which is what counts it, then the failure that refreshes the
// clock. Calling Failed alone would count nothing.
func wrongPassword(th *Throttle, client string) {
	th.Begin(client)
	th.Failed(client)
}

// clock lets these tests assert about minutes without spending them.
func withClock(t *testing.T) (*Throttle, *time.Time) {
	t.Helper()
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	th := NewThrottle()
	th.now = func() time.Time { return now }
	return th, &now
}

func TestAFewWrongAttemptsAreFreeAndThenTheWaitBegins(t *testing.T) {
	th, now := withClock(t)

	for i := 0; i < freeAttempts; i++ {
		wait, ok := th.Begin("1.2.3.4")
		if !ok || wait != 0 {
			t.Fatalf("attempt %d was made to wait %s; the first %d are free", i+1, wait, freeAttempts)
		}
		th.Failed("1.2.3.4")
	}
	// freeAttempts failures are now recorded, and the allowance is spent.

	wait := th.Wait("1.2.3.4")
	if wait <= 0 {
		t.Fatal("guessing was still free after the allowance was spent")
	}

	// Waiting it out works — the answer is always "later", never "never".
	*now = now.Add(wait)
	if again := th.Wait("1.2.3.4"); again != 0 {
		t.Errorf("still %s to wait after the full delay had passed", again)
	}
}

// Each failure past the allowance costs more than the last, so a patient
// attacker is stopped rather than merely slowed.
//
// The attacker here is patient in the exact way that matters: he waits out each
// delay in full and then tries again. That is the best case available to him,
// and it still escalates to the cap. (A test with a frozen clock proves nothing
// here — every attempt is simply refused, and refused attempts are not what the
// escalation is made of.)
func TestTheWaitGrowsForAnAttackerWhoWaitsItOut(t *testing.T) {
	th, now := withClock(t)

	for i := 0; i < freeAttempts; i++ {
		wrongPassword(th, "1.2.3.4")
	}

	var previous time.Duration
	for i := 0; i < maxShift+2; i++ {
		wait := th.Wait("1.2.3.4")
		if wait <= 0 {
			t.Fatalf("round %d: guessing was free again", i)
		}
		if wait < previous {
			t.Errorf("round %d: the wait went backwards, %s after %s", i, wait, previous)
		}
		previous = wait

		*now = now.Add(wait) // he waits, exactly as long as he was told
		if w, ok := th.Begin("1.2.3.4"); !ok {
			t.Fatalf("round %d: still refused %s after the full wait elapsed", i, w)
		}
		th.Failed("1.2.3.4")
	}
	if previous != maxDelay {
		t.Errorf("the wait settled at %s, want the cap of %s", previous, maxDelay)
	}
}

// The property this design exists for: one client being throttled must never
// lock another one out. With a single password and a single owner, a global
// counter would hand any stranger a button that locks the owner out of his own
// machine.
func TestOneClientBeingThrottledDoesNotLockOutAnother(t *testing.T) {
	th, _ := withClock(t)

	for i := 0; i < freeAttempts+5; i++ {
		wrongPassword(th, "attacker")
	}
	if th.Wait("attacker") == 0 {
		t.Fatal("the attacker was not throttled, so the test proves nothing")
	}
	if wait := th.Wait("the owner"); wait != 0 {
		t.Errorf("the owner was made to wait %s because somebody else guessed wrong", wait)
	}
}

// Getting in proves the failures were the owner mistyping.
func TestSigningInSuccessfullyForgetsTheFailures(t *testing.T) {
	th, _ := withClock(t)

	for i := 0; i < freeAttempts+3; i++ {
		wrongPassword(th, "1.2.3.4")
	}
	if th.Wait("1.2.3.4") == 0 {
		t.Fatal("not throttled before the successful sign-in")
	}
	th.Succeeded("1.2.3.4")
	if wait := th.Wait("1.2.3.4"); wait != 0 {
		t.Errorf("still throttled after signing in: %s", wait)
	}
}

// Old failures fall out of the window, so a mistake made this morning is not
// still being counted this evening.
func TestFailuresAreForgottenAfterTheWindow(t *testing.T) {
	th, now := withClock(t)

	for i := 0; i < freeAttempts+3; i++ {
		wrongPassword(th, "1.2.3.4")
	}
	if th.Wait("1.2.3.4") == 0 {
		t.Fatal("not throttled to begin with")
	}
	*now = now.Add(failureWindow + time.Minute)
	if wait := th.Wait("1.2.3.4"); wait != 0 {
		t.Errorf("failures outside the window still cost %s", wait)
	}
}
