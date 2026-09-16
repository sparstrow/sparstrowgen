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
	// Appearance travels with the account, so the app can paint a person's
	// own theme as soon as it knows who they are.
	Appearance Appearance
}

// ErrEmailTaken is the unique constraint on users.email, named.
var ErrEmailTaken = errors.New("that address already has an account — sign in instead")

// ErrNoAccount covers both "no such email" and "wrong password", deliberately
// as one error. Telling them apart tells a stranger which emails exist.
var ErrNoAccount = errors.New("that email and password do not match an account")

const uniqueViolation = "23505"

// UserCount is how many accounts exist. Tests use it to assert that a flow
// created exactly the accounts it should have.
func (s *Store) UserCount(ctx context.Context) (int64, error) {
	return s.q.CountUsers(ctx)
}

// UserByEmail finds an account by address, and reports whether there is one.
func (s *Store) UserByEmail(ctx context.Context, email string) (User, bool, error) {
	row, err := s.q.GetUserByEmail(ctx, NormaliseEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	return User{ID: uuidToString(row.ID), Email: row.Email, Appearance: appearanceOf(row.AppearanceMode, row.AppearanceSurface, row.AppearanceAccent)}, true, nil
}

// CreateUser makes an account directly. People create accounts through a
// confirmation link (CompleteRegistration); this is for tests, which need an
// account without an inbox.
func (s *Store) CreateUser(ctx context.Context, email, password string) (User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return User{}, err
	}
	row, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Email:        NormaliseEmail(email),
		PasswordHash: hash,
	})
	if isUniqueViolation(err) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}
	return User{ID: uuidToString(row.ID), Email: row.Email, Appearance: appearanceOf(row.AppearanceMode, row.AppearanceSurface, row.AppearanceAccent)}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
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

	row, err := q.GetUserByEmailForSignIn(ctx, NormaliseEmail(email))
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

	token, expires, err := newSession(ctx, q, row.ID, userAgent, ip)
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, "", time.Time{}, err
	}
	return User{ID: uuidToString(row.ID), Email: row.Email, Appearance: appearanceOf(row.AppearanceMode, row.AppearanceSurface, row.AppearanceAccent)}, token, expires, nil
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

// NormaliseEmail trims and lowercases.
//
// The column is citext so the database already compares case-insensitively;
// this stops a leading space from creating an address that looks identical to
// the one the owner thinks he typed, and lets the invitation list be compared
// in Go the same way. Nothing clever beyond that — "normalising" an email any
// further (stripping dots, cutting +tags) is a decision about somebody else's
// mail server that is not ours to make.
func NormaliseEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// DeleteEveryUser exists for tests, and only for tests.
//
// A suite needs a way back to "nobody has an account". Conversations reference
// their account without a cascade (D-009), so they and their children go first;
// sessions go with their users by ON DELETE CASCADE; links and access requests
// are keyed by address and are emptied too, or one run's request would be the
// next run's repeat. One transaction, so a failure halfway leaves the database
// as it was.
func (s *Store) DeleteEveryUser(ctx context.Context) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	if err := q.DeleteEveryEntry(ctx); err != nil {
		return 0, err
	}
	if err := q.DeleteEveryProviderSession(ctx); err != nil {
		return 0, err
	}
	if err := q.DeleteEveryConversation(ctx); err != nil {
		return 0, err
	}
	if err := q.DeleteEveryEmailLink(ctx); err != nil {
		return 0, err
	}
	if err := q.DeleteEveryAccessRequest(ctx); err != nil {
		return 0, err
	}
	n, err := q.DeleteEveryUser(ctx)
	if err != nil {
		return 0, err
	}
	return n, tx.Commit(ctx)
}
