package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// errDaemonOffline is a real outcome, not an internal error. Everything already
// said stays readable when the machine is unreachable; only sending something
// new is impossible, and the surface must say so before the owner types rather
// than after.
var errDaemonOffline = errors.New("your machine is unreachable, so nothing new can be sent")

// errTurnNotRunning answers a stop for a turn that has already ended. Ordinary
// rather than exceptional: the button and the last delta race every time.
var errTurnNotRunning = errors.New("that turn has already finished")

// errMachineWentAway is written into the transcript of a turn that was running
// when the daemon disconnected. Phrased as what happened rather than as a fault:
// a laptop closing mid-answer is ordinary, and the turn can simply be sent
// again.
// Deliberately one clause. The surface already adds "what arrived before it
// stopped is kept above" under every failure, and saying it here too printed
// the same reassurance twice in one box.
var errMachineWentAway = errors.New("your machine disconnected before this turn finished")

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
	// hub has to know whose it is in order to close it when that session ends
	// (docs/Decisions.md D-030).
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
	if !a.daemonAuthorised(r) {
		a.log.Warn("refused a daemon connection", "remote", r.RemoteAddr)
		a.fail(w, errors.New("this machine is not authorised"), http.StatusUnauthorized)
		return
	}
	up := a.upgrader()
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	a.log.Info("daemon connected", "remote", r.RemoteAddr)
	a.hub.SetDaemon(conn)
	defer func() {
		a.log.Info("daemon disconnected")
		// Only if this socket was still the current daemon. One that has already
		// reconnected has replaced it, and abandoning turns then would kill the
		// new connection's work on the strength of the old one's teardown.
		if a.hub.ClearDaemon(conn) {
			a.abandonTurns(errMachineWentAway.Error())
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
		if a.hub.Deliver(msg) {
			continue
		}
		a.handleDaemonMessage(msg)
	}
}
