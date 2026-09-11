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
func (s *Store) StartSession(ctx context.Context, userAgent, ip string) (string, time.Time, error) {
	token, hash, err := auth.NewToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(auth.Lifetime)
	if _, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: hash,
		ExpiresAt: stamp(expires),
		UserAgent: userAgent,
		Ip:        ip,
	}); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// ValidSession reports whether a presented token is a live session, and marks it
// used in the same statement.
//
// A token that never existed, one that was signed out, one that is too old and
// one that has gone unused too long all return false. The caller cannot tell
// them apart, and neither can an attacker.
func (s *Store) ValidSession(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	_, err := s.q.TouchSession(ctx, db.TouchSessionParams{
		TokenHash: auth.HashToken(token),
		IdleSince: stamp(time.Now().Add(-auth.IdleLifetime)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// EndSession signs one session out. Signing out a token that is already gone is
// not an error — the caller wanted it gone, and it is.
func (s *Store) EndSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.q.DeleteSession(ctx, auth.HashToken(token))
}

// EndAllSessions signs every device out.
func (s *Store) EndAllSessions(ctx context.Context) (int64, error) {
	return s.q.DeleteAllSessions(ctx)
}

// SweepSessions deletes rows that can no longer authenticate anything. Purely
// housekeeping: ValidSession already refuses them, so this is about the table
// not growing forever rather than about security.
func (s *Store) SweepSessions(ctx context.Context) (int64, error) {
	return s.q.DeleteDeadSessions(ctx, stamp(time.Now().Add(-auth.IdleLifetime)))
}

// Sessions lists the live logins, newest use first.
func (s *Store) Sessions(ctx context.Context) ([]Session, error) {
	rows, err := s.q.ListSessions(ctx)
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
