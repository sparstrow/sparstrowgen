package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/agent"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* The watchdog, tested against a fake agent rather than a real one.

A CLI that wedges is exactly what cannot be arranged on demand, and a test that
resolved `claude` from PATH would spend the owner's quota on every run
(docs/KnownGaps.md G-12). What the watchdog actually does is observable without
either: after a budget of silence it cancels the turn's context, and that
cancellation is what `launch` turns into a dead process tree.

The daemon under test has no connection, so its sends fail and are discarded —
which is fine here, because the question is whether the agent gets stopped, not
what the server is told about it. */

// scriptedAgent emits the messages it is given, spaced by gap, and then waits to
// be cancelled. It never exits on its own, so any turn that ends here ended
// because something stopped it.
type scriptedAgent struct {
	beats     int
	gap       time.Duration
	cancelled chan struct{}
}

func (scriptedAgent) ID() string { return "fake" }

func (a scriptedAgent) Execute(ctx context.Context, _ string, _ agent.ExecOptions) (*agent.Session, error) {
	msgs := make(chan agent.Message)
	res := make(chan agent.Result, 1)
	go func() {
		defer close(msgs)
		defer func() { res <- agent.Result{Text: "whatever arrived"} }()
		for i := 0; i < a.beats; i++ {
			select {
			case <-ctx.Done():
				close(a.cancelled)
				return
			case <-time.After(a.gap):
			}
			select {
			case msgs <- agent.Message{Type: agent.MessageDelta, Text: "."}:
			case <-ctx.Done():
				close(a.cancelled)
				return
			}
		}
		<-ctx.Done() // silent from here, waiting to be stopped
		close(a.cancelled)
	}()
	return &agent.Session{Messages: msgs, Result: res}, nil
}

func testDaemon(t *testing.T, backend agent.Backend) *daemon {
	t.Helper()
	return &daemon{
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		backends: map[string]agent.Backend{"fake": backend},
		turns:    newRunningTurns(),
	}
}

// withBudget shrinks the watchdog for the duration of a test. Fifteen minutes is
// the right production value and an impossible test.
func withBudget(t *testing.T, d time.Duration) {
	t.Helper()
	original := idleBudget
	idleBudget = d
	t.Cleanup(func() { idleBudget = original })
}

func aTurn() protocol.RunTurn {
	return protocol.RunTurn{TurnID: "t1", Provider: "fake", Prompt: "hello"}
}

// The hole this closes: a CLI that stops responding used to leave the turn
// running forever, and with it a locked composer and a working indicator ticking
// against nothing.
func TestATurnThatGoesQuietIsEndedRatherThanWaitedOnForever(t *testing.T) {
	withBudget(t, 100*time.Millisecond)
	backend := scriptedAgent{cancelled: make(chan struct{})}
	d := testDaemon(t, backend)

	done := make(chan struct{})
	go func() { defer close(done); d.runTurn(context.Background(), aTurn()) }()

	select {
	case <-backend.cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("a silent agent was never stopped")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runTurn did not return after the watchdog fired")
	}
}

// The half that matters more, because getting it wrong is worse: an agent that
// IS working must never be killed for taking a while. Every message resets the
// budget, so a turn producing output steadily outlives a budget far shorter than
// the turn itself — here, ten beats spaced wider than half the budget each.
func TestAnAgentThatKeepsTalkingIsNeverKilledForRunningLong(t *testing.T) {
	withBudget(t, 200*time.Millisecond)
	backend := scriptedAgent{beats: 10, gap: 120 * time.Millisecond, cancelled: make(chan struct{})}
	d := testDaemon(t, backend)

	done := make(chan struct{})
	go func() { defer close(done); d.runTurn(context.Background(), aTurn()) }()

	// 10 × 120ms = 1.2s of work under a 200ms budget. A total timeout would have
	// killed this six times over; an inactivity watchdog must not touch it.
	select {
	case <-backend.cancelled:
		t.Fatal("an agent that was still producing output was killed for running long")
	case <-time.After(1 * time.Second):
	}

	// And it is still stoppable afterwards, so resetting the timer has not
	// detached the turn from its cancellation.
	if !d.turns.stop("t1") {
		t.Error("the turn was no longer registered as running")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runTurn did not return after the turn was stopped")
	}
}

// An unusable setting is worth saying out loud rather than silently ignoring:
// someone who sets it meant something by it.
func TestAnUnusableIdleTimeoutFallsBackLoudly(t *testing.T) {
	for _, bad := range []string{"soon", "0", "-5m"} {
		t.Setenv("TURN_IDLE_TIMEOUT", bad)
		if got := envDuration("TURN_IDLE_TIMEOUT", time.Minute); got != time.Minute {
			t.Errorf("%q gave %s, want the fallback", bad, got)
		}
	}
	t.Setenv("TURN_IDLE_TIMEOUT", "45s")
	if got := envDuration("TURN_IDLE_TIMEOUT", time.Minute); got != 45*time.Second {
		t.Errorf("a usable value was ignored: got %s", got)
	}
}
