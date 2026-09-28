package store

import (
	"context"
	"errors"
	"strings"

	"github.com/sparstrow/sparstrowgen/server/internal/db"
)

// FavouriteModel is one starred model in the agent and model picker: which
// agent runs it, and the id that agent reports for it. Nothing else — the label
// and description come from the computer, which is the one that knows them.
type FavouriteModel struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// MaxFavouriteModels is far more than anyone stars by hand, and small enough
// that a runaway script cannot grow one account's list without bound.
const MaxFavouriteModels = 100

var (
	// ErrBadFavourite is a star with no agent or no model, or one too long to
	// be an id any CLI reports.
	ErrBadFavourite = errors.New("that is not a model that can be starred")
	// ErrTooManyFavourites is a star past MaxFavouriteModels.
	ErrTooManyFavourites = errors.New("you have starred as many models as can be kept; unstar one first")
)

// Valid reports whether both halves are present and of a sane length. They are
// not checked against a list of known agents or models: a newer daemon may
// offer one this server has never heard of.
func (f FavouriteModel) Valid() bool {
	p, m := strings.TrimSpace(f.Provider), strings.TrimSpace(f.Model)
	return p != "" && m != "" && p == f.Provider && m == f.Model && len(p) <= 64 && len(m) <= 200
}

// FavouriteModels lists one account's starred models, oldest first.
func (s *Store) FavouriteModels(ctx context.Context, userID string) ([]FavouriteModel, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListFavouriteModels(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]FavouriteModel, 0, len(rows))
	for _, r := range rows {
		out = append(out, FavouriteModel{Provider: r.Provider, Model: r.Model})
	}
	return out, nil
}

// SetFavouriteModel stars or unstars one model and answers with the whole list,
// so a tab that sent it holds exactly what the account now has. Starring one
// already starred, or unstarring one that is not, changes nothing.
func (s *Store) SetFavouriteModel(ctx context.Context, userID string, f FavouriteModel, starred bool) ([]FavouriteModel, error) {
	if !f.Valid() {
		return nil, ErrBadFavourite
	}
	u, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	if starred {
		n, err := s.q.CountFavouriteModels(ctx, u)
		if err != nil {
			return nil, err
		}
		if n >= MaxFavouriteModels {
			current, err := s.FavouriteModels(ctx, userID)
			if err != nil {
				return nil, err
			}
			for _, c := range current {
				if c == f {
					return current, nil
				}
			}
			return nil, ErrTooManyFavourites
		}
		err = s.q.AddFavouriteModel(ctx, db.AddFavouriteModelParams{UserID: u, Provider: f.Provider, Model: f.Model})
		if err != nil {
			return nil, err
		}
	} else {
		err = s.q.RemoveFavouriteModel(ctx, db.RemoveFavouriteModelParams{UserID: u, Provider: f.Provider, Model: f.Model})
		if err != nil {
			return nil, err
		}
	}
	return s.FavouriteModels(ctx, userID)
}
