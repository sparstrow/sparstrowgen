//go:build windows

package agent

import (
	"context"
	"io"
	"os/exec"
	"testing"
	"time"
)

/* Does stopping a turn actually stop what the agent was running?

The point of owning the tree is the process the agent spawned, not the agent. A
test that only checked the CLI died would pass on the broken version too —
killing the leader has always worked. So the subject here is a GRANDCHILD.

cmd.exe running ping is that shape in two Windows built-ins: cmd.exe is the
child we launch, ping is the grandchild it spawns, and it runs for a minute so
it cannot exit on its own and fake a pass. No agent CLI is involved — a default
test that resolved `claude` from PATH would spend the owner's quota every time
anyone ran go test (docs/KnownGaps.md G-12).

**How a surviving grandchild is observed:** it inherited stdout, so while it is
alive the pipe does not reach EOF even though the leader is dead. That is not a
proxy for the real symptom, it IS the real symptom — a descendant holding the
pipe open is what wedges the parser. */

// grandchildTree starts cmd.exe running a one-minute ping and returns its
// stdout. Nothing is read from it: the question is only whether it ends.
func grandchildTree(t *testing.T, ctx context.Context) (*exec.Cmd, io.ReadCloser) {
	t.Helper()
	cmd := command(ctx, "", "cmd.exe", "/c", "ping", "-n", "60", "127.0.0.1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	return cmd, stdout
}

// eofWithin reports whether the pipe ends within the timeout. A live grandchild
// holding the write end is exactly what stops that happening.
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

func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for %s", timeout, what)
}

// The problem, demonstrated. This is what the daemon did before D-021, and it
// is why owning the tree is not over-engineering: the grandchild outlives the
// kill and keeps the pipe open behind it.
func TestKillingOnlyTheLeaderLeavesTheGrandchildRunning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd, stdout := grandchildTree(t, ctx)
	if err := cmd.Start(); err != nil { // deliberately NOT launch
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); _, _ = cmd.Process.Wait() })

	// Wait until the grandchild is genuinely running before killing the leader.
	// Killing straight after Start proves nothing: cmd.exe has not spawned ping
	// yet, so the pipe ends because there was never a grandchild to survive.
	// ping writes its own banner, so a byte on stdout is that proof.
	if _, err := io.ReadFull(stdout, make([]byte, 1)); err != nil {
		t.Fatalf("ping produced no output to wait on: %v", err)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_, _ = cmd.Process.Wait()

	if eofWithin(stdout, 3*time.Second) {
		t.Skip("ping did not outlive cmd.exe here, so this machine cannot show the problem")
	}
	// Reaching this line IS the assertion: the leader is dead and something it
	// spawned is still holding the pipe.
}

// The fix. Same tree, stopped through launch, and this time it goes.
func TestCancellingATurnKillsWhatTheAgentSpawned(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd, stdout := grandchildTree(t, ctx)
	proc, err := launch(ctx, cmd, stdout)
	if err != nil {
		t.Fatal(err)
	}
	if proc.tree == nil {
		t.Fatal("no ownership was taken, so this test would prove nothing")
	}

	// Two processes in the job says the grandchild was captured rather than
	// escaping while the child was starting.
	waitFor(t, 10*time.Second, "cmd.exe to spawn ping inside the job", func() bool {
		n, err := proc.tree.active()
		return err == nil && n >= 2
	})

	waited := make(chan error, 1)
	go func() { waited <- proc.Wait() }()
	cancel()

	select {
	case <-waited:
	case <-time.After(15 * time.Second):
		t.Fatal("Wait did not return after the turn was cancelled")
	}
	if !proc.treeGone {
		t.Error("the stop was not confirmed clean: something in the tree outlived it")
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

	cmd := command(ctx, "", "cmd.exe", "/c", "exit", "0")
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
