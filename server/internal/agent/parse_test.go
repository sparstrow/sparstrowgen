package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/* Every fixture here is REAL output, captured 2026-09-10 by running the CLI and
   saving what it printed. Invented fixtures would only prove the parser agrees
   with my idea of the format, which is the thing most likely to be wrong.

   No test in this package executes an agent CLI. We drive the same binaries the
   owner uses, on his machine and his account, so a test that resolved `claude`
   from PATH would spend his quota every time anyone ran `go test ./...`. The
   rule is borrowed from Multica, which hit this at scale first. */

// drain collects messages while a parser runs, so a blocking send cannot
// deadlock the test.
func drain(t *testing.T, fn func(chan<- Message) parsed) (parsed, []Message) {
	t.Helper()
	ch := make(chan Message, 256)
	done := make(chan []Message, 1)
	go func() {
		var got []Message
		for m := range ch {
			got = append(got, m)
		}
		done <- got
	}()
	p := fn(ch)
	close(ch)
	return p, <-done
}

// ---------------------------------------------------------------------------
// claude
// ---------------------------------------------------------------------------

const claudeStream = `{"type":"system","subtype":"init","session_id":"52e935ea-102a-428b-940a-1797de25b332","tools":[],"mcp_servers":[],"model":"claude-sonnet-4-6","apiKeySource":"none"}
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"When "}}}
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"many "}}}
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"clients retry"}}}
{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":1789083000,"rateLimitType":"five_hour","overageStatus":"rejected","isUsingOverage":false}}
{"type":"assistant","message":{"content":[{"type":"text","text":"When many clients retry at once they synchronise."}]}}
{"type":"result","subtype":"success","is_error":false,"result":"When many clients retry at once they synchronise.","total_cost_usd":0.03630945,"usage":{"input_tokens":3,"output_tokens":4,"cache_creation_input_tokens":10,"cache_read_input_tokens":100}}`

func TestParseClaude(t *testing.T) {
	p, msgs := drain(t, func(ch chan<- Message) parsed {
		return parseClaude(strings.NewReader(claudeStream), ch)
	})

	if p.Err != nil {
		t.Fatalf("unexpected error: %v", p.Err)
	}
	// The assistant event supersedes the deltas of its OWN message rather than
	// adding to them. Getting this wrong doubles every streamed answer.
	if want := "When many clients retry at once they synchronise."; p.Text != want {
		t.Errorf("text = %q, want %q", p.Text, want)
	}
	if p.SessionID != "52e935ea-102a-428b-940a-1797de25b332" {
		t.Errorf("session id = %q", p.SessionID)
	}
	if p.Tokens != 117 {
		t.Errorf("tokens = %d, want 117 (all four usage fields summed)", p.Tokens)
	}
	// 0.03630945 USD in ticks of 1e-10.
	if p.SpendTicks != 363094500 {
		t.Errorf("spend ticks = %d, want 363094500", p.SpendTicks)
	}

	var deltas, started, limits int
	for _, m := range msgs {
		switch m.Type {
		case MessageDelta:
			deltas++
		case MessageStarted:
			started++
		case MessageLimit:
			limits++
			if m.Headroom.Window != "five_hour" || m.Headroom.ResetsAt != 1789083000 {
				t.Errorf("headroom = %+v", m.Headroom)
			}
		}
	}
	if deltas != 3 || started != 1 || limits != 1 {
		t.Errorf("deltas=%d started=%d limits=%d, want 3/1/1", deltas, started, limits)
	}
}

