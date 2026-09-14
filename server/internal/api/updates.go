package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

// Keeping each computer's daemon updated (spec US3). The server decides nothing
// about an update: it stores the preference, passes requests to the computer,
// and reports what the computer says.

var (
	errComputerOffline  = errors.New("that computer is offline; it checks for updates when it reconnects")
	errCannotSelfUpdate = errors.New("this computer's version cannot update itself — download the installer and open it on that computer")
	errUpdateNoAnswer   = errors.New("that computer did not answer in time; try again")
	// errMachineTooOld refuses work for a computer below MinDaemonProtocol, saying
	// what to do rather than failing like an unreachable machine.
	errMachineTooOld = errors.New("your computer's sparstrowgen is too old for this app, so nothing new can be sent — update it in Settings → Updates. Everything already said stays readable")
)

// updateAskTimeout covers a check that goes on to download an installer.
const updateAskTimeout = 90 * time.Second

func (a *API) setAutomaticUpdates(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&body); err != nil || body.Enabled == nil {
		a.fail(w, errors.New("say whether automatic updates are on"), http.StatusBadRequest)
		return
	}
	m, err := a.store.SetAutomaticUpdates(r.Context(), u.ID, chi.URLParam(r, "id"), *body.Enabled)
	if errors.Is(err, store.ErrPairingUnavailable) {
		a.fail(w, errors.New("that machine is not available"), http.StatusNotFound)
		return
	}
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	// A computer that is not connected is told when it next connects.
	a.hub.SendToMachine(m.ID, protocol.ServerMessage{Type: protocol.ServerUpdatePreference, Automatic: body.Enabled})
	a.hub.BroadcastTo(u.ID, protocol.ClientEvent{Type: protocol.EventMachines})
	writeJSON(w, a.viewMachine(m))
}

func (a *API) checkForUpdates(w http.ResponseWriter, r *http.Request) {
	a.askForUpdate(w, r, protocol.ServerCheckUpdate)
}

func (a *API) applyUpdate(w http.ResponseWriter, r *http.Request) {
	a.askForUpdate(w, r, protocol.ServerApplyUpdate)
}

func (a *API) askForUpdate(w http.ResponseWriter, r *http.Request, kind string) {
	u, _ := userFrom(r.Context())
	m, err := a.store.Machine(r.Context(), u.ID, chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrPairingUnavailable) {
		a.fail(w, errors.New("that machine is not available"), http.StatusNotFound)
		return
	}
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	if !a.hub.MachineOnline(m.ID) {
		a.fail(w, errComputerOffline, http.StatusServiceUnavailable)
		return
	}
	if !a.hub.MachineRuntime(m.ID).SelfUpdates {
		a.fail(w, errCannotSelfUpdate, http.StatusConflict)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), updateAskTimeout)
	defer cancel()
	reply, err := a.hub.AskMachine(ctx, u.ID, m.ID, protocol.ServerMessage{Type: kind})
	if errors.Is(err, hub.ErrDaemonOffline) {
		a.fail(w, errComputerOffline, http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		a.fail(w, errUpdateNoAnswer, http.StatusGatewayTimeout)
		return
	}
	if reply.Update != nil {
		a.hub.SetMachineUpdate(u.ID, m.ID, *reply.Update)
	}
	writeJSON(w, a.viewMachine(m))
}
