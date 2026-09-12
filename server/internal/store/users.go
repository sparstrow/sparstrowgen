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

// ErrNoAccount covers both "no such email" and "wrong password", deliberately
// as one error. Telling them apart tells a stranger which emails exist.
var ErrNoAccount = errors.New("that email and password do not match an account")

const uniqueViolation = "23505"

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

// CreateUser makes the account.
//
// Two simultaneous sign-ups can both pass the Claimed check — there is no lock
// between reading a count and inserting a row. The UNIQUE constraint on email
// is what actually decides it, and a second account with a DIFFERENT email is
// prevented by the caller re-checking inside the same request. Belt and braces,
// because the cost of being wrong here is somebody else owning the machine.
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
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}
	return User{ID: uuidToString(row.ID), Email: row.Email}, nil
}

// Authenticate checks an email and password together.
//
// A missing email and a wrong password return the SAME error, and both pay for
// a password hash — the miss hashes against a dummy so that "no such account"
// and "wrong password" take the same time. Without that, the response time is
// an oracle for which emails have accounts.
func (s *Store) Authenticate(ctx context.Context, email, password string) (User, error) {
	row, err := s.q.GetUserByEmail(ctx, normaliseEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		// Deliberately wasted work. The cost of the hash is the point.
		_, _ = auth.VerifyPassword(decoyHash, password)
		return User{}, ErrNoAccount
	}
	if err != nil {
		return User{}, err
	}
	ok, err := auth.VerifyPassword(row.PasswordHash, password)
	if err != nil {
		return User{}, err
	}
	if !ok {
		return User{}, ErrNoAccount
	}
	return User{ID: uuidToString(row.ID), Email: row.Email}, nil
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
	row, err := s.q.GetUser(ctx, uid)
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
	// One transaction: a password changed without its sessions ending would
	// leave the old credential's sessions working, which is the failure this
	// whole function exists to prevent.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)

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
