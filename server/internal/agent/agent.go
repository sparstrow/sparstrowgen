// Package agent drives the coding-agent CLIs installed on the machine.
//
// Shape borrowed from Multica's server/pkg/agent: one Backend interface, a
// Session carrying a message channel plus a single result, and per-provider
// files that translate each CLI's stream into the same events. Ours is far
// smaller — three providers, text only — because the design shows text, code,
// usage and failure and nothing else (AGENTS.md rule 4).
package agent

import (
	"context"
	"os"
	"os/exec"
	"strings"

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
)

type Message struct {
	Type      MessageType
	Text      string
	SessionID string
	Headroom  *protocol.Headroom
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
	return cmd
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

// scrubbedEnv drops every variable that lets one agent CLI notice it is running
// inside another. PATH, HOME and the provider's own home directories stay:
// CODEX_HOME is how codex finds its credentials even with --ignore-user-config.
func scrubbedEnv() []string {
	const dropPrefix = "CLAUDE"
	out := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		key, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		upper := strings.ToUpper(key)
		switch {
		case keepAnyway[upper]:
		case strings.HasPrefix(upper, dropPrefix):
			continue
		case upper == "ANTHROPIC_BASE_URL", upper == "ANTHROPIC_API_KEY":
			// A base URL inherited from a host session would silently point the
			// spawned CLI somewhere we did not choose.
			continue
		}
		out = append(out, kv)
	}
	return out
}
