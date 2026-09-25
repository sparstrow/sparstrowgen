// Package agent drives the coding-agent CLIs installed on the machine.
//
// Shape borrowed from Multica's server/pkg/agent: one Backend interface, a
// Session carrying a message channel plus a single result, and per-provider
// files that translate each CLI's stream into the same events. Ours is far
// smaller: three providers, and the parsers keep only the text, usage and
// failure the chat shows. Everything else a CLI prints is kept verbatim in the
// turn's record instead (record.go), rather than interpreted here.
package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

type MessageType string

const (
	// MessageDelta is incremental text. agy emits these; codex is verified
	// never to; claude has not produced one in any capture yet. Nothing may
	// depend on receiving them.
	MessageDelta MessageType = "delta"
	// MessageStarted carries the provider's own session id as soon as it is
	// known, so a resume pointer survives a turn that later fails.
	MessageStarted MessageType = "started"
	// MessageLimit reports a usage window. Only claude emits one, and only
	// during a turn — there is no way to ask a provider for it.
	MessageLimit MessageType = "limit"
	// MessageLine is one line the CLI printed, on stdout or stderr, for the
	// turn's record (record.go). Every line, including the ones the parser
	// also turned into one of the messages above.
	MessageLine MessageType = "line"
)

type Message struct {
	Type      MessageType
	Text      string
	SessionID string
	Headroom  *protocol.Headroom
	Line      *Line
}

type Result struct {
	Text      string
	SessionID string
	Tokens    int64
	// SpendTicks is 1e-10 USD. Zero means the provider did not state a cost,
	// which is not the same as free — only claude reports one.
	SpendTicks int64
	Err        error
}

// Session streams one turn. Messages is closed before Result is sent, so a
// consumer can drain it and then take the result without racing.
type Session struct {
	Messages <-chan Message
	Result   <-chan Result
	// Sent is exactly what the CLI was handed, for the turn's record.
	Sent protocol.ExchangeSent
}

// parsed is what one turn's stream yielded. Every provider's parser returns
// this, which is what lets them be tested against captured output instead of a
// live CLI.
type parsed struct {
	Text       string
	SessionID  string
	Tokens     int64
	SpendTicks int64
	// Err is a failure the PROVIDER reported, as distinct from the process
	// exiting badly. A turn can fail cleanly with exit code 0.
	Err error
}

// send blocks until the consumer takes the message.
//
// Dropping instead would be worse than waiting: a discarded delta is text the
// reader never sees arrive, and back-pressure through to the CLI's stdout pipe
// is the correct response to a slow consumer. Execute's caller always drains
// Messages until it is closed, so this cannot deadlock in production. A nil
// channel is allowed so a parser can run with nobody listening.
func send(out chan<- Message, msg Message) {
	if out == nil {
		return
	}
	out <- msg
}

type ExecOptions struct {
	Cwd   string
	Model string
	// Empty starts a fresh provider session; set resumes one.
	ResumeSessionID string
	// Folders outside Cwd the agent may read: the conversation's own files
	// folder (D-056). claude and agy are given them with --add-dir; codex's
	// sandbox cannot be widened this way, so it is given pictures instead.
	AddDirs []string
	// Pictures sent with this message, as full paths. Only codex uses them,
	// with --image: it cannot read files under its sandbox, but it sees these.
	Images []string
}

type Backend interface {
	// ID is the provider id used everywhere else: "claude", "codex", "agy".
	ID() string
	Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error)
}

// command builds an exec.Cmd with a scrubbed environment.
//
// This is not optional. Anything spawned from inside a Claude Code session
// inherits CLAUDECODE=1 and CLAUDE_CODE_MESSAGING_SOCKET, and a nested
// `claude -p` then hangs forever — verified 2026-09-10, two runs killed at
// 120s. The daemon is exactly the kind of process someone starts from a
// terminal inside an agent session. See docs/Decisions.md D-016.
func command(ctx context.Context, cwd, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	cmd.Env = scrubbedEnv()
	hideConsole(cmd)
	return cmd
}

const (
	// terminateGrace is how long a stopped tree gets to disappear — on Unix
	// between SIGTERM and SIGKILL, and everywhere as the window for confirming
	// it actually went.
	terminateGrace = 3 * time.Second
	// waitDelay is the backstop for a Wait that a descendant is holding open by
	// still owning an inherited pipe.
	waitDelay = 10 * time.Second
)

// process is a launched CLI together with ownership of everything it spawns.
type process struct {
	cmd  *exec.Cmd
	tree *processTree
	// done is closed once the process has been reaped, so the terminator can
	// tell "finished on its own" from "cancelled".
	done chan struct{}
	// terminated is closed when the terminator goroutine has stopped touching
	// the tree, which is what makes releasing it safe.
	terminated chan struct{}
	// treeGone records whether the tree was CONFIRMED empty after a stop.
	// False also means "could not be confirmed" — an unowned tree can never be
	// observed — so it is never read as proof of a leak, only as absence of
	// proof of a clean stop. Safe to read once Wait has returned.
	treeGone bool
}

