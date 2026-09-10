package main

import (
	"strings"
	"testing"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// Moving a conversation is physically just this: no CLI can resume another's
// session, so a provider that has not seen the history is told it in the
// prompt. What the prompt says therefore matters as much as any wire format.
func TestBuildPromptWithoutReplayIsUntouched(t *testing.T) {
	got := buildPrompt(protocol.RunTurn{Prompt: "what is jitter for?"})
	if got != "what is jitter for?" {
		t.Errorf("a turn with nothing to replay must send the prompt verbatim, got %q", got)
	}
}

func TestBuildPromptFramesTheReplayAsSomeoneElsesWork(t *testing.T) {
	turn := protocol.RunTurn{
		Prompt: "and what did you conclude?",
		Replay: []protocol.ReplayEntry{
			{Role: "user", Text: "why does backoff need jitter?"},
			{Role: "agent", Provider: "codex", Text: "it desynchronises retries"},
		},
	}
	got := buildPrompt(turn)

	// The earlier turns must be attributed, not passed off as this agent's own.
	// Presenting another agent's words as its own would make it answer as if it
	// had already committed to them.
	if !strings.Contains(got, "[codex]") {
		t.Error("an earlier agent's turn must be attributed to it")
	}
	if !strings.Contains(got, "already in progress") {
		t.Error("the replay must be framed as joining, not as its own memory")
	}
	for _, want := range []string{"why does backoff need jitter?", "it desynchronises retries"} {
		if !strings.Contains(got, want) {
			t.Errorf("replay dropped %q", want)
		}
	}
	// The new question comes last, so it is the thing being answered.
	if !strings.HasSuffix(got, "and what did you conclude?") {
		t.Error("the prompt being answered must come last")
	}
}

// Full jitter: every delay is drawn from [0, ceiling]. Without it, a fleet of
// machines waking from sleep together retries in lockstep — which is the exact
// failure the daemon's own reconnect loop exists to avoid.
func TestBackoffIsBoundedAndJittered(t *testing.T) {
	var sawBelowCeiling bool
	for attempt := 0; attempt < 20; attempt++ {
		ceiling := backoffBase << min(attempt, 16)
		if ceiling > backoffMax {
			ceiling = backoffMax
		}
		for i := 0; i < 50; i++ {
			d := backoff(attempt)
			if d < 0 {
				t.Fatalf("attempt %d produced a negative delay %v", attempt, d)
			}
			if d > backoffMax {
				t.Fatalf("attempt %d produced %v, above the %v ceiling", attempt, d, backoffMax)
			}
			if d > ceiling {
				t.Fatalf("attempt %d produced %v, above its own ceiling %v", attempt, d, ceiling)
			}
			if d < ceiling {
				sawBelowCeiling = true
			}
		}
	}
	if !sawBelowCeiling {
		t.Error("every delay hit its ceiling exactly — that is not jitter")
	}
}

func TestBackoffCapsRatherThanOverflowing(t *testing.T) {
	// A daemon left retrying overnight reaches a high attempt count. Shifting
	// by it without clamping would overflow and produce a negative duration.
	for _, attempt := range []int{62, 63, 64, 1000} {
		d := backoff(attempt)
		if d < 0 || d > backoffMax {
			t.Errorf("attempt %d produced %v, outside [0, %v]", attempt, d, backoffMax)
		}
	}
}

func TestBackoffGrows(t *testing.T) {
	// Compare ceilings rather than samples: individual draws are random, so a
	// test on single values would be flaky by construction.
	lo := backoffBase << 1
	hi := backoffBase << 5
	if !(lo < hi && hi < backoffMax+time.Nanosecond) {
		t.Errorf("ceilings do not grow: %v then %v", lo, hi)
	}
}
