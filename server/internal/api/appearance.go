package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

// maxAppearanceBody is generous for three short names and small enough that a
// mistaken upload is refused rather than read.
const maxAppearanceBody = 1 << 12

// getAppearance answers with this account's saved choices. The session endpoint
// carries them too, for the first paint; this is what a later tab re-reads.
func (a *API) getAppearance(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	appearance, err := a.store.Appearance(r.Context(), user.ID)
	if err != nil {
		a.fail(w, errors.New("your appearance could not be loaded"), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, appearance)
}

// setAppearance saves all three choices together.
func (a *API) setAppearance(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAppearanceBody)
	user, _ := userFrom(r.Context())

	var body store.Appearance
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, errors.New("that request could not be read"), http.StatusBadRequest)
		return
	}
	saved, err := a.store.SetAppearance(r.Context(), user.ID, body)
	switch {
	case errors.Is(err, store.ErrUnknownAppearance):
		a.fail(w, err, http.StatusBadRequest)
		return
	case errors.Is(err, store.ErrNoAccount):
		a.fail(w, errors.New("that account no longer exists"), http.StatusUnauthorized)
		return
	case err != nil:
		// The page stays readable and says the choice was not saved, rather
		// than showing it as kept.
		a.fail(w, errors.New("your appearance could not be saved"), http.StatusServiceUnavailable)
		return
	}
	// Told to this account's other open tabs and devices, so they converge on
	// the choice instead of keeping the look they were loaded with.
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventAppearance})
	writeJSON(w, saved)
}
