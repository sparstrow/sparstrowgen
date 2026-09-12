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

// Session is a login, as the owner would read it back.
type Session struct {
	Created   time.Time
	LastSeen  time.Time
	Expires   time.Time
	UserAgent string
	IP        string
}

func stamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// StartSession records a login and returns the token to hand the browser.
//
// The token is returned and never stored; what goes in the row is its hash. The
// caller gets the only copy that will ever exist, which is why it goes straight
// into a cookie and nowhere else — not a log line, not an error message.
func (s *Store) StartSession(ctx context.Context, userID, userAgent, ip string) (string, time.Time, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return "", time.Time{}, err
	}
	token, hash, err := auth.NewToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(auth.Lifetime)
	if _, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: hash,
		UserID:    uid,
		ExpiresAt: stamp(expires),
		UserAgent: userAgent,
		Ip:        ip,
	}); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// SessionUser returns who a presented token belongs to, and marks it used in
// the same statement.
//
// A token that never existed, one that was signed out, one that is too old and
// one that has gone unused too long all come back the same way: no user, no
// error. The caller cannot tell them apart, and neither can an attacker.
//
// The user comes back from the session row itself rather than from a second
// lookup, so there is no window where the session is valid and the account it
// belonged to has gone — deleting the account cascades to the session.
func (s *Store) SessionUser(ctx context.Context, token string) (User, bool, error) {
	if token == "" {
		return User{}, false, nil
	}
	row, err := s.q.TouchSession(ctx, db.TouchSessionParams{
		TokenHash: auth.HashToken(token),
		IdleSince: stamp(time.Now().Add(-auth.IdleLifetime)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	user, err := s.q.GetUser(ctx, row.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	return User{ID: uuidToString(user.ID), Email: user.Email}, true, nil
}

// EndSession signs one session out. Signing out a token that is already gone is
// not an error — the caller wanted it gone, and it is.
func (s *Store) EndSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.q.DeleteSession(ctx, auth.HashToken(token))
}

// EndAllSessions signs every one of this user's devices out.
func (s *Store) EndAllSessions(ctx context.Context, userID string) (int64, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return 0, err
	}
	return s.q.DeleteUserSessions(ctx, uid)
}

// SweepSessions deletes rows that can no longer authenticate anything. Purely
// housekeeping: SessionUser already refuses them, so this is about the table
// not growing forever rather than about security.
func (s *Store) SweepSessions(ctx context.Context) (int64, error) {
	return s.q.DeleteDeadSessions(ctx, stamp(time.Now().Add(-auth.IdleLifetime)))
}

// Sessions lists this user's live logins, newest use first.
func (s *Store) Sessions(ctx context.Context, userID string) ([]Session, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListUserSessions(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(rows))
	for _, r := range rows {
		out = append(out, Session{
			Created:   r.CreatedAt.Time,
			LastSeen:  r.LastSeenAt.Time,
			Expires:   r.ExpiresAt.Time,
			UserAgent: r.UserAgent,
			IP:        r.Ip,
		})
	}
	return out, nil
}
