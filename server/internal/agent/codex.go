package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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

func (c Codex) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	// `resume` takes the thread id and the prompt positionally; a fresh run
	// takes the prompt alone. The flags are otherwise identical.
	args := []string{"exec", "--json", "--ignore-user-config", "--skip-git-repo-check"}
	if opts.Model != "" {
		args = append(args, "-m", opts.Model)
	}
	if opts.ResumeSessionID != "" {
		args = append(args, "resume", opts.ResumeSessionID, prompt)
	} else {
		args = append(args, prompt)
	}

	cmd := command(ctx, opts.Cwd, "codex", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	// codex prints "Reading additional input from stdin..." and waits unless
	// stdin is already closed.
	cmd.Stdin = strings.NewReader("")
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	messages := make(chan Message, 8)
	result := make(chan Result, 1)

	go func() {
		defer close(result)

		var (
			full      strings.Builder
			sessionID string
			tokens    int64
			apiErr    string
		)

		scanner := bufio.NewScanner(stdout)
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
					sessionID = ev.ThreadID
					messages <- Message{Type: MessageStarted, SessionID: sessionID}
				}
			case "item.completed":
				if ev.Item.Type == "agent_message" && ev.Item.Text != "" {
					full.WriteString(ev.Item.Text)
				}
			case "turn.completed":
				tokens = ev.Usage.total()
			case "error":
				apiErr = ev.Message
			}
		}
		close(messages)

		waitErr := cmd.Wait()
		if apiErr != "" {
			waitErr = fmt.Errorf("codex: %s", apiErr)
		}
		result <- Result{
			Text:      full.String(),
			SessionID: sessionID,
			Tokens:    tokens,
			// codex reports tokens only, never currency. Leaving SpendTicks at
			// zero is what makes "no cost data" distinguishable from "free".
			Err: waitErr,
		}
	}()

	return &Session{Messages: messages, Result: result}, nil
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
