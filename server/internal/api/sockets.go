package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// errDaemonOffline is a real outcome, not an internal error. Everything already
// said stays readable when the machine is unreachable; only sending something
// new is impossible, and the surface must say so before the owner types rather
// than after.
var errDaemonOffline = errors.New("your machine is unreachable, so nothing new can be sent")

func defaultFolder() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func (a *API) browserSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	a.hub.AddClient(conn)
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	a.log.Info("daemon connected", "remote", r.RemoteAddr)
	a.hub.SetDaemon(conn)
	defer func() {
		a.log.Info("daemon disconnected")
		a.hub.ClearDaemon(conn)
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
