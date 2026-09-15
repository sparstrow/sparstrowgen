package main

import (
	"context"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

func TestInstalledAgentsAreReportedAgainOnlyWhenTheyChange(t *testing.T) {
	ctx := context.Background()
	fp := "claude:1;codex:missing;agy:missing;"
	detections := 0
	providers := []protocol.Provider{{ID: "claude", Availability: protocol.Available}}
	w := &providerWatch{
		fingerprint: func() string { return fp },
		detect:      func(context.Context) []protocol.Provider { detections++; return providers },
		last:        fp,
		providers:   providers,
	}

	if _, changed := w.check(ctx); changed || detections != 0 {
		t.Fatalf("nothing changed: reported %v after %d detections, want neither", changed, detections)
	}

	// claude updated in place, and the app would show exactly the same thing.
	fp = "claude:2;codex:missing;agy:missing;"
	if _, changed := w.check(ctx); changed || detections != 1 {
		t.Fatalf("an update that changes nothing shown: reported %v after %d detections, want 1 detection and no report", changed, detections)
	}

	// agy installed.
	fp = "claude:2;codex:missing;agy:1;"
	providers = []protocol.Provider{{ID: "claude", Availability: protocol.Available}, {ID: "agy", Availability: protocol.Available}}
	next, changed := w.check(ctx)
	if !changed || len(next) != 2 || detections != 2 {
		t.Fatalf("agy installed: reported %v with %d providers after %d detections", changed, len(next), detections)
	}

	// And that is now what was last reported.
	if _, changed := w.check(ctx); changed || detections != 2 {
		t.Fatalf("reported again with nothing new: %v after %d detections", changed, detections)
	}
}
