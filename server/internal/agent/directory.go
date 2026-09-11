package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// Listing answers "what is in this directory, and can a conversation run here?"
//
// It lives on the daemon because the daemon is the only part of the system that
// is on the owner's machine. The server is meant to run somewhere else, so it
// cannot stat a path even today, when both happen to be on the same laptop.
//
// Files are deliberately never returned. A conversation runs in a directory, so
// listing files would only offer a choice that cannot be made.
func Listing(path string) protocol.DirListing {
	if path == "" {
		return roots()
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return protocol.DirListing{Path: path, Reason: protocol.DirNotAbsolute}
	}
	// Resolve "..", a trailing separator, and the case of what was typed, so
	// what gets stored is what the agent will actually run in rather than
	// whatever spelling reached the box.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	abs = filepath.Clean(abs)

	out := protocol.DirListing{Path: abs, Parent: parentOf(abs), Entries: []protocol.DirEntry{}}

	info, err := os.Stat(abs)
	switch {
	case os.IsNotExist(err):
		out.Reason = protocol.DirNotFound
		return out
	case err != nil:
		out.Reason = protocol.DirNotReadable
		return out
	case !info.IsDir():
		// Pointing at a file is a common near-miss — the path of a source file
		// gets pasted instead of the project. Naming the parent lets the picker
		// offer it rather than just refusing.
		out.Reason = protocol.DirNotDirectory
		return out
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		out.Reason = protocol.DirNotReadable
		return out
	}

	for _, e := range entries {
		if !e.IsDir() || hidden(e.Name()) {
			continue
		}
		full := filepath.Join(abs, e.Name())
		// A directory we cannot open is worse than one we do not show: it would
		// look selectable and then fail on the first turn.
		if f, err := os.Open(full); err == nil {
			_ = f.Close()
			out.Entries = append(out.Entries, protocol.DirEntry{Name: e.Name(), Path: full})
		}
	}
	sort.Slice(out.Entries, func(i, j int) bool {
		return strings.ToLower(out.Entries[i].Name) < strings.ToLower(out.Entries[j].Name)
	})

	out.IsGitRepo = insideGitWorkTree(abs)
	return out
}

// roots is what to show when nothing has been chosen yet: the drive letters on
// Windows, the home directory elsewhere. A browser has no way to know what the
// machine's filesystem even looks like, so it has to be told where to start.
func roots() protocol.DirListing {
	out := protocol.DirListing{Entries: []protocol.DirEntry{}}
	if home, err := os.UserHomeDir(); err == nil {
		out.Entries = append(out.Entries, protocol.DirEntry{Name: "Home", Path: home})
	}
	if runtime.GOOS == "windows" {
		for c := 'A'; c <= 'Z'; c++ {
			drive := string(c) + `:\`
			if f, err := os.Open(drive); err == nil {
				_ = f.Close()
				out.Entries = append(out.Entries, protocol.DirEntry{Name: drive, Path: drive})
			}
		}
	}
	return out
}

// parentOf is empty at a root, so the picker can hide "up" without doing path
// arithmetic in the browser — where the separator may not even match the
// machine the daemon is running on.
func parentOf(path string) string {
	parent := filepath.Dir(path)
	if parent == path {
		return ""
	}
	return parent
}

// hidden skips dot-directories and the two Windows attic folders. They are
// never the answer, and on a home directory they are most of the list.
func hidden(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch strings.ToLower(name) {
	case "$recycle.bin", "system volume information":
		return true
	}
	return false
}

// insideGitWorkTree walks up looking for .git, the way git itself resolves a
// working tree, so a subdirectory of a repo reports true.
//
// .git is accepted as a file as well as a directory: in a linked worktree it is
// a file holding a gitdir pointer. Advisory only — it is shown so that picking
// a folder outside any repo is noticeable, never to prevent it.
func insideGitWorkTree(path string) bool {
	for {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}
