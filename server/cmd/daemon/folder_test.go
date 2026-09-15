package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/agent"
)

// countingAgent records whether a turn got as far as launching an agent.
type countingAgent struct{ calls int }

func (*countingAgent) ID() string { return "fake" }

func (a *countingAgent) Execute(context.Context, string, agent.ExecOptions) (*agent.Session, error) {
	a.calls++
	return nil, errors.New("launched")
}

func TestAMissingOrReplacedFolderIsNamedInsteadOfTheAgent(t *testing.T) {
	root := t.TempDir()
	gone := filepath.Join(root, "work")
	file := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name, cwd, want string
	}{
		{"deleted folder", gone, "no longer exists on this computer"},
		{"a file where the folder was", file, "is a file on this computer now"},
		{"folder that exists", root, ""},
		{"no folder given", "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := folderProblem(c.cwd)
			if c.want == "" {
				if got != "" {
					t.Fatalf("folderProblem(%q) = %q, want none", c.cwd, got)
				}
				return
			}
			if !strings.Contains(got, c.want) || !strings.Contains(got, c.cwd) {
				t.Fatalf("folderProblem(%q) = %q, want it to name the folder and say %q", c.cwd, got, c.want)
			}
		})
	}
}

// The turn must end before an agent is launched: Windows' refusal of a missing
// working directory is what produced the misleading error.
func TestATurnInADeletedFolderNeverLaunchesTheAgent(t *testing.T) {
	backend := &countingAgent{}
	d := testDaemon(t, backend)
	turn := aTurn()
	turn.Cwd = filepath.Join(t.TempDir(), "deleted")

	d.runTurn(context.Background(), turn)

	if backend.calls != 0 {
		t.Fatalf("the agent was launched %d time(s) for a folder that does not exist", backend.calls)
	}
	if d.turns.stop(turn.TurnID) {
		t.Fatal("the refused turn was left registered as running")
	}
}
