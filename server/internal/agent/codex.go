package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Codex drives the `codex` CLI.
//
// It emits NO incremental text — verified: no delta event type exists in the
// --json stream, and one whole item.completed arrives per turn. Anything that
// treats silence as a stall will be wrong on this provider one time in three.
//
// --ignore-user-config is mandatory: without it the run loads the owner's
// global CODEX_HOME config and boots unrelated MCP servers. Auth still works
// with it, because the flag skips config.toml but keeps using CODEX_HOME for
// credentials.
type Codex struct{}

func (Codex) ID() string { return "codex" }

func codexArgs(prompt string, opts ExecOptions) []string {
	args := []string{"exec", "--json", "--ignore-user-config", "--skip-git-repo-check"}
	if opts.Model != "" {
		args = append(args, "-m", opts.Model)
	}
	// `resume` takes the thread id and the prompt positionally; a fresh run
	// takes the prompt alone.
	if opts.ResumeSessionID != "" {
		return append(args, "resume", opts.ResumeSessionID, prompt)
	}
	return append(args, prompt)
}

func (c Codex) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	cmd := command(ctx, opts.Cwd, "codex", codexArgs(prompt, opts)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	// codex prints "Reading additional input from stdin..." and waits unless
	// stdin is already closed.
	cmd.Stdin = strings.NewReader("")
	// launch rather than cmd.Start: a stop has to take the tool subprocesses
	// with it, not just the CLI (D-021).
	proc, err := launch(ctx, cmd, stdout)
	if err != nil {
		return nil, err
	}

	messages := make(chan Message, 8)
	result := make(chan Result, 1)

	go func() {
		defer close(result)
		p := parseCodex(stdout, messages)
		close(messages)

		waitErr := proc.Wait()
		if p.Err != nil {
			waitErr = p.Err
		}
		result <- Result{
			Text:      p.Text,
			SessionID: p.SessionID,
			Tokens:    p.Tokens,
			// codex reports tokens only, never currency. Leaving SpendTicks at
			// zero is what makes "no cost data" distinguishable from "free".
			Err: waitErr,
		}
	}()

	return &Session{Messages: messages, Result: result}, nil
}

func parseCodex(r io.Reader, out chan<- Message) parsed {
	var (
		p    parsed
		full strings.Builder
	)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		var ev codexEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "thread.started":
			// This is the resume handle: `codex exec resume <thread_id>`.
			if ev.ThreadID != "" {
				p.SessionID = ev.ThreadID
				send(out, Message{Type: MessageStarted, SessionID: p.SessionID})
			}
		case "item.completed":
			if ev.Item.Type == "agent_message" && ev.Item.Text != "" {
				// One turn can produce several agent_message items, and they are
				// separate messages rather than pieces of one. Concatenating them
				// edge-to-edge welds the last line of one to the first line of the
				// next — and a fence that does not begin a line is not a fence, so
				// a reply whose preamble and code arrive as two items rendered as
				// shredded inline code. Verified against a real capture:
				// "…deletion, and traversal." + "```cpp\n#include…"
				// (testdata/codex-two-messages.jsonl, docs/Bugs.md B-5).
				if full.Len() > 0 {
					full.WriteString("\n\n")
				}
				full.WriteString(strings.Trim(ev.Item.Text, "\n"))
			}
		case "turn.completed":
			p.Tokens = ev.Usage.total()
		case "error":
			p.Err = fmt.Errorf("codex: %s", ev.Message)
		}
	}

	p.Text = full.String()
	return p
}

type codexEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Message  string `json:"message"`
	Item     struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"item"`
	Usage codexUsage `json:"usage"`
}

type codexUsage struct {
	InputTokens           int64 `json:"input_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
	CacheWriteInputTokens int64 `json:"cache_write_input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
}

func (u codexUsage) total() int64 {
	return u.InputTokens + u.CachedInputTokens + u.CacheWriteInputTokens +
		u.OutputTokens + u.ReasoningOutputTokens
}
