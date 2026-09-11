package main

import (
	"context"
	"sync"
)

/* Which turns this daemon is running, so one can be stopped by id.

The two maps are not redundant. A stop can arrive before the turn it names has
registered: run_turn and stop_turn are read off the same socket and each turn
runs on its own goroutine, so the order they reach this map is not the order
they were sent. `asked` is what remembers a stop that overtook its start, and a
turn that finds itself already asked never launches the CLI at all — which is
the only way to make "I stopped it immediately" mean anything. */

type runningTurns struct {
	mu     sync.Mutex
	cancel map[string]context.CancelFunc
	asked  map[string]bool
}

func newRunningTurns() *runningTurns {
	return &runningTurns{
		cancel: map[string]context.CancelFunc{},
		asked:  map[string]bool{},
	}
}

// begin registers a turn about to start. It reports false when that turn has
// already been asked to stop, in which case the caller must not run anything.
func (r *runningTurns) begin(id string, cancel context.CancelFunc) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.asked[id] {
		// Nothing will run, so there is nothing left to stop. Dropping it here
		// is also what keeps `asked` from accumulating ids forever.
		delete(r.asked, id)
		return false
	}
	r.cancel[id] = cancel
	return true
}

// stop ends a running turn, and reports whether it was actually running. An
// unknown id is remembered anyway: its start may still be in flight.
func (r *runningTurns) stop(id string) bool {
	r.mu.Lock()
	cancel, running := r.cancel[id]
	r.asked[id] = true
	r.mu.Unlock()
	// Outside the lock: cancelling wakes the turn's goroutine, which will want
	// this mutex on its way out.
	if running {
		cancel()
	}
	return running
}

// end deregisters a finished turn and reports whether it ended because someone
// asked it to. That answer is what separates "stopped" from "failed", and it
// has to be read exactly once, by the goroutine sending the turn's last
// message.
func (r *runningTurns) end(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	asked := r.asked[id]
	delete(r.cancel, id)
	delete(r.asked, id)
	return asked
}

// cancelAll stops every running turn, without recording any of them as
// deliberately stopped.
//
// That distinction is the whole reason this is not just stop() in a loop: the
// machine losing its connection is not the owner changing their mind, and the
// transcript should not claim it was. Each turn still reports through its own
// path — which will fail to send, because the socket is why we are here.
//
// The point is quota. The server has already given up on these turns
// (docs/Bugs.md B-8), so anything still running is work nobody will ever
// receive, and on an agent CLI that is real money.
func (r *runningTurns) cancelAll() int {
	r.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(r.cancel))
	for _, c := range r.cancel {
		cancels = append(cancels, c)
	}
	r.mu.Unlock()
	// Outside the lock: each cancel wakes a turn that wants this mutex to end.
	for _, c := range cancels {
		c()
	}
	return len(cancels)
}
