package api

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* One account's work is invisible and unreachable to every other account
(docs/KnownGaps.md G-27).

Before accounts could be more than one, "every conversation" and "the owner's
conversations" were the same set, and the hub sent every event to every browser.
Each test here fails on that shape: a second account seeing a list, opening an
id, receiving an event, or using the owner's machine. */

// secondAccount signs a different person in on the same server.
func (r *rig) secondAccount() *rig {
	r.t.Helper()
	user, err := r.store.CreateUser(context.Background(), testSecond, testPassword)
	if err != nil {
		r.t.Fatalf("create a second account: %v", err)
	}
	other := &rig{t: r.t, api: r.api, store: r.store, http: r.http,
		client: newJarClient(r.t), userID: user.ID, mail: r.mail}
	if res := other.post("/api/auth/login", map[string]any{
		"email": testSecond, "password": testPassword,
	}); res.StatusCode != http.StatusOK {
		r.t.Fatalf("sign in as the second account: %s", res.Status)
	}
	return other
}

func TestAnotherAccountCannotReachTheOwnersConversations(t *testing.T) {
	r := newRig(t)
	c := r.conversation("claude")
	if _, err := r.store.AppendUser(context.Background(), c.ID, "the owner's private plan"); err != nil {
		t.Fatal(err)
	}
	other := r.secondAccount()

	for _, path := range []string{"/api/conversations", "/api/conversations?q=" + url.QueryEscape("private plan")} {
		res := other.get(path)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: %s", path, res.Status)
		}
		var list []protocol.Conversation
		decodeInto(t, res, &list)
		if len(list) != 0 {
			t.Errorf("GET %s showed another account %d of the owner's conversations", path, len(list))
		}
	}

	var folders map[string][]string
	decodeInto(t, other.get("/api/folders/recent"), &folders)
	if len(folders["folders"]) != 0 {
		t.Errorf("another account sees the owner's folders: %v", folders["folders"])
	}

	for _, attempt := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/conversations/" + c.ID, nil},
		{http.MethodPatch, "/api/conversations/" + c.ID, map[string]any{"title": "taken over"}},
		{http.MethodPatch, "/api/conversations/" + c.ID, map[string]any{}},
		{http.MethodDelete, "/api/conversations/" + c.ID, nil},
		{http.MethodGet, "/api/conversations/" + c.ID + "/switch-cost?provider=codex", nil},
		{http.MethodPost, "/api/conversations/" + c.ID + "/messages", map[string]any{"text": "hello", "provider": "claude"}},
	} {
		if res := other.do(attempt.method, attempt.path, attempt.body); res.StatusCode != http.StatusNotFound {
			t.Errorf("%s %s by another account: %s, want 404", attempt.method, attempt.path, res.Status)
		}
	}

	// And nothing it tried changed anything.
	after, err := r.store.Get(context.Background(), r.userID, c.ID)
	if err != nil {
		t.Fatalf("the owner's conversation is gone: %v", err)
	}
	if after.Title != "" || len(after.Entries) != 1 {
		t.Errorf("another account changed the owner's conversation: title=%q entries=%d", after.Title, len(after.Entries))
	}
}

func TestAnotherAccountHearsNothingAboutTheOwnersWork(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	owner := r.watch()
	other := r.secondAccount()
	theirs := other.watch()

	// Greeted with ITS OWN machine state: no machine, nothing installed — even
	// though the owner's machine is connected.
	greeting := theirs.await("daemon", func(ev protocol.ClientEvent) bool { return ev.Type == protocol.EventDaemon })
	if greeting.Online {
		t.Error("another account was told the owner's machine is its machine")
	}
	installed := theirs.await("providers", func(ev protocol.ClientEvent) bool { return ev.Type == protocol.EventProviders })
	if len(installed.Providers) != 0 {
		t.Errorf("another account was shown the owner's providers: %v", installed.Providers)
	}

	c := r.conversation("claude")
	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "only the owner should see this", "provider": "claude",
		"model": protocol.Model{ID: "m1", Label: "M One"},
	})
	turn := d.nextTurn()
	d.send(protocol.DaemonMessage{Type: protocol.DaemonDelta, TurnID: turn.TurnID, Text: "streaming"})
	owner.await("delta", func(ev protocol.ClientEvent) bool { return ev.Type == protocol.EventEntryDelta })

	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case ev := <-theirs.events:
			if ev.ConversationID == c.ID || (ev.Conversation != nil && ev.Conversation.ID == c.ID) ||
				ev.Type == protocol.EventEntryDelta {
				t.Fatalf("another account received an event about the owner's work: %+v", ev)
			}
		case <-deadline:
			return
		}
	}
}

func TestTheOwnersMachineDoesNotWorkForAnotherAccount(t *testing.T) {
	r := newRig(t)
	r.connectDaemon()
	other := r.secondAccount()
	mine := other.createConversation()

	res := other.post("/api/conversations/"+mine.ID+"/messages", map[string]any{
		"text": "run this on somebody else's laptop", "provider": "claude",
		"model": protocol.Model{ID: "m1", Label: "M One"},
	})
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("another account sending with only the owner's machine connected: %s, want 503", res.Status)
	}

	var providers []protocol.Provider
	decodeInto(t, other.get("/api/providers"), &providers)
	if len(providers) != 0 {
		t.Errorf("another account sees the owner's machine's providers: %v", providers)
	}
	if res := other.get("/api/directories?path=" + url.QueryEscape("C:\\")); res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("another account browsing folders: %s, want 503 — it has no machine", res.Status)
	}
}

func TestAnotherAccountCannotStopTheOwnersTurn(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")
	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "a long job", "provider": "claude",
		"model": protocol.Model{ID: "m1", Label: "M One"},
	})
	turn := d.nextTurn()

	other := r.secondAccount()
	if res := other.post("/api/turns/"+turn.TurnID+"/stop", nil); res.StatusCode != http.StatusConflict {
		t.Errorf("another account stopping the owner's turn: %s, want 409 — as if it had finished", res.Status)
	}

	deadline := time.After(300 * time.Millisecond)
	for {
		select {
		case msg := <-d.work:
			if msg.Type == protocol.ServerStopTurn {
				t.Fatal("the owner's machine was told to stop a turn by another account")
			}
		case <-deadline:
			return
		}
	}
}
