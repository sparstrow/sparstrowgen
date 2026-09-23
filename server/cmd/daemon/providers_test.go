package main

import (
	"context"
	"testing"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

func TestInstalledAgentsAreReportedAgainOnlyWhenTheyChange(t *testing.T) {
	ctx := context.Background()
	fp := "claude:1;codex:missing;agy:missing;"
	detections := 0
	providers := []protocol.Provider{{ID: "claude", Availability: protocol.Available}}
	// The clock never moves here, so only the fingerprint can cause a check.
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	w := &providerWatch{
		fingerprint: func() string { return fp },
		detect:      func(context.Context) []protocol.Provider { detections++; return providers },
		last:        fp,
		providers:   providers,
		detected:    now,
		refresh:     30 * time.Minute,
		now:         func() time.Time { return now },
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

// B-52: a model Anthropic ships reaches the app although nothing on this
// computer changed. Which models claude offers is decided for the account, so
// the fingerprint of the installed CLI cannot notice it; time has to.
func TestANewModelIsReportedWithoutTheCLIChanging(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 22, 9, 0, 0, 0, time.UTC)
	opus := func(models ...string) []protocol.Provider {
		p := protocol.Provider{ID: "claude", Availability: protocol.Available}
		for _, m := range models {
			p.Models = append(p.Models, protocol.Model{ID: m, Label: m})
		}
		return []protocol.Provider{p}
	}
	offered := opus("claude-opus-5", "claude-sonnet-5")
	detections := 0
	w := &providerWatch{
		fingerprint: func() string { return "claude:1;" },
		detect:      func(context.Context) []protocol.Provider { detections++; return offered },
		last:        "claude:1;",
		providers:   offered,
		detected:    now,
		refresh:     30 * time.Minute,
		now:         func() time.Time { return now },
	}

	// Launch day: Anthropic starts offering Opus 5.5 to this account.
	offered = opus("claude-opus-5-5", "claude-opus-5", "claude-sonnet-5")

	now = now.Add(29 * time.Minute)
	if _, changed := w.check(ctx); changed || detections != 0 {
		t.Fatalf("inside the interval: reported %v after %d detections, want the CLI left alone", changed, detections)
	}

	now = now.Add(time.Minute)
	next, changed := w.check(ctx)
	if !changed || detections != 1 || next[0].Models[0].ID != "claude-opus-5-5" {
		t.Fatalf("after the interval: reported %v after %d detections, want Opus 5.5 reported once", changed, detections)
	}

	// The next interval asks again and, with nothing new, says nothing.
	now = now.Add(30 * time.Minute)
	if _, changed := w.check(ctx); changed || detections != 2 {
		t.Fatalf("nothing new: reported %v after %d detections, want one quiet check", changed, detections)
	}
}