// A failed turn reports subtype "success" with is_error true. Reading the
// subtype turns a total authentication failure into an apparent success, which
// is exactly the mistake this test exists to prevent recurring.
// One claude turn, four assistant messages: thinking, a sentence of prose, a
// tool call, then the answer. Captured 2026-09-10 from "First write one
// sentence of prose saying which file you are about to open. Then read
// package.json. Then give me a fenced json code block…".
//
// A file rather than an inline constant: the real stream is 61 lines, most of
// them deltas and thinking signatures the parser has to ignore, and that is
// precisely the part worth keeping honest.
func TestParseClaudeKeepsEveryMessage(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "claude-two-messages.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	p, _ := drain(t, func(ch chan<- Message) parsed {
		return parseClaude(bytes.NewReader(raw), ch)
	})
	if p.Err != nil {
		t.Fatalf("unexpected error: %v", p.Err)
	}

	// The bug this exists for (docs/Bugs.md B-6): the assistant event reset the
	// whole buffer, so this sentence — a complete message, sent before the tool
	// call — was dropped without trace.
	const preamble = "Opening the root package.json to check the package identity."
	if !strings.Contains(p.Text, preamble) {
		t.Errorf("the message before the tool call was dropped; text = %q", p.Text)
	}
	if !strings.Contains(p.Text, "```json") {
		t.Errorf("the final message is missing; text = %q", p.Text)
	}
	if !strings.Contains(p.Text, preamble+"\n\n```json") {
		t.Errorf("messages are not separated by a blank line; text = %q", p.Text)
	}

	// Reasoning and tool arguments stream as thinking_delta and
	// input_json_delta, neither of which carries `.text`. If that ever changes,
	// claude's private reasoning starts appearing in the transcript.
	if strings.Contains(p.Text, "package identity.Opening") ||
		strings.Contains(strings.ToLower(p.Text), "let me") {
		t.Errorf("non-answer content leaked into the text: %q", p.Text)
	}

	for i, line := range strings.Split(p.Text, "\n") {
		if strings.Contains(line, "```") && !strings.HasPrefix(strings.TrimLeft(line, " "), "```") {
			t.Fatalf("line %d has a fence that does not start it: %q", i+1, line)
		}
	}
}

func TestParseClaudeAuthFailureIsNotSuccess(t *testing.T) {
	const stream = `{"type":"system","subtype":"api_retry","attempt":1,"error_status":401,"error":"authentication_failed"}
{"type":"result","subtype":"success","is_error":true,"result":"Failed to authenticate. API Error: 401 {\"error\":{\"message\":\"OAuth access token has expired. Re-authenticate to continue.\"}}","total_cost_usd":0}`

	p, _ := drain(t, func(ch chan<- Message) parsed {
		return parseClaude(strings.NewReader(stream), ch)
	})

	if p.Err == nil {
		t.Fatal("a turn with is_error true must be an error, whatever the subtype says")
	}
	if !strings.Contains(p.Err.Error(), "setup-token") {
		t.Errorf("an auth failure should say how to fix it, got: %v", p.Err)
	}
}

func TestParseClaudeIgnoresNonJSONLines(t *testing.T) {
	const stream = `Warning: something on stderr got interleaved
{"type":"assistant","message":{"content":[{"type":"text","text":"ok"}]}}
not json either`

	p, _ := drain(t, func(ch chan<- Message) parsed {
		return parseClaude(strings.NewReader(stream), ch)
	})
	if p.Text != "ok" {
		t.Errorf("text = %q, want %q", p.Text, "ok")
	}
}

// ---------------------------------------------------------------------------
// codex
// ---------------------------------------------------------------------------

const codexStream = `{"type": "thread.started", "thread_id": "01a08caf-32e6-7531-a336-c4e5c756fde2"}
{"type": "turn.started"}
{"type": "item.completed", "item": {"id": "item_0", "type": "agent_message", "text": "ok"}}
{"type": "turn.completed", "usage": {"input_tokens": 19753, "cached_input_tokens": 12288, "cache_write_input_tokens": 0, "output_tokens": 5, "reasoning_output_tokens": 0}}`

