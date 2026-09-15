package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Fingerprint identifies which agent CLIs are installed, and which build of each,
// by the file each name resolves to: its path, size and modification time.
//
// It costs three PATH lookups and three stats, so a connected daemon can compare
// it every few minutes and run the much slower Detect only when it changes
// (Multica re-checks the same way, MUL-5439). A CLI updated behind a launcher
// file that itself stays the same is noticed at the next connection instead.
func Fingerprint() string {
	var b strings.Builder
	for _, name := range []string{"claude", "codex", "agy"} {
		path, err := exec.LookPath(name)
		if err != nil {
			fmt.Fprintf(&b, "%s:missing;", name)
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(&b, "%s:%s;", name, path)
			continue
		}
		fmt.Fprintf(&b, "%s:%s:%d:%d;", name, path, info.Size(), info.ModTime().UnixNano())
	}
	return b.String()
}
