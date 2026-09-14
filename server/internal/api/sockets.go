package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// errDaemonOffline is a real outcome, not an internal error. Everything already
// said stays readable when the machine is unreachable; only sending something
// new is impossible, and the surface must say so before the person types rather
// than after.
var errDaemonOffline = errors.New("your machine is unreachable, so nothing new can be sent")

// errTurnNotRunning answers a stop for a turn that has already ended. Ordinary
// rather than exceptional: the button and the last delta race every time.
var errTurnNotRunning = errors.New("that turn has already finished")

// errMachineWentAway is written into the transcript of a turn that was running
// when the daemon disconnected. Phrased as what happened rather than as a fault:
// a laptop closing mid-answer is ordinary, and the turn can simply be sent
// again.
var errMachineWentAway = errors.New("your machine disconnected before this turn finished")

// errNoOwnerAccount refuses the shared-token daemon before the account it works
// for exists. Nothing it did could be shown to anybody.
var errNoOwnerAccount = errors.New("the owner account does not exist yet — create it, then restart the daemon")

// errMachineDisconnected tells a computer its credential can never work again —
// disconnected, or declined or expired before approval — so it stops retrying.
var errMachineDisconnected = errors.New("this computer was disconnected, or its pairing was declined or expired — pair it again to connect")

func defaultFolder() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func (a *API) browserSocket(w http.ResponseWriter, r *http.Request) {
	// Behind requireSession, so this is always present. Registered WITH the
	// socket rather than merely checked before it: a websocket is authenticated
	// once, at the handshake, and then lives as long as the tab does — so the
	// hub has to know whose it is, both to close it when that session ends
	// (docs/Decisions.md D-030) and to send it only that account's events.
	user, token := userFrom(r.Context())

	up := a.upgrader()
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	a.hub.AddClient(conn, user.ID, string(auth.HashToken(token)))
	defer a.hub.RemoveClient(conn)

	// Browsers only listen. Everything they can do goes through HTTP, so a
	// dropped socket costs live updates and never an action.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (a *API) daemonSocket(w http.ResponseWriter, r *http.Request) {
	// Checked BEFORE the upgrade, so an unauthorised dialler gets a plain 401
	// rather than a working websocket that is then closed — and so that nothing
	// is registered in the hub on the strength of a connection we are about to
	// reject.
	presented := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	legacy := a.daemonAuthorised(r)
	paired, pairedUserID, pairedOK, err := a.store.MachineForCredential(r.Context(), auth.HashToken(presented))
	if err != nil {
		a.fail(w, errors.New("could not verify this machine"), http.StatusServiceUnavailable)
		return
	}
	if !legacy && !pairedOK {
		// 403 and 401 mean different things to the daemon. A credential that
		// can never work again must be forgotten; one still waiting for
		// approval must keep trying. Both are refused either way.
		refused, err := a.store.CredentialRefused(r.Context(), auth.HashToken(presented))
		if err != nil {
			a.fail(w, errors.New("could not verify this machine"), http.StatusServiceUnavailable)
			return
		}
		if refused {
			a.fail(w, errMachineDisconnected, http.StatusForbidden)
			return
		}
		a.log.Warn("refused a daemon connection", "remote", r.RemoteAddr)
		a.fail(w, errors.New("this machine is not authorised"), http.StatusUnauthorized)
		return
	}

	// The shared token proves "the owner's machine", so this machine works for
	// the owner's account. Pairing (US2) replaces this with a credential per
	// computer that names its own account.
	owner, exists, err := a.store.UserByEmail(r.Context(), a.cfg.OwnerEmail)
	if err != nil {
		a.fail(w, errors.New("could not reach the database"), http.StatusServiceUnavailable)
		return
	}
	if !legacy {
		owner.ID = pairedUserID
	} else if !exists {
		a.fail(w, errNoOwnerAccount, http.StatusConflict)
		return
	}

	up := a.upgrader()
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	a.log.Info("daemon connected", "remote", r.RemoteAddr, "account", owner.Email)
	if pairedOK {
		a.hub.SetPairedDaemon(owner.ID, paired.ID, conn)
		a.hub.BroadcastTo(owner.ID, protocol.ClientEvent{Type: protocol.EventMachines})
	} else {
		a.hub.SetDaemon(owner.ID, conn)
	}
	defer func() {
		a.log.Info("daemon disconnected", "account", owner.Email)
		// Only if this socket was still the account's current machine. One that
		// has already reconnected has replaced it, and abandoning turns then
		// would kill the new connection's work on the strength of the old one's
		// teardown.
		current := false
		if pairedOK {
			current = a.hub.ClearPairedDaemon(owner.ID, paired.ID, conn)
			if current {
				a.hub.BroadcastTo(owner.ID, protocol.ClientEvent{Type: protocol.EventMachines})
			}
		} else {
			current = a.hub.ClearDaemon(owner.ID, conn)
		}
		if current {
			a.abandonTurns(owner.ID, errMachineWentAway.Error())
		}
	}()

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg protocol.DaemonMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			a.log.Warn("bad daemon message", "err", err)
			continue
		}
		// A reply to something the server asked (a directory listing) belongs to
		// whoever is waiting for it, not to the event handling below.
		if a.hub.Deliver(owner.ID, msg) {
			continue
		}
		if pairedOK && msg.Type == protocol.DaemonHello {
			a.hub.SetPairedProviders(owner.ID, paired.ID, msg.Providers)
			a.hub.SetMachineHello(owner.ID, paired.ID, msg.Version, msg.Protocol, msg.SelfUpdates)
			if msg.Version != "" {
				if err := a.store.RecordMachineVersion(r.Context(), paired.ID, msg.Version); err != nil {
					a.log.Warn("could not record the daemon version", "machine", paired.ID, "err", err)
				}
			}
			// Told on every connection, so a change made while it was offline
			// arrives. A daemon older than US3 ignores it.
			automatic := paired.AutomaticUpdates
			a.hub.SendToMachine(paired.ID, protocol.ServerMessage{Type: protocol.ServerUpdatePreference, Automatic: &automatic})
			a.hub.BroadcastTo(owner.ID, protocol.ClientEvent{Type: protocol.EventMachines})
			continue
		}
		if pairedOK && msg.Type == protocol.DaemonUpdateStatus && msg.Update != nil {
			a.hub.SetMachineUpdate(owner.ID, paired.ID, *msg.Update)
			continue
		}
		a.handleDaemonMessage(owner.ID, msg)
	}
}
