package agent

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// chunked hands a string over a few bytes at a time, the way a pipe does:
// lines arrive split across reads, and the recorder must not care.
type chunked struct {
	s string
	n int
}

func (c *chunked) Read(p []byte) (int, error) {
	if c.s == "" {
		return 0, io.EOF
	}
	k := min(c.n, len(c.s), len(p))
	copy(p, c.s[:k])
	c.s = c.s[k:]
	return k, nil
}

func collect(stream string) (*splitter, *[]Line) {
	var got []Line
	return &splitter{stream: stream, emit: func(l Line) { got = append(got, l) }}, &got
}

func TestTheRecordKeepsEveryLineExactlyWhateverTheReadsLookLike(t *testing.T) {
	s, got := collect(protocol.StreamStdout)
	tap := &tapped{r: &chunked{s: "{\"a\":1}\r\n\n{\"b\":2}\npartial, killed mid-line", n: 3}, s: s}

	// The parser still gets every byte, in order.
	passed, err := io.ReadAll(tap)
	if err != nil {
		t.Fatal(err)
	}
	if string(passed) != "{\"a\":1}\r\n\n{\"b\":2}\npartial, killed mid-line" {
		t.Fatalf("the reader changed what the parser sees: %q", passed)
	}

	want := []string{`{"a":1}`, "", `{"b":2}`, "partial, killed mid-line"}
	if len(*got) != len(want) {
		t.Fatalf("lines = %d, want %d: %+v", len(*got), len(want), *got)
	}
	for i, l := range *got {
		if l.Text != want[i] || l.Stream != protocol.StreamStdout || l.Cut != 0 {
			t.Errorf("line %d = %+v, want %q on stdout", i, l, want[i])
		}
	}
}

func TestAnOverLongLineIsCutAndTheCutIsCounted(t *testing.T) {
	s, got := collect(protocol.StreamStdout)
	long := strings.Repeat("x", maxLine+10)
	s.write([]byte(long + "\nnext\n"))

	if len(*got) != 2 {
		t.Fatalf("lines = %d, want 2", len(*got))
	}
	if first := (*got)[0]; len(first.Text) != maxLine || first.Cut != 10 {
		t.Errorf("kept %d bytes and cut %d, want %d and 10", len(first.Text), first.Cut, maxLine)
	}
	// The cut belongs to that line only.
	if next := (*got)[1]; next.Text != "next" || next.Cut != 0 {
		t.Errorf("the line after it = %+v", next)
	}
}

func TestStderrIsRecordedOnItsOwnStream(t *testing.T) {
	var got []Line
	w := &lineWriter{s: &splitter{stream: protocol.StreamStderr, emit: func(l Line) { got = append(got, l) }}}
	_, _ = w.Write([]byte("CreateProcess ... rejected: blocked by pol"))
	_, _ = w.Write([]byte("icy\nsecond"))
	w.end()

	if len(got) != 2 || got[0].Text != "CreateProcess ... rejected: blocked by policy" || got[1].Text != "second" {
		t.Fatalf("stderr lines = %+v", got)
	}
	for _, l := range got {
		if l.Stream != protocol.StreamStderr {
			t.Errorf("stream = %q, want stderr", l.Stream)
		}
	}
}

// ---------------------------------------------------------------------------
// reports, read from real captures
// ---------------------------------------------------------------------------

func stdoutLines(t *testing.T, text string) []protocol.ExchangeLine {
	t.Helper()
	var out []protocol.ExchangeLine
	for i, l := range strings.Split(strings.TrimSpace(text), "\n") {
		out = append(out, protocol.ExchangeLine{Seq: int32(i + 1), Stream: protocol.StreamStdout, Text: strings.TrimRight(l, "\r")})
	}
	return out
}

