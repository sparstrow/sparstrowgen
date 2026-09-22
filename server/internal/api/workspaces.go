package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* Workspaces: separate areas of work inside one account (docs/Decisions.md
D-050).

A workspace is named by the routes that need one — `?workspace=` on the two
that list, a field in the body of the one that creates — and never by a header.
Multica resolves it from any of `X-Workspace-Slug`, `?workspace_slug`,
`X-Workspace-ID` or `?workspace_id`, whichever arrives; four spellings of one
thing is four places for the answer to differ from what the statement then
does. Here the workspace id goes into the statement that does the work, beside
the account, and there is nothing in between to disagree with. */

// failWorkspace answers a store error about a workspace. A workspace this
// account is not in and one that never existed are the same 404: telling them
// apart would confirm that an id belongs to somebody.
func (a *API) failWorkspace(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNoWorkspace):
		a.fail(w, err, http.StatusNotFound)
	case errors.Is(err, store.ErrWorkspaceName):
		a.fail(w, err, http.StatusBadRequest)
	default:
		a.fail(w, err, http.StatusInternalServerError)
	}
}

// workspaceFor reads the workspace a request is about and checks that this
// account is in it, before anything is read or written.
//
// Missing is a 400 rather than a silent default. "Whichever workspace we happen
// to pick" is how a conversation gets started in the wrong one, and the browser
// always knows which one it is looking at.
func (a *API) workspaceFor(w http.ResponseWriter, r *http.Request, id string) (protocol.Workspace, bool) {
	user, _ := userFrom(r.Context())
	if id == "" {
		a.fail(w, errors.New("which workspace? none was named"), http.StatusBadRequest)
		return protocol.Workspace{}, false
	}
	ws, err := a.store.Workspace(r.Context(), user.ID, id)
	if err != nil {
		a.failWorkspace(w, err)
		return protocol.Workspace{}, false
	}
	return ws, true
}

func (a *API) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	list, err := a.store.Workspaces(r.Context(), user.ID)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (a *API) createWorkspace(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	ws, err := a.store.CreateWorkspace(r.Context(), user.ID, body.Name)
	if err != nil {
		a.failWorkspace(w, err)
		return
	}
	// Every other tab this account has open gets the new one in its switcher.
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventWorkspaces})
	writeJSON(w, ws)
}

func (a *API) renameWorkspace(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	ws, err := a.store.RenameWorkspace(r.Context(), user.ID, chi.URLParam(r, "id"), body.Name)
	if err != nil {
		a.failWorkspace(w, err)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventWorkspaces})
	writeJSON(w, ws)
}
