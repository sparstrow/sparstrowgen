package api

import (
	"net/http"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* A turn's record (docs/specs/2026-09-23-raw-exchange.md): what the daemon
   reports is stored as it arrives, told to the account's browsers, and read back
   whole, with what the CLI said about itself. */

const claudeInit = `{"type":"system","subtype":"init","cwd":"D:\\work","session_id":"s-1","tools":["Read","Bash"],"mcp_servers":[],"model":"claude-sonnet-5","permissionMode":"default","claude_code_version":"2.1.280","skills":["testing"],"agents":["Plan"]}`
const claudeResult = `{"type":"result","subtype":"success","is_error":false,"result":"hi","total_cost_usd":0.01,"usage":{"input_tokens":3,"output_tokens":4,"cache_creation_input_tokens":10,"cache_read_input_tokens":100}}`

// sendTurn starts a claude turn and returns what the daemon was handed.
func sendTurn(r *rig, d *daemon, c protocol.Conversation) protocol.RunTurn {
	r.t.Helper()
	res := r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "say hi", "provider": "claude", "model": protocol.Model{ID: "claude-sonnet-5", Label: "Sonnet 5"},
	})
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("send: %s", res.Status)
	}
	return d.nextTurn()
}

func (r *rig) exchange(entryID string) (protocol.Exchange, int) {
	r.t.Helper()
	res := r.get("/api/turns/" + entryID + "/exchange")
	var ex protocol.Exchange
	if res.StatusCode == http.StatusOK {
		decodeInto(r.t, res, &ex)
	} else {
		res.Body.Close()
	}
	return ex, res.StatusCode
}

func TestATurnsRecordIsStoredToldAndReadBack(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	b := r.watch()
	c := r.conversation("claude")
	turn := sendTurn(r, d, c)

	d.send(protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: turn.TurnID, Sent: &protocol.ExchangeSent{
		Program: `C:\bin\claude.exe`, Args: []string{"-p", "--model", "claude-sonnet-5"}, Cwd: `D:\work`,
		Prompt: "say hi", Stdin: `{"type":"user"}` + "\n", Launched: true,
	}})
	d.send(protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: turn.TurnID, Lines: []protocol.ExchangeLine{
		{Seq: 1, AtMs: 900, Stream: protocol.StreamStdout, Text: claudeInit},
		{Seq: 2, AtMs: 950, Stream: protocol.StreamStderr, Text: "a warning of its own"},
	}})

	// A browser watching learns what the CLI loaded while the turn still runs.
	live := b.await("exchange with a report", func(ev protocol.ClientEvent) bool {
		return ev.Type == protocol.EventExchange && ev.Exchange != nil && ev.Exchange.Report != nil
	})
	if live.EntryID != turn.EntryID || live.ConversationID != c.ID || len(live.Exchange.Lines) != 2 {
		t.Errorf("live event = %+v", live)
	}
	if live.Exchange.Report.Model != "claude-sonnet-5" {
		t.Errorf("live report model = %q", live.Exchange.Report.Model)
	}

	d.send(protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: turn.TurnID,
		Lines:   []protocol.ExchangeLine{{Seq: 3, AtMs: 4000, Stream: protocol.StreamStdout, Text: claudeResult}},
		Dropped: &protocol.ExchangeDropped{Lines: 2, Bytes: 4096},
	})
	d.send(protocol.DaemonMessage{Type: protocol.DaemonDone, TurnID: turn.TurnID, Full: "hi", Tokens: 117})
	r.awaitEntry(c.ID, turn.EntryID, func(e protocol.Entry) bool { return e.Usage != nil })

	ex, status := r.exchange(turn.EntryID)
	if status != http.StatusOK || !ex.Recorded {
		t.Fatalf("status %d, recorded %v", status, ex.Recorded)
	}
	if ex.Sent == nil || ex.Sent.Program != `C:\bin\claude.exe` || len(ex.Sent.Args) != 3 || !ex.Sent.Launched || ex.Sent.Stdin == "" {
		t.Errorf("sent = %+v", ex.Sent)
	}
	if len(ex.Lines) != 3 || ex.Lines[0].Text != claudeInit || ex.Lines[1].Stream != protocol.StreamStderr || ex.Lines[2].AtMs != 4000 {
		t.Errorf("lines = %+v", ex.Lines)
	}
	if ex.Dropped == nil || ex.Dropped.Lines != 2 || ex.Dropped.Bytes != 4096 {
		t.Errorf("dropped = %+v", ex.Dropped)
	}
	rep := ex.Report
	if rep == nil || rep.CLIVersion != "2.1.280" || len(rep.Tools) != 2 || rep.Usage == nil || rep.Usage.Total != 117 {
		t.Errorf("report = %+v", rep)
	}

	// Another account reads nothing, and is told nothing it could learn from.
	other := r.secondAccount()
	if _, status := other.exchange(turn.EntryID); status != http.StatusNotFound {
		t.Errorf("another account reading the record got %d, want 404", status)
	}

	// How big it is, without reading it: what the Raw view labels turns with.
	var sums []protocol.ExchangeSummary
	decodeInto(t, r.get("/api/conversations/"+c.ID+"/exchanges"), &sums)
	want := int64(len(claudeInit) + len("a warning of its own") + len(claudeResult))
	if len(sums) != 1 || sums[0].EntryID != turn.EntryID || sums[0].Lines != 3 || sums[0].Bytes != want ||
		sums[0].LastAtMs != 4000 || !sums[0].Launched || sums[0].Dropped == nil {
		t.Errorf("summaries = %+v, want 3 lines, %d bytes, 4000ms, launched, dropped", sums, want)
	}
	if res := other.get("/api/conversations/" + c.ID + "/exchanges"); res.StatusCode != http.StatusNotFound {
		t.Errorf("another account read the summaries: %s", res.Status)
	}
}

