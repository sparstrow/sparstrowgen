package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

// maxProfileBody covers a name and a short description with room to spare, and
// refuses a mistaken upload rather than reading it.
const maxProfileBody = 1 << 13

func (a *API) getProfile(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	profile, err := a.store.Profile(r.Context(), user.ID)
	if err != nil {
		a.fail(w, errors.New("your profile could not be loaded"), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, profile)
}

func (a *API) setProfile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxProfileBody)
	user, _ := userFrom(r.Context())

	var body store.Profile
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, errors.New("that request could not be read"), http.StatusBadRequest)
		return
	}
	saved, err := a.store.SetProfile(r.Context(), user.ID, body)
	switch {
	case errors.Is(err, store.ErrNameTooLong), errors.Is(err, store.ErrBioTooLong):
		a.fail(w, err, http.StatusBadRequest)
		return
	case errors.Is(err, store.ErrNoAccount):
		a.fail(w, errors.New("that account no longer exists"), http.StatusUnauthorized)
		return
	case err != nil:
		a.fail(w, errors.New("your profile could not be saved"), http.StatusServiceUnavailable)
		return
	}
	// This account's other tabs and devices converge on the new name, the same
	// way they do on a new appearance.
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventProfile})
	writeJSON(w, saved)
}

// getAvatar serves this account's own picture. It is deliberately only ever the
// signed-in account's: there is no id in the path, so no endpoint here can be
// walked to enumerate other people's pictures.
func (a *API) getAvatar(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	avatar, err := a.store.Avatar(r.Context(), user.ID)
	if errors.Is(err, store.ErrNoAvatar) {
		a.fail(w, err, http.StatusNotFound)
		return
	}
	if err != nil {
		a.fail(w, errors.New("that picture could not be loaded"), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", avatar.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(avatar.Bytes)))
	// Private, because it belongs to whoever this session is — a shared cache
	// must never hand one account's picture to the next request. The browser may
	// keep it, and the URL carries the upload time, so a new picture is a new
	// URL and an unchanged one is not re-fetched.
	w.Header().Set("Cache-Control", "private, max-age=300")
	// The bytes came from a person and are served from our own origin. Even
	// though only image types are stored, this says outright that the browser
	// must not reinterpret them as something executable.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")
	_, _ = w.Write(avatar.Bytes)
}

// putAvatar takes the raw image as the body, with its type in Content-Type.
// Not multipart: there is exactly one file and no fields beside it, so a form
// encoding would be ceremony around a single blob.
//
// POST rather than PUT, though replacing the one picture is what PUT is for.
// The CORS preflight allows GET, POST, PATCH and DELETE, so a PUT from the
// browser is refused before it is sent — curl never sees it, because curl does
// not preflight. Widening the allowed methods for this one route is a bigger
// change than using the verb every other write here already uses.
func (a *API) putAvatar(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	contentType := r.Header.Get("Content-Type")
	if !store.AvatarTypeAllowed(contentType) {
		a.fail(w, store.ErrAvatarType, http.StatusUnsupportedMediaType)
		return
	}
	// One more than the limit, so a body exactly at the limit is accepted and
	// anything past it is caught here rather than after reading all of it.
	body, err := io.ReadAll(io.LimitReader(r.Body, store.MaxAvatarBytes+1))
	if err != nil {
		a.fail(w, errors.New("that picture could not be read"), http.StatusBadRequest)
		return
	}
	if len(body) > store.MaxAvatarBytes {
		a.fail(w, store.ErrAvatarTooBig, http.StatusRequestEntityTooLarge)
		return
	}
	switch err := a.store.SetAvatar(r.Context(), user.ID, contentType, body); {
	case errors.Is(err, store.ErrAvatarType):
		a.fail(w, err, http.StatusUnsupportedMediaType)
		return
	case errors.Is(err, store.ErrAvatarTooBig):
		a.fail(w, err, http.StatusRequestEntityTooLarge)
		return
	case err != nil:
		a.fail(w, errors.New("that picture could not be saved"), http.StatusServiceUnavailable)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventProfile})
	a.getProfile(w, r)
}

func (a *API) deleteAvatar(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	if err := a.store.DeleteAvatar(r.Context(), user.ID); err != nil {
		a.fail(w, errors.New("that picture could not be removed"), http.StatusServiceUnavailable)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventProfile})
	a.getProfile(w, r)
}
