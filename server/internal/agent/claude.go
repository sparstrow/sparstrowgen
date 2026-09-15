package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// Claude drives the `claude` CLI.
//
// Scoping is mandatory and is NOT `--bare`: that flag never reads OAuth or the
// keychain, so it cannot authenticate a subscription account at all. The
// working pair is --strict-mcp-config (0 MCP servers) and --setting-sources
// project (inherited skills 72 → 17). Verified 2026-09-10; see D-016.
type Claude struct{}

func (Claude) ID() string { return "claude" }

func claudeArgs(opts ExecOptions) []string {
	args := []string{
		"-p",
		// The prompt goes to stdin, never the command line: Windows refuses a
		// command line over 32,767 characters, and a catch-up after switching
		// agent is easily longer (docs/Bugs.md B-32).
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		// --verbose is REQUIRED with stream-json in print mode. Its absence is
		// not documented in --help and produces an unhelpful error.
		"--verbose",
		// Without this claude sends one whole message per turn. With it, ~89
		// deltas of ~8 chars for a four-sentence answer.
		"--include-partial-messages",
		"--strict-mcp-config",
		"--setting-sources", "project",
		// Its question tool has nowhere to show the question here, so a call
		// comes back unanswered and claude guesses silently. Multica found the
		// same (its GitHub #2588).
		"--disallowedTools", "AskUserQuestion",
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.ResumeSessionID != "" {
		args = append(args, "--resume", opts.ResumeSessionID)
	}
	return args
}

// claudeInput is the one stream-json user message a turn sends on stdin.
func claudeInput(prompt string) ([]byte, error) {
	data, err := json.Marshal(map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": []map[string]string{{"type": "text", "text": prompt}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("encode the prompt for claude: %w", err)
	}
	return append(data, '\n'), nil
}

func (c Claude) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	input, err := claudeInput(prompt)
	if err != nil {
		return nil, err
	}
	cmd := command(ctx, opts.Cwd, "claude", claudeArgs(opts)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	// One user message, then end of input: claude runs that turn and exits
	// rather than waiting for another.
	cmd.Stdin = bytes.NewReader(input)
	// launch rather than cmd.Start: a stop has to take the tool subprocesses
	// with it, not just the CLI (D-021).
	proc, err := launch(ctx, cmd, stdout)
	if err != nil {
		return nil, err
	}

	messages := make(chan Message, 64)
	result := make(chan Result, 1)

	go func() {
		defer close(result)
		p := parseClaude(stdout, messages)
		close(messages)

		waitErr := proc.Wait()
		if p.Err != nil {
			waitErr = p.Err
		} else if waitErr != nil && p.Text == "" {
			waitErr = fmt.Errorf("claude exited without answering: %w", waitErr)
		}
		result <- Result{
			Text:       p.Text,
			SessionID:  p.SessionID,
			Tokens:     p.Tokens,
			SpendTicks: p.SpendTicks,
			Err:        waitErr,
		}
	}()

	return &Session{Messages: messages, Result: result}, nil
}

// parseClaude reads one turn's stream-json output.
//
// Separated from Execute so it can be tested against captured output without
// spawning anything — a test that resolves `claude` from PATH would spend the
// owner's quota. See docs/KnownGaps.md G-12.
func parseClaude(r io.Reader, out chan<- Message) parsed {
	var (
		p parsed
		// One turn can contain several assistant messages — narration, then a
		// tool call, then the answer. `done` holds the ones already completed;
		// `streaming` accumulates the deltas of the one still arriving, and is
		// superseded by that message's own `assistant` event when it lands.
		//
		// Collapsing these two into one buffer is what caused B-6: the assistant
		// event reset the buffer, so every message before the last was thrown
		// away. Verified against a real capture — a 60-character sentence
		// vanished ahead of a tool call.
		done      []string
		streaming strings.Builder
		retries   int
	)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		var ev claudeEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue // stdout is shared; a non-JSON line is not corruption
		}
		switch ev.Type {
		case "system":
			switch ev.Subtype {
			case "init":
				if ev.SessionID != "" && p.SessionID == "" {
					p.SessionID = ev.SessionID
					send(out, Message{Type: MessageStarted, SessionID: p.SessionID})
				}
			case "api_retry":
				// Repeated 401s while the CLI refreshes OAuth on its own are
				// normal when it is not launched by the desktop app. Only worth
				// reporting if the turn then fails.
				retries++
			}

		case "rate_limit_event":
			// The only usage-window signal any of the three emits, and it only
			// ever arrives mid-turn.
			if ev.RateLimitInfo.RateLimitType != "" {
				send(out, Message{Type: MessageLimit, Headroom: &protocol.Headroom{
					Status:   ev.RateLimitInfo.Status,
					ResetsAt: ev.RateLimitInfo.ResetsAt,
					Window:   ev.RateLimitInfo.RateLimitType,
				}})
			}

		case "stream_event":
			// Verified: ~89 deltas of ~8 characters for a four-sentence answer.
			// Finer-grained than agy's 25-35.
			// Verified: only text_delta carries `.text`. thinking_delta and
			// input_json_delta do not, so reasoning and tool arguments cannot
			// leak into the answer.
			if ev.Event.Type == "content_block_delta" && ev.Event.Delta.Text != "" {
				streaming.WriteString(ev.Event.Delta.Text)
				send(out, Message{Type: MessageDelta, Text: ev.Event.Delta.Text})
			}

		case "assistant":
			// One whole message, and authoritative for it: when deltas also
			// arrived they were pieces of this same message, so it supersedes
			// them rather than adding to them. Earlier messages are kept.
			//
			// Messages carrying only thinking or tool_use have no text and are
			// skipped, which leaves any deltas already in flight alone.
			if text := ev.Message.text(); text != "" {
				done = append(done, strings.Trim(text, "\n"))
				streaming.Reset()
			}

		case "result":
			p.Tokens = ev.Usage.total()
			if ev.TotalCostUSD > 0 {
				p.SpendTicks = protocol.USDToTicks(ev.TotalCostUSD)
			}
			// Read is_error, NOT subtype. A failed turn reports
			// subtype:"success" with is_error:true, which is how this was first
			// misread as working — see docs/Capabilities.md.
			if ev.IsError {
				p.Err = claudeError(ev.Result, retries)
			}
		}
	}

	// A turn that died mid-message leaves deltas with no assistant event behind
	// them. Keeping them is the difference between showing a half-written answer
	// and showing nothing at all.
	if tail := strings.Trim(streaming.String(), "\n"); tail != "" {
		done = append(done, tail)
	}
	// A blank line between messages, for the same reason as codex (B-5): they
	// are separate messages, and glued together a fence stops beginning its line
	// and stops being a fence.
	p.Text = strings.Join(done, "\n\n")
	return p
}