func TestClaudeReportsWhatItLoadedAndWhatItUsed(t *testing.T) {
	data, err := os.ReadFile("testdata/claude-two-messages.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	r := Report("claude", stdoutLines(t, string(data)))

	if r.CLIVersion != "2.1.90" || r.Model != "claude-sonnet-4-6" || r.PermissionMode != "default" {
		t.Errorf("version, model, permissions = %q, %q, %q", r.CLIVersion, r.Model, r.PermissionMode)
	}
	if r.Cwd != `D:\sparstrowgen` || r.SessionID != "4af6f16f-10fb-4767-b7a9-4363d3d777f1" {
		t.Errorf("folder, session = %q, %q", r.Cwd, r.SessionID)
	}
	if len(r.Tools) != 24 || len(r.Agents) != 4 || len(r.Skills) == 0 || len(r.MCPServers) != 0 {
		t.Errorf("tools %d, agents %d, skills %d, MCP servers %d", len(r.Tools), len(r.Agents), len(r.Skills), len(r.MCPServers))
	}
	u := r.Usage
	if u == nil {
		t.Fatal("no usage read from the result line")
	}
	// 4 fresh + 9,177 written to the cache + 30,219 read from it.
	if u.Input != 39400 || *u.FromCache != 30219 || *u.ToCache != 9177 || u.Output != 165 || u.Total != 39565 {
		t.Errorf("usage = %+v (from cache %d, to cache %d)", u, *u.FromCache, *u.ToCache)
	}
	if u.Reasoning != nil {
		t.Error("claude does not break reasoning out, so none may be reported")
	}
}

func TestCodexReportsItsThreadAndUsageWithCachedInputInsideInput(t *testing.T) {
	data, err := os.ReadFile("testdata/codex-two-messages.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	r := Report("codex", stdoutLines(t, string(data)))

	if r.SessionID != "01a08db6-8502-7043-bd0a-e7a491a96c9f" {
		t.Errorf("session = %q", r.SessionID)
	}
	// codex reports nothing else about itself, and nothing may be made up.
	if r.Model != "" || r.CLIVersion != "" || r.Tools != nil {
		t.Errorf("codex reported %+v, which it never prints", r)
	}
	u := r.Usage
	if u == nil || u.Input != 39387 || *u.FromCache != 31744 || u.Output != 721 || *u.Reasoning != 25 || u.Total != 40108 {
		t.Errorf("usage = %+v", u)
	}
}

func TestAgyReportsItsConversationAndFolder(t *testing.T) {
	r := Report("agy", stdoutLines(t, agyStream))

	if r.SessionID != "74a2cdae-09a8-4846-8738-0e4eadc22f0d" || r.Cwd != `D:\sparstrowgen` {
		t.Errorf("session, folder = %q, %q", r.SessionID, r.Cwd)
	}
	// Its thinking is outside output_tokens (input + output + thinking is its
	// own total), so Output takes both and names the thinking as a part.
	u := r.Usage
	if u == nil || u.Input != 15917 || u.Output != 247 || *u.Reasoning != 123 || u.Total != 16164 {
		t.Errorf("usage = %+v", u)
	}
}

func TestAReportReadsOnlyStdoutAndSaysWhenItChanged(t *testing.T) {
	x := NewReporter("claude")
	if x.Add(protocol.StreamStderr, `{"type":"system","subtype":"init","model":"from stderr"}`) {
		t.Error("a stderr line changed the report")
	}
	if x.Add(protocol.StreamStdout, "not json at all") {
		t.Error("a line that is not an event changed the report")
	}
	if !x.Add(protocol.StreamStdout, `{"type":"system","subtype":"init","model":"claude-opus-5-5","mcp_servers":[{"name":"github","status":"connected"}]}`) {
		t.Fatal("the init line did not change the report")
	}
	if r := x.Report(); r.Model != "claude-opus-5-5" || len(r.MCPServers) != 1 || r.MCPServers[0] != "github" {
		t.Errorf("report = %+v", r)
	}
	if NewReporter("gemini").Add(protocol.StreamStdout, `{"type":"system"}`) {
		t.Error("a provider with no reading claimed a change")
	}
}
