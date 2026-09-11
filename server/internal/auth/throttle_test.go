package auth

import (
	"testing"
	"time"
)

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
		if wait := th.Wait("1.2.3.4"); wait != 0 {
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
func TestTheWaitGrowsWithEveryFurtherFailure(t *testing.T) {
	th, _ := withClock(t)

	var previous time.Duration
	// Far enough past the allowance to reach the cap: 2s doubled eight times is
	// already beyond five minutes.
	for i := 0; i < freeAttempts+maxShift+1; i++ {
		th.Failed("1.2.3.4")
		wait := th.Wait("1.2.3.4")
		if i >= freeAttempts && wait <= previous && wait < maxDelay {
			t.Errorf("failure %d waits %s, no more than the previous %s", i+1, wait, previous)
		}
		previous = wait
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
		th.Failed("attacker")
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
		th.Failed("1.2.3.4")
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
		th.Failed("1.2.3.4")
	}
	if th.Wait("1.2.3.4") == 0 {
		t.Fatal("not throttled to begin with")
	}
	*now = now.Add(failureWindow + time.Minute)
	if wait := th.Wait("1.2.3.4"); wait != 0 {
		t.Errorf("failures outside the window still cost %s", wait)
	}
}
