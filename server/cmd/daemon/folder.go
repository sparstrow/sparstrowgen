package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// folderProblem says why a turn cannot start in its conversation's folder, or ""
// when it can. Checked before the agent is launched, because Windows' own refusal
// names the executable instead of the folder (docs/Bugs.md B-30). An empty cwd
// is left alone: the agent then runs where the daemon does, as it always has.
func folderProblem(cwd string) string {
	if cwd == "" {
		return ""
	}
	info, err := os.Stat(cwd)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Sprintf("this conversation's folder, %s, no longer exists on this computer. Put the folder back, or start a new conversation in another folder.", cwd)
	case err != nil:
		return fmt.Sprintf("this conversation's folder, %s, cannot be opened on this computer: %v", cwd, err)
	case !info.IsDir():
		return fmt.Sprintf("this conversation's folder, %s, is a file on this computer now, not a folder. Start a new conversation in another folder.", cwd)
	}
	return ""
}
