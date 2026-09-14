// Package hub owns every live connection: the browsers watching, and the
// machines doing the work — each belonging to one account.
//
// The daemon dials out to us and nothing ever dials in to anyone's machine
// (AGENTS.md §3), so from here a daemon is just a websocket that appeared and
// proved which account it works for.
package hub

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// ErrDaemonOffline is returned by Ask when there is no machine to ask. It is a
// normal outcome and not a fault — a machine is allowed to be asleep — so
// callers answer it differently from a real error: a picker says "machine
// unreachable", not "something went wrong".
var ErrDaemonOffline = errors.New("the machine is not connected")

/* Every method that sends, reads or changes machine state takes an ACCOUNT.

There is no broadcast to everyone, and that is the point. Before accounts could
be more than one, "every browser" and "the owner's browsers" were the same set,
and a conversation event went to all of them. With a second account that is a
stream of somebody else's transcript. So the only way to send an event is to
name whose it is (docs/KnownGaps.md G-27). */

type Hub struct {
	mu sync.RWMutex

	// Browsers, and WHOSE they are. The session is carried here so that ending
	// a session can close the socket it opened: a websocket is authenticated
	// once, at the handshake, and then stays open for as long as the tab does.
	clients map[*websocket.Conn]client

	// One machine per account. A second connection for the same account
	// replaces the first rather than being rejected: a daemon that restarted
	// after a network drop should not be locked out by its own stale socket.
	// Several machines per account arrive with pairing (US2), where each gets
	// its own credential and identity.
	machines    map[string]*machine
	paired      map[string]*machine
	pairedUsers map[string]string
	primary     map[string]string

	// Waiters for daemon replies, by request id. See Ask.
	pending map[string]waiter
	// Request ids only have to be unique among the requests in flight in this
	// process, so a counter does the job.
	nextRequest atomic.Uint64

	log *slog.Logger
}

// client is who is on the other end of a browser socket.
type client struct {
	// UserID is the account. Used by DisconnectUser, which is what a password
	// change and "sign out everywhere" need, and by every event to decide who
	// receives it.
	UserID string
	// SessionHash identifies the one session, so an ordinary sign-out closes
	// only the tab that pressed it and not the account's other devices.
	SessionHash string
}

type machine struct {
	conn      *websocket.Conn
	providers []protocol.Provider
	runtime   Runtime
}

// waiter is a request waiting for a machine's answer, and the account whose
// machine may give it. A reply from any other account's machine is ignored,
// however it came by the request id.
type waiter struct {
	userID string
	reply  chan protocol.DaemonMessage
}

func New(log *slog.Logger) *Hub {
	return &Hub{
		clients:  map[*websocket.Conn]client{},
		machines: map[string]*machine{},
		paired:   map[string]*machine{}, pairedUsers: map[string]string{}, primary: map[string]string{},
		pending: map[string]waiter{},
		log:     log,
	}
}

// ---------------------------------------------------------------------------
// browsers
// ---------------------------------------------------------------------------

func (h *Hub) AddClient(c *websocket.Conn, userID, sessionHash string) {
	h.mu.Lock()
	h.clients[c] = client{UserID: userID, SessionHash: sessionHash}
	h.mu.Unlock()

	// Tell it what it needs to render immediately, rather than leaving the
	// surface guessing until the next event happens to arrive.
	h.sendTo(c, h.daemonEvent(userID))
	h.sendTo(c, protocol.ClientEvent{Type: protocol.EventProviders, Providers: h.Providers(userID)})
}

func (h *Hub) RemoveClient(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	_ = c.Close()
}

// DisconnectUser closes every browser socket belonging to one account.
//
// Called after the sessions have been deleted, not instead of deleting them.
// The database is what decides whether a token is valid; this only stops a
// connection that was authorised before that decision from outliving it.
func (h *Hub) DisconnectUser(userID string) int {
	return h.disconnect(func(c client) bool { return c.UserID == userID })
}

// DisconnectSession closes the socket or sockets opened under one session.
func (h *Hub) DisconnectSession(sessionHash string) int {
	if sessionHash == "" {
		return 0
	}
	return h.disconnect(func(c client) bool { return c.SessionHash == sessionHash })
}

func (h *Hub) disconnect(match func(client) bool) int {
	h.mu.Lock()
	var closing []*websocket.Conn
	for conn, c := range h.clients {
		if match(c) {
			closing = append(closing, conn)
			delete(h.clients, conn)
		}
	}
	h.mu.Unlock()

	// Closed outside the lock: Close can block, and holding the hub's lock
	// while it does would stall every other browser's events.
	for _, conn := range closing {
		_ = conn.Close()
	}
	if len(closing) > 0 {
		h.log.Info("closed revoked browser sockets", "count", len(closing))
	}
	return len(closing)
}

