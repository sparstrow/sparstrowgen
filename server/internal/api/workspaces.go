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

// ---------------------------------------------------------------------------
// which computers a workspace may use
// ---------------------------------------------------------------------------

// machinesIn is the ids of the computers a workspace is allowed to use. Every
// path that runs work, lists agents or browses folders goes through it, so the
// assignment decides the work rather than merely labelling it.
func (a *API) machinesIn(r *http.Request, workspaceID string) ([]string, error) {
	user, _ := userFrom(r.Context())
	machines, err := a.store.WorkspaceMachines(r.Context(), user.ID, workspaceID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(machines))
	for _, m := range machines {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// workspaceMachines answers both surfaces: the computers assigned to this
// workspace, and the account's other computers that could be.
func (a *API) workspaceMachines(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	ws, ok := a.workspaceFor(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	assigned, err := a.store.WorkspaceMachines(r.Context(), user.ID, ws.ID)
	if err != nil {
		a.failWorkspace(w, err)
		return
	}
	// Every computer the account has, each saying whether this workspace may
	// use it. One list rather than two, because the surface is a set of
	// checkboxes over all of them and computing the difference in the browser
	// is how the two get out of step.
	all, err := a.store.Machines(r.Context(), user.ID)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	in := map[string]bool{}
	for _, m := range assigned {
		in[m.ID] = true
	}
	type row struct {
		store.Machine
		Assigned bool `json:"assigned"`
		Online   bool `json:"online"`
	}
	out := make([]row, 0, len(all))
	for _, m := range all {
		out = append(out, row{Machine: m, Assigned: in[m.ID], Online: a.hub.MachineOnline(m.ID)})
	}
	writeJSON(w, out)
}

// setWorkspaceMachine adds or removes one computer from one workspace.
//
// POST with the wanted state rather than POST-to-add and DELETE-to-remove: a
// checkbox has a state, not an event, and sending the state means a click that
// arrives twice leaves the same answer.
func (a *API) setWorkspaceMachine(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	ws, ok := a.workspaceFor(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var body struct {
		Assigned bool `json:"assigned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	if err := a.store.SetWorkspaceMachine(r.Context(), user.ID, ws.ID, chi.URLParam(r, "machineId"), body.Assigned); err != nil {
		a.failWorkspace(w, err)
		return
	}
	// Every tab: this changes what can be run where, and the provider strip is
	// on screen in all of them.
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventWorkspaces})
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventMachines})
	a.workspaceMachines(w, r)
}

// failMachine answers a store error about a computer. Somebody else's computer
// and one that never existed are the same 404, for the reason every other
// lookup here gives.
func (a *API) failMachine(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrPairingUnavailable) {
		a.fail(w, errors.New("that computer does not exist"), http.StatusNotFound)
		return
	}
	a.fail(w, err, http.StatusInternalServerError)
}

// machineWorkspaces is the same assignment from the computer's end, which is
// the direction the owner asked for second: "or vice versa whick workspace
// needs to added to the machines".
func (a *API) machineWorkspaces(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	id := chi.URLParam(r, "id")
	// The computer has to be this account's before anything is said about it.
	if _, err := a.store.Machine(r.Context(), user.ID, id); err != nil {
		a.failMachine(w, err)
		return
	}
	in, err := a.store.MachineWorkspaces(r.Context(), user.ID, id)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	spaces, err := a.store.Workspaces(r.Context(), user.ID)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	using := map[string]bool{}
	for _, id := range in {
		using[id] = true
	}
	type row struct {
		protocol.Workspace
		Assigned bool `json:"assigned"`
	}
	out := make([]row, 0, len(spaces))
	for _, s := range spaces {
		out = append(out, row{Workspace: s, Assigned: using[s.ID]})
	}
	writeJSON(w, out)
}

func (a *API) setMachineWorkspace(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	id := chi.URLParam(r, "id")
	if _, err := a.store.Machine(r.Context(), user.ID, id); err != nil {
		a.failMachine(w, err)
		return
	}
	var body struct {
		Assigned bool `json:"assigned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	if err := a.store.SetWorkspaceMachine(r.Context(), user.ID, chi.URLParam(r, "workspaceId"), id, body.Assigned); err != nil {
		a.failWorkspace(w, err)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventWorkspaces})
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventMachines})
	a.machineWorkspaces(w, r)
}
