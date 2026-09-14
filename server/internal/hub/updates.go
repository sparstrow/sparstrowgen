package hub

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// What each paired computer said about itself, and where its updates stand
// (spec US3). Held only while it is connected, like its providers.

type Runtime struct {
	Version     string
	Protocol    int
	SelfUpdates bool
	Update      protocol.UpdateStatus
	// Reported is false until the computer's hello arrives. Nothing is
	// concluded about a computer that has not said anything yet.
	Reported bool
}

func (m *machine) tooOld() bool {
	return m.runtime.Reported && m.runtime.Protocol < protocol.MinDaemonProtocol
}

// SetMachineHello records a paired computer's version, protocol and whether it
// can update itself, and tells the account's browsers whether it is too old.
func (h *Hub) SetMachineHello(userID, machineID, version string, proto int, selfUpdates bool) {
	h.mu.Lock()
	m := h.paired[machineID]
	if m != nil {
		m.runtime.Version, m.runtime.Protocol, m.runtime.SelfUpdates, m.runtime.Reported = version, proto, selfUpdates, true
	}
	h.mu.Unlock()
	if m != nil {
		h.BroadcastTo(userID, h.daemonEvent(userID))
	}
}

// MachineRuntime is what a connected computer reported. A computer that is not
// connected reports nothing.
func (h *Hub) MachineRuntime(machineID string) Runtime {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var r Runtime
	if m := h.paired[machineID]; m != nil {
		r = m.runtime
	}
	if r.Update.Kind == "" {
		r.Update.Kind = protocol.UpdateUnchecked
	}
	return r
}

func (h *Hub) MachineTooOld(machineID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := h.paired[machineID]
	return m != nil && m.tooOld()
}

// SetMachineUpdate records where a computer's update stands.
func (h *Hub) SetMachineUpdate(userID, machineID string, s protocol.UpdateStatus) {
	h.mu.Lock()
	m := h.paired[machineID]
	if m != nil {
		m.runtime.Update = s
	}
	h.mu.Unlock()
	if m != nil {
		h.BroadcastTo(userID, protocol.ClientEvent{Type: protocol.EventMachines})
	}
}

// DaemonTooOld is whether the computer this account's work goes to is too old
// to be sent any.
func (h *Hub) DaemonTooOld(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := h.machines[userID]
	if m == nil && h.primary[userID] != "" {
		m = h.paired[h.primary[userID]]
	}
	return m != nil && m.tooOld()
}

func (h *Hub) daemonEvent(userID string) protocol.ClientEvent {
	return protocol.ClientEvent{Type: protocol.EventDaemon, Online: h.DaemonOnline(userID), TooOld: h.DaemonTooOld(userID)}
}

// SendToMachine sends to one paired computer, and reports false when it is not
// connected.
func (h *Hub) SendToMachine(machineID string, msg protocol.ServerMessage) bool {
	h.mu.RLock()
	m := h.paired[machineID]
	h.mu.RUnlock()
	if m == nil {
		return false
	}
	return h.write(m.conn, msg)
}

func (h *Hub) write(conn *websocket.Conn, msg protocol.ServerMessage) bool {
	payload, err := json.Marshal(msg)
	if err != nil {
		h.log.Error("marshal server message", "err", err)
		return false
	}
	h.mu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, payload)
	h.mu.Unlock()
	return err == nil
}

// AskMachine is Ask for one particular computer rather than the account's
// current one.
func (h *Hub) AskMachine(ctx context.Context, userID, machineID string, msg protocol.ServerMessage) (protocol.DaemonMessage, error) {
	return h.ask(ctx, userID, msg, func(m protocol.ServerMessage) bool { return h.SendToMachine(machineID, m) })
}

func (h *Hub) ask(ctx context.Context, userID string, msg protocol.ServerMessage, send func(protocol.ServerMessage) bool) (protocol.DaemonMessage, error) {
	id := strconv.FormatUint(h.nextRequest.Add(1), 10)
	msg.RequestID = id

	// Buffered, so a reply that lands after the caller's context expired is
	// dropped by the garbage collector rather than blocking the read loop.
	reply := make(chan protocol.DaemonMessage, 1)
	h.mu.Lock()
	h.pending[id] = waiter{userID: userID, reply: reply}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.pending, id)
		h.mu.Unlock()
	}()

	if !send(msg) {
		return protocol.DaemonMessage{}, ErrDaemonOffline
	}

	select {
	case m := <-reply:
		return m, nil
	case <-ctx.Done():
		return protocol.DaemonMessage{}, ctx.Err()
	}
}
