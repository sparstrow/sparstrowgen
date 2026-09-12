package store

import (
	"context"
	"testing"
	"time"
)

/* The row lock that stops a revoked password from minting a session.

Signing in verifies a password and then creates a session. Verifying costs ~50ms
of argon2, and changing a password deletes every session — so with those two
steps unsynchronised, a sign-in could read the OLD hash, spend that time while
the change committed, and then insert a session for a password that no longer
exists.

Trying to prove that by racing goroutines through the HTTP API does not work:
the window is real but narrow, and a test that fires requests at it passes just
as happily with the fix removed, which makes it a test of nothing. So this goes
at the mechanism instead, where the behaviour is deterministic — ChangePassword
must WAIT for a sign-in that is holding the row.
*/

func newAccount(t *testing.T, s *Store) User {
	t.Helper()
	ctx := context.Background()
	if _, err := s.DeleteEveryUser(ctx); err != nil {
		t.Fatal(err)
	}
	user, err := s.CreateUser(ctx, "lock@sparstrow.test", "the-first-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	return user
}

// A password change waits for an in-flight sign-in on the same account.
//
// Fails when GetUserForChange loses its FOR UPDATE: the change then commits
// straight through the sign-in's FOR SHARE, deletes the sessions, and the
// sign-in inserts its own afterwards.
func TestChangingAPasswordWaitsForASignInHoldingTheRow(t *testing.T) {
	s := testStore(t)
	user := newAccount(t, s)
	ctx := context.Background()

	// Stand in for a sign-in that has read the row and is busy hashing: hold
	// exactly the lock SignIn holds, for a known length of time.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	uid, err := parseUUID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.q.WithTx(tx).GetUserByEmailForSignIn(ctx, "lock@sparstrow.test"); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	_ = uid

	const held = 400 * time.Millisecond
	changed := make(chan time.Duration, 1)
	failed := make(chan error, 1)

	go func() {
		began := time.Now()
		if _, err := s.ChangePassword(ctx, user.ID, "the-first-passphrase", "the-second-passphrase"); err != nil {
			failed <- err
			return
		}
		changed <- time.Since(began)
	}()

	// Give the change a moment to reach the lock and block on it.
	time.Sleep(held)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-failed:
		t.Fatalf("change password: %v", err)
	case took := <-changed:
		// Some tolerance: the assertion is "it waited for the holder", not a
		// measurement of how long scheduling took.
		if took < held/2 {
			t.Errorf("the change took %v and did not wait for the sign-in holding the row "+
				"(it was held for %v)", took, held)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the change never completed")
	}
}

// Two password changes at once cannot both verify against the same old hash.
//
// Without FOR UPDATE they both read it, both succeed, and the second silently
// overwrites the first — so the owner is told twice that the password is now
// what they typed, and only one of those is true.
func TestTwoPasswordChangesAtOnceCannotBothSucceed(t *testing.T) {
	s := testStore(t)
	user := newAccount(t, s)
	ctx := context.Background()

	type outcome struct{ err error }
	results := make(chan outcome, 2)

	for _, next := range []string{"change-one-passphrase", "change-two-passphrase"} {
		go func(next string) {
			_, err := s.ChangePassword(ctx, user.ID, "the-first-passphrase", next)
			results <- outcome{err}
		}(next)
	}

	succeeded := 0
	for i := 0; i < 2; i++ {
		if (<-results).err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Errorf("%d of 2 simultaneous password changes succeeded; exactly one may", succeeded)
	}
}
