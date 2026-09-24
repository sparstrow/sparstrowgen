package main

import (
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/agent"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/sessionstore"
)

// readContext reads what a CLI fed its model from its own session store. A
// variable so a test's fake agent is not looked for in the real stores.
var readContext = sessionstore.Read

/* A turn's record, on its way to the server (docs/specs/2026-09-23-raw-exchange.md).

Lines are sent in batches rather than one message each: claude prints close to a
hundred lines for a four-sentence answer, and a message per line would double the
socket traffic for no reader's benefit. A batch goes when it is big enough, or a
quarter of a second after it started, so a Raw view watching a slow turn still
sees it move. Everything is flushed before the turn's own ending is sent, and the
server handles one socket's messages in order, so a finished turn's record is
already complete when the turn is. */

var (
	// exchangeBudget is how much of one turn's output is kept. A runaway
	// command can print gigabytes; the record keeps everything up to here and
	// then counts, so it says how much it did not keep rather than stopping
	// quietly. The turn itself is untouched either way.
	exchangeBudget int64 = 16 << 20
	flushEvery           = 250 * time.Millisecond
)

const (
	flushLines = 200
	flushBytes = 256 << 10
)

type exchangeRecord struct {
	send   func(protocol.DaemonMessage) error
	turnID string

	seq          int32
	kept         int64
	pending      []protocol.ExchangeLine
	pendingBytes int
	dropped      protocol.ExchangeDropped
	// droppedNew says the count moved since it was last sent.
	droppedNew bool
}

func newExchangeRecord(turnID string, send func(protocol.DaemonMessage) error) *exchangeRecord {
	return &exchangeRecord{send: send, turnID: turnID}
}

// sent records what the CLI was handed. First, before any line.
func (r *exchangeRecord) sent(s protocol.ExchangeSent) {
	if s.Args == nil {
		s.Args = []string{}
	}
	_ = r.send(protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: r.turnID, Sent: &s})
}

// add keeps one line, or counts it if the budget is spent. It reports whether
// the batch is now big enough to go.
func (r *exchangeRecord) add(l agent.Line) bool {
	size := int64(len(l.Text))
	if l.Cut > 0 {
		r.dropped.Bytes += l.Cut
		r.droppedNew = true
	}
	if r.kept+size > exchangeBudget {
		r.dropped.Lines++
		r.dropped.Bytes += size
		r.droppedNew = true
		return false
	}
	r.seq++
	r.kept += size
	r.pending = append(r.pending, protocol.ExchangeLine{Seq: r.seq, AtMs: l.AtMs, Stream: l.Stream, Text: l.Text})
	r.pendingBytes += len(l.Text)
	return len(r.pending) >= flushLines || r.pendingBytes >= flushBytes
}

func (r *exchangeRecord) flush() {
	if len(r.pending) == 0 && !r.droppedNew {
		return
	}
	msg := protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: r.turnID, Lines: r.pending}
	if r.droppedNew {
		d := r.dropped
		msg.Dropped = &d
	}
	// A failed send loses this batch, and the browser notices the gap in seq.
	// Retrying would only reorder it behind lines sent after it.
	_ = r.send(msg)
	r.pending, r.pendingBytes, r.droppedNew = nil, 0, false
}

// context records what the CLI fed its model, once, after the last lines. A
// store that could not be read still sends its reason, so the record can say
// why it has nothing rather than looking like a daemon that never tried.
func (r *exchangeRecord) context(c protocol.ContextRecord) {
	_ = r.send(protocol.DaemonMessage{Type: protocol.DaemonExchange, TurnID: r.turnID, Context: &c})
}

// notLaunched records a turn the daemon refused before starting the CLI: what
// would have been sent, and that it never was. The reason travels as the turn's
// failure, where it is already shown.
func notLaunched(t protocol.RunTurn, prompt string) protocol.ExchangeSent {
	return protocol.ExchangeSent{
		Program:         t.Provider,
		Args:            []string{},
		Cwd:             t.Cwd,
		ResumeSessionID: t.ResumeSessionID,
		Prompt:          prompt,
		Launched:        false,
	}
}
