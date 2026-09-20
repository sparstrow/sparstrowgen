package api

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* Updates (spec US3), through the API a browser uses, with a websocket client
standing in for the daemon. The server decides nothing about an update; these
prove it keeps the preference, relays requests to the right computer, reports
what that computer says, and refuses work for one that is too old. */

type machineUpdatesSeen struct {
	ID               string                `json:"id"`
	Online           bool                  `json:"online"`
	Version          string                `json:"version"`
	TooOld           bool                  `json:"tooOld"`
	SelfUpdates      bool                  `json:"selfUpdates"`
	AutomaticUpdates bool                  `json:"automaticUpdates"`
	Update           protocol.UpdateStatus `json:"update"`
}

func (r *rig) machineUpdates(id string) machineUpdatesSeen {
	r.t.Helper()
	res := r.get("/api/machines/" + id)
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("get machine: %s", res.Status)
	}
	var m machineUpdatesSeen
	decodeInto(r.t, res, &m)
	return m
}

func (r *rig) awaitMachineUpdates(id string, done func(machineUpdatesSeen) bool) machineUpdatesSeen {
	r.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		m := r.machineUpdates(id)
		if done(m) {
			return m
		}
		if time.Now().After(deadline) {
			r.t.Fatalf("machine never reached the expected state: %+v", m)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// updatingComputer pairs a computer, has it say the given hello, and returns
// what the server sends it.
func (r *rig) updatingComputer(name string, hello protocol.DaemonMessage) (string, *websocket.Conn, chan protocol.ServerMessage) {
	r.t.Helper()
	id, _, conn := r.pairComputer(name)
	inbox := make(chan protocol.ServerMessage, 16)
	go func() {
		defer close(inbox)
		for {
			var m protocol.ServerMessage
			if err := conn.ReadJSON(&m); err != nil {
				return
			}
			inbox <- m
		}
	}()
	hello.Type, hello.Machine = protocol.DaemonHello, name
	hello.Providers = []protocol.Provider{{ID: "claude", Label: "claude", Availability: protocol.Available}}
	if err := conn.WriteJSON(hello); err != nil {
		r.t.Fatal(err)
	}
	return id, conn, inbox
}

func awaitServerMessage(t *testing.T, inbox chan protocol.ServerMessage, want func(protocol.ServerMessage) bool) protocol.ServerMessage {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case m, ok := <-inbox:
			if !ok {
				t.Fatal("the computer's socket closed")
			}
			if want(m) {
				return m
			}
		case <-deadline:
			t.Fatal("the expected message never reached the computer")
		}
	}
}

var currentDaemon = protocol.DaemonMessage{Version: "0.2.0", Protocol: protocol.DaemonProtocol, SelfUpdates: true}

func TestAComputerReportsItsVersionAndUpdatesAutomaticallyByDefault(t *testing.T) {
	r := newRig(t)
	id, conn, inbox := r.updatingComputer("DESKTOP-GJ8NLB8", currentDaemon)

	m := r.awaitMachineUpdates(id, func(m machineUpdatesSeen) bool { return m.Version == "0.2.0" })
	if !m.SelfUpdates || !m.AutomaticUpdates || m.TooOld || m.Update.Kind != protocol.UpdateUnchecked {
		t.Errorf("a freshly connected computer: %+v", m)
	}
	pref := awaitServerMessage(t, inbox, func(m protocol.ServerMessage) bool { return m.Type == protocol.ServerUpdatePreference })
	if pref.Automatic == nil || !*pref.Automatic {
		t.Errorf("the computer was not told automatic updates are on: %+v", pref)
	}

	_ = conn.Close()
	off := r.awaitMachineUpdates(id, func(m machineUpdatesSeen) bool { return !m.Online })
	if off.Version != "0.2.0" {
		t.Errorf("an offline computer no longer says what it runs: %+v", off)
	}
}

func TestTurningAutomaticUpdatesOffIsKeptAndToldToTheComputer(t *testing.T) {
	r := newRig(t)
	id, _, inbox := r.updatingComputer("DESKTOP-GJ8NLB8", currentDaemon)

	res := r.post("/api/machines/"+id+"/automatic-updates", map[string]any{"enabled": false})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("turn off: %s", res.Status)
	}
	var m machineUpdatesSeen
	decodeInto(t, res, &m)
	if m.AutomaticUpdates {
		t.Error("the answer still says automatic updates are on")
	}
	awaitServerMessage(t, inbox, func(m protocol.ServerMessage) bool {
		return m.Type == protocol.ServerUpdatePreference && m.Automatic != nil && !*m.Automatic
	})
	if r.machineUpdates(id).AutomaticUpdates {
		t.Error("the preference was not kept")
	}
	if res := r.post("/api/machines/"+id+"/automatic-updates", map[string]any{}); res.StatusCode != http.StatusBadRequest {
		t.Errorf("a request that says nothing: %s, want 400", res.Status)
	}
}

