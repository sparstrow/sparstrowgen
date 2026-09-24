package agent

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* The fixtures are the U-32 test turns' own session files, trimmed: real
   structure, shortened text (docs/Capabilities.md, "What each agent loaded,
   word for word"). The agy record is built here in the layout agy 1.2.3 writes,
   because its real one is a binary blob holding the owner's own rules. */

// docsFrom reads a fixture the way internal/sessionstore does: one document per
// context record, in file order.
func claudeDocs(t *testing.T) []protocol.ContextDocument {
	t.Helper()
	data, err := os.ReadFile("testdata/claude-session-context.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	var docs []protocol.ContextDocument
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var l struct {
			Type       string          `json:"type"`
			Attachment json.RawMessage `json:"attachment"`
			Rendered   json.RawMessage `json:"rendered"`
		}
		if json.Unmarshal([]byte(line), &l) != nil || l.Type != "attachment" {
			continue
		}
		if len(l.Rendered) == 0 {
			l.Rendered = json.RawMessage("null")
		}
		body, _ := json.Marshal(map[string]json.RawMessage{"attachment": l.Attachment, "rendered": l.Rendered})
		docs = append(docs, protocol.ContextDocument{Kind: "claude.attachment", Body: string(body)})
	}
	return docs
}

func byKind(pieces []protocol.ContextPiece) map[string][]protocol.ContextPiece {
	out := map[string][]protocol.ContextPiece{}
	for _, p := range pieces {
		out[p.Kind] = append(out[p.Kind], p)
	}
	return out
}

func pieceNames(pieces []protocol.ContextPiece) []string {
	var out []string
	for _, p := range pieces {
		out = append(out, p.Name)
	}
	return out
}

func TestClaudeFedItsPromptToolsFilesSkillsAndServers(t *testing.T) {
	pieces, missing := FedContext("claude", claudeDocs(t))
	k := byKind(pieces)

	// The system prompt in its sections, the opening line first. Two snapshots
	// were recorded; the second replaced the first rather than doubling it.
	ins := k[KindInstructions]
	if len(ins) != 16 || ins[0].Name != "opening line" {
		t.Errorf("instructions = %d %v", len(ins), pieceNames(ins))
	}
	if !strings.HasPrefix(ins[1].Text, "\nYou are an interactive agent") {
		t.Errorf("first section = %q", ins[1].Text[:40])
	}
	// Tools come from the snapshot that carried them.
	if tools := k[KindTool]; len(tools) != 3 || tools[0].Name != "Agent" || !strings.Contains(tools[0].Text, `"name": "Agent"`) {
		t.Errorf("tools = %v", pieceNames(tools))
	}
	// Each instruction file names the file on the computer.
	files := k[KindFile]
	if len(files) != 3 || files[0].Name != "CLAUDE.md · project" || files[0].Source != `D:\work\CLAUDE.md` ||
		files[1].Name != "CLAUDE.md · user" || files[2].Source != `D:\work\Reference\CLAUDE.md` {
		t.Errorf("files = %+v", files)
	}
	if skills := k[KindSkill]; len(skills) != 4 || skills[0].Name != "dataviz" || !strings.HasPrefix(skills[0].Text, "- dataviz:") {
		t.Errorf("skills = %v", pieceNames(skills))
	}
	// One server was added and a later record removed the other.
	if servers := k[KindServer]; len(servers) != 1 || servers[0].Name != "context7" {
		t.Errorf("servers = %v", pieceNames(servers))
	}
	if d := k[KindDeferred]; len(d) != 8 || d[0].Name != "CronCreate" {
		t.Errorf("deferred = %v", pieceNames(d))
	}
	if a := k[KindAgent]; len(a) != 5 {
		t.Errorf("agents = %v", pieceNames(a))
	}
	// What claude rendered is what is shown, not the attachment's fields.
	for _, e := range k[KindEnvironment] {
		if e.Name == "environment" && !strings.HasPrefix(e.Text, "<system-reminder>") {
			t.Errorf("environment = %q", e.Text[:40])
		}
	}
	if len(missing) != 1 || !strings.Contains(missing[0], "which file each skill") {
		t.Errorf("missing = %v", missing)
	}
	for _, p := range pieces {
		if p.Chars != len([]rune(p.Text)) {
			t.Errorf("%s: chars %d for %d runes", p.Name, p.Chars, len([]rune(p.Text)))
		}
	}
}

func codexDocs(t *testing.T) []protocol.ContextDocument {
	t.Helper()
	data, err := os.ReadFile("testdata/codex-rollout-context.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	var docs []protocol.ContextDocument
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var l struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		_ = json.Unmarshal([]byte(line), &l)
		var m struct {
			Type, Role string
			Content    []struct{ Text string }
		}
		_ = json.Unmarshal(l.Payload, &m)
		switch {
		case l.Type == "session_meta":
			docs = append(docs, protocol.ContextDocument{Kind: "codex.session_meta", Body: string(l.Payload)})
		case l.Type == "world_state":
			docs = append(docs, protocol.ContextDocument{Kind: "codex.world_state", Body: string(l.Payload)})
		case l.Type == "turn_context":
			docs = append(docs, protocol.ContextDocument{Kind: "codex.turn_context", Body: string(l.Payload)})
		case m.Type == "message" && (m.Role == "developer" || m.Role == "user" && strings.HasPrefix(m.Content[0].Text, "<")):
			docs = append(docs, protocol.ContextDocument{Kind: "codex.message", Body: string(l.Payload)})
		}
	}
	return docs
}

func TestCodexFedItsBaseInstructionsAndSkillsWithTheirFiles(t *testing.T) {
	pieces, missing := FedContext("codex", codexDocs(t))
	k := byKind(pieces)

	ins := k[KindInstructions]
	if len(ins) < 3 || ins[0].Name != "base instructions for gpt-5.6-sol" {
		t.Errorf("instructions = %v", pieceNames(ins))
	}
	// Each skill carries the file it lives in, with its short root expanded.
	var super *protocol.ContextPiece
	for i, s := range k[KindSkill] {
		if s.Name == "superpowers:using-superpowers" {
			super = &k[KindSkill][i]
		}
	}
	if super == nil {
		t.Fatalf("skills = %v", pieceNames(k[KindSkill]))
	}
	if super.Source != "C:/Users/gsrih/.agents/skills/superpowers-main/superpowers-main/skills/using-superpowers/SKILL.md" {
		t.Errorf("source = %q", super.Source)
	}
	if o := k[KindOther]; len(o) != 1 || o[0].Name != "plugins codex recommends installing" {
		t.Errorf("other = %v", pieceNames(o))
	}
	// Its environment block, and the sandbox it ran in. One message holds
	// several blocks as separate parts, and each is its own piece.
	env := k[KindEnvironment]
	if len(env) != 2 || env[0].Name != "environment" || !strings.HasPrefix(env[0].Text, "<environment_context>") ||
		!strings.Contains(env[1].Text, `"read-only"`) {
		t.Errorf("environment = %+v", env)
	}
	for _, name := range []string{"how codex uses skills", "plugins instructions", "multi agent mode"} {
		found := false
		for _, p := range ins {
			found = found || p.Name == name
		}
		if !found {
			t.Errorf("no %q among %v", name, pieceNames(ins))
		}
	}
	if len(k[KindTool]) != 0 || len(missing) != 1 || !strings.Contains(missing[0], "tool definitions") {
		t.Errorf("tools %d, missing %v", len(k[KindTool]), missing)
	}
}

// ---------------------------------------------------------------------------
// agy, in its wire format
// ---------------------------------------------------------------------------

func pbVar(v uint64) []byte {
	var b []byte
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func pbStr(num uint64, s string) []byte { return pbBytes(num, []byte(s)) }

func pbBytes(num uint64, b []byte) []byte {
	return append(append(pbVar(num<<3|2), pbVar(uint64(len(b)))...), b...)
}

func pbInt(num, v uint64) []byte { return append(pbVar(num<<3), pbVar(v)...) }

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func agyRecord() string {
	section := func(name, text string) []byte { return pbBytes(16, cat(pbStr(1, name), pbStr(2, text))) }
	tool := func(name, desc, schema string) []byte {
		return pbBytes(8, cat(pbStr(1, name), pbStr(2, desc), pbStr(3, schema)))
	}
	count := func(name string, n uint64) []byte { return pbBytes(5, cat(pbStr(1, name), pbInt(3, n))) }
	breakdown := pbBytes(9, pbBytes(10, pbBytes(3, pbBytes(1, cat(pbStr(1, "System Prompt"), pbInt(4, 4136),
		count("identity", 159), count("user_rules", 3977))))))
	req := cat(
		pbStr(1, "<identity>\nYou are Antigravity.</identity>\n<user_rules>\nAlways cite.</user_rules>"),
		section("identity", "<identity>\nYou are Antigravity.</identity>"),
		section("user_rules", "<user_rules>\nAlways cite.</user_rules>"),
		section("skills", "<skills>\nimpeccable\n</skills>"),
		section("mcp_servers", "<mcp_servers>\nshadcn\n</mcp_servers>"),
		tool("view_file", "View the contents of a file.", `{"type":"object","properties":{"AbsolutePath":{"type":"string"}}}`),
		tool("search_web", "Search the web.", `{"type":"object"}`),
		breakdown,
		pbStr(19, "gemini-pro-default"),
	)
	return base64.StdEncoding.EncodeToString(cat(pbBytes(1, req), pbStr(4, "5ac6970a")))
}

func TestAgyFedItsSectionsWithItsOwnTokenCountsAndItsTools(t *testing.T) {
	pieces, missing := FedContext("agy", []protocol.ContextDocument{{Kind: "agy.gen_metadata", Body: agyRecord()}})
	k := byKind(pieces)

	if ins := k[KindInstructions]; len(ins) != 1 || ins[0].Name != "identity" || ins[0].Tokens == nil || *ins[0].Tokens != 159 {
		t.Errorf("instructions = %+v", ins)
	}
	// The owner's rules are a file of his, with agy's own count of them.
	if f := k[KindFile]; len(f) != 1 || f[0].Name != "user rules" || f[0].Tokens == nil || *f[0].Tokens != 3977 {
		t.Errorf("files = %+v", f)
	}
	if len(k[KindSkill]) != 1 || len(k[KindServer]) != 1 {
		t.Errorf("skills %v, servers %v", pieceNames(k[KindSkill]), pieceNames(k[KindServer]))
	}
	tools := k[KindTool]
	if len(tools) != 2 || tools[0].Name != "view_file" || !strings.Contains(tools[0].Text, `"AbsolutePath"`) {
		t.Errorf("tools = %+v", tools)
	}
	if len(missing) != 1 || !strings.Contains(missing[0], "user rules") {
		t.Errorf("missing = %v", missing)
	}
}

func TestARecordInAShapeNobodyKnowsIsShownNotDropped(t *testing.T) {
	pieces, _ := FedContext("agy", []protocol.ContextDocument{
		{Kind: "agy.gen_metadata", Body: base64.StdEncoding.EncodeToString([]byte("not protobuf at all \xff"))},
		{Kind: "gemini.something", Body: "a record from a CLI this build has never heard of"},
	})
	if len(pieces) != 2 || pieces[0].Kind != KindOther || pieces[1].Name != "gemini.something" {
		t.Errorf("pieces = %+v", pieces)
	}
}

// The server is Linux and the paths come from Windows computers.
func TestAPathIsNamedByItsFileWhicheverComputerWroteIt(t *testing.T) {
	for path, want := range map[string]string{
		`D:\work\CLAUDE.md`:            "CLAUDE.md",
		`/home/me/.codex/AGENTS.md`:    "AGENTS.md",
		`C:/Users/me/.agents/SKILL.md`: "SKILL.md",
		`CLAUDE.md`:                    "CLAUDE.md",
	} {
		if got := fileName(path); got != want {
			t.Errorf("fileName(%q) = %q, want %q", path, got, want)
		}
	}
}
