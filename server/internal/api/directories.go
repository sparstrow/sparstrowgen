package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// listDirectories asks the daemon what is inside a path.
//
// The server never touches the filesystem itself. Today it happens to run on
// the same machine, so os.ReadDir would appear to work — and would quietly
// start listing the SERVER's disk the day this is deployed to Coolify, showing
// a container's directories as if they were the owner's. Going through the
// daemon is the same amount of work and cannot drift into that.
func (a *API) listDirectories(w http.ResponseWriter, r *http.Request) {
	// Long enough for a spun-down drive, short enough that a wedged daemon does
	// not hold the request open.
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	reply, err := a.hub.Ask(ctx, protocol.ServerMessage{
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

// recentFolders backs the picker's shortcut list.
func (a *API) recentFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := a.store.RecentFolders(r.Context(), 8)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string][]string{"folders": folders})
}
