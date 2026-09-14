package api

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* Pairing a computer, end to end through the API a browser and a daemon use.

The browser starts a request and approves it; the daemon claims it through the
sparstrowgen:// link and dials with the credential it earns. Nothing here is
faked except the daemon process, which is a websocket client. */

type pairingStarted struct {
	Pairing   store.Pairing `json:"pairing"`
	LaunchURI string        `json:"launchUri"`
}

type machineSeen struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Approved  bool                `json:"approved"`
	Online    bool                `json:"online"`
	Providers []protocol.Provider `json:"providers"`
}

func (r *rig) startPairing() (id, request string) {
	r.t.Helper()
	res := r.post("/api/machines/pairings", map[string]any{})
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("start pairing: %s", res.Status)
	}
	var out pairingStarted
	decodeInto(r.t, res, &out)
	const prefix = "sparstrowgen://pair?request="
	if !strings.HasPrefix(out.LaunchURI, prefix) {
		r.t.Fatalf("launch link = %q", out.LaunchURI)
	}
	return out.Pairing.ID, strings.TrimPrefix(out.LaunchURI, prefix)
}

// claimPairing is the daemon spending the request. It carries no session.
func (r *rig) claimPairing(request, name string) (int, string) {
	r.t.Helper()
	res := r.stranger().post("/daemon/pair", map[string]string{"request": request, "name": name})
	var out struct {
		Credential string `json:"credential"`
	}
	if res.StatusCode == http.StatusOK {
		decodeInto(r.t, res, &out)
	}
	return res.StatusCode, out.Credential
}

func (r *rig) dialPaired(credential string) (*websocket.Conn, int) {
	r.t.Helper()
	conn, res, err := websocket.DefaultDialer.Dial(r.ws("/daemon"), http.Header{
		"Authorization": []string{"Bearer " + credential},
	})
	if err != nil {
		if res == nil {
			r.t.Fatalf("dial /daemon: %v", err)
		}
		return nil, res.StatusCode
	}
	r.t.Cleanup(func() { _ = conn.Close() })
	return conn, http.StatusSwitchingProtocols
}

func (r *rig) machines() []machineSeen {
	r.t.Helper()
	res := r.get("/api/machines")
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("list machines: %s", res.Status)
	}
	var list []machineSeen
	decodeInto(r.t, res, &list)
	return list
}

