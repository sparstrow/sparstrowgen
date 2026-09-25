package hub

import "github.com/sparstrow/sparstrowgen/server/internal/protocol"

// Files in a conversation (D-056) need a computer that understands them.

// SetChatsDir records where a computer keeps conversations' files. machineID
// empty is the legacy account-wide daemon (D-038).
func (h *Hub) SetChatsDir(userID, machineID, dir string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m := h.machineFor(userID, machineID); m != nil {
		m.runtime.ChatsDir = dir
	}
}

// ChatsDir is where a connected computer keeps conversations' files, or empty.
func (h *Hub) ChatsDir(userID, machineID string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if m := h.machineFor(userID, machineID); m != nil {
		return m.runtime.ChatsDir
	}
	return ""
}

// SupportsFiles is whether a connected computer can be given files and asked
// for its working folder. The legacy daemon is only ever run from source, so it
// is as new as the server; a paired one says which protocol it speaks.
func (h *Hub) SupportsFiles(userID, machineID string) bool {
	if machineID == "" {
		return true
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := h.paired[machineID]
	return m != nil && m.runtime.Reported && m.runtime.Protocol >= protocol.FilesProtocol
}

func (h *Hub) machineFor(userID, machineID string) *machine {
	if machineID == "" {
		return h.machines[userID]
	}
	if h.pairedUsers[machineID] != userID {
		return nil
	}
	return h.paired[machineID]
}