// launch starts cmd and takes ownership of every process it goes on to create,
// so cancelling ctx stops the agent AND whatever the agent was itself running —
// a build, an npm install, an MCP server. Killing the leader alone leaves those
// behind, which is a stop that visibly stopped and did not (D-021).
//
// stdout is closed only AFTER the tree has been terminated. A wedged descendant
// that inherited the pipe can otherwise keep the parser's scanner blocked
// forever, and closing it earlier would race the parser against processes still
// able to write into it.
func launch(ctx context.Context, cmd *exec.Cmd, stdout io.Closer) (*process, error) {
	prepare(cmd)
	// Take cancellation away from os/exec, which would otherwise kill the
	// leader the instant ctx is done and race the tree-wide stop below.
	// WaitDelay stays as the hard backstop.
	cmd.Cancel = func() error { return nil }
	cmd.WaitDelay = waitDelay

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	tree, err := own(cmd)
	if err != nil {
		// Degraded, not failed: an unowned agent still answers, and stopping it
		// still kills the CLI itself. Said out loud because this warning is the
		// only signal that stopping on this machine has quietly stopped being
		// thorough.
		slog.Warn("could not take ownership of the agent process tree; stopping a turn will kill "+
			"only the CLI and may leave what it spawned running",
			"err", err, "pid", cmd.Process.Pid, "exe", cmd.Path)
	}
	// Always resume, owned or not — on Windows the child is created suspended,
	// so skipping this would hang forever on a process that never ran.
	if err := resume(cmd.Process.Pid); err != nil {
		tree.release()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("resume suspended child: %w", err)
	}

	p := &process{
		cmd: cmd, tree: tree,
		done: make(chan struct{}), terminated: make(chan struct{}),
	}
	go func() {
		defer close(p.terminated)
		select {
		case <-p.done:
			return // finished on its own; there is nothing to stop
		case <-ctx.Done():
		}
		p.tree.terminate(cmd)
		// Wait for the tree to actually be gone before unblocking the reader:
		// until then a survivor could still write into the pipe.
		p.treeGone = p.tree.gone(terminateGrace)
		if !p.treeGone {
			slog.Warn("a stopped turn could not be confirmed clean; something the agent "+
				"spawned may still be running", "pid", cmd.Process.Pid, "exe", cmd.Path)
		}
		if stdout != nil {
			_ = stdout.Close()
		}
	}()
	return p, nil
}

// Wait reaps the process and gives up ownership of its tree.
func (p *process) Wait() error {
	err := p.cmd.Wait()
	close(p.done)
	// The terminator may still be inside terminate/gone, both of which read the
	// tree handle that release is about to invalidate.
	<-p.terminated
	p.tree.release()
	return err
}

// keepAnyway are the CLAUDE_* variables that must survive scrubbing.
//
// CLAUDE_CODE_OAUTH_TOKEN is the long-lived token from `claude setup-token`,
// and it is the only way a spawned claude authenticates: the credentials in
// ~/.claude/.credentials.json hold an access token that the desktop app
// refreshes, so a standalone process eventually meets "OAuth access token has
// expired" and cannot recover. Scrubbing this would delete the fix.
var keepAnyway = map[string]bool{
	"CLAUDE_CODE_OAUTH_TOKEN": true,
}

// userEnvironment is the user's environment as it is now, not as this process
// inherited it. A variable so tests can supply one; nil outside Windows.
var userEnvironment = readUserEnvironment

// scrubbedEnv drops every variable that lets one agent CLI notice it is running
// inside another. PATH, HOME and the provider's own home directories stay:
// CODEX_HOME is how codex finds its credentials even with --ignore-user-config.
//
// It then fills in from the user's current environment (docs/Bugs.md B-28). The
// daemon inherits the environment of whatever started it — Windows sign-in, the
// copy it updated from, the browser that opened a pairing link, a terminal — so
// a token set with setx after that never arrived, and every claude turn spent
// three minutes retrying 401s before failing. A variable the process lacks is
// taken from the user environment, and CLAUDE_CODE_OAUTH_TOKEN is taken from it
// even when the process has one, because that is where a replacement is written.
func scrubbedEnv() []string {
	out := make([]string, 0, len(os.Environ()))
	at := map[string]int{}
	add := func(key, kv string) {
		upper := strings.ToUpper(key)
		if upper == "" {
			out = append(out, kv) // Windows' per-drive "=C:" entries, never deduplicated
			return
		}
		if !passes(upper) {
			return
		}
		if i, ok := at[upper]; ok {
			out[i] = kv
			return
		}
		at[upper] = len(out)
		out = append(out, kv)
	}
	for _, kv := range os.Environ() {
		key, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		add(key, kv)
	}
	for key, value := range userEnvironment() {
		upper := strings.ToUpper(key)
		if value == "" || upper == "PATH" {
			continue // the process PATH already joins the machine's and the user's
		}
		if _, has := at[upper]; has && upper != "CLAUDE_CODE_OAUTH_TOKEN" {
			continue
		}
		add(key, key+"="+value)
	}
	return out
}

// passes reports whether a variable may reach a spawned agent CLI.
func passes(upper string) bool {
	switch {
	case keepAnyway[upper]:
		return true
	case strings.HasPrefix(upper, "CLAUDE"):
		return false
	case upper == "ANTHROPIC_BASE_URL", upper == "ANTHROPIC_API_KEY":
		// A base URL inherited from a host session would silently point the
		// spawned CLI somewhere we did not choose.
		return false
	}
	return true
}
