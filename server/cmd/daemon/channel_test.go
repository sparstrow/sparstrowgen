package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A computer nobody has told otherwise follows the stream everybody is on.
func TestAComputerFollowsStableUnlessItWasTold(t *testing.T) {
	t.Setenv("SPARSTROWGEN_HOME", t.TempDir())
	if got := channel(); got != channelStable {
		t.Errorf("channel = %q, want %q", got, channelStable)
	}
}

func TestTheChannelSurvivesInTheComputersHome(t *testing.T) {
	t.Setenv("SPARSTROWGEN_HOME", t.TempDir())

	if err := setChannel(channelCandidate); err != nil {
		t.Fatal(err)
	}
	if got := channel(); got != channelCandidate {
		t.Errorf("channel = %q after being set, want %q", got, channelCandidate)
	}
	// Back to stable leaves nothing behind to go stale.
	if err := setChannel(channelStable); err != nil {
		t.Fatal(err)
	}
	if got := channel(); got != channelStable {
		t.Errorf("channel = %q after being set back, want %q", got, channelStable)
	}
	home := os.Getenv("SPARSTROWGEN_HOME")
	if _, err := os.Stat(filepath.Join(home, channelFile)); !os.IsNotExist(err) {
		t.Errorf("the channel file is still there after going back to stable (%v)", err)
	}
}

// Anything unrecognised is stable. An installed daemon will one day be older
// than whatever wrote this file (AGENTS.md §3), and the safe reading of a
// channel it does not know is the ordinary one, never an experimental stream.
func TestAnUnknownChannelIsStable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPARSTROWGEN_HOME", home)

	for _, written := range []string{"", "   ", "beta", "nightly\n", "CANDIDATE-2"} {
		if err := os.WriteFile(filepath.Join(home, channelFile), []byte(written), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := channel(); got != channelStable {
			t.Errorf("channel file %q read as %q, want %q", written, got, channelStable)
		}
	}
}

// Case and stray whitespace are how a person writes it by hand.
func TestTheChannelIsReadLeniently(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPARSTROWGEN_HOME", home)

	for _, written := range []string{"candidate", "Candidate\n", "  CANDIDATE  "} {
		if err := os.WriteFile(filepath.Join(home, channelFile), []byte(written), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := channel(); got != channelCandidate {
			t.Errorf("channel file %q read as %q, want %q", written, got, channelCandidate)
		}
	}
}

// The environment variable is how a test computer is put on the candidate
// channel without installing anything.
func TestTheEnvironmentOverridesTheFile(t *testing.T) {
	t.Setenv("SPARSTROWGEN_HOME", t.TempDir())
	if err := setChannel(channelCandidate); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPARSTROWGEN_CHANNEL", "stable")
	if got := channel(); got != channelStable {
		t.Errorf("channel = %q, want the environment to win", got)
	}
}

// The reason the channel is not compiled in: the promoted release is the same
// executable the owner tested, so it must not carry the candidate's update URL
// into production. Which manifest it reads is decided here, per computer.
func TestTheUpdateSourceFollowsTheChannel(t *testing.T) {
	t.Setenv("SPARSTROWGEN_HOME", t.TempDir())

	stable, candidate := releaseUpdateURL, releaseCandidateURL
	t.Cleanup(func() { releaseUpdateURL, releaseCandidateURL = stable, candidate })
	releaseUpdateURL = "https://example.test/latest/download/sparstrowgen-update.json"
	releaseCandidateURL = "https://example.test/download/daemon-candidate/sparstrowgen-update.json"

	if got := updateSource(); got != releaseUpdateURL {
		t.Errorf("stable computer reads %q, want %q", got, releaseUpdateURL)
	}
	t.Setenv("SPARSTROWGEN_CHANNEL", channelCandidate)
	if got := updateSource(); got != releaseCandidateURL {
		t.Errorf("candidate computer reads %q, want %q", got, releaseCandidateURL)
	}

	// A build with no candidate URL keeps updating from the stream it knows.
	releaseCandidateURL = ""
	if got := updateSource(); got != releaseUpdateURL {
		t.Errorf("a build with no candidate URL reads %q, want it to fall back to %q", got, releaseUpdateURL)
	}
}
