package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* Which computers a workspace may use (migration 00018). The owner: "I need a
way and a settings to be able to add the machine to the workspace. If I have
multiple machine in my account. I need to choose which machine needs added to
that workspace or vice versa whick workspace needs to added to the machines."

The test that earns this feature is the routing one at the bottom: taking a
computer out of a workspace has to stop that workspace's work reaching it. An
assignment that only labels would be a setting that lies. */

type assignedMachine struct {
	store.Machine
	Assigned bool `json:"assigned"`
	Online   bool `json:"online"`
}

type assignedWorkspace struct {
	protocol.Workspace
	Assigned bool `json:"assigned"`
}

func (r *rig) workspaceMachines(workspaceID string) []assignedMachine {
	r.t.Helper()
	res := r.get("/api/workspaces/" + workspaceID + "/machines")
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("workspace machines: %s", res.Status)
	}
	var out []assignedMachine
	decodeInto(r.t, res, &out)
	return out
}

func (r *rig) machineWorkspaces(machineID string) []assignedWorkspace {
	r.t.Helper()
	res := r.get("/api/machines/" + machineID + "/workspaces")
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("machine workspaces: %s", res.Status)
	}
	var out []assignedWorkspace
	decodeInto(r.t, res, &out)
	return out
}

// assign sets it from the workspace's end; assignFromMachine from the
// computer's. Both directions exist because the owner asked for both, and both
// write the same row — which is the thing worth asserting.
func (r *rig) assign(workspaceID, machineID string, assigned bool) *http.Response {
	r.t.Helper()
	return r.post("/api/workspaces/"+workspaceID+"/machines/"+machineID, map[string]any{"assigned": assigned})
}

func (r *rig) assignFromMachine(machineID, workspaceID string, assigned bool) *http.Response {
	r.t.Helper()
	return r.post("/api/machines/"+machineID+"/workspaces/"+workspaceID, map[string]any{"assigned": assigned})
}

func assignedIn(list []assignedMachine, id string) bool {
	for _, m := range list {
		if m.ID == id {
			return m.Assigned
		}
	}
	return false
}

// A computer arrives offered everywhere, because a computer that has just been
// paired and can run nothing is a computer somebody has to go and repair.
func TestANewComputerIsOfferedInEveryWorkspace(t *testing.T) {
	r := newRig(t)
	work := r.newWorkspace("Work")
	personal := r.workspaces()[0]

	machineID, _, conn := r.pairComputer("DESKTOP-RIVER")
	defer conn.Close()

	for _, ws := range []protocol.Workspace{personal, work} {
		if !assignedIn(r.workspaceMachines(ws.ID), machineID) {
			t.Errorf("%s was not offered the computer that was just paired", ws.Name)
		}
	}

	// And a workspace made AFTER the computer gets it too, or making one would
	// be followed by a settings trip nobody expects.
	later := r.newWorkspace("Later")
	if !assignedIn(r.workspaceMachines(later.ID), machineID) {
		t.Error("a workspace made after the computer was paired did not get it")
	}
}

func TestAComputerIsAssignedFromEitherEnd(t *testing.T) {
	r := newRig(t)
	work := r.newWorkspace("Work")
	machineID, _, conn := r.pairComputer("DESKTOP-RIVER")
	defer conn.Close()

	// Out, from the workspace's end.
	if res := r.assign(work.ID, machineID, false); res.StatusCode != http.StatusOK {
		t.Fatalf("removing: %s", res.Status)
	}
	if assignedIn(r.workspaceMachines(work.ID), machineID) {
		t.Error("the computer is still assigned after being removed")
	}
	for _, w := range r.machineWorkspaces(machineID) {
		if w.ID == work.ID && w.Assigned {
			t.Error("the computer's own list still claims that workspace")
		}
	}

	// Back in, from the computer's end. The same row, so the other direction
	// has to agree.
	if res := r.assignFromMachine(machineID, work.ID, true); res.StatusCode != http.StatusOK {
		t.Fatalf("adding from the machine: %s", res.Status)
	}
	if !assignedIn(r.workspaceMachines(work.ID), machineID) {
		t.Error("assigning from the computer's end did not show up on the workspace's")
	}

	// Assigning twice is not an error: a checkbox has a state, not an event.
	if res := r.assignFromMachine(machineID, work.ID, true); res.StatusCode != http.StatusOK {
		t.Errorf("assigning again: %s, want it to be fine", res.Status)
	}
}