func TestParseCodex(t *testing.T) {
	p, msgs := drain(t, func(ch chan<- Message) parsed {
		return parseCodex(strings.NewReader(codexStream), ch)
	})

	if p.Err != nil {
		t.Fatalf("unexpected error: %v", p.Err)
	}
	if p.Text != "ok" {
		t.Errorf("text = %q", p.Text)
	}
	// thread_id is the resume handle. Losing it silently starts a fresh session
	// on the next turn and replays the whole history again.
	if p.SessionID != "01a08caf-32e6-7531-a336-c4e5c756fde2" {
		t.Errorf("session id = %q", p.SessionID)
	}
	// Input plus output. The 12,288 cached tokens are part of the 19,753 read,
	// and adding them again showed a turn as using far more than it did (B-56).
	if p.Tokens != 19758 {
		t.Errorf("tokens = %d, want 19758 (input_tokens + output_tokens)", p.Tokens)
	}

	// codex is verified to emit no deltas. If this ever starts failing, the CLI
	// has changed and the "sends its reply in one piece" copy is now wrong.
	for _, m := range msgs {
		if m.Type == MessageDelta {
			t.Errorf("codex emitted a delta: %+v — the UI copy assumes it never does", m)
		}
	}
}

// One turn, two agent_message items: a preamble sentence and then the code.
// Captured 2026-09-10 from `give me code block with dsa tree in c++`. Kept as a
// file rather than an inline constant only because the second message is 2.4kB
// of C++ and would swamp this one.
func TestParseCodexSeparatesMessages(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "codex-two-messages.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	p, _ := drain(t, func(ch chan<- Message) parsed {
		return parseCodex(bytes.NewReader(raw), ch)
	})
	if p.Err != nil {
		t.Fatalf("unexpected error: %v", p.Err)
	}

	// The bug this exists for (docs/Bugs.md B-5): joined with no separator, the
	// preamble's last line and the opening fence share a line, at which point
	// CommonMark stops seeing a fence at all and the whole reply renders as
	// shredded inline code.
	for i, line := range strings.Split(p.Text, "\n") {
		trimmed := strings.TrimLeft(line, " ")
		if strings.Contains(line, "```") && !strings.HasPrefix(trimmed, "```") {
			t.Fatalf("line %d has a fence that does not start it: %q", i+1, line)
		}
	}

	if !strings.Contains(p.Text, "traversal.\n\n```cpp") {
		t.Errorf("the two messages are not separated by a blank line; text begins:\n%s",
			p.Text[:min(len(p.Text), 200)])
	}
}

// ---------------------------------------------------------------------------
// agy
// ---------------------------------------------------------------------------

const agyStream = `{"event": "init", "conversation_id": "74a2cdae-09a8-4846-8738-0e4eadc22f0d", "init": {"cwd": "D:\\sparstrowgen"}}
{"event": "step_update", "step_update": {"step_index": 0, "state": "ACTIVE", "text_delta": "Jitter "}}
{"event": "step_update", "step_update": {"step_index": 0, "state": "ACTIVE", "text_delta": "spreads retries"}}
{"event": "step_update", "step_update": {"step_index": 0, "state": "DONE", "step_type": "user_input"}}
{"event": "result", "result": {"conversation_id": "74a2cdae-09a8-4846-8738-0e4eadc22f0d", "status": "SUCCESS", "response": "Jitter spreads retries.\n", "usage": {"input_tokens": 15917, "output_tokens": 124, "thinking_tokens": 123, "cache_read_tokens": 0, "total_tokens": 16164}}}`

func TestParseAgy(t *testing.T) {
	p, msgs := drain(t, func(ch chan<- Message) parsed {
		return parseAgy(strings.NewReader(agyStream), ch)
	})

	if p.Err != nil {
		t.Fatalf("unexpected error: %v", p.Err)
	}
	// result.response wins over the accumulated deltas, and its trailing
	// newline is trimmed.
	if p.Text != "Jitter spreads retries." {
		t.Errorf("text = %q", p.Text)
	}
	if p.SessionID != "74a2cdae-09a8-4846-8738-0e4eadc22f0d" {
		t.Errorf("session id = %q", p.SessionID)
	}
	// total_tokens is authoritative when present, not the sum of the parts.
	if p.Tokens != 16164 {
		t.Errorf("tokens = %d, want 16164", p.Tokens)
	}

	var deltas int
	for _, m := range msgs {
		if m.Type == MessageDelta {
			deltas++
		}
	}
	if deltas != 2 {
		t.Errorf("deltas = %d, want 2", deltas)
	}
}

