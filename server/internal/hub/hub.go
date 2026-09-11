// Package hub owns every live connection: the browsers watching, and the one
// daemon doing the work.
//
// The daemon dials out to us and nothing ever dials in to the owner's machine
// (AGENTS.md §3), so from here a daemon is just a websocket that appeared.
package hub

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// ErrDaemonOffline is returned by Ask when there is no machine to ask. It is a
// normal outcome and not a fault — the owner's machine is allowed to be asleep —
// so callers answer it differently from a real error: a picker says "machine
// unreachable", not "something went wrong".
var ErrDaemonOffline = errors.New("the machine is not connected")

type Hub struct {
	mu sync.RWMutex

	clients map[*websocket.Conn]bool

	// One daemon, because one machine. A second connection replaces the first
	// rather than being rejected: a daemon that restarted after a network drop
	// should not be locked out by its own stale socket.
	daemon    *websocket.Conn
	providers []protocol.Provider

	// Waiters for daemon replies, by request id. See Ask.
	pending map[string]chan protocol.DaemonMessage
	// Request ids only have to be unique among the requests in flight in this
	// process, so a counter does the job and saves a uuid dependency the module
	// does not otherwise have — every other id here comes from Postgres.
	nextRequest atomic.Uint64

	log *slog.Logger

	// OnDaemonMessage is set by the server so the hub stays free of database
	// and turn-handling concerns.
	OnDaemonMessage func(protocol.DaemonMessage)
}

func New(log *slog.Logger) *Hub {
	return &Hub{clients: map[*websocket.Conn]bool{}, log: log, providers: []protocol.Provider{}}
}

// ---------------------------------------------------------------------------
// browsers
// ---------------------------------------------------------------------------

func (h *Hub) AddClient(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	// Tell it what it needs to render immediately, rather than leaving the
	// surface guessing until the next event happens to arrive.
	h.sendTo(c, protocol.ClientEvent{Type: protocol.EventDaemon, Online: h.DaemonOnline()})
	h.sendTo(c, protocol.ClientEvent{Type: protocol.EventProviders, Providers: h.Providers()})
}

func (h *Hub) RemoveClient(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	_ = c.Close()
}

func (h *Hub) Broadcast(ev protocol.ClientEvent) {
	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		conns = append(conns, c)
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
// daemon
// ---------------------------------------------------------------------------

func (h *Hub) SetDaemon(c *websocket.Conn) {
	h.mu.Lock()
	old := h.daemon
	h.daemon = c
	h.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	h.Broadcast(protocol.ClientEvent{Type: protocol.EventDaemon, Online: true})
}

func (h *Hub) ClearDaemon(c *websocket.Conn) {
	h.mu.Lock()
	if h.daemon == c {
		h.daemon = nil
		// Availability is not knowledge we still have. Reporting the last
		// providers we saw would claim the machine is answering when it is not.
		h.providers = []protocol.Provider{}
	}
	h.mu.Unlock()
	_ = c.Close()
	h.Broadcast(protocol.ClientEvent{Type: protocol.EventDaemon, Online: false})
	h.Broadcast(protocol.ClientEvent{Type: protocol.EventProviders, Providers: h.Providers()})
}

func (h *Hub) DaemonOnline() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.daemon != nil
}

func (h *Hub) SetProviders(p []protocol.Provider) {
	h.mu.Lock()
	h.providers = p
	h.mu.Unlock()
	h.Broadcast(protocol.ClientEvent{Type: protocol.EventProviders, Providers: p})
}

// SetHeadroom records a usage window for one provider and tells every browser.
//
// It arrives mid-turn because that is the only time a provider mentions one, so
// this is the one piece of provider state that is learned by doing work rather
// than by probing.
func (h *Hub) SetHeadroom(provider string, head *protocol.Headroom) {
	h.mu.Lock()
	changed := false
	for i := range h.providers {
		if h.providers[i].ID == provider {
			h.providers[i].Headroom = head
			changed = true
			break
		}
	}
	h.mu.Unlock()
	if changed {
		h.Broadcast(protocol.ClientEvent{Type: protocol.EventProviders, Providers: h.Providers()})
	}
}

func (h *Hub) Providers() []protocol.Provider {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]protocol.Provider, len(h.providers))
	copy(out, h.providers)
	return out
}

// SendToDaemon returns false when the machine is unreachable. The caller must
// treat that as a real outcome and say so, not queue silently: a message that
// looks sent and never runs is worse than one that was refused.
func (h *Hub) SendToDaemon(msg protocol.ServerMessage) bool {
	h.mu.RLock()
	d := h.daemon
	h.mu.RUnlock()
	if d == nil {
		return false
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		h.log.Error("marshal server message", "err", err)
		return false
	}
	h.mu.Lock()
	err = d.WriteMessage(websocket.TextMessage, payload)
	h.mu.Unlock()
	return err == nil
}

// ---------------------------------------------------------------------------
// asking the daemon something
// ---------------------------------------------------------------------------

// Ask sends a message and waits for the daemon's reply to it.
//
// Every other exchange here is one-way: the server tells the daemon to run a
// turn, the daemon streams back what happens. Directory browsing is the first
// thing that needs an answer, because only the daemon can see the filesystem —
// the server is meant to run somewhere else entirely.
//
// Replies are matched by request id rather than by order, since a browser can
// have several pickers open and the daemon may answer them in any sequence.
// The waiter is always removed, on every path, or a client that gave up would
// leak a channel per keystroke.
func (h *Hub) Ask(ctx context.Context, msg protocol.ServerMessage) (protocol.DaemonMessage, error) {
	id := strconv.FormatUint(h.nextRequest.Add(1), 10)
	msg.RequestID = id

	// Buffered, so a reply that lands after the caller's context expired is
	// dropped by the garbage collector rather than blocking the read loop.
	reply := make(chan protocol.DaemonMessage, 1)
	h.mu.Lock()
	if h.pending == nil {
		h.pending = map[string]chan protocol.DaemonMessage{}
	}
	h.pending[id] = reply
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.pending, id)
		h.mu.Unlock()
	}()

	if !h.SendToDaemon(msg) {
		return protocol.DaemonMessage{}, ErrDaemonOffline
	}

	select {
	case m := <-reply:
		return m, nil
	case <-ctx.Done():
		return protocol.DaemonMessage{}, ctx.Err()
	}
}

// Deliver hands a daemon reply to whoever is waiting for it, and reports
// whether anyone was. A false means the message is an ordinary event and the
// caller should handle it normally — including a late reply to a request that
// has already been abandoned.
func (h *Hub) Deliver(m protocol.DaemonMessage) bool {
	if m.RequestID == "" {
		return false
	}
	h.mu.RLock()
	ch, ok := h.pending[m.RequestID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case ch <- m:
	default:
		// Already answered. Only reachable if the daemon replied twice.
	}
	return true
}
