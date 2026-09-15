package agent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// Drives the real CLIs, so it spends the owner's agent quota and is skipped
// unless asked for (docs/KnownGaps.md G-12):
//
//	SPARSTROWGEN_LIVE_AGENTS=claude,codex,agy go test -run TestLive ./internal/agent/
//
// SPARSTROWGEN_LIVE_DIR is the folder the agents run in; keep it a scratch one.

// B-32: a prompt longer than the whole Windows command line reaches each agent.
func TestLiveAPromptLongerThanTheWindowsCommandLineIsAnswered(t *testing.T) {
	wanted := os.Getenv("SPARSTROWGEN_LIVE_AGENTS")
	if wanted == "" {
		t.Skip("set SPARSTROWGEN_LIVE_AGENTS to run the real agent CLIs")
	}
	dir := os.Getenv("SPARSTROWGEN_LIVE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	filler := strings.Repeat("This line is filler and needs no reply. ", 1_000)
	prompt := "Ignore the filler below and reply with exactly: ok\n\n" + filler + "\nReply with exactly: ok"
	if len(prompt) <= 32_767 {
		t.Fatalf("prompt is %d characters; it must exceed the Windows limit to prove anything", len(prompt))
	}
	models := map[string]string{"claude": "claude-haiku-4-5-20251001", "agy": "gemini-3.8-flash-low"}

	for _, id := range strings.Split(wanted, ",") {
		backend, ok := Backends()[id]
		if !ok {
			t.Fatalf("unknown agent %q", id)
		}
		t.Run(id, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			start := time.Now()
			session, err := backend.Execute(ctx, prompt, ExecOptions{Cwd: dir, Model: models[id]})
			if err != nil {
				t.Fatalf("did not start: %v", err)
			}
			for range session.Messages {
			}
			res := <-session.Result
			if res.Err != nil {
				t.Fatalf("failed after %s: %v (text %q)", time.Since(start).Round(time.Second), res.Err, res.Text)
			}
			if !strings.Contains(strings.ToLower(res.Text), "ok") {
				t.Fatalf("answer = %q", res.Text)
			}
			t.Logf("%s answered %q to a %d-character prompt in %s, %d tokens",
				id, res.Text, len(prompt), time.Since(start).Round(time.Second), res.Tokens)
		})
	}
}