// BroadcastTo sends an event to every browser signed in to one account, and to
// nobody else.
func (h *Hub) BroadcastTo(userID string, ev protocol.ClientEvent) {
	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for conn, c := range h.clients {
		if c.UserID == userID {
			conns = append(conns, conn)
		}
	}
	h.mu.RUnlock()

	for _, c := range conns {
		h.sendTo(c, ev)
	}
}

func (h *Hub) sendTo(c *websocket.Conn, ev protocol.ClientEvent) {
	payload, err := json.Marshal(ev)
	if err != nil {
		h.log.Error("marshal client event", "err", err)
		return
	}
	h.mu.Lock()
	err = c.WriteMessage(websocket.TextMessage, payload)
	h.mu.Unlock()
	if err != nil {
		// A write failure means this browser is gone. Dropping it here keeps a
		// closed tab from being retried on every future event.
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
		_ = c.Close()
	}
}

// ---------------------------------------------------------------------------
// machines
// ---------------------------------------------------------------------------

// SetDaemon records the machine now connected for an account, closing any
// socket it replaces.
func (h *Hub) SetDaemon(userID string, c *websocket.Conn) {
	h.mu.Lock()
	var old *websocket.Conn
	if m := h.machines[userID]; m != nil {
		old = m.conn
	}
	h.machines[userID] = &machine{conn: c, providers: []protocol.Provider{}}
	h.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	h.BroadcastTo(userID, protocol.ClientEvent{Type: protocol.EventDaemon, Online: true})
}

// ClearDaemon drops a machine's connection, and reports whether it was still
// that account's current one.
//
// The answer matters. A daemon that reconnects has already replaced this socket
// through SetDaemon, and the old socket's read loop only notices afterwards —
// so a late teardown from a superseded connection must not announce that the
// machine has gone, and must not let its caller abandon the new daemon's work
// (docs/Bugs.md B-8).
func (h *Hub) ClearDaemon(userID string, c *websocket.Conn) bool {
	h.mu.Lock()
	m := h.machines[userID]
	current := m != nil && m.conn == c
	if current {
		// Availability is not knowledge we still have. Reporting the last
		// providers we saw would claim the machine is answering when it is not.
		delete(h.machines, userID)
	}
	h.mu.Unlock()
	_ = c.Close()
	if !current {
		return false
	}
	h.BroadcastTo(userID, protocol.ClientEvent{Type: protocol.EventDaemon, Online: false})
	h.BroadcastTo(userID, protocol.ClientEvent{Type: protocol.EventProviders, Providers: h.Providers(userID)})
	return true
}

func (h *Hub) DaemonOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.machines[userID] != nil {
		return true
	}
	for id, owner := range h.pairedUsers {
		if owner == userID && h.paired[id] != nil {
			return true
		}
	}
	return false
}

// SetPairedDaemon keeps every approved computer independently connected. The
// legacy owner token remains a separate compatibility route until migration.
func (h *Hub) SetPairedDaemon(userID, machineID string, c *websocket.Conn) {
	h.mu.Lock()
	old := h.paired[machineID]
	h.paired[machineID] = &machine{conn: c, providers: []protocol.Provider{}}
	h.pairedUsers[machineID] = userID
	h.primary[userID] = machineID
	h.mu.Unlock()
	if old != nil {
		_ = old.conn.Close()
	}
	h.BroadcastTo(userID, protocol.ClientEvent{Type: protocol.EventDaemon, Online: true})
}
func (h *Hub) ClearPairedDaemon(userID, machineID string, c *websocket.Conn) bool {
	h.mu.Lock()
	m := h.paired[machineID]
	current := m != nil && m.conn == c
	if current {
		delete(h.paired, machineID)
		delete(h.pairedUsers, machineID)
		if h.primary[userID] == machineID {
			delete(h.primary, userID)
			for id, owner := range h.pairedUsers {
				if owner == userID && h.paired[id] != nil {
					h.primary[userID] = id
					break
				}
			}
		}
	}
	h.mu.Unlock()
	_ = c.Close()
	if current {
		h.BroadcastTo(userID, h.daemonEvent(userID))
		h.BroadcastTo(userID, protocol.ClientEvent{Type: protocol.EventProviders, Providers: h.Providers(userID)})
	}
	return current
}
func (h *Hub) MachineOnline(machineID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.paired[machineID] != nil
}
func (h *Hub) MachineProviders(machineID string) []protocol.Provider {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := h.paired[machineID]
	if m == nil {
		return []protocol.Provider{}
	}
	out := make([]protocol.Provider, len(m.providers))
	copy(out, m.providers)
	return out
}
func (h *Hub) SetPairedProviders(userID, machineID string, p []protocol.Provider) {
	h.mu.Lock()
	m := h.paired[machineID]
	if m != nil {
		m.providers = p
	}
	h.mu.Unlock()
	if m != nil {
		h.BroadcastTo(userID, protocol.ClientEvent{Type: protocol.EventProviders, Providers: h.Providers(userID)})
	}
}
func (h *Hub) DisconnectMachine(machineID string) {
	h.mu.RLock()
	m := h.paired[machineID]
	h.mu.RUnlock()
	if m != nil {
		_ = m.conn.Close()
	}
}

