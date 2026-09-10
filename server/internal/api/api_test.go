package api

import (
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// The replay quote is the number the owner decides on before committing to a
// switch, so what it is derived from matters. It used to be a flat 1300 tokens
// per message with nothing behind it — that quoted 2.6k for a switch that
// really cost 53.
func TestEstimateTokensScalesWithActualText(t *testing.T) {
	short := []protocol.Entry{{Text: "hi"}}
	long := []protocol.Entry{{Text: string(make([]byte, 4000))}}

	if got := estimateTokens(nil); got != 0 {
		t.Errorf("nothing to replay must cost 0, got %d", got)
	}
	if estimateTokens(short) >= estimateTokens(long) {
		t.Error("a longer conversation must be quoted higher than a shorter one")
	}
	// chars/4 is the standard rough ratio; it is an estimate, and the surface
	// says so with a ~.
	if got := estimateTokens(long); got != 1000 {
		t.Errorf("4000 characters = %d tokens, want 1000", got)
	}
}

func TestEstimateTokensSumsEveryEntry(t *testing.T) {
	entries := []protocol.Entry{
		{Text: "aaaa"}, {Text: "bbbb"}, {Text: "cccc"},
	}
	if got := estimateTokens(entries); got != 3 {
		t.Errorf("got %d, want 3 — the whole replay is charged, not just the last message", got)
	}
}
