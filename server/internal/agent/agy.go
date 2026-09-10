package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Agy drives the `agy` CLI (Antigravity).
//
// Two things make it different from the other two. Its stream keys events on
// "event" rather than "type". And it is a ROUTER, not a vendor: `agy models`
// returns Gemini, Claude and GPT-OSS models behind one CLI, with the reasoning
// effort baked into the model id (gemini-3.1-pro-high vs -low) rather than
// offered as a separate setting.
//
// It is the only one of the three verified to stream incrementally — ~25-35
// character text_delta chunks. Short answers still arrive in one piece; that is
// length, not absence of streaming.
type Agy struct{}

func (Agy) ID() string { return "agy" }

func (a Agy) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	args := []string{"-p", prompt, "--output-format", "stream-json"}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.ResumeSessionID != "" {
		args = append(args, "--conversation", opts.ResumeSessionID)
	}

	cmd := command(ctx, opts.Cwd, "agy", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdin = strings.NewReader("")
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	messages := make(chan Message, 64)
	result := make(chan Result, 1)

	go func() {
		defer close(result)

		var (
			streamed  strings.Builder
			final     string
			sessionID string
			tokens    int64
			status    string
		)

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
		for scanner.Scan() {
			var ev agyEvent
			if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
				continue
			}
			switch ev.Event {
			case "init":
				if ev.ConversationID != "" {
					sessionID = ev.ConversationID
					messages <- Message{Type: MessageStarted, SessionID: sessionID}
				}
			case "step_update":
				if ev.StepUpdate.TextDelta != "" {
					streamed.WriteString(ev.StepUpdate.TextDelta)
					messages <- Message{Type: MessageDelta, Text: ev.StepUpdate.TextDelta}
				}
			case "result":
				status = ev.Result.Status
				tokens = ev.Result.Usage.total()
				// result.response is authoritative: it is the whole answer,
				// whether or not deltas arrived.
				final = ev.Result.Response
				if ev.Result.ConversationID != "" {
					sessionID = ev.Result.ConversationID
				}
			}
		}
		close(messages)

		text := strings.TrimRight(final, "\n")
		if text == "" {
			text = streamed.String()
		}

		waitErr := cmd.Wait()
		if waitErr == nil && status != "" && status != "SUCCESS" {
			waitErr = fmt.Errorf("agy finished with status %s", status)
		}
		result <- Result{
			Text:      text,
			SessionID: sessionID,
			Tokens:    tokens,
			// Tokens only — agy states no currency figure.
			Err: waitErr,
		}
	}()

	return &Session{Messages: messages, Result: result}, nil
}

type agyEvent struct {
	Event          string `json:"event"`
	ConversationID string `json:"conversation_id"`
	StepUpdate     struct {
		State     string `json:"state"`
		TextDelta string `json:"text_delta"`
	} `json:"step_update"`
	Result struct {
		ConversationID string   `json:"conversation_id"`
		Status         string   `json:"status"`
		Response       string   `json:"response"`
		Usage          agyUsage `json:"usage"`
	} `json:"result"`
}

type agyUsage struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ThinkingTokens  int64 `json:"thinking_tokens"`
	CacheReadTokens int64 `json:"cache_read_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

func (u agyUsage) total() int64 {
	if u.TotalTokens > 0 {
		return u.TotalTokens
	}
	return u.InputTokens + u.OutputTokens + u.ThinkingTokens + u.CacheReadTokens
}
