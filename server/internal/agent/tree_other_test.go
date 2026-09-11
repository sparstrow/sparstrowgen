//go:build !windows

package agent

import (
	"context"
	"io"
	"testing"
	"time"
)

/* The Unix half of the question tree_windows_test.go asks: when a turn is
stopped, does what the AGENT was running stop too?

Same shape, same reasoning — a grandchild that outlives its parent and goes on
holding the inherited stdout pipe — built from `sh` and `sleep` instead of
cmd.exe and ping. No agent CLI runs here either (docs/KnownGaps.md G-12).

Note `sleep 60; echo done` rather than plain `sleep 60`: a shell whose last
command is the only command execs it and BECOMES it, leaving no grandchild at
all and a test that proves nothing. The trailing command is what keeps sh a
separate process. */

func grandchildTree(t *testing.T, ctx context.Context) (*process, io.ReadCloser) {
	t.Helper()
	cmd := command(ctx, "", "sh", "-c", "sleep 60; echo done")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	proc, err := launch(ctx, cmd, stdout)
	if err != nil {
		t.Fatal(err)
	}
	return proc, stdout
}

func eofWithin(r io.Reader, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(io.Discard, r)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// The problem, demonstrated: killing the leader alone leaves the grandchild
// running, and it keeps the pipe open behind it.
func TestKillingOnlyTheLeaderLeavesTheGrandchildRunning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := command(ctx, "", "sh", "-c", "sleep 60; echo done")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil { // deliberately NOT launch
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })

	// Give sh time to actually fork sleep. Killing before it has proves nothing:
	// the pipe would end because there was never a grandchild.
	time.Sleep(500 * time.Millisecond)

	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_, _ = cmd.Process.Wait()

	if eofWithin(stdout, 3*time.Second) {
		t.Skip("sleep did not outlive sh here, so this machine cannot show the problem")
	}
	// Reaching this line is the assertion: the leader is dead and something it
	// spawned still holds the pipe.
}

// The fix. Same tree, stopped through launch, and this time the group goes.
func TestCancellingATurnKillsWhatTheAgentSpawned(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proc, stdout := grandchildTree(t, ctx)
	time.Sleep(500 * time.Millisecond) // let sh fork sleep

	waited := make(chan error, 1)
	go func() { waited <- proc.Wait() }()
	cancel()

	select {
	case <-waited:
	case <-time.After(15 * time.Second):
		t.Fatal("Wait did not return after the turn was cancelled")
	}
	if !proc.treeGone {
		t.Error("the stop was not confirmed clean: something in the group outlived it")
	}
	if !eofWithin(stdout, 3*time.Second) {
		t.Error("the pipe never ended, so a descendant is still holding it open")
	}
}

// A turn that finishes normally must not be terminated, and Wait must come back
// promptly rather than sitting out the grace window.
func TestATurnThatFinishesOnItsOwnIsNotKilled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := command(ctx, "", "sh", "-c", "exit 0")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	proc, err := launch(ctx, cmd, stdout)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- proc.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("a command that exited 0 reported: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Wait blocked on a process that had already exited")
	}
}
