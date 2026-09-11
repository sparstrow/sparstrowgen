package main

import (
	"context"
	"testing"
)

func TestStoppingARunningTurnCancelsIt(t *testing.T) {
	r := newRunningTurns()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !r.begin("t1", cancel) {
		t.Fatal("a fresh turn should be allowed to start")
	}
	if !r.stop("t1") {
		t.Error("stop reported the turn was not running")
	}
	if ctx.Err() == nil {
		t.Error("the turn's context was not cancelled")
	}
	if !r.end("t1") {
		t.Error("end should report the turn was stopped on purpose")
	}
}

// The case that is easy to get wrong. run_turn and stop_turn arrive on the same
// socket but each turn runs on its own goroutine, so a stop sent immediately
// after a send can reach the registry first. Without `asked`, the cancel would
// land on nothing and the CLI would run to completion after the owner had
// already stopped it.
func TestAStopThatOvertakesItsOwnStartPreventsTheTurn(t *testing.T) {
	r := newRunningTurns()

	if r.stop("t1") {
		t.Error("stop should report the turn was not running yet")
	}
	if r.begin("t1", func() {}) {
		t.Fatal("a turn already asked to stop must not start")
	}
	// And the request is not left lying around to kill a later turn that
	// happens to reuse the id.
	if !r.begin("t1", func() {}) {
		t.Error("the stop request outlived the turn it was meant for")
	}
}

func TestATurnThatFinishesNormallyIsNotReportedAsStopped(t *testing.T) {
	r := newRunningTurns()
	if !r.begin("t1", func() {}) {
		t.Fatal("begin")
	}
	if r.end("t1") {
		t.Error("end reported a stop that never happened")
	}
}

// Stopping one turn must not touch another. The daemon runs turns concurrently
// and a shared context would have taken them all down together.
func TestStoppingOneTurnLeavesTheOthersAlone(t *testing.T) {
	r := newRunningTurns()
	ctxA, cancelA := context.WithCancel(context.Background())
	ctxB, cancelB := context.WithCancel(context.Background())
	defer cancelA()
	defer cancelB()

	r.begin("a", cancelA)
	r.begin("b", cancelB)
	r.stop("a")

	if ctxA.Err() == nil {
		t.Error("the stopped turn was not cancelled")
	}
	if ctxB.Err() != nil {
		t.Error("stopping one turn cancelled another")
	}
	if r.end("b") {
		t.Error("the untouched turn was reported as stopped")
	}
}
