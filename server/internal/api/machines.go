package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sparstrow/sparstrowgen/server/internal/auth"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

const pairingLifetime = 10 * time.Minute

type machineView struct {
	store.Machine
	Online    bool                `json:"online"`
	Providers []protocol.Provider `json:"providers"`
	// Updates (spec US3). Version is what the computer reports while it is
	// connected, and what it last reported otherwise.
	Version     string                `json:"version,omitempty"`
	TooOld      bool                  `json:"tooOld"`
	SelfUpdates bool                  `json:"selfUpdates"`
	Update      protocol.UpdateStatus `json:"update"`
}

func (a *API) viewMachine(m store.Machine) machineView {
	rt := a.hub.MachineRuntime(m.ID)
	v := machineView{
		Machine: m, Online: a.hub.MachineOnline(m.ID), Providers: a.hub.MachineProviders(m.ID),
		Version: m.Version, TooOld: a.hub.MachineTooOld(m.ID), SelfUpdates: rt.SelfUpdates, Update: rt.Update,
	}
	if rt.Version != "" {
		v.Version = rt.Version
	}
	return v
}
func (a *API) listMachines(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	ms, err := a.store.Machines(r.Context(), u.ID)
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	out := make([]machineView, 0, len(ms))
	for _, m := range ms {
		out = append(out, a.viewMachine(m))
	}
	writeJSON(w, out)
}
func (a *API) getMachine(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	m, err := a.store.Machine(r.Context(), u.ID, chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrPairingUnavailable) {
		a.fail(w, errors.New("that machine is not available"), 404)
		return
	}
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	writeJSON(w, a.viewMachine(m))
}
func (a *API) createPairing(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	token, hash, err := auth.NewToken()
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	p, err := a.store.CreatePairing(r.Context(), u.ID, hash, time.Now().Add(pairingLifetime))
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	writeJSON(w, map[string]any{"pairing": p, "launchUri": "sparstrowgen://pair?request=" + token})
}
func (a *API) getPairing(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	p, err := a.store.Pairing(r.Context(), u.ID, chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrPairingUnavailable) {
		a.fail(w, errors.New("that pairing request is no longer available"), 404)
		return
	}
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	writeJSON(w, p)
}
func (a *API) approvePairing(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	m, err := a.store.ApprovePairing(r.Context(), u.ID, chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrPairingUnavailable) {
		a.fail(w, errors.New("that pairing request cannot be approved"), 409)
		return
	}
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	a.hub.BroadcastTo(u.ID, protocol.ClientEvent{Type: protocol.EventMachines})
	writeJSON(w, a.viewMachine(m))
}
func (a *API) revokeMachine(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	m, err := a.store.RevokeMachine(r.Context(), u.ID, chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrPairingUnavailable) {
		a.fail(w, errors.New("that machine is not available"), 404)
		return
	}
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	a.hub.DisconnectMachine(m.ID)
	a.hub.BroadcastTo(u.ID, protocol.ClientEvent{Type: protocol.EventMachines})
	w.WriteHeader(http.StatusNoContent)
}

// daemonPair is intentionally outside session auth. The opaque one-time
// request is the credential, and its server-side hash is spent atomically.
func (a *API) daemonPair(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Request string `json:"request"`
		Name    string `json:"name"`
		// Current is the credential this computer already holds, if any. An
		// older daemon sends none, which is treated as a first pairing.
		Current string `json:"current"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		a.fail(w, errors.New("that pairing request could not be read"), 400)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		body.Name = "This computer"
	}
	if len(body.Name) > 120 {
		a.fail(w, errors.New("computer name is too long"), 400)
		return
	}
	credential, credentialHash, err := auth.NewToken()
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	var currentHash []byte
	if body.Current != "" {
		currentHash = auth.HashToken(body.Current)
	}
	_, machineID, alreadyPaired, err := a.store.ClaimPairing(r.Context(), auth.HashToken(body.Request), credentialHash, currentHash, body.Name)
	if errors.Is(err, store.ErrPairingUnavailable) {
		a.fail(w, errors.New("that pairing request has expired or was already used"), 409)
		return
	}
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	if alreadyPaired {
		writeJSON(w, map[string]any{"machineId": machineID, "alreadyPaired": true})
		return
	}
	writeJSON(w, map[string]string{"machineId": machineID, "credential": credential})
}

func (a *API) declinePairing(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	err := a.store.DeclinePairing(r.Context(), u.ID, chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrPairingUnavailable) {
		a.fail(w, errors.New("that pairing request was already approved, declined or has expired"), 409)
		return
	}
	if err != nil {
		a.fail(w, err, 500)
		return
	}
	a.hub.BroadcastTo(u.ID, protocol.ClientEvent{Type: protocol.EventMachines})
	w.WriteHeader(http.StatusNoContent)
}
