package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// B-8. Only a message carrying a turn id ever finished a turn, and when the
// daemon goes there is nothing left to send one — so the entry stayed an empty
// placeholder, the composer stayed locked, and the working indicator ticked
// forever, through a refresh.
func TestATurnIsClosedOutWhenTheMachineGoesAway(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")

	res := r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text":     "this one never comes back",
		"provider": "claude",
		"model":    protocol.Model{ID: "m1", Label: "M One"},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("send: %s", res.Status)
	}
	turn := d.nextTurn()

	// Some of the answer arrives, and then the machine goes.
	d.send(protocol.DaemonMessage{
		Type: protocol.DaemonDelta, TurnID: turn.TurnID, Text: "I had started to say",
	})
	d.vanish()

	got := r.awaitEntry(c.ID, turn.EntryID, func(e protocol.Entry) bool {
		return e.Failure != ""
	})
	if !strings.Contains(got.Failure, "disconnected") {
		t.Errorf("failure = %q, want it to say the machine went away", got.Failure)
	}
	if got.Stopped {
		t.Error("stopped = true; a machine going away is not the owner stopping a turn")
	}
	// The same rule as a failed turn: what arrived is still worth reading, and
	// deleting it would hide how far the turn got.
	if got.Text != "I had started to say" {
		t.Errorf("text = %q, want the part that arrived before the machine went", got.Text)
	}
}

// B-9. The conversation really is on the new provider after a send, but nothing
// said so, and the browser's cached copy still named the old one — which is what
// the composer reads, so it snapped back to claude after sending to codex.
func TestSendingToADifferentProviderAnnouncesTheChange(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	b := r.watch()
	c := r.conversation("claude")

	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text":     "over to codex",
		"provider": "codex",
		"model":    protocol.Model{ID: "gpt", Label: "GPT"},
	})
	d.nextTurn()

	ev := b.await("conversation", func(ev protocol.ClientEvent) bool {
		return ev.Type == protocol.EventConversation &&
			ev.Conversation != nil && ev.Conversation.ID == c.ID &&
			ev.Conversation.Provider == "codex"
	})
	if ev.Conversation.Model.Label != "GPT" {
		t.Errorf("model = %q, want the one that was sent with", ev.Conversation.Model.Label)
	}
}

// The other half of L-12: the replay decision itself, which is where B-7 lived.
// It gated the catch-up on a provider CHANGE, which quietly assumed a switch is
// the only way a session goes missing — so after a folder move the next turn ran
// with no history at all.
func TestAProviderThatHasNotSeenTheConversationIsCaughtUp(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")

	// One exchange, so there is something to catch up on.
	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "remember this", "provider": "claude",
		"model": protocol.Model{ID: "m1", Label: "M One"},
	})
	first := d.nextTurn()
	if len(first.Replay) != 0 {
		t.Errorf("a fresh conversation replayed %d messages, want none", len(first.Replay))
	}
	d.send(protocol.DaemonMessage{
		Type: protocol.DaemonDone, TurnID: first.TurnID, Full: "noted", Tokens: 5,
	})
	// Waiting for the entry is enough: the seen-marker is written in the same
	// transaction, so there is no gap to slip into. It used to need a separate
	// wait, which is how G-18 was found.
	r.awaitEntry(c.ID, first.EntryID, func(e protocol.Entry) bool { return e.Text != "" })

	// The SAME provider, with its session dropped — which is what moving a
	// conversation to another folder does, because claude keys sessions by
	// project directory. Under the old condition this replayed nothing.
	if _, err := r.store.SetFolder(context.Background(), c.ID, "D:\\test\\elsewhere"); err != nil {
		t.Fatal(err)
	}
	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "what did I say?", "provider": "claude",
		"model": protocol.Model{ID: "m1", Label: "M One"},
	})
	second := d.nextTurn()

	if len(second.Replay) == 0 {
		t.Fatal("the agent started blind: no replay after its session was dropped")
	}
	var replayed []string
	for _, e := range second.Replay {
		replayed = append(replayed, e.Text)
	}
	joined := strings.Join(replayed, " | ")
	if !strings.Contains(joined, "remember this") || !strings.Contains(joined, "noted") {
		t.Errorf("replay = %q, want both halves of the earlier exchange", joined)
	}
}

// Stopping a turn is a request to the machine, not a local edit: the turn ends
// when the daemon says it has. A stop for a turn that already finished is an
// ordinary race rather than an error, and answers 409 so the client can stay
// quiet about it.
func TestStoppingATurnAsksTheDaemonAndWaitsForIt(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")

	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "go on then", "provider": "claude",
		"model": protocol.Model{ID: "m1", Label: "M One"},
	})
	turn := d.nextTurn()
	d.send(protocol.DaemonMessage{
		Type: protocol.DaemonDelta, TurnID: turn.TurnID, Text: "half of an",
	})

	if res := r.post("/api/turns/"+turn.TurnID+"/stop", nil); res.StatusCode != http.StatusAccepted {
		t.Fatalf("stop: %s, want 202 — accepted, not done", res.Status)
	}

	// Accepted is not finished. Nothing may be written until the daemon reports.
	if got := r.entry(c.ID, turn.EntryID); got.Stopped {
		t.Error("the turn was marked stopped before the machine confirmed it")
	}

	var stop protocol.ServerMessage
	for msg := range d.work {
		if msg.Type == protocol.ServerStopTurn {
			stop = msg
			break
		}
	}
	if stop.TurnID != turn.TurnID {
		t.Fatalf("the daemon was asked to stop %q, want %q", stop.TurnID, turn.TurnID)
	}

	d.send(protocol.DaemonMessage{
		Type: protocol.DaemonStopped, TurnID: turn.TurnID, Full: "half of an",
	})
	got := r.awaitEntry(c.ID, turn.EntryID, func(e protocol.Entry) bool { return e.Stopped })
	if got.Failure != "" {
		t.Errorf("failure = %q; being stopped is not going wrong", got.Failure)
	}
	if got.Text != "half of an" {
		t.Errorf("text = %q, want what had arrived", got.Text)
	}

	// And now that it is over, stopping it again is a race the client lost, not
	// a fault.
	if res := r.post("/api/turns/"+turn.TurnID+"/stop", nil); res.StatusCode != http.StatusConflict {
		t.Errorf("second stop: %s, want 409", res.Status)
	}
}

