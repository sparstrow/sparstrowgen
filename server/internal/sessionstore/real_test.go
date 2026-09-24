package sessionstore

import (
	"os"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/agent"
)

// Reads this computer's real session stores, never runs a CLI. Off unless
// SPARSTROWGEN_REAL_STORES names sessions to read, as provider=id pairs:
//
//	SPARSTROWGEN_REAL_STORES="claude=<session> codex=<thread> agy=<conversation>" go test ./internal/sessionstore -run Real -v
//
// It is how the readers are checked against a CLI's current format after an
// update, which the fixtures cannot tell anyone.
func TestReadRealStores(t *testing.T) {
	spec := os.Getenv("SPARSTROWGEN_REAL_STORES")
	if spec == "" {
		t.Skip("set SPARSTROWGEN_REAL_STORES to read this computer's own session stores")
	}
	for _, pair := range strings.Fields(spec) {
		provider, id, _ := strings.Cut(pair, "=")
		rec := Read(provider, id)
		if rec.Error != "" {
			t.Errorf("%s: %s", provider, rec.Error)
		}
		pieces, missing := agent.FedContext(provider, rec.Documents)
		total, byKind := 0, map[string]int{}
		for _, p := range pieces {
			total += p.Chars
			byKind[p.Kind]++
		}
		t.Logf("%s: %d records from %s -> %d pieces, %d characters, %v; missing %v",
			provider, len(rec.Documents), rec.From, len(pieces), total, byKind, missing)
		for _, p := range pieces {
			tokens := ""
			if p.Tokens != nil {
				tokens = " tokens"
			}
			t.Logf("    %-12s %-44.44s %7d chars%s  %s", p.Kind, p.Name, p.Chars, tokens, p.Source)
		}
		if len(pieces) == 0 {
			t.Errorf("%s: nothing read", provider)
		}
	}
}
