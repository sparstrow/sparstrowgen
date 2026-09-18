package main

import (
	"os"
	"path/filepath"
	"strings"
)

/* Which stream of releases this computer follows.

Production computers follow `stable`: the release marked latest, which is what
every installed copy has always read. A computer testing a release candidate
follows `candidate`: a pointer that moves as each candidate is published and
which no production computer reads (WORKFLOW.md phase 2, and it is what
docs/Unverified.md U-8 and U-10 needed a channel for).

The channel is resolved at RUN time, from a file beside the machine credential,
and is deliberately NOT compiled into the executable. If it were, promoting a
candidate to stable could not reuse the bytes the owner tested — the promoted
copy would carry the candidate's own update URL and quietly put every production
computer on the candidate channel. Keeping it out of the binary is what lets the
release protocol's "build once, promote exactly" actually hold for the daemon. */

// Baked by the release build (scripts/package-windows.ps1), alongside
// releaseUpdateURL which is the stable one. Empty in a development build, which
// never updates itself.
var releaseCandidateURL string

const (
	channelStable    = "stable"
	channelCandidate = "candidate"
	// channelFile sits in the daemon's home, so it survives the executable
	// being replaced by an update and belongs to one computer rather than one
	// build.
	channelFile = "channel"
)

// channel is what this computer follows. Anything unrecognised — an empty file,
// a channel a newer version knows about, a typo — is stable: the safe answer is
// the stream everybody else is on, never an experimental one.
func channel() string {
	if v := os.Getenv("SPARSTROWGEN_CHANNEL"); v != "" {
		return normaliseChannel(v)
	}
	dir, err := dataDir()
	if err != nil {
		return channelStable
	}
	body, err := os.ReadFile(filepath.Join(dir, channelFile))
	if err != nil {
		return channelStable
	}
	return normaliseChannel(string(body))
}

func normaliseChannel(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), channelCandidate) {
		return channelCandidate
	}
	return channelStable
}

// setChannel records the channel for this computer. Writing stable removes the
// file rather than writing the word, so the ordinary case leaves nothing behind
// to go stale.
func setChannel(name string) error {
	dir, err := dataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, channelFile)
	if normaliseChannel(name) == channelStable {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(path, []byte(channelCandidate+"\n"), 0o644)
}

// updateSource is the manifest this computer checks for updates.
//
// A candidate computer whose build has no candidate URL falls back to stable
// rather than to nothing: an older daemon that finds itself on the candidate
// channel should keep updating from the stream it does understand.
func updateSource() string {
	if channel() == channelCandidate && releaseCandidateURL != "" {
		return releaseCandidateURL
	}
	return releaseUpdateURL
}
