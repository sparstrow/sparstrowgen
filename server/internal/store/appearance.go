package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/sparstrow/sparstrowgen/server/internal/db"
)

// Appearance is how one account wants the app to look. It belongs to the
// account rather than the browser, so the same choice follows a person
// everywhere they sign in (spec 2026-09-12-appearance-preferences).
type Appearance struct {
	// Mode is light, dark, or system — follow the computer.
	Mode string `json:"mode"`
	// Surface is the character of the neutral background.
	Surface string `json:"surface"`
	// Accent is the colour of interactive emphasis. It never recolours success,
	// warning, danger or provider meanings.
	Accent string `json:"accent"`
}

// DefaultAppearance is what a new account starts with, and what an unreadable
// saved choice falls back to.
var DefaultAppearance = Appearance{Mode: "system", Surface: "mono", Accent: "neutral"}

// The names this version understands. A newer version may save one this one has
// never heard of, which is why reading falls back instead of failing.
var (
	AppearanceModes    = []string{"light", "dark", "system"}
	AppearanceSurfaces = []string{"paper", "slate", "soft", "mono"}
	AppearanceAccents  = []string{"neutral", "amber", "violet", "blue", "teal", "rose"}
)

// ErrUnknownAppearance is a save naming something this version does not offer.
// Reading is forgiving; writing is not, because the only writer is this app's
// own settings screen and a name it does not offer is a bug, not a preference.
var ErrUnknownAppearance = errors.New("that is not an appearance this version offers")

func known(value string, allowed []string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

// Valid reports whether every part is a name this version offers.
func (a Appearance) Valid() bool {
	return known(a.Mode, AppearanceModes) &&
		known(a.Surface, AppearanceSurfaces) &&
		known(a.Accent, AppearanceAccents)
}

// readable replaces anything this version does not understand with the default,
// part by part, so one unknown name does not discard the other two choices.
func (a Appearance) readable() Appearance {
	if !known(a.Mode, AppearanceModes) {
		a.Mode = DefaultAppearance.Mode
	}
	if !known(a.Surface, AppearanceSurfaces) {
		a.Surface = DefaultAppearance.Surface
	}
	if !known(a.Accent, AppearanceAccents) {
		a.Accent = DefaultAppearance.Accent
	}
	return a
}

// appearanceOf builds the appearance carried on a user row.
func appearanceOf(mode, surface, accent string) Appearance {
	return Appearance{Mode: mode, Surface: surface, Accent: accent}.readable()
}

// SetAppearance saves all three choices together. A save always carries the
// whole appearance, so a browser tab holding stale values cannot merge half of
// them into the account later.
func (s *Store) SetAppearance(ctx context.Context, userID string, want Appearance) (Appearance, error) {
	if !want.Valid() {
		return Appearance{}, ErrUnknownAppearance
	}
	u, err := parseUUID(userID)
	if err != nil {
		return Appearance{}, err
	}
	row, err := s.q.SetUserAppearance(ctx, db.SetUserAppearanceParams{
		ID:                u,
		AppearanceMode:    want.Mode,
		AppearanceSurface: want.Surface,
		AppearanceAccent:  want.Accent,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Appearance{}, ErrNoAccount
	}
	if err != nil {
		return Appearance{}, err
	}
	return appearanceOf(row.AppearanceMode, row.AppearanceSurface, row.AppearanceAccent), nil
}

// Appearance reads one account's saved choices.
func (s *Store) Appearance(ctx context.Context, userID string) (Appearance, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return Appearance{}, err
	}
	row, err := s.q.GetUser(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return Appearance{}, ErrNoAccount
	}
	if err != nil {
		return Appearance{}, err
	}
	return appearanceOf(row.AppearanceMode, row.AppearanceSurface, row.AppearanceAccent), nil
}
