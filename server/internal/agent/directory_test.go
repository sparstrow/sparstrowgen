package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

func TestListingReportsWhyAPathIsUnusable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name, path, want string
	}{
		// Pasting the path of a source file instead of its project is the
		// common near-miss, and it must not read the same as a typo.
		{"a file is not a directory", file, protocol.DirNotDirectory},
		{"a typo is not found", filepath.Join(dir, "no-such-place"), protocol.DirNotFound},
		{"a real directory is fine", dir, protocol.DirOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Listing(c.path).Reason; got != c.want {
				t.Errorf("reason = %q, want %q", got, c.want)
			}
		})
	}
}

func TestListingReturnsOnlyReachableDirectories(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"server", "apps", ".git", ".next"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := Listing(dir)
	if got.Reason != protocol.DirOK {
		t.Fatalf("reason = %q", got.Reason)
	}

	var names []string
	for _, e := range got.Entries {
		names = append(names, e.Name)
	}
	// Files would be an unusable choice; dot-directories are noise and, in a
	// home directory, most of the list.
	if want := []string{"apps", "server"}; strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("entries = %v, want %v (sorted, no files, no dot-directories)", names, want)
	}
	// .git here makes this a work tree.
	if !got.IsGitRepo {
		t.Error("isGitRepo = false, want true — a .git directory is right here")
	}
	if got.Parent == "" {
		t.Error("parent is empty, so the picker cannot walk up")
	}
}

func TestListingWithNoPathOffersSomewhereToStart(t *testing.T) {
	// A browser cannot know what the machine's filesystem looks like, so an
	// empty path has to answer with the places a person could begin.
	got := Listing("")
	if len(got.Entries) == 0 {
		t.Fatal("no roots offered; the picker would open on nothing")
	}
	if got.Parent != "" {
		t.Errorf("parent = %q, want empty at the root level", got.Parent)
	}
	if runtime.GOOS == "windows" {
		var hasDrive bool
		for _, e := range got.Entries {
			if strings.HasSuffix(e.Path, `:\`) {
				hasDrive = true
			}
		}
		if !hasDrive {
			t.Error("no drive letters offered on Windows")
		}
	}
}

func TestListingResolvesThePathItReports(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	// What gets stored has to be what the agent will actually run in, not
	// whatever spelling was typed into the box.
	messy := filepath.Join(dir, "project", "..", "project") + string(filepath.Separator)
	got := Listing(messy)
	if got.Reason != protocol.DirOK {
		t.Fatalf("reason = %q", got.Reason)
	}
	if strings.Contains(got.Path, "..") {
		t.Errorf("path = %q, still holds a relative segment", got.Path)
	}
}
