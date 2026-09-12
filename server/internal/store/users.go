package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
	"github.com/sparstrow/sparstrowgen/server/internal/db"
)

// User is an account, as everything above the store sees it. There is no
// password on it, ever — the hash does not leave this package.
type User struct {
	ID    string
	Email string
}

// ErrEmailTaken is the unique constraint on users.email, named.
var ErrEmailTaken = errors.New("that email already has an account")

// ErrAlreadyClaimed is the users_only_one index, named. It is the loser of a
// race between two sign-ups, which the database decided.
var ErrAlreadyClaimed = errors.New("this app already has an account")

// ErrNoAccount covers both "no such email" and "wrong password", deliberately
// as one error. Telling them apart tells a stranger which emails exist.
var ErrNoAccount = errors.New("that email and password do not match an account")

const uniqueViolation = "23505"

// singletonIndex is the unique index that permits exactly one row in users.
// Named here because the two constraints on that table mean different things to
// a caller: one is "pick another email", the other is "you are not the owner".
const singletonIndex = "users_only_one"

// Claimed reports whether the app has an owner yet.
//
// Sign-up is reachable only while this is false. It is the whole gate: what is
// behind this login runs coding agents on the owner's machine, so a sign-up
// form that stays open is a form that hands a stranger a shell.
func (s *Store) Claimed(ctx context.Context) (bool, error) {
	n, err := s.q.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// UserCount is how many accounts exist. Claimed answers the question the app
// asks; this answers the one a test asks, which is whether the "exactly one"
// rule actually held.
func (s *Store) UserCount(ctx context.Context) (int64, error) {
	return s.q.CountUsers(ctx)
}

// CreateUser makes the account.
//
// Two simultaneous sign-ups can both pass the Claimed check — there is no lock
// between counting rows and inserting one, and they may be on different
// connections seeing different snapshots. So the check is not what enforces
// "one account": the users_only_one index is, and this reads which constraint
// refused in order to say the true thing to the loser.
func (s *Store) CreateUser(ctx context.Context, email, password string) (User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return User{}, err
	}
	row, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Email:        normaliseEmail(email),
		PasswordHash: hash,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		if pgErr.ConstraintName == singletonIndex {
			return User{}, ErrAlreadyClaimed
		}
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}
	return User{ID: uuidToString(row.ID), Email: row.Email}, nil
}

// SignIn checks an email and password and, if they match, starts the session —
// both inside ONE transaction.
//
// The two halves cannot be separated. A password change deletes every session,
// and a sign-in that verified before the change but inserted after it would
// hand out a working session for a password that no longer exists. Since
// verifying costs ~50ms of argon2, that window is wide enough to hit by
// accident, never mind on purpose. The row lock (FOR SHARE, see
// queries/users.sql) is what closes it: a change waits for this to commit, and
// then deletes the session this created.
//
// A missing email and a wrong password return the SAME error, and both pay for
// a password hash — the miss hashes against a dummy so that "no such account"
// and "wrong password" take the same time. Without that, the response time is
// an oracle for which emails have accounts.
func (s *Store) SignIn(ctx context.Context, email, password, userAgent, ip string) (User, string, time.Time, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)

	row, err := q.GetUserByEmailForSignIn(ctx, normaliseEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		// Deliberately wasted work. The cost of the hash is the point.
		_, _ = auth.VerifyPassword(decoyHash, password)
		return User{}, "", time.Time{}, ErrNoAccount
	}
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	ok, err := auth.VerifyPassword(row.PasswordHash, password)
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	if !ok {
		return User{}, "", time.Time{}, ErrNoAccount
	}

	token, hash, err := auth.NewToken()
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	expires := time.Now().Add(auth.Lifetime)
	if _, err := q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: hash,
		UserID:    row.ID,
		ExpiresAt: stamp(expires),
		UserAgent: userAgent,
		Ip:        ip,
	}); err != nil {
		return User{}, "", time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, "", time.Time{}, err
	}
	return User{ID: uuidToString(row.ID), Email: row.Email}, token, expires, nil
}

// decoyHash is a real argon2id hash of a value nobody knows, used only to spend
// the same time on a missing account as on a wrong password. Generated once at
// startup rather than written down, so it cannot match anything.
var decoyHash = func() string {
	h, err := auth.HashPassword(time.Now().String())
	if err != nil {
		// A decoy that cannot be built is not worth failing startup over; the
		// timing difference is a weakness, not a hole.
		return ""
	}
	return h
}()

// ChangePassword replaces it, having first proved the current one, and ends
// EVERY session including the one asking.
//
// Keeping the current session alive is the tempting version and it is wrong.
// The reason to change a password is usually that somebody else may hold
// something — and a session token is a something. An attacker who copied the
// current session's cookie would survive a password change that spared it,
// which defeats the point of changing it. So all of them go, and the caller
// immediately issues a fresh session for the browser that asked, which keeps
// the owner signed in without keeping the old token alive.
//
// Returns how many sessions were ended.
func (s *Store) ChangePassword(ctx context.Context, userID, current, next string) (int64, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return 0, err
	}
	// One transaction for the whole thing, opened BEFORE the current password
	// is read. Reading outside it and writing inside would let a sign-in that
	// is already verifying against the old hash slip its session in after the
	// delete — the exact hole this function exists to close.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)

	// FOR UPDATE: waits for any in-flight sign-in on this row, and serialises
	// two simultaneous changes.
	row, err := q.GetUserForChange(ctx, uid)
	if err != nil {
		return 0, err
	}
	ok, err := auth.VerifyPassword(row.PasswordHash, current)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrNoAccount
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return 0, err
	}
	if _, err := q.SetUserPassword(ctx, db.SetUserPasswordParams{ID: uid, PasswordHash: hash}); err != nil {
		return 0, err
	}
	ended, err := q.DeleteUserSessions(ctx, uid)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return ended, nil
}

// normaliseEmail trims and lowercases.
//
// The column is citext so the database already compares case-insensitively;
// this stops a leading space from creating an address that looks identical to
// the one the owner thinks he typed. Nothing clever beyond that — "normalising"
// an email any further (stripping dots, cutting +tags) is a decision about
// somebody else's mail server that is not ours to make.
func normaliseEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// DeleteEveryUser exists for tests, and only for tests.
//
// The app can be claimed once, so a suite that runs more than one test needs a
// way back to unclaimed. Deleting a user cascades to their sessions, so this
// leaves nothing behind.
func (s *Store) DeleteEveryUser(ctx context.Context) (int64, error) {
	return s.q.DeleteEveryUser(ctx)
}
