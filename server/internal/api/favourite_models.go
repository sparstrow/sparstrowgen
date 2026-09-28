package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

// maxFavouriteBody fits one agent id and one model id with room to spare.
const maxFavouriteBody = 1 << 10

// getFavouriteModels answers with the models this account has starred.
func (a *API) getFavouriteModels(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	favs, err := a.store.FavouriteModels(r.Context(), user.ID)
	if err != nil {
		a.fail(w, errors.New("your starred models could not be loaded"), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, favs)
}

// setFavouriteModel stars or unstars one model. One endpoint for both, carrying
// the state wanted rather than toggling, so a click sent twice lands where it
// was meant to.
func (a *API) setFavouriteModel(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFavouriteBody)
	user, _ := userFrom(r.Context())

	var body struct {
		store.FavouriteModel
		Starred bool `json:"starred"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, errors.New("that request could not be read"), http.StatusBadRequest)
		return
	}
	favs, err := a.store.SetFavouriteModel(r.Context(), user.ID, body.FavouriteModel, body.Starred)
	switch {
	case errors.Is(err, store.ErrBadFavourite), errors.Is(err, store.ErrTooManyFavourites):
		a.fail(w, err, http.StatusBadRequest)
		return
	case err != nil:
		a.fail(w, errors.New("that star could not be saved"), http.StatusServiceUnavailable)
		return
	}
	// The picker in another tab or on another device shows the same list.
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventFavouriteModels})
	writeJSON(w, favs)
}
