package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// listDirectories asks this account's machine what is inside a path.
//
// The server never touches the filesystem itself. os.ReadDir would list the
// SERVER's disk the day this is deployed, showing a container's directories as
// if they were the person's. Going through the daemon is the same amount of
// work and cannot drift into that — and only ever reaches the machine of the
// account asking.
func (a *API) listDirectories(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	// Long enough for a spun-down drive, short enough that a wedged daemon does
	// not hold the request open.
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	reply, err := a.hub.Ask(ctx, user.ID, protocol.ServerMessage{
		Type: protocol.ServerListDir,
		Path: r.URL.Query().Get("path"),
	})
	switch {
	case errors.Is(err, hub.ErrDaemonOffline):
		// Not a fault. The picker says the machine is unreachable, which is a
		// different sentence from "something went wrong".
		a.fail(w, errDaemonOffline, http.StatusServiceUnavailable)
		return
	case errors.Is(err, context.DeadlineExceeded):
		a.fail(w, errors.New("your machine did not answer in time"), http.StatusGatewayTimeout)
		return
	case err != nil:
		a.fail(w, err, http.StatusInternalServerError)
		return
	case reply.Listing == nil:
		a.fail(w, errors.New("the machine sent an empty directory listing"), http.StatusInternalServerError)
		return
	}
	writeJSON(w, reply.Listing)
}

// recentFolders backs the picker's shortcut list: this account's folders only.
func (a *API) recentFolders(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	folders, err := a.store.RecentFolders(r.Context(), user.ID, 8)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string][]string{"folders": folders})
}
