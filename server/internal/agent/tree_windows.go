//go:build windows

package agent

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

/* Owning an agent's whole process tree on Windows, via a Job Object.

Adapted from Multica's server/pkg/agent/proc_windows.go, which drives these same
CLIs on this same machine. Trimmed to what stopping a turn needs: the console
handling there solves a popup problem we do not have, since our daemon runs in a
terminal its children inherit.

The load-bearing part is the ORDER, and it is not obvious: the child is created
suspended, assigned to the job, and only then resumed. Windows grants job
membership only to processes created after the assignment and never
retroactively — so assigning after a plain Start leaves a window in which the
agent has already spawned tool subprocesses outside the job. That is worse than
owning nothing, because the job then reports an empty tree while the escaped
processes are still running, and a stop would be reported as confirmed when it
is not. */

// createSuspended starts the child with its initial thread suspended, so it can
// be placed in the job before it executes a single instruction.
const createSuspended = 0x00000004

type processTree struct {
	job windows.Handle
}

// prepare configures cmd before it is started.
func prepare(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createSuspended
}

// own takes ownership of a started-but-suspended child. A non-nil error means
// the caller must still resume the child: it runs unowned rather than not at
// all, exactly as it did before any of this existed.
func own(cmd *exec.Cmd) (*processTree, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create job object: %w", err)
	}
	// KILL_ON_JOB_CLOSE makes the tree die with the last handle, so a daemon
	// that crashes cannot strand an agent's subprocesses. It also means release
	// must run only once the tree is finished with.
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("set KILL_ON_JOB_CLOSE: %w", err)
	}
	handle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("open process: %w", err)
	}
	defer windows.CloseHandle(handle)
	if err := windows.AssignProcessToJobObject(job, handle); err != nil {
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("assign process to job object: %w", err)
	}
	return &processTree{job: job}, nil
}

// resume releases the initial thread of a suspended child. A freshly created
// process has exactly one thread; the snapshot is filtered by owning pid so a
// concurrent launch cannot resume somebody else's.
func resume(pid int) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return fmt.Errorf("snapshot threads: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ThreadEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	resumed := 0
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != uint32(pid) {
			continue
		}
		thread, openErr := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if openErr != nil {
			return fmt.Errorf("open thread %d: %w", entry.ThreadID, openErr)
		}
		_, resumeErr := windows.ResumeThread(thread)
		_ = windows.CloseHandle(thread)
		if resumeErr != nil {
			return fmt.Errorf("resume thread %d: %w", entry.ThreadID, resumeErr)
		}
		resumed++
	}
	if resumed == 0 {
		return fmt.Errorf("no threads found for pid %d", pid)
	}
	return nil
}

// terminate kills every process in the tree.
//
// Windows has no SIGTERM equivalent and no process-group signalling, so there
// is only one way to stop a job and it is the hard one. The graceful attempt an
// agent gets is the EOF on its stdin, which happens before this.
func (t *processTree) terminate(cmd *exec.Cmd) {
	if t == nil || t.job == 0 {
		// Ownership was never taken. Killing the leader alone is all that is
		// left, and it is what the caller was told to expect.
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return
	}
	if err := windows.TerminateJobObject(t.job, 1); err != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// gone reports whether every process in the tree has exited, polling until the
// timeout. Without an owned tree there is nothing to observe, so it reports
// false — "could not be confirmed", never "confirmed clean".
func (t *processTree) gone(timeout time.Duration) bool {
	if t == nil || t.job == 0 {
		return false
	}
	deadline := time.Now().Add(timeout)
	for {
		active, err := t.active()
		if err != nil {
			return false
		}
		if active == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// jobObjectBasicAccountingInformation mirrors
// JOBOBJECT_BASIC_ACCOUNTING_INFORMATION. x/sys/windows exports the info class
// but not the struct.
type jobObjectBasicAccountingInformation struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

func (t *processTree) active() (uint32, error) {
	var info jobObjectBasicAccountingInformation
	if err := windows.QueryInformationJobObject(
		t.job,
		windows.JobObjectBasicAccountingInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil,
	); err != nil {
		return 0, fmt.Errorf("query job object: %w", err)
	}
	return info.ActiveProcesses, nil
}

// release drops ownership. Closing the handle is what kills anything still
// inside, so this runs only after the process has been reaped.
func (t *processTree) release() {
	if t == nil || t.job == 0 {
		return
	}
	_ = windows.CloseHandle(t.job)
	t.job = 0
}
