package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
	"github.com/sparstrow/sparstrowgen/server/internal/db"
)

/* Account access: creating an account from a confirmation link, resetting a
forgotten password from a reset link, and recording requests from people who
were not invited.

Who may register is NOT decided here. The invitation list is deployment
configuration, and the API checks it before anything in this file runs. */

// LinkKind is what an emailed link is for.
type LinkKind string

const (
	LinkVerify LinkKind = "verify"
	LinkReset  LinkKind = "reset"
)

// Why a link no longer works. Values rather than prose, so the page that shows
// them chooses the words.
const (
	LinkExpired    = "expired"
	LinkUsed       = "used"
	LinkSuperseded = "superseded"
)

// ErrLinkUnusable is a link that was valid when its page opened and is not any
// more by the time the form was submitted: used in another tab, superseded by a
// newer email, or expired while the person was typing.
var ErrLinkUnusable = errors.New("this link no longer works — open the newest email or ask for another")

// LinkState is what a link is still good for.
type LinkState struct {
	Usable bool
	// Email is set only for a usable link. An unusable link's page has nothing
	// to do with the address, and does not need to show it.
	Email  string
	Reason string
	// AccountReady says the address has an account. For a used confirmation
	// link it turns "already used" into "your account is already set up".
	AccountReady bool
}

// IssueLink mints a link for an address and supersedes every earlier unused one
// of the same kind — in one transaction, so there is never a moment with two
// live links, or with none.
//
// Returns the token for the email. Only its hash is stored.
func (s *Store) IssueLink(ctx context.Context, kind LinkKind, email string, lifetime time.Duration) (string, error) {
	token, hash, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	address := NormaliseEmail(email)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)

	if err := q.SupersedeEmailLinks(ctx, db.SupersedeEmailLinksParams{Email: address, Kind: string(kind)}); err != nil {
		return "", err
	}
	if _, err := q.CreateEmailLink(ctx, db.CreateEmailLinkParams{
		TokenHash: hash,
		Kind:      string(kind),
		Email:     address,
		ExpiresAt: stamp(time.Now().Add(lifetime)),
	}); err != nil {
		return "", err
	}
	return token, tx.Commit(ctx)
}

// InspectLink reports what a link is still good for, without spending it.
//
// A token that was never issued is reported as expired rather than as its own
// case: the person's next step is the same, and "no such link" would only help
// somebody guessing.
func (s *Store) InspectLink(ctx context.Context, kind LinkKind, token string) (LinkState, error) {
	if token == "" {
		return LinkState{Reason: LinkExpired}, nil
	}
	row, err := s.q.GetEmailLink(ctx, db.GetEmailLinkParams{TokenHash: auth.HashToken(token), Kind: string(kind)})
	if errors.Is(err, pgx.ErrNoRows) {
		return LinkState{Reason: LinkExpired}, nil
	}
	if err != nil {
		return LinkState{}, err
	}

	_, exists, err := s.UserByEmail(ctx, row.Email)
	if err != nil {
		return LinkState{}, err
	}
	switch {
	case row.UsedAt.Valid:
		return LinkState{Reason: LinkUsed, AccountReady: exists}, nil
	case row.SupersededAt.Valid:
		return LinkState{Reason: LinkSuperseded, AccountReady: exists}, nil
	case !row.ExpiresAt.Time.After(time.Now()):
		return LinkState{Reason: LinkExpired, AccountReady: exists}, nil
	}
	return LinkState{Usable: true, Email: row.Email}, nil
}

// CompleteRegistration spends a confirmation link, creates the account with
// the chosen password, and starts its first session — one transaction, so a
// link is never spent without an account to show for it, and an account never
// exists without the link that proved its address.
//
// The password is hashed BEFORE the transaction opens. argon2 takes tens of
// milliseconds, and there is no reason to hold a row lock through it.
func (s *Store) CompleteRegistration(ctx context.Context, token, password, userAgent, ip string) (User, string, time.Time, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return User{}, "", time.Time{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)

	link, err := q.UseEmailLink(ctx, db.UseEmailLinkParams{TokenHash: auth.HashToken(token), Kind: string(LinkVerify)})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", time.Time{}, ErrLinkUnusable
	}
	if err != nil {
		return User{}, "", time.Time{}, err
	}

	row, err := q.CreateUser(ctx, db.CreateUserParams{Email: link.Email, PasswordHash: hash})
	if isUniqueViolation(err) {
		// Rolled back, so the link is not spent on an account that already
		// existed; the person is told to sign in instead.
		return User{}, "", time.Time{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	if err := q.DeleteAccessRequest(ctx, link.Email); err != nil {
		return User{}, "", time.Time{}, err
	}

	session, expires, err := newSession(ctx, q, row.ID, userAgent, ip)
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, "", time.Time{}, err
	}
	return User{ID: uuidToString(row.ID), Email: row.Email}, session, expires, nil
}

// ResetPassword spends a reset link, replaces the password, ends every session
// the account had, and starts one for the browser that completed the reset.
//
// The account row is taken FOR UPDATE, the same lock a password change takes,
// so a sign-in verifying against the old password at this moment finishes
// first — and its session is among the ones deleted.
func (s *Store) ResetPassword(ctx context.Context, token, password, userAgent, ip string) (User, string, time.Time, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return User{}, "", time.Time{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)

	link, err := q.UseEmailLink(ctx, db.UseEmailLinkParams{TokenHash: auth.HashToken(token), Kind: string(LinkReset)})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", time.Time{}, ErrLinkUnusable
	}
	if err != nil {
		return User{}, "", time.Time{}, err
	}

	row, err := q.GetUserByEmailForChange(ctx, link.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		// The account went away after the link was sent.
		return User{}, "", time.Time{}, ErrLinkUnusable
	}
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	if _, err := q.SetUserPassword(ctx, db.SetUserPasswordParams{ID: row.ID, PasswordHash: hash}); err != nil {
		return User{}, "", time.Time{}, err
	}
	if _, err := q.DeleteUserSessions(ctx, row.ID); err != nil {
		return User{}, "", time.Time{}, err
	}

	session, expires, err := newSession(ctx, q, row.ID, userAgent, ip)
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, "", time.Time{}, err
	}
	return User{ID: uuidToString(row.ID), Email: row.Email}, session, expires, nil
}

// RecordAccessRequest notes that an uninvited address asked for an account, and
// reports whether this is the first time it has — the only time the owner is
// told.
func (s *Store) RecordAccessRequest(ctx context.Context, email string) (bool, error) {
	row, err := s.q.RecordAccessRequest(ctx, NormaliseEmail(email))
	if err != nil {
		return false, err
	}
	return row.TimesRequested == 1, nil
}

// SweepEmailLinks deletes links that can no longer do anything. Housekeeping;
// InspectLink already refuses them.
func (s *Store) SweepEmailLinks(ctx context.Context) (int64, error) {
	return s.q.DeleteDeadEmailLinks(ctx)
}

// newSession records a login inside a caller's transaction and returns the
// token to hand the browser. Only its hash is stored.
func newSession(ctx context.Context, q *db.Queries, userID pgtype.UUID, userAgent, ip string) (string, time.Time, error) {
	token, hash, err := auth.NewToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(auth.Lifetime)
	if _, err := q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: stamp(expires),
		UserAgent: userAgent,
		Ip:        ip,
	}); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}
