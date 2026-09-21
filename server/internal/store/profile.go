package store

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/sparstrow/sparstrowgen/server/internal/db"
)

// Profile is who a person is, as opposed to how they sign in. It travels with
// the account rather than the browser, like appearance (D-032).
type Profile struct {
	// Empty means "not set". There is deliberately no separate NULL: both
	// would mean the same thing — fall back to the email address — and a value
	// that can be absent two ways invites code that checks only one.
	DisplayName string `json:"displayName"`
	Bio         string `json:"bio"`
	// AvatarUpdatedAt is absent when there is no picture. It doubles as the
	// cache key for the image, so a new upload is fetched and an unchanged one
	// is not.
	AvatarUpdatedAt *time.Time `json:"avatarUpdatedAt,omitempty"`
}

// Limits. Generous enough not to be met in normal use, small enough that a
// single request cannot be used to store a book.
const (
	MaxDisplayName = 80
	MaxBio         = 400
	// MaxAvatarBytes is what the server accepts. The browser resizes to a small
	// square before sending, so this is the backstop for a client that does
	// not — not the expected size.
	MaxAvatarBytes = 512 * 1024
)

var (
	ErrNameTooLong   = errors.New("that name is too long")
	ErrBioTooLong    = errors.New("that description is too long")
	ErrAvatarTooBig  = errors.New("that picture is too large")
	ErrAvatarType    = errors.New("a picture must be a PNG, JPEG or WebP")
	ErrNoAvatar      = errors.New("this account has no picture")
	avatarMediaTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true}
)

// AvatarTypeAllowed reports whether bytes of this media type may be stored. The
// list is closed rather than "anything image/*": an SVG is a document that can
// carry script, and serving one back from our own origin would run it there.
func AvatarTypeAllowed(mediaType string) bool { return avatarMediaTypes[mediaType] }

func profileOf(name, bio string) Profile {
	return Profile{DisplayName: name, Bio: bio}
}

// Profile reads the account's profile, including whether it has a picture. It
// never reads the picture itself — that is a separate call, because this one
// runs whenever a page wants a name and the bytes are not wanted.
func (s *Store) Profile(ctx context.Context, userID string) (Profile, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return Profile{}, err
	}
	row, err := s.q.GetUser(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNoAccount
	}
	if err != nil {
		return Profile{}, err
	}
	p := profileOf(row.DisplayName, row.Bio)
	at, err := s.q.GetUserAvatarUpdatedAt(ctx, u)
	switch {
	case err == nil && at.Valid:
		t := at.Time
		p.AvatarUpdatedAt = &t
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		return Profile{}, err
	}
	return p, nil
}

// SetProfile saves the name and the description together, for the same reason
// appearance saves all three of its choices at once: the form submits both, so
// a tab holding one stale value cannot merge it into the account on its own.
//
// Both are trimmed. A name that is only spaces is not a name, and letting one
// through would show an account with a blank label everywhere instead of
// falling back to the email address.
func (s *Store) SetProfile(ctx context.Context, userID string, want Profile) (Profile, error) {
	name := strings.TrimSpace(want.DisplayName)
	bio := strings.TrimSpace(want.Bio)
	// Counted in runes, not bytes: a limit in bytes cuts a name with accents or
	// non-Latin script shorter than the same name in ASCII.
	if utf8.RuneCountInString(name) > MaxDisplayName {
		return Profile{}, ErrNameTooLong
	}
	if utf8.RuneCountInString(bio) > MaxBio {
		return Profile{}, ErrBioTooLong
	}
	u, err := parseUUID(userID)
	if err != nil {
		return Profile{}, err
	}
	row, err := s.q.SetUserProfile(ctx, db.SetUserProfileParams{ID: u, DisplayName: name, Bio: bio})
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNoAccount
	}
	if err != nil {
		return Profile{}, err
	}
	out := profileOf(row.DisplayName, row.Bio)
	at, err := s.q.GetUserAvatarUpdatedAt(ctx, u)
	switch {
	case err == nil && at.Valid:
		t := at.Time
		out.AvatarUpdatedAt = &t
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		return Profile{}, err
	}
	return out, nil
}

// Avatar is the stored picture and when it was stored.
type Avatar struct {
	ContentType string
	Bytes       []byte
	UpdatedAt   time.Time
}

func (s *Store) Avatar(ctx context.Context, userID string) (Avatar, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return Avatar{}, err
	}
	row, err := s.q.GetUserAvatar(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return Avatar{}, ErrNoAvatar
	}
	if err != nil {
		return Avatar{}, err
	}
	return Avatar{ContentType: row.ContentType, Bytes: row.Bytes, UpdatedAt: row.UpdatedAt.Time}, nil
}

func (s *Store) SetAvatar(ctx context.Context, userID, contentType string, bytes []byte) error {
	if !AvatarTypeAllowed(contentType) {
		return ErrAvatarType
	}
	if len(bytes) == 0 {
		return ErrAvatarType
	}
	if len(bytes) > MaxAvatarBytes {
		return ErrAvatarTooBig
	}
	u, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return s.q.SetUserAvatar(ctx, db.SetUserAvatarParams{UserID: u, ContentType: contentType, Bytes: bytes})
}

func (s *Store) DeleteAvatar(ctx context.Context, userID string) error {
	u, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return s.q.DeleteUserAvatar(ctx, u)
}