func TestParseAgyFailedStatusIsAnError(t *testing.T) {
	const stream = `{"event": "result", "result": {"status": "ERROR", "response": "", "usage": {}}}`
	p, _ := drain(t, func(ch chan<- Message) parsed {
		return parseAgy(strings.NewReader(stream), ch)
	})
	if p.Err == nil {
		t.Fatal("a non-SUCCESS status must surface as an error")
	}
}

// agy falls back to the streamed text when result.response is empty, so a turn
// cut short still shows what arrived.
func TestParseAgyFallsBackToStreamedText(t *testing.T) {
	const stream = `{"event": "step_update", "step_update": {"state": "ACTIVE", "text_delta": "partial answer"}}`
	p, _ := drain(t, func(ch chan<- Message) parsed {
		return parseAgy(strings.NewReader(stream), ch)
	})
	if p.Text != "partial answer" {
		t.Errorf("text = %q, want the streamed text", p.Text)
	}
}

// ---------------------------------------------------------------------------
// argument construction
// ---------------------------------------------------------------------------

func contains(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

// The scoping flags are a security boundary, not a preference: without them a
// spawned claude gets working MCP tool access to whatever the owner has
// configured. See docs/Decisions.md D-016.
func TestClaudeArgsAlwaysScoped(t *testing.T) {
	args := claudeArgs(ExecOptions{})
	for _, want := range []string{"--strict-mcp-config", "--setting-sources", "--verbose", "AskUserQuestion"} {
		if !contains(args, want) {
			t.Errorf("claude args missing %s: %v", want, args)
		}
	}
	// --bare cannot authenticate a subscription account at all.
	if contains(args, "--bare") {
		t.Error("--bare must never be passed: it never reads OAuth")
	}
}

func TestCodexArgsAlwaysScoped(t *testing.T) {
	if !contains(codexArgs(ExecOptions{}), "--ignore-user-config") {
		t.Error("codex must always pass --ignore-user-config")
	}
}

// resume is positional for codex and a flag for the other two. Getting the
// order wrong sends the thread id to the model as the prompt. The prompt itself
// is "-", read from stdin (B-32).
func TestCodexResumeIsPositional(t *testing.T) {
	args := codexArgs(ExecOptions{ResumeSessionID: "thread-1"})
	last3 := args[len(args)-3:]
	want := []string{"resume", "thread-1", "-"}
	for i := range want {
		if last3[i] != want[i] {
			t.Fatalf("resume args = %v, want %v", last3, want)
		}
	}
}

func TestAgyResumeUsesConversationFlag(t *testing.T) {
	args := agyArgs(ExecOptions{ResumeSessionID: "conv-1"}, "")
	if !contains(args, "--conversation") || !contains(args, "conv-1") {
		t.Errorf("agy resume args = %v", args)
	}
}

// ---------------------------------------------------------------------------
// environment
// ---------------------------------------------------------------------------

// B-28: a daemon started before the token was set, or from a program that never
// had it, must still hand claude the user's current token.
func TestScrubbedEnvTakesTheUsersCurrentEnvironment(t *testing.T) {
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "sk-ant-oat01-replaced")
	t.Setenv("CODEX_HOME", `C:\from\process`)
	old := userEnvironment
	userEnvironment = func() map[string]string {
		return map[string]string{
			"CLAUDE_CODE_OAUTH_TOKEN":        "sk-ant-oat01-current",
			"CODEX_HOME":                     `C:\from\user`,
			"SPARSTROWGEN_ONLY_IN_USER_ENV":  "yes",
			"CLAUDECODE":                     "1",
			"ANTHROPIC_BASE_URL":             "https://example.invalid",
			"Path":                           `C:\only\user`,
			"SPARSTROWGEN_EMPTY_IN_USER_ENV": "",
		}
	}
	t.Cleanup(func() { userEnvironment = old })

	values := map[string][]string{}
	for _, kv := range scrubbedEnv() {
		key, value, _ := strings.Cut(kv, "=")
		values[strings.ToUpper(key)] = append(values[strings.ToUpper(key)], value)
	}

	if got := values["CLAUDE_CODE_OAUTH_TOKEN"]; len(got) != 1 || got[0] != "sk-ant-oat01-current" {
		t.Errorf("token = %v; the user's current token must replace the one the process inherited", got)
	}
	if got := values["SPARSTROWGEN_ONLY_IN_USER_ENV"]; len(got) != 1 || got[0] != "yes" {
		t.Errorf("a variable only the user environment has = %v, want it filled in", got)
	}
	if got := values["CODEX_HOME"]; len(got) != 1 || got[0] != `C:\from\process` {
		t.Errorf("CODEX_HOME = %v; a variable the process already has must keep its value", got)
	}
	if _, ok := values["CLAUDECODE"]; ok {
		t.Error("the user environment must not bring back a scrubbed variable")
	}
	if _, ok := values["ANTHROPIC_BASE_URL"]; ok {
		t.Error("the user environment must not bring back ANTHROPIC_BASE_URL")
	}
	for _, p := range values["PATH"] {
		if p == `C:\only\user` {
			t.Error("PATH must stay the process's, which already joins machine and user")
		}
	}
	if _, ok := values["SPARSTROWGEN_EMPTY_IN_USER_ENV"]; ok {
		t.Error("an empty user variable must not be added")
	}
}