// claudeError turns the CLI's own words into something worth showing.
//
// An expired OAuth token is the one failure with a known, specific remedy, and
// the raw 401 does not say what to do about it. Everything else is passed
// through unchanged — the CLI's wording is usually better than ours.
func claudeError(result string, retries int) error {
	if strings.Contains(result, "OAuth") || strings.Contains(result, "authenticate") {
		return fmt.Errorf(
			"claude is not authenticated (%d retries). Its stored token has expired and only the "+
				"desktop app refreshes it. Run `claude setup-token` and save the token with "+
				"`setx CLAUDE_CODE_OAUTH_TOKEN`; the next turn uses it, no restart needed — "+
				"see docs/runbooks/claude-headless-auth.md", retries)
	}
	if result != "" {
		return fmt.Errorf("claude: %s", result)
	}
	return fmt.Errorf("claude reported an error after %d retries", retries)
}

type claudeEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	SessionID string `json:"session_id"`
	IsError   bool   `json:"is_error"`
	// The human-readable outcome. On failure this is where the reason lives.
	Result string `json:"result"`
	Event  struct {
		Type  string `json:"type"`
		Delta struct {
			Text string `json:"text"`
		} `json:"delta"`
	} `json:"event"`
	Message      claudeMessage `json:"message"`
	Usage        claudeUsage   `json:"usage"`
	TotalCostUSD float64       `json:"total_cost_usd"`
	// Note what is NOT here: any percentage. The payload carries a status, a
	// reset timestamp and a window name, and nothing about how much is left.
	RateLimitInfo struct {
		Status        string `json:"status"`
		ResetsAt      int64  `json:"resetsAt"`
		RateLimitType string `json:"rateLimitType"`
	} `json:"rate_limit_info"`
}

type claudeMessage struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// text is every text block in one message, joined with a blank line.
//
// A message holds several text blocks only when something sits between them —
// thinking, or a tool call — so they are separate paragraphs, and gluing them
// edge-to-edge is what shredded codex replies (B-5). Unlike that one this is
// unverified in either direction: every claude capture we hold has a single
// text block per message, so this path has never actually run.
func (m claudeMessage) text() string {
	parts := make([]string, 0, len(m.Content))
	for _, c := range m.Content {
		if c.Type == "text" && c.Text != "" {
			parts = append(parts, strings.Trim(c.Text, "\n"))
		}
	}
	return strings.Join(parts, "\n\n")
}

type claudeUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

func (u claudeUsage) total() int64 {
	return u.InputTokens + u.OutputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}