// B-10. A failure used to be written from the deltas the server had accumulated,
// which is empty on a provider that does not stream — so codex losing a turn
// lost everything it had produced, silently. The daemon recovers whole messages
// from the stream it parsed, and on those providers that is the only copy.
func TestAFailedTurnKeepsTextFromAProviderThatDoesNotStream(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("codex")

	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "something long", "provider": "codex",
		"model": protocol.Model{ID: "gpt", Label: "GPT"},
	})
	turn := d.nextTurn()

	// No deltas at all — that is what "does not stream" means. The whole answer
	// so far arrives with the failure.
	d.send(protocol.DaemonMessage{
		Type: protocol.DaemonFailed, TurnID: turn.TurnID,
		Full:  "I had got this far before it broke",
		Error: "codex stopped responding",
	})

	got := r.awaitEntry(c.ID, turn.EntryID, func(e protocol.Entry) bool { return e.Failure != "" })
	if got.Text != "I had got this far before it broke" {
		t.Errorf("text = %q, want what the parser recovered — there were no deltas to fall back on", got.Text)
	}
}

// And a failure with nothing to show keeps the deltas that did arrive, rather
// than blanking them because the daemon sent no Full.
func TestAFailedTurnStillFallsBackToTheDeltasItStreamed(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")

	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "go", "provider": "claude",
		"model": protocol.Model{ID: "m1", Label: "M One"},
	})
	turn := d.nextTurn()
	d.send(protocol.DaemonMessage{
		Type: protocol.DaemonDelta, TurnID: turn.TurnID, Text: "streamed this much",
	})
	d.send(protocol.DaemonMessage{
		Type: protocol.DaemonFailed, TurnID: turn.TurnID, Error: "died",
	})

	got := r.awaitEntry(c.ID, turn.EntryID, func(e protocol.Entry) bool { return e.Failure != "" })
	if got.Text != "streamed this much" {
		t.Errorf("text = %q, want the deltas kept", got.Text)
	}
}

// L-9. Every conversation was "Untitled conversation" until somebody renamed
// one by hand, so the sidebar was a column of identical rows told apart only by
// folder and age — and the search had to look inside message bodies precisely
// because the titles said nothing.
//
// The name comes from the first thing said, which is free and right most of the
// time, and it rides along on the conversation event the send already
// broadcasts rather than needing one of its own.
func TestAConversationIsNamedByTheFirstThingSaidInIt(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	b := r.watch()
	c := r.conversation("claude")

	if c.Title != "" {
		t.Fatalf("a new conversation arrived named %q; it should have no name yet", c.Title)
	}

	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text":     "Name this conversation after me\n\nand not after this second line.",
		"provider": "claude", "model": protocol.Model{ID: "m1", Label: "M One"},
	})
	first := d.nextTurn()

	ev := b.await("conversation", func(ev protocol.ClientEvent) bool {
		return ev.Type == protocol.EventConversation &&
			ev.Conversation != nil && ev.Conversation.ID == c.ID &&
			ev.Conversation.Title != ""
	})
	if ev.Conversation.Title != "Name this conversation after me" {
		t.Errorf("title = %q, want the first line of the first message", ev.Conversation.Title)
	}

	// The second message is not a second chance. A name describes where a
	// conversation started, and rewriting it on every send would make the
	// sidebar move under the owner while he reads it.
	d.send(protocol.DaemonMessage{
		Type: protocol.DaemonDone, TurnID: first.TurnID, Full: "done", Tokens: 1,
	})
	r.awaitEntry(c.ID, first.EntryID, func(e protocol.Entry) bool { return e.Text != "" })

	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "Something else entirely", "provider": "claude",
		"model": protocol.Model{ID: "m1", Label: "M One"},
	})
	d.nextTurn()

	after, err := r.store.Get(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Title != "Name this conversation after me" {
		t.Errorf("title = %q after a second message; a name is taken once", after.Title)
	}
}

// And a name the owner typed outranks any name we would have derived — however
// the two arrive. The guard is in the statement, so a rename landing between the
// send and the naming still wins.
func TestANameTheOwnerTypedIsNeverOverwritten(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")

	if _, err := r.store.Rename(context.Background(), c.ID, "The one about job objects"); err != nil {
		t.Fatal(err)
	}

	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text":     "a first message that would have named it something else",
		"provider": "claude", "model": protocol.Model{ID: "m1", Label: "M One"},
	})
	d.nextTurn()

	after, err := r.store.Get(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Title != "The one about job objects" {
		t.Errorf("title = %q, want the one the owner typed", after.Title)
	}
}