// The one that matters. A workspace's work goes to that workspace's computers
// and nowhere else — so taking the only one out stops it, while another
// workspace that still has it carries on.
func TestAWorkspaceRunsOnlyOnItsOwnComputers(t *testing.T) {
	r := newRig(t)
	personal := r.workspaces()[0]
	work := r.newWorkspace("Work")

	machineID, _, conn := r.pairComputer("DESKTOP-RIVER")
	defer conn.Close()

	// Take it out of Work only.
	if res := r.assign(work.ID, machineID, false); res.StatusCode != http.StatusOK {
		t.Fatalf("removing: %s", res.Status)
	}

	mine := r.conversationIn(personal.ID, "D:\\home\\notes")
	theirs := r.conversationIn(work.ID, "D:\\clients\\contoso")

	send := func(id string) int {
		return r.post("/api/conversations/"+id+"/messages", map[string]any{
			"text": "do something", "provider": "claude",
			"model": protocol.Model{ID: "m1", Label: "M One"},
		}).StatusCode
	}
	if got := send(theirs.ID); got != http.StatusServiceUnavailable {
		t.Errorf("sending in a workspace with no computer = %d, want 503", got)
	}
	if got := send(mine.ID); got != http.StatusOK {
		t.Errorf("sending in the workspace that still has the computer = %d, want 200", got)
	}

	// The agents on offer follow the same rule: a workspace that cannot reach a
	// computer must not advertise what that computer can run.
	var none, some []protocol.Provider
	decodeInto(t, r.get("/api/providers?workspace="+work.ID), &none)
	decodeInto(t, r.get("/api/providers?workspace="+personal.ID), &some)
	if len(none) != 0 {
		t.Errorf("a workspace with no computer offers %d providers: %v", len(none), none)
	}
	if len(some) == 0 {
		t.Error("the workspace that has the computer offers no providers")
	}

	// And so does browsing folders, or somebody would pick a path off a
	// computer the next message cannot reach.
	if res := r.get("/api/directories?workspace=" + work.ID + "&path=D:%5C"); res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("browsing folders in a workspace with no computer = %s, want 503", res.Status)
	}
}

func TestOnlyThisAccountAssignsItsComputers(t *testing.T) {
	r := newRig(t)
	machineID, _, conn := r.pairComputer("DESKTOP-RIVER")
	defer conn.Close()
	mine := r.workspaces()[0]

	other := r.secondAccount()

	// Neither id is permission on its own, from either direction.
	if res := other.assign(mine.ID, machineID, false); res.StatusCode != http.StatusNotFound {
		t.Errorf("another account removing a computer from this workspace: %s, want 404", res.Status)
	}
	if res := other.assignFromMachine(machineID, mine.ID, false); res.StatusCode != http.StatusNotFound {
		t.Errorf("another account assigning this computer: %s, want 404", res.Status)
	}
	if res := other.get("/api/workspaces/" + mine.ID + "/machines"); res.StatusCode != http.StatusNotFound {
		t.Errorf("another account reading this workspace's computers: %s, want 404", res.Status)
	}
	if res := other.get("/api/machines/" + machineID + "/workspaces"); res.StatusCode != http.StatusNotFound {
		t.Errorf("another account reading this computer's workspaces: %s, want 404", res.Status)
	}

	// And it is still assigned where it was.
	if !assignedIn(r.workspaceMachines(mine.ID), machineID) {
		t.Error("another account's attempt changed the assignment")
	}
}

// Stopping a turn has to reach the computer it is RUNNING on. With one computer
// this was always true by accident; with a workspace's own set it has to be
// true on purpose, so the turn records where it went.
func TestStoppingATurnReachesTheComputerRunningIt(t *testing.T) {
	r := newRig(t)
	personal := r.workspaces()[0]
	machineID, _, conn := r.pairComputer("DESKTOP-RIVER")
	defer conn.Close()
	_ = machineID

	c := r.conversationIn(personal.ID, "D:\\home\\notes")
	res := r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "a long job", "provider": "claude",
		"model": protocol.Model{ID: "m1", Label: "M One"},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("send: %s", res.Status)
	}
	var started struct {
		TurnID string `json:"turnId"`
	}
	decodeInto(t, res, &started)

	// The computer it went to is the one that is asked to stop it.
	var asked protocol.ServerMessage
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("the computer running the turn was never asked to stop it: %v", err)
		}
		if err := json.Unmarshal(raw, &asked); err != nil {
			t.Fatal(err)
		}
		if asked.Type == protocol.ServerRunTurn {
			if got := r.post("/api/turns/"+started.TurnID+"/stop", nil).StatusCode; got != http.StatusAccepted {
				t.Fatalf("stop = %d, want 202", got)
			}
			continue
		}
		if asked.Type == protocol.ServerStopTurn {
			if asked.TurnID != started.TurnID {
				t.Errorf("stopped turn %q, want %q", asked.TurnID, started.TurnID)
			}
			return
		}
	}
}
