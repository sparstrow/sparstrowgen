package auth

import (
	"sync"
	"testing"
)

// Codex found this by reading, and it is real: Wait() and Failed() each take the
// lock, but the gap BETWEEN them is unprotected — and that gap is where the
// argon2 hash runs, which is the slowest thing in the request. A hundred
// requests arriving together all ask "may I?", all get told yes because none of
// them has failed yet, and all proceed to hash.
//
// Two consequences, both bad. The allowance stops being five attempts and
// becomes five attempts per round of however many an attacker sends at once.
// And a hundred concurrent argon2 hashes is about 1.9 GiB of working memory on
// a VPS that does not have it.
//
// This test is written against the CONCURRENT behaviour, so it fails on the
// check-then-act version and passes once admission reserves the attempt.
func TestSimultaneousAttemptsCannotAllSlipThroughTheAllowance(t *testing.T) {
	th, _ := withClock(t)

	const attackers = 100
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	start := make(chan struct{})
	for i := 0; i < attackers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // all at once, which is the whole point
			if wait, ok := th.Begin("one attacker"); ok && wait == 0 {
				mu.Lock()
				allowed++
				mu.Unlock()
				th.Failed("one attacker")
			}
		}()
	}
	close(start)
	wg.Wait()

	if allowed > freeAttempts {
		t.Errorf("%d of %d simultaneous attempts were admitted; the allowance is %d",
			allowed, attackers, freeAttempts)
	}
	if allowed == 0 {
		t.Error("none were admitted at all, so the throttle now blocks the owner too")
	}
}

// And the owner, arriving once with the right password, must still get through.
func TestAdmissionDoesNotBlockAnHonestFirstAttempt(t *testing.T) {
	th, _ := withClock(t)

	wait, ok := th.Begin("the owner")
	if !ok || wait != 0 {
		t.Fatalf("a first attempt was refused: wait=%s ok=%v", wait, ok)
	}
	th.Succeeded("the owner")

	// And again afterwards, because succeeding must not leave a reservation
	// behind that counts against the next sign-in.
	if wait, ok := th.Begin("the owner"); !ok || wait != 0 {
		t.Errorf("the attempt after a successful one was refused: wait=%s ok=%v", wait, ok)
	}
}