// Postgres text refuses one byte, NUL. A CLI printing binary must not lose
// the batch around it.
func TestALineWithANulByteIsKeptRatherThanLosingItsBatch(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")
	turn := sendTurn(r, d, c)
	d.send(protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: turn.TurnID,
		Sent: &protocol.ExchangeSent{Program: "claude", Prompt: "say hi", Launched: true}})
	d.send(protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: turn.TurnID, Lines: []protocol.ExchangeLine{
		{Seq: 1, Stream: protocol.StreamStderr, Text: "before"},
		{Seq: 2, Stream: protocol.StreamStderr, Text: "bin\x00ary"},
		{Seq: 3, Stream: protocol.StreamStderr, Text: "after"},
	}})
	d.send(protocol.DaemonMessage{Type: protocol.DaemonDone, TurnID: turn.TurnID, Full: "hi", Tokens: 5})
	r.awaitEntry(c.ID, turn.EntryID, func(e protocol.Entry) bool { return e.Usage != nil })

	ex, _ := r.exchange(turn.EntryID)
	if len(ex.Lines) != 3 || ex.Lines[1].Text != "bin\uFFFDary" || ex.Lines[2].Text != "after" {
		t.Errorf("lines = %+v", ex.Lines)
	}
}

func TestATurnWithNoRecordSaysSoRatherThanFailing(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")
	// A daemon from before recording: it answers and sends no record at all.
	turn := sendTurn(r, d, c)
	d.send(protocol.DaemonMessage{Type: protocol.DaemonDone, TurnID: turn.TurnID, Full: "hi", Tokens: 5})
	r.awaitEntry(c.ID, turn.EntryID, func(e protocol.Entry) bool { return e.Usage != nil })

	ex, status := r.exchange(turn.EntryID)
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if ex.Recorded || ex.Sent != nil || ex.Report != nil {
		t.Errorf("an unrecorded turn claimed a record: %+v", ex)
	}
	if ex.Lines == nil {
		t.Error("lines must be an empty list, never null")
	}
}

func TestReadingARecordNeedsATurnThatIsYours(t *testing.T) {
	r := newRig(t)
	c := r.conversation("claude")
	for _, id := range []string{"not-a-uuid", "00000000-0000-0000-0000-000000000000", c.ID} {
		if _, status := r.exchange(id); status != http.StatusNotFound {
			t.Errorf("%q: status %d, want 404", id, status)
		}
	}
}

// Records reference their entries without a cascade (D-009), so deleting a
// conversation has to take them first or the delete fails outright.
func TestDeletingAConversationDeletesItsRecords(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")
	turn := sendTurn(r, d, c)
	d.send(protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: turn.TurnID,
		Sent: &protocol.ExchangeSent{Program: "claude", Prompt: "say hi", Launched: true}})
	d.send(protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: turn.TurnID,
		Lines: []protocol.ExchangeLine{{Seq: 1, Stream: protocol.StreamStdout, Text: claudeInit}}})
	d.send(protocol.DaemonMessage{Type: protocol.DaemonDone, TurnID: turn.TurnID, Full: "hi", Tokens: 5})
	r.awaitEntry(c.ID, turn.EntryID, func(e protocol.Entry) bool { return e.Usage != nil })

	res := r.do(http.MethodDelete, "/api/conversations/"+c.ID, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %s", res.Status)
	}
	if _, status := r.exchange(turn.EntryID); status != http.StatusNotFound {
		t.Errorf("the record outlived its conversation: status %d", status)
	}
}
