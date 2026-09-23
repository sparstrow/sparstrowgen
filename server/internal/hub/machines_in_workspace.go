package hub

import (
	"context"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* Which computer a piece of work runs on.

Until workspaces there was no choice to make: the hub kept one "primary"
machine per account — whichever connected most recently — and every turn,
folder listing and provider strip used it. With several computers paired that
was already arbitrary; it simply never came up, because nobody had a way to say
which they meant.

The owner gave it one: "I need to choose which machine needs added to that
workspace or vice versa whick workspace needs to added to the machines." So a
workspace's assigned computers (migration 00018) are the set, and everything
below picks from inside it. The assignment is not a label — it is what the work
is routed by, which is the only thing that makes it worth setting. */

// Target is the computer a workspace's work should go to.
type Target struct {
	// MachineID is the paired computer chosen. EMPTY means the legacy
	// account-wide daemon: the shared DAEMON_TOKEN route, which is off on
	// anything deployed (docs/Decisions.md D-038) and has no machine row to
	// scope by. It predates machines entirely, so a workspace cannot narrow it
	// and this is the one thing that ignores the assignment.
	MachineID string
}

// Target picks the computer for a workspace, from the computers that workspace
// is allowed to use. Reports false when none of them is connected.
//
// The choice among several online is the account's primary — the same rule as
// before, now narrowed to the workspace's set — falling back to any of them.
// Which one it lands on is not yet something a person can see or change; that
// is docs/KnownGaps.md G-44, and the honest place for it is the conversation
// rather than a global preference.
func (h *Hub) Target(userID string, allowed []string) (Target, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// The legacy route first: it is the whole account's machine and no
	// workspace owns it.
	if h.machines[userID] != nil {
		return Target{}, true
	}
	if len(allowed) == 0 {
		return Target{}, false
	}
	// The primary, when the workspace is allowed to use it.
	primary := h.primary[userID]
	for _, id := range allowed {
		if id == primary && h.paired[id] != nil && h.pairedUsers[id] == userID {
			return Target{MachineID: id}, true
		}
	}
	for _, id := range allowed {
		if h.paired[id] != nil && h.pairedUsers[id] == userID {
			return Target{MachineID: id}, true
		}
	}
	return Target{}, false
}

// OnlineIn reports whether any of a workspace's computers is connected.
func (h *Hub) OnlineIn(userID string, allowed []string) bool {
	_, ok := h.Target(userID, allowed)
	return ok
}

// ProvidersIn is what a workspace can run: the agents on the computer its work
// would go to. A workspace with no computer assigned, or none connected, has an
// empty list — which the surfaces already render as their own state.
func (h *Hub) ProvidersIn(userID string, allowed []string) []protocol.Provider {
	target, ok := h.Target(userID, allowed)
	if !ok {
		return []protocol.Provider{}
	}
	if target.MachineID == "" {
		return h.Providers(userID)
	}
	return h.MachineProviders(target.MachineID)
}

// SendToWorkspaceDaemon sends to the computer a workspace's work should go to,
// and reports which one it went to.
//
// The id comes back because the caller has to remember it: stopping a turn has
// to reach the computer that is running it, and "the account's current machine"
// is a different computer the moment another one connects.
func (h *Hub) SendToWorkspaceDaemon(userID string, allowed []string, msg protocol.ServerMessage) (string, bool) {
	target, ok := h.Target(userID, allowed)
	if !ok {
		return "", false
	}
	if target.MachineID == "" {
		return "", h.SendToDaemon(userID, msg)
	}
	return target.MachineID, h.SendToMachine(target.MachineID, msg)
}

// SendToRunningMachine sends to the computer a turn is already on. An empty id
// is the legacy account-wide daemon, which is where that turn started.
func (h *Hub) SendToRunningMachine(userID, machineID string, msg protocol.ServerMessage) bool {
	if machineID == "" {
		return h.SendToDaemon(userID, msg)
	}
	return h.SendToMachine(machineID, msg)
}

// AskIn is Ask, restricted to a workspace's computers.
func (h *Hub) AskIn(ctx context.Context, userID string, allowed []string, msg protocol.ServerMessage) (protocol.DaemonMessage, error) {
	target, ok := h.Target(userID, allowed)
	if !ok {
		return protocol.DaemonMessage{}, ErrDaemonOffline
	}
	if target.MachineID == "" {
		return h.Ask(ctx, userID, msg)
	}
	return h.AskMachine(ctx, userID, target.MachineID, msg)
}
