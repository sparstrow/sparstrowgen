package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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

func (c Claude) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	args := []string{
		"-p", prompt,
		"--output-format", "stream-json",
		// --verbose is REQUIRED with stream-json in print mode. Its absence is
		// not documented in --help and produces an unhelpful error.
		"--verbose",
		"--include-partial-messages",
		"--strict-mcp-config",
		"--setting-sources", "project",
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.ResumeSessionID != "" {
		args = append(args, "--resume", opts.ResumeSessionID)
	}

	cmd := command(ctx, opts.Cwd, "claude", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	// Print mode reads stdin even with the prompt as an argument; handing it a
	// closed stdin is what stops it waiting forever.
	cmd.Stdin = strings.NewReader("")
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	messages := make(chan Message, 64)
	result := make(chan Result, 1)

	go func() {
		defer close(result)

		var (
			full       strings.Builder
			sessionID  string
			tokens     int64
			spendTicks int64
			// Repeated 401s while the CLI refreshes OAuth on its own are normal
			// when it is not launched by the desktop app. Only worth reporting
			// if the turn then fails.
			retries int
		)

		scanner := bufio.NewScanner(stdout)
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
					if ev.SessionID != "" && sessionID == "" {
						sessionID = ev.SessionID
						messages <- Message{Type: MessageStarted, SessionID: sessionID}
					}
				case "api_retry":
					retries++
				}
			case "stream_event":
				// Verified: ~89 deltas of ~8 characters for a four-sentence
				// answer. Finer-grained than agy's 25-35, so this is the
				// chattiest of the three by some margin.
				if ev.Event.Type == "content_block_delta" && ev.Event.Delta.Text != "" {
					full.WriteString(ev.Event.Delta.Text)
					messages <- Message{Type: MessageDelta, Text: ev.Event.Delta.Text}
				}
			case "assistant":
				// The whole answer, sent once. When deltas did arrive this
				// repeats them, so it replaces rather than appends.
				if text := ev.Message.text(); text != "" {
					full.Reset()
					full.WriteString(text)
				}
			case "result":
				tokens = ev.Usage.total()
				if ev.TotalCostUSD > 0 {
					spendTicks = protocol.USDToTicks(ev.TotalCostUSD)
				}
				// Read is_error, NOT subtype. A failed turn reports
				// subtype:"success" with is_error:true, which is how this was
				// first misread as working — see docs/KnownGaps.md G-9.
				if ev.IsError {
					close(messages)
					result <- Result{
						Text:      full.String(),
						SessionID: sessionID,
						Err:       claudeError(ev.Result, retries),
					}
					_ = cmd.Wait()
					return
				}
			}
		}
		close(messages)

		waitErr := cmd.Wait()
		if waitErr != nil && full.Len() == 0 {
			waitErr = fmt.Errorf("claude exited without answering (%d auth retries): %w", retries, waitErr)
		}
		result <- Result{
			Text:       full.String(),
			SessionID:  sessionID,
			Tokens:     tokens,
			SpendTicks: spendTicks,
			Err:        waitErr,
		}
	}()

	return &Session{Messages: messages, Result: result}, nil
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
				"desktop app refreshes it. Run `claude setup-token`, set CLAUDE_CODE_OAUTH_TOKEN, "+
				"and restart the daemon from a NEW terminal so it inherits the variable — "+
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
	Event     struct {
		Type  string `json:"type"`
		Delta struct {
			Text string `json:"text"`
		} `json:"delta"`
	} `json:"event"`
	Message      claudeMessage `json:"message"`
	Usage        claudeUsage   `json:"usage"`
	TotalCostUSD float64       `json:"total_cost_usd"`
}

type claudeMessage struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (m claudeMessage) text() string {
	var b strings.Builder
	for _, c := range m.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
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