func (r *rig) awaitMachine(done func([]machineSeen) bool) []machineSeen {
	r.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		list := r.machines()
		if done(list) {
			return list
		}
		if time.Now().After(deadline) {
			r.t.Fatalf("machines never reached the expected state: %+v", list)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// pairComputer runs the whole journey and returns a connected, approved computer.
func (r *rig) pairComputer(name string) (machineID, credential string, conn *websocket.Conn) {
	r.t.Helper()
	id, request := r.startPairing()
	status, credential := r.claimPairing(request, name)
	if status != http.StatusOK || credential == "" {
		r.t.Fatalf("claim: status %d, credential present %v", status, credential != "")
	}
	res := r.post("/api/machines/pairings/"+id+"/approve", map[string]any{})
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("approve: %s", res.Status)
	}
	var m machineSeen
	decodeInto(r.t, res, &m)
	conn, code := r.dialPaired(credential)
	if conn == nil {
		r.t.Fatalf("an approved computer was refused: %d", code)
	}
	if err := conn.WriteJSON(protocol.DaemonMessage{
		Type: protocol.DaemonHello, Machine: name,
		Providers: []protocol.Provider{{ID: "claude", Label: "claude", Availability: protocol.Available}},
	}); err != nil {
		r.t.Fatal(err)
	}
	r.awaitMachine(func(list []machineSeen) bool {
		for _, x := range list {
			if x.ID == m.ID && x.Online && len(x.Providers) == 1 {
				return true
			}
		}
		return false
	})
	return m.ID, credential, conn
}

func TestAComputerConnectsOnlyAfterItsPairingIsApproved(t *testing.T) {
	r := newRig(t)
	id, request := r.startPairing()

	status, credential := r.claimPairing(request, "DESKTOP-RIVER")
	if status != http.StatusOK || credential == "" {
		t.Fatalf("claim: status %d, credential present %v", status, credential != "")
	}
	var p store.Pairing
	decodeInto(t, r.get("/api/machines/pairings/"+id), &p)
	if p.Status != "claimed" {
		t.Fatalf("pairing status after claim = %q, want claimed", p.Status)
	}

	if conn, code := r.dialPaired(credential); conn != nil || code != http.StatusUnauthorized {
		t.Fatalf("an unapproved computer dialled: conn %v, status %d; want 401", conn != nil, code)
	}

	browser := r.watch()
	if res := r.post("/api/machines/pairings/"+id+"/approve", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("approve: %s", res.Status)
	}
	conn, code := r.dialPaired(credential)
	if conn == nil {
		t.Fatalf("an approved computer was refused: %d", code)
	}
	if err := conn.WriteJSON(protocol.DaemonMessage{
		Type: protocol.DaemonHello, Machine: "DESKTOP-RIVER",
		Providers: []protocol.Provider{{ID: "codex", Label: "codex", Availability: protocol.Available}},
	}); err != nil {
		t.Fatal(err)
	}

	list := r.awaitMachine(func(list []machineSeen) bool {
		return len(list) == 1 && list[0].Online && len(list[0].Providers) == 1
	})
	if list[0].Name != "DESKTOP-RIVER" || !list[0].Approved {
		t.Errorf("machine = %+v", list[0])
	}
	if !r.api.hub.DaemonOnline(r.userID) {
		t.Error("Chat would not see the paired computer as online")
	}
	browser.awaitEvent(t, protocol.EventMachines)
}

func TestAPairingRequestCanBeClaimedOnce(t *testing.T) {
	r := newRig(t)
	_, request := r.startPairing()
	if status, _ := r.claimPairing(request, "first"); status != http.StatusOK {
		t.Fatalf("first claim: %d", status)
	}
	if status, credential := r.claimPairing(request, "second"); status != http.StatusConflict || credential != "" {
		t.Fatalf("second claim: status %d, credential present %v; want 409 and none", status, credential != "")
	}
	if status, _ := r.claimPairing("not-a-real-request", "third"); status != http.StatusConflict {
		t.Fatalf("made-up request: status %d, want 409", status)
	}
}

func TestADisconnectedComputerIsToldToStopAndOthersKeepWorking(t *testing.T) {
	r := newRig(t)
	gone, goneCredential, goneConn := r.pairComputer("FINANCE-LAPTOP")
	kept, _, _ := r.pairComputer("DESKTOP-RIVER")

	res := r.do(http.MethodDelete, "/api/machines/"+gone, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("disconnect: %s", res.Status)
	}

	_ = goneConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		if _, _, err := goneConn.ReadMessage(); err != nil {
			if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
				t.Fatal("the disconnected computer's socket stayed open")
			}
			break
		}
	}

	redial, res2, _ := websocket.DefaultDialer.Dial(r.ws("/daemon"), http.Header{
		"Authorization": []string{"Bearer " + goneCredential},
	})
	if redial != nil {
		_ = redial.Close()
		t.Fatal("a disconnected computer reconnected with its old credential")
	}
	if res2 == nil || res2.StatusCode != http.StatusForbidden {
		t.Fatalf("redial status = %v, want 403 so the daemon stops retrying", res2)
	}
	body, _ := io.ReadAll(res2.Body)
	if !strings.Contains(string(body), "disconnected") {
		t.Errorf("403 body does not say the computer was disconnected: %s", body)
	}

	list := r.awaitMachine(func(list []machineSeen) bool { return len(list) == 1 })
	if list[0].ID != kept || !list[0].Online {
		t.Errorf("the other computer did not keep working: %+v", list)
	}
}

func TestAnotherAccountCannotSeeApproveOrDisconnectAComputer(t *testing.T) {
	r := newRig(t)
	machineID, _, _ := r.pairComputer("DESKTOP-RIVER")
	pendingID, request := r.startPairing()
	if status, _ := r.claimPairing(request, "WAITING-PC"); status != http.StatusOK {
		t.Fatalf("claim: %d", status)
	}

	other := r.secondAccount()
	if list := other.machines(); len(list) != 0 {
		t.Errorf("another account listed %d of the owner's computers", len(list))
	}
	for _, c := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/machines/" + machineID, http.StatusNotFound},
		{http.MethodDelete, "/api/machines/" + machineID, http.StatusNotFound},
		{http.MethodGet, "/api/machines/pairings/" + pendingID, http.StatusNotFound},
		{http.MethodPost, "/api/machines/pairings/" + pendingID + "/approve", http.StatusConflict},
	} {
		if res := other.do(c.method, c.path, map[string]any{}); res.StatusCode != c.want {
			t.Errorf("%s %s by another account: %s, want %d", c.method, c.path, res.Status, c.want)
		}
	}
	if list := r.machines(); len(list) != 2 {
		t.Fatalf("owner machines = %d, want 2", len(list))
	}
}

// awaitEvent waits for one event of a type, discarding others.
func (b *browser) awaitEvent(t *testing.T, eventType string) protocol.ClientEvent {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev, ok := <-b.events:
			if !ok {
				t.Fatalf("the browser socket closed before a %q event", eventType)
			}
			if ev.Type == eventType {
				return ev
			}
		case <-deadline:
			t.Fatalf("no %q event reached the browser", eventType)
		}
	}
}
