package agent

import (
	"encoding/json"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* What a CLI said about itself in one turn, read from the lines it printed.

This is provider knowledge, so it lives beside the parsers that already hold the
rest of it. It runs over the STORED lines whenever a record is read, and over
live lines as they arrive, and never writes anything back: the lines are the
record, and this is only one reading of them. Only what a CLI actually printed
is reported. codex says almost nothing about itself, and its report is then
almost empty, which is the truth (docs/Capabilities.md, "The raw exchange"). */

// Reporter accumulates one turn's report, line by line.
type Reporter struct {
	provider string
	r        protocol.ExchangeReport
}

func NewReporter(provider string) *Reporter {
	return &Reporter{provider: provider}
}

// Report reads a whole record at once.
func Report(provider string, lines []protocol.ExchangeLine) protocol.ExchangeReport {
	x := NewReporter(provider)
	for _, l := range lines {
		x.Add(l.Stream, l.Text)
	}
	return x.Report()
}

func (x *Reporter) Report() protocol.ExchangeReport {
	return x.r
}

// Add reads one line, and says whether it changed the report. Only stdout is
// read: all three CLIs print their events there, and stderr is prose.
func (x *Reporter) Add(stream, text string) bool {
	if stream != protocol.StreamStdout || text == "" || text[0] != '{' {
		return false
	}
	switch x.provider {
	case "claude":
		return x.claude(text)
	case "codex":
		return x.codex(text)
	case "agy":
		return x.agy(text)
	default:
		// A provider this build does not know yet. Its lines are still in the
		// record; there is just no reading of them here.
		return false
	}
}

func (x *Reporter) claude(text string) bool {
	var ev struct {
		Type           string            `json:"type"`
		Subtype        string            `json:"subtype"`
		Cwd            string            `json:"cwd"`
		SessionID      string            `json:"session_id"`
		Model          string            `json:"model"`
		PermissionMode string            `json:"permissionMode"`
		Version        string            `json:"claude_code_version"`
		Tools          []string          `json:"tools"`
		Skills         []string          `json:"skills"`
		Agents         []string          `json:"agents"`
		MCPServers     []json.RawMessage `json:"mcp_servers"`
		Usage          *claudeUsage      `json:"usage"`
	}
	if json.Unmarshal([]byte(text), &ev) != nil {
		return false
	}
	switch {
	case ev.Type == "system" && ev.Subtype == "init":
		x.r.CLIVersion, x.r.Model, x.r.PermissionMode = ev.Version, ev.Model, ev.PermissionMode
		x.r.Cwd, x.r.SessionID = ev.Cwd, ev.SessionID
		x.r.Tools, x.r.Skills, x.r.Agents = ev.Tools, ev.Skills, ev.Agents
		x.r.MCPServers = names(ev.MCPServers)
		return true
	case ev.Type == "result" && ev.Usage != nil:
		// The result's usage covers every model call in the turn, and its four
		// fields do not overlap: fresh input, input written to the cache, input
		// read from it, and output.
		u := ev.Usage
		input := u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
		x.r.Usage = &protocol.ExchangeUsage{
			Input:     input,
			FromCache: ptr(u.CacheReadInputTokens),
			ToCache:   ptr(u.CacheCreationInputTokens),
			Output:    u.OutputTokens,
			Total:     input + u.OutputTokens,
		}
		return true
	}
	return false
}

func (x *Reporter) codex(text string) bool {
	var ev struct {
		Type     string      `json:"type"`
		ThreadID string      `json:"thread_id"`
		Usage    *codexUsage `json:"usage"`
	}
	if json.Unmarshal([]byte(text), &ev) != nil {
		return false
	}
	switch {
	case ev.Type == "thread.started" && ev.ThreadID != "":
		x.r.SessionID = ev.ThreadID
		return true
	case ev.Type == "turn.completed" && ev.Usage != nil:
		// Cached input is part of input, and reasoning part of output (see
		// codexUsage.total).
		u := ev.Usage
		x.r.Usage = &protocol.ExchangeUsage{
			Input:     u.InputTokens,
			FromCache: ptr(u.CachedInputTokens),
			ToCache:   ptr(u.CacheWriteInputTokens),
			Output:    u.OutputTokens,
			Reasoning: ptr(u.ReasoningOutputTokens),
			Total:     u.total(),
		}
		return true
	}
	return false
}

func (x *Reporter) agy(text string) bool {
	var ev struct {
		Event          string `json:"event"`
		ConversationID string `json:"conversation_id"`
		Init           struct {
			Cwd string `json:"cwd"`
		} `json:"init"`
		Result struct {
			Usage *agyUsage `json:"usage"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(text), &ev) != nil {
		return false
	}
	switch {
	case ev.Event == "init":
		x.r.SessionID, x.r.Cwd = ev.ConversationID, ev.Init.Cwd
		return true
	case ev.Event == "result" && ev.Result.Usage != nil:
		// agy's total_tokens is input + output + thinking, so its thinking is
		// NOT inside output_tokens; here it is folded in, to keep Output the
		// whole of what was written. Whether cache_read_tokens is inside
		// input_tokens is unverified: every capture so far has it at 0
		// (docs/KnownGaps.md G-49).
		u := ev.Result.Usage
		x.r.Usage = &protocol.ExchangeUsage{
			Input:     u.InputTokens,
			FromCache: ptr(u.CacheReadTokens),
			Output:    u.OutputTokens + u.ThinkingTokens,
			Reasoning: ptr(u.ThinkingTokens),
			Total:     u.total(),
		}
		return true
	}
	return false
}

// names reads MCP servers as claude lists them: objects carrying a name, or,
// should that ever change, plain strings.
func names(raw []json.RawMessage) []string {
	var out []string
	for _, r := range raw {
		var named struct {
			Name string `json:"name"`
		}
		var plain string
		switch {
		case json.Unmarshal(r, &named) == nil && named.Name != "":
			out = append(out, named.Name)
		case json.Unmarshal(r, &plain) == nil && plain != "":
			out = append(out, plain)
		}
	}
	return out
}

func ptr(n int64) *int64 { return &n }
