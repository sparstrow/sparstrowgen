//go:build !windows

package agent

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
)

/* Owning an agent's whole process tree everywhere that is not Windows, via a
process group.

Same contract as tree_windows.go — see the comment there for why any of this is
needed. This side is much simpler because a process group is requested before
the child starts, so there is no suspend-assign-resume dance and `own` and
`resume` have nothing to do.

NOT VERIFIED. The daemon runs on the owner's Windows machine and that is the
only place this has been exercised; this file exists so the module still builds
and behaves sanely on Linux, where the server (but not the daemon) is headed.
See docs/KnownGaps.md. */

// prepare puts the child in its own process group, so signals can be sent to
// the group rather than to the leader alone.
func prepare(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// hideConsole is Windows-only: nothing here opens a window for a child.
func hideConsole(*exec.Cmd) {}

type processTree struct{ pgid int }

// own records the group. Membership was already granted by prepare, so this
// cannot fail — the error exists only to match the Windows signature.
func own(cmd *exec.Cmd) (*processTree, error) {
	return &processTree{pgid: cmd.Process.Pid}, nil
}

// resume is a no-op: the child was never suspended.
func resume(int) error { return nil }

// terminate asks the group to stop, then insists.
//
// Unlike Windows there is a real graceful signal here, so it is used: SIGTERM
// first, and SIGKILL only for whatever ignored it.
func (t *processTree) terminate(cmd *exec.Cmd) {
	if t == nil || t.pgid == 0 {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return
	}
	if err := syscall.Kill(-t.pgid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return // already gone
		}
	}
	if t.gone(terminateGrace) {
		return
	}
	if err := syscall.Kill(-t.pgid, syscall.SIGKILL); err != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// gone reports whether every process in the group has exited. Signal 0 checks
// for existence without delivering anything.
func (t *processTree) gone(timeout time.Duration) bool {
	if t == nil || t.pgid == 0 {
		return false
	}
	deadline := time.Now().Add(timeout)
	for {
		if err := syscall.Kill(-t.pgid, 0); errors.Is(err, syscall.ESRCH) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// release has nothing to drop: a process group needs no handle.
func (t *processTree) release() {}