// SetProviders records what an account's machine can run. Ignored when that
// account has no machine connected: a hello from a socket that has since been
// replaced must not resurrect it.
func (h *Hub) SetProviders(userID string, p []protocol.Provider) {
	h.mu.Lock()
	m := h.machines[userID]
	if m == nil && h.primary[userID] != "" {
		m = h.paired[h.primary[userID]]
	}
	if m != nil {
		m.providers = p
	}
	h.mu.Unlock()
	if m == nil {
		return
	}
	h.BroadcastTo(userID, protocol.ClientEvent{Type: protocol.EventProviders, Providers: h.Providers(userID)})
}

// SetHeadroom records a usage window for one provider on one account's machine.
//
// It arrives mid-turn because that is the only time a provider mentions one, so
// this is the one piece of provider state that is learned by doing work rather
// than by probing.
func (h *Hub) SetHeadroom(userID, provider string, head *protocol.Headroom) {
	h.mu.Lock()
	changed := false
	if m := h.machines[userID]; m != nil {
		for i := range m.providers {
			if m.providers[i].ID == provider {
				m.providers[i].Headroom = head
				changed = true
				break
			}
		}
	}
	h.mu.Unlock()
	if changed {
		h.BroadcastTo(userID, protocol.ClientEvent{Type: protocol.EventProviders, Providers: h.Providers(userID)})
	}
}

// Providers is what an account's machine can run. Never nil: an account with no
// machine has an empty list, and the surface renders that as its own state.
func (h *Hub) Providers(userID string) []protocol.Provider {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := h.machines[userID]
	if m == nil && h.primary[userID] != "" {
		m = h.paired[h.primary[userID]]
	}
	if m == nil {
		return []protocol.Provider{}
	}
	out := make([]protocol.Provider, len(m.providers))
	copy(out, m.providers)
	return out
}

// SendToDaemon returns false when the account's machine is unreachable. The
// caller must treat that as a real outcome and say so, not queue silently: a
// message that looks sent and never runs is worse than one that was refused.
func (h *Hub) SendToDaemon(userID string, msg protocol.ServerMessage) bool {
	h.mu.RLock()
	m := h.machines[userID]
	if m == nil && h.primary[userID] != "" {
		m = h.paired[h.primary[userID]]
	}
	h.mu.RUnlock()
	if m == nil {
		return false
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		h.log.Error("marshal server message", "err", err)
		return false
	}
	h.mu.Lock()
	err = m.conn.WriteMessage(websocket.TextMessage, payload)
	h.mu.Unlock()
	return err == nil
}

// ---------------------------------------------------------------------------
// asking a machine something
// ---------------------------------------------------------------------------

// Ask sends a message to an account's machine and waits for its reply.
//
// Every other exchange here is one-way: the server tells the daemon to run a
// turn, the daemon streams back what happens. Directory browsing is the first
// thing that needs an answer, because only the daemon can see the filesystem.
//
// Replies are matched by request id rather than by order, since a browser can
// have several pickers open and the daemon may answer them in any sequence.
// The waiter is always removed, on every path, or a client that gave up would
// leak a channel per keystroke.
func (h *Hub) Ask(ctx context.Context, userID string, msg protocol.ServerMessage) (protocol.DaemonMessage, error) {
	return h.ask(ctx, userID, msg, func(m protocol.ServerMessage) bool { return h.SendToDaemon(userID, m) })
}

// Deliver hands a machine's reply to whoever is waiting for it, and reports
// whether anyone was. A false means the message is an ordinary event and the
// caller should handle it normally — including a late reply to a request that
// has already been abandoned.
//
// Only a machine of the account that asked may answer. Request ids are a
// counter and easy to guess; the account is what makes a reply belong.
func (h *Hub) Deliver(userID string, m protocol.DaemonMessage) bool {
	if m.RequestID == "" {
		return false
	}
	h.mu.RLock()
	w, ok := h.pending[m.RequestID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	if w.userID != userID {
		h.log.Warn("ignored a reply from another account's machine", "request", m.RequestID)
		return true
	}
	select {
	case w.reply <- m:
	default:
		// Already answered. Only reachable if the daemon replied twice.
	}
	return true
}