func TestScrubbedEnvKeepsTheOAuthTokenAndDropsTheRest(t *testing.T) {
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "sk-ant-oat01-test")
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CLAUDE_CODE_MESSAGING_SOCKET", "\\\\.\\pipe\\x")
	t.Setenv("ANTHROPIC_BASE_URL", "https://example.invalid")
	t.Setenv("CODEX_HOME", "C:\\Users\\x\\.codex")

	env := scrubbedEnv()
	has := func(prefix string) bool {
		for _, kv := range env {
			if strings.HasPrefix(kv, prefix) {
				return true
			}
		}
		return false
	}

	// Without this, claude cannot authenticate at all from a spawned process.
	if !has("CLAUDE_CODE_OAUTH_TOKEN=") {
		t.Error("the OAuth token must survive scrubbing — scrubbing it deletes the fix")
	}
	// With these, a nested claude -p hangs indefinitely.
	if has("CLAUDECODE=") || has("CLAUDE_CODE_MESSAGING_SOCKET=") {
		t.Error("nested-session variables must be scrubbed or claude hangs")
	}
	// An inherited base URL would silently redirect the spawned CLI.
	if has("ANTHROPIC_BASE_URL=") {
		t.Error("ANTHROPIC_BASE_URL must be scrubbed")
	}
	// codex finds its credentials here even with --ignore-user-config.
	if !has("CODEX_HOME=") {
		t.Error("CODEX_HOME must survive: it is how codex authenticates")
	}
}

// Pictures reach codex as --image=<path>, the value attached, so the flag's
// several values cannot swallow the thread id or the "-" that reads stdin.
func TestCodexGetsPicturesWithoutLosingItsThread(t *testing.T) {
	fresh := strings.Join(codexArgs(ExecOptions{Images: []string{`C:\a.png`, `C:\b b.jpg`}}), " ")
	if !strings.HasSuffix(fresh, `--image=C:\a.png --image=C:\b b.jpg -`) {
		t.Errorf("new session: %s", fresh)
	}
	resumed := strings.Join(codexArgs(ExecOptions{ResumeSessionID: "thread-1", Images: []string{`C:\a.png`}}), " ")
	if !strings.HasSuffix(resumed, `resume --image=C:\a.png thread-1 -`) {
		t.Errorf("resumed: %s", resumed)
	}
}

func TestClaudeAndAgyAreGivenTheFilesFolder(t *testing.T) {
	opts := ExecOptions{Cwd: `D:\proj`, AddDirs: []string{`C:\chats\c1`}}
	if got := strings.Join(claudeArgs(opts), " "); !strings.Contains(got, `--add-dir C:\chats\c1`) {
		t.Errorf("claude: %s", got)
	}
	if got := strings.Join(agyArgs(opts, ""), " "); !strings.Contains(got, `--add-dir D:\proj --add-dir C:\chats\c1`) {
		t.Errorf("agy: %s", got)
	}
}
