package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/agent"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// wired is a test daemon whose sends reach a real socket, so a test reads what
// the server would have been told, in the order it would have been told it.
func wired(t *testing.T, backend agent.Backend) (*daemon, <-chan protocol.DaemonMessage) {
	t.Helper()
	got := make(chan protocol.DaemonMessage, 256)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg protocol.DaemonMessage
			if json.Unmarshal(payload, &msg) == nil {
				got <- msg
			}
		}
	}))
	t.Cleanup(srv.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	d := testDaemon(t, backend)
	d.conn = conn
	return d, got
}

// until reads messages until one of the given type, and returns all of them.
func until(t *testing.T, got <-chan protocol.DaemonMessage, last string) []protocol.DaemonMessage {
	t.Helper()
	var out []protocol.DaemonMessage
	for {
		select {
		case msg := <-got:
			out = append(out, msg)
			if msg.Type == last {
				return out
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("no %q after %d messages: %+v", last, len(out), out)
		}
	}
}

// printingAgent prints the lines it is given and finishes.
type printingAgent struct{ lines []agent.Line }

func (printingAgent) ID() string { return "fake" }

func (a printingAgent) Execute(context.Context, string, agent.ExecOptions) (*agent.Session, error) {
	msgs := make(chan agent.Message)
	res := make(chan agent.Result, 1)
	go func() {
		for _, l := range a.lines {
			l := l
			msgs <- agent.Message{Type: agent.MessageLine, Line: &l}
		}
		close(msgs)
		res <- agent.Result{Text: "ok", Tokens: 12, SessionID: "s-9"}
	}()
	return &agent.Session{Messages: msgs, Result: res, Sent: protocol.ExchangeSent{
		Program: `C:\bin\fake.exe`, Args: []string{"-p"}, Prompt: "hello", Stdin: "hello", Launched: true,
	}}, nil
}

func TestATurnsRecordIsSentFirstAndCompleteBeforeTheTurnEnds(t *testing.T) {
	d, got := wired(t, printingAgent{lines: []agent.Line{
		{Stream: protocol.StreamStdout, AtMs: 5, Text: `{"type":"system"}`},
		{Stream: protocol.StreamStderr, AtMs: 9, Text: "warning: something"},
		{Stream: protocol.StreamStdout, AtMs: 12, Text: `{"type":"result"}`},
	}})
	d.runTurn(context.Background(), aTurn())
	msgs := until(t, got, protocol.DaemonDone)

	if msgs[0].Type != protocol.DaemonExchange || msgs[0].Sent == nil || !msgs[0].Sent.Launched || msgs[0].Sent.Program != `C:\bin\fake.exe` {
		t.Fatalf("first message = %+v, want the sent record", msgs[0])
	}
	var lines []protocol.ExchangeLine
	for _, m := range msgs[1 : len(msgs)-1] {
		if m.Type == protocol.DaemonExchange {
			lines = append(lines, m.Lines...)
		}
	}
	if len(lines) != 3 {
		t.Fatalf("lines before done = %d, want all 3: %+v", len(lines), msgs)
	}
	for i, l := range lines {
		if l.Seq != int32(i+1) {
			t.Errorf("line %d has seq %d", i, l.Seq)
		}
	}
	if lines[1].Stream != protocol.StreamStderr || lines[1].AtMs != 9 {
		t.Errorf("stderr line = %+v", lines[1])
	}
}

func TestWhatTheAgentWasFedIsSentAfterItsLinesAndBeforeTheEnd(t *testing.T) {
	original := readContext
	readContext = func(provider, session string) protocol.ContextRecord {
		return protocol.ContextRecord{From: provider + ":" + session,
			Documents: []protocol.ContextDocument{{Kind: "claude.attachment", Body: "{}"}}}
	}
	t.Cleanup(func() { readContext = original })

	d, got := wired(t, printingAgent{lines: []agent.Line{{Stream: protocol.StreamStdout, Text: `{"type":"result"}`}}})
	d.runTurn(context.Background(), aTurn())
	msgs := until(t, got, protocol.DaemonDone)

	lastLines, ctx := -1, -1
	for i, m := range msgs {
		if m.Type == protocol.DaemonExchange && len(m.Lines) > 0 {
			lastLines = i
		}
		if m.Type == protocol.DaemonExchange && m.Context != nil {
			ctx = i
		}
	}
	if ctx < 0 || ctx < lastLines || ctx != len(msgs)-2 {
		t.Fatalf("context at %d, last lines at %d, of %d messages", ctx, lastLines, len(msgs))
	}
	// Read for the session the agent reported, which is how its store is found.
	if c := msgs[ctx].Context; c.From != "fake:s-9" || len(c.Documents) != 1 {
		t.Errorf("context = %+v", c)
	}
}

func TestATurnRefusedForItsFolderStillRecordsWhatWouldHaveBeenSent(t *testing.T) {
	d, got := wired(t, &countingAgent{})
	turn := aTurn()
	turn.Cwd = filepath.Join(t.TempDir(), "deleted")
	turn.Replay = []protocol.ReplayEntry{{Role: "user", Text: "earlier"}}

	d.runTurn(context.Background(), turn)
	msgs := until(t, got, protocol.DaemonFailed)

	sent := msgs[0].Sent
	if msgs[0].Type != protocol.DaemonExchange || sent == nil {
		t.Fatalf("first message = %+v, want the sent record", msgs[0])
	}
	if sent.Launched {
		t.Error("a turn that never started was recorded as launched")
	}
	// The prompt as it would have gone, catch-up included.
	if !strings.Contains(sent.Prompt, "earlier") || !strings.HasSuffix(sent.Prompt, "hello") {
		t.Errorf("prompt = %q", sent.Prompt)
	}
	if sent.Args == nil {
		t.Error("args must be an empty list, never null, for the browser")
	}
}

func TestARecordPastItsBudgetCountsWhatItDidNotKeep(t *testing.T) {
	original := exchangeBudget
	exchangeBudget = 10
	t.Cleanup(func() { exchangeBudget = original })

	var sent []protocol.DaemonMessage
	r := newExchangeRecord("t1", func(m protocol.DaemonMessage) error { sent = append(sent, m); return nil })
	r.add(agent.Line{Stream: protocol.StreamStdout, Text: "123456"})
	r.add(agent.Line{Stream: protocol.StreamStdout, Text: "7890"})
	r.add(agent.Line{Stream: protocol.StreamStdout, Text: "over"})
	r.add(agent.Line{Stream: protocol.StreamStdout, Text: "x", Cut: 3})
	r.flush()

	if len(sent) != 1 {
		t.Fatalf("messages = %d, want one batch", len(sent))
	}
	if n := len(sent[0].Lines); n != 2 {
		t.Errorf("kept %d lines, want the 2 that fit", n)
	}
	// "over" and "x" did not fit; "x" had also lost 3 bytes to being over-long.
	if d := sent[0].Dropped; d == nil || d.Lines != 2 || d.Bytes != 4+1+3 {
		t.Errorf("dropped = %+v, want 2 lines and 8 bytes", d)
	}

	// Nothing new: nothing sent.
	r.flush()
	if len(sent) != 1 {
		t.Error("an empty flush sent a message")
	}
}