func TestCheckNowAndUpdateNowAreAnsweredByTheComputer(t *testing.T) {
	r := newRig(t)
	id, conn, inbox := r.updatingComputer("DESKTOP-GJ8NLB8", currentDaemon)
	// A websocket connection allows one writer at a time, and this test writes
	// from the goroutine answering the server and again from the test itself.
	var write sync.Mutex
	send := func(m protocol.DaemonMessage) error {
		write.Lock()
		defer write.Unlock()
		return conn.WriteJSON(m)
	}
	go func() {
		for msg := range inbox {
			var s protocol.UpdateStatus
			switch msg.Type {
			case protocol.ServerCheckUpdate:
				s = protocol.UpdateStatus{Kind: protocol.UpdateAvailable, Version: "0.2.1"}
			case protocol.ServerApplyUpdate:
				s = protocol.UpdateStatus{Kind: protocol.UpdateWaiting, Version: "0.2.1", ActiveTasks: 2}
			default:
				continue
			}
			_ = send(protocol.DaemonMessage{Type: protocol.DaemonUpdateStatus, RequestID: msg.RequestID, Update: &s})
		}
	}()

	res := r.post("/api/machines/"+id+"/updates/check", map[string]any{})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("check: %s", res.Status)
	}
	var checked machineUpdatesSeen
	decodeInto(t, res, &checked)
	if checked.Update.Kind != protocol.UpdateAvailable || checked.Update.Version != "0.2.1" {
		t.Errorf("after checking: %+v", checked.Update)
	}

	res = r.post("/api/machines/"+id+"/updates/apply", map[string]any{})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update now: %s", res.Status)
	}
	var applied machineUpdatesSeen
	decodeInto(t, res, &applied)
	if applied.Update.Kind != protocol.UpdateWaiting || applied.Update.ActiveTasks != 2 {
		t.Errorf("after Update now: %+v", applied.Update)
	}

	// A change the computer reports by itself reaches the account's browsers.
	b := r.watch()
	b.await("daemon", func(ev protocol.ClientEvent) bool { return ev.Type == protocol.EventDaemon })
	current := protocol.UpdateStatus{Kind: protocol.UpdateCurrent}
	if err := send(protocol.DaemonMessage{Type: protocol.DaemonUpdateStatus, Update: &current}); err != nil {
		t.Fatal(err)
	}
	b.await("machines", func(ev protocol.ClientEvent) bool { return ev.Type == protocol.EventMachines })
	r.awaitMachineUpdates(id, func(m machineUpdatesSeen) bool { return m.Update.Kind == protocol.UpdateCurrent })
}

func TestUpdatesNeedAComputerThatIsConnectedAndCanUpdateItself(t *testing.T) {
	r := newRig(t)
	// A daemon from before US3 says nothing about versions or updating.
	id, conn, _ := r.updatingComputer("OLDER-PC", protocol.DaemonMessage{})
	r.awaitMachineUpdates(id, func(m machineUpdatesSeen) bool { return m.Online })
	if res := r.post("/api/machines/"+id+"/updates/check", map[string]any{}); res.StatusCode != http.StatusConflict {
		t.Errorf("checking a computer that cannot update itself: %s, want 409", res.Status)
	}

	_ = conn.Close()
	r.awaitMachineUpdates(id, func(m machineUpdatesSeen) bool { return !m.Online })
	if res := r.post("/api/machines/"+id+"/updates/check", map[string]any{}); res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("checking an offline computer: %s, want 503", res.Status)
	}
}

func TestAnotherAccountCannotChangeOrCheckMyComputersUpdates(t *testing.T) {
	r := newRig(t)
	id, _, _ := r.updatingComputer("DESKTOP-GJ8NLB8", currentDaemon)
	other := r.secondAccount()
	for _, path := range []string{"/automatic-updates", "/updates/check", "/updates/apply"} {
		if res := other.post("/api/machines/"+id+path, map[string]any{"enabled": false}); res.StatusCode != http.StatusNotFound {
			t.Errorf("POST %s by another account: %s, want 404", path, res.Status)
		}
	}
	if !r.machineUpdates(id).AutomaticUpdates {
		t.Error("another account turned my automatic updates off")
	}
}

// Spec US3 and the phase 1 exit gate: an older daemon gets a useful message,
// not a mysterious failure, and existing work stays readable.
func TestATooOldComputerIsToldToUpdateAndSentNoWork(t *testing.T) {
	original := protocol.MinDaemonProtocol
	protocol.MinDaemonProtocol = protocol.DaemonProtocol + 1
	t.Cleanup(func() { protocol.MinDaemonProtocol = original })

	r := newRig(t)
	b := r.watch()
	id, _, inbox := r.updatingComputer("OLD-PC", currentDaemon)
	b.await("a too-old daemon", func(ev protocol.ClientEvent) bool { return ev.Type == protocol.EventDaemon && ev.Online && ev.TooOld })
	if m := r.awaitMachineUpdates(id, func(m machineUpdatesSeen) bool { return m.Version == "0.2.0" }); !m.TooOld {
		t.Errorf("the machine is not marked too old: %+v", m)
	}

	c := r.conversation("claude")
	res := r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "anything", "provider": "claude", "model": protocol.Model{ID: "m1", Label: "M One"},
	})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("sending to a too-old computer: %s, want 409", res.Status)
	}
	var body struct {
		Error string `json:"error"`
	}
	decodeInto(t, res, &body)
	if !strings.Contains(body.Error, "too old") || !strings.Contains(body.Error, "Settings → Updates") {
		t.Errorf("the refusal does not say what to do: %q", body.Error)
	}
	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case m := <-inbox:
			if m.Type == protocol.ServerRunTurn {
				t.Fatal("work was sent to a computer too old to run it")
			}
		case <-deadline:
			return
		}
	}
}
