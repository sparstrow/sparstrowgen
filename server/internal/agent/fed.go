package agent

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* What a CLI fed its model on a turn, read from the records it keeps about
   itself (internal/sessionstore collects them on the computer; this reads
   them). docs/specs/2026-09-23-raw-exchange.md, US4.

Every piece is text the record actually holds, never a reconstruction. Where
a CLI keeps no record of something, that is said in Missing, so an empty group
is never read as "it had none". The formats are private to each CLI and change
with its versions; whatever this does not recognise is shown under "other"
rather than dropped. */

const (
	KindInstructions = "instructions" // the CLI's own system prompt, in its sections
	KindFile         = "file"         // instruction files and rules the owner controls
	KindSkill        = "skill"
	KindTool         = "tool"
	KindDeferred     = "deferred" // tools named but held back until searched for
	KindServer       = "server"   // MCP servers
	KindAgent        = "agent"    // sub-agents it may call
	KindEnvironment  = "environment"
	KindOther        = "other"
)

var kindOrder = []string{KindInstructions, KindFile, KindSkill, KindTool, KindDeferred, KindServer, KindAgent, KindEnvironment, KindOther}

// FedContext reads a turn's context records into pieces.
func FedContext(provider string, docs []protocol.ContextDocument) (pieces []protocol.ContextPiece, missing []string) {
	f := &fed{byKey: map[string]int{}}
	for _, d := range docs {
		switch d.Kind {
		case "claude.attachment":
			f.claude(d.Body)
		case "codex.session_meta", "codex.message", "codex.world_state", "codex.turn_context":
			f.codex(d.Kind, d.Body)
		case "agy.gen_metadata":
			f.agy(d.Body)
		default:
			f.put(KindOther, d.Kind, "", d.Body, nil)
		}
	}
	switch provider {
	case "claude":
		if f.count(KindSkill) > 0 {
			missing = append(missing, "claude does not record which file each skill comes from.")
		}
	case "codex":
		missing = append(missing, "codex does not record its tool definitions, so none are shown.")
	case "agy":
		if f.hasName(KindFile, "user rules") {
			missing = append(missing, "agy does not record which files its user rules came from.")
		}
	}
	return f.sorted(), missing
}

// fed holds pieces by kind and name, so a later record of the same thing (the
// next snapshot, a delta that removes a server) replaces or removes it.
type fed struct {
	list  []protocol.ContextPiece
	byKey map[string]int
	gone  map[int]bool
}

// key identifies a piece: by its file where it has one, since two files can
// share a name (a project's CLAUDE.md and a subfolder's), and otherwise by name.
func key(kind, name, source string) string {
	if source != "" {
		return kind + "\x00" + source
	}
	return kind + "\x00" + name
}

func (f *fed) put(kind, name, source, text string, tokens *int64) {
	p := protocol.ContextPiece{Kind: kind, Name: name, Source: source, Text: text, Chars: len([]rune(text)), Tokens: tokens}
	k := key(kind, name, source)
	if i, ok := f.byKey[k]; ok && !f.gone[i] {
		f.list[i] = p
		return
	}
	f.byKey[k] = len(f.list)
	f.list = append(f.list, p)
}

func (f *fed) remove(kind, name string) {
	f.drop(key(kind, name, ""))
}

func (f *fed) drop(k string) {
	if i, ok := f.byKey[k]; ok {
		if f.gone == nil {
			f.gone = map[int]bool{}
		}
		f.gone[i] = true
		delete(f.byKey, k)
	}
}

// clear removes every piece of a kind: a new whole snapshot replaces the last.
func (f *fed) clear(kind string) {
	for _, p := range f.list {
		if p.Kind == kind {
			f.drop(key(kind, p.Name, p.Source))
		}
	}
}

func (f *fed) count(kind string) int {
	n := 0
	for i, p := range f.list {
		if p.Kind == kind && !f.gone[i] {
			n++
		}
	}
	return n
}

func (f *fed) hasName(kind, name string) bool {
	i, ok := f.byKey[key(kind, name, "")]
	return ok && !f.gone[i]
}

func (f *fed) sorted() []protocol.ContextPiece {
	out := make([]protocol.ContextPiece, 0, len(f.list))
	for _, kind := range kindOrder {
		for i, p := range f.list {
			if p.Kind == kind && !f.gone[i] {
				out = append(out, p)
			}
		}
	}
	return out
}

// title names a section by its first heading or tag, or its first words.
func title(text string) string {
	line := strings.TrimSpace(text)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	line = strings.TrimLeft(line, "# ")
	if m := openTag.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	if r := []rune(line); len(r) > 60 {
		line = string(r[:60]) + "…"
	}
	if line == "" {
		return "untitled"
	}
	return line
}

var openTag = regexp.MustCompile(`^<([A-Za-z_][\w-]*)>`)

func pretty(raw []byte) string {
	var out bytes.Buffer
	if json.Indent(&out, raw, "", "  ") != nil {
		return string(raw)
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// claude: attachments in its session file, each with the text it rendered to
// ---------------------------------------------------------------------------

func (f *fed) claude(body string) {
	var doc struct {
		Attachment json.RawMessage `json:"attachment"`
		Rendered   []struct {
			Content json.RawMessage `json:"content"`
		} `json:"rendered"`
	}
	if json.Unmarshal([]byte(body), &doc) != nil {
		return
	}
	// What claude actually put in front of the model for this attachment.
	var rendered []string
	for _, r := range doc.Rendered {
		if s := textOf(r.Content); s != "" {
			rendered = append(rendered, s)
		}
	}
	shown := strings.Join(rendered, "\n\n")

	var a struct {
		Type         string            `json:"type"`
		SystemPrompt []json.RawMessage `json:"systemPrompt"`
		Tools        []json.RawMessage `json:"tools"`
		CliPrefix    string            `json:"cliPrefix"`
		Files        []claudeFile      `json:"files"`
		Path         string            `json:"path"`
		Content      json.RawMessage   `json:"content"`
		IsInitial    bool              `json:"isInitial"`
		AddedNames   []string          `json:"addedNames"`
		AddedBlocks  []string          `json:"addedBlocks"`
		AddedLines   []string          `json:"addedLines"`
		AddedTypes   []string          `json:"addedTypes"`
		RemovedNames []string          `json:"removedNames"`
		RemovedTypes []string          `json:"removedTypes"`
	}
	if json.Unmarshal(doc.Attachment, &a) != nil {
		return
	}

	switch a.Type {
	case "prompt_snapshot":
		// Each snapshot is the whole prompt: it replaces the last, section by
		// section, and its tools replace the last tools when it has any.
		f.clear(KindInstructions)
		if a.CliPrefix != "" {
			f.put(KindInstructions, "opening line", "", a.CliPrefix, nil)
		}
		for i, raw := range a.SystemPrompt {
			text := textOf(raw)
			name := title(text)
			if f.hasName(KindInstructions, name) {
				name = name + " (" + strconv.Itoa(i+1) + ")"
			}
			f.put(KindInstructions, name, "", text, nil)
		}
		if len(a.Tools) > 0 {
			f.clear(KindTool)
			for _, raw := range a.Tools {
				var t struct {
					Name string `json:"name"`
				}
				_ = json.Unmarshal(raw, &t)
				f.put(KindTool, orName(t.Name, "unnamed tool"), "", pretty(raw), nil)
			}
		}

	case "instructions":
		for _, file := range a.Files {
			f.file(file)
		}
	case "nested_memory":
		var file claudeFile
		if json.Unmarshal(a.Content, &file) == nil && file.Content != "" {
			f.file(file)
		} else {
			f.put(KindFile, fileName(a.Path), a.Path, shown, nil)
		}

	case "skill_listing":
		if a.IsInitial {
			f.clear(KindSkill)
		}
		var content string
		_ = json.Unmarshal(a.Content, &content)
		for _, line := range strings.Split(content, "\n") {
			if m := claudeSkill.FindStringSubmatch(line); m != nil {
				f.put(KindSkill, m[1], "", strings.TrimSpace(line), nil)
			}
		}
	case "invoked_skills":
		f.put(KindSkill, "skills loaded in full", "", shown, nil)

	case "mcp_instructions_delta":
		for i, name := range a.AddedNames {
			if i < len(a.AddedBlocks) {
				f.put(KindServer, name, "", a.AddedBlocks[i], nil)
			}
		}
		for _, name := range a.RemovedNames {
			f.remove(KindServer, name)
		}
	case "deferred_tools_delta":
		for i, name := range a.AddedNames {
			text := name
			if i < len(a.AddedLines) && a.AddedLines[i] != "" {
				text = a.AddedLines[i]
			}
			f.put(KindDeferred, name, "", text, nil)
		}
		for _, name := range a.RemovedNames {
			f.remove(KindDeferred, name)
		}
	case "agent_listing_delta":
		for i, name := range a.AddedTypes {
			text := name
			if i < len(a.AddedLines) {
				text = a.AddedLines[i]
			}
			f.put(KindAgent, name, "", text, nil)
		}
		for _, name := range a.RemovedTypes {
			f.remove(KindAgent, name)
		}

	case "environment", "session_context", "model", "date", "command_permissions", "auto_mode":
		f.put(KindEnvironment, strings.ReplaceAll(a.Type, "_", " "), "", orText(shown, doc.Attachment), nil)
	case "hook_additional_context":
		f.put(KindOther, "added by a hook", "", orText(shown, doc.Attachment), nil)
	default:
		f.put(KindOther, a.Type, "", orText(shown, doc.Attachment), nil)
	}
}

type claudeFile struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

func (f *fed) file(file claudeFile) {
	name := fileName(file.Path)
	if file.Type != "" {
		name += " · " + strings.ToLower(file.Type)
	}
	f.put(KindFile, name, file.Path, file.Content, nil)
}

// A skill's name runs to the first ": ". Names can hold a colon of their own:
// a plugin's skill is "plugin:skill".
// fileName is the last part of a path from either kind of computer. This runs
// on the server, which is Linux, and reads paths the owner's Windows computer
// wrote: filepath.Base there does not treat a backslash as a separator, and
// would name "D:\work\CLAUDE.md" by all of it.
func fileName(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}
	return path
}

var claudeSkill = regexp.MustCompile(`^- ([^\s\x60]\S*?):(?:\s|$)`)

// textOf reads a rendered block, which claude writes as a string or as a list
// of text parts.
func textOf(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(p.Text)
		}
		return b.String()
	}
	var part struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &part) == nil {
		return part.Text
	}
	return ""
}

func orText(shown string, raw json.RawMessage) string {
	if shown != "" {
		return shown
	}
	return pretty(raw)
}

func orName(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}


// ---------------------------------------------------------------------------
// codex: its rollout's session header, developer messages and turn settings
// ---------------------------------------------------------------------------

var (
	wholeTag   = regexp.MustCompile(`(?s)^\s*<([A-Za-z_][\w-]*)>(.*)</([A-Za-z_][\w-]*)>\s*$`)
	skillRoot  = regexp.MustCompile("^- `(r\\d+)` = `([^`]+)`\\s*$")
	codexSkill = regexp.MustCompile(`^- ([^\s\x60]\S*?): (.*) \(file: ([^)]+)\)\s*$`)
)

func (f *fed) codex(kind, body string) {
	switch kind {
	case "codex.session_meta":
		var m struct {
			Base struct {
				Text       string `json:"text"`
				Provenance struct {
					Model string `json:"model"`
				} `json:"provenance"`
			} `json:"base_instructions"`
		}
		if json.Unmarshal([]byte(body), &m) == nil && m.Base.Text != "" {
			name := "base instructions"
			if m.Base.Provenance.Model != "" {
				name += " for " + m.Base.Provenance.Model
			}
			f.put(KindInstructions, name, "", m.Base.Text, nil)
		}

	case "codex.message":
		var m struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if json.Unmarshal([]byte(body), &m) != nil {
			return
		}
		// One message holds several blocks, each its own part: skills,
		// plugins, environment. Each is read on its own.
		for _, c := range m.Content {
			if strings.TrimSpace(c.Text) != "" {
				f.codexMessage(c.Text)
			}
		}

	case "codex.world_state":
		var w struct {
			State struct {
				AgentsMD map[string]json.RawMessage `json:"agents_md"`
			} `json:"state"`
		}
		if json.Unmarshal([]byte(body), &w) != nil {
			return
		}
		for path, raw := range w.State.AgentsMD {
			text := textOf(raw)
			if text == "" {
				text = pretty(raw)
			}
			f.put(KindFile, fileName(path), path, text, nil)
		}

	case "codex.turn_context":
		var t map[string]json.RawMessage
		if json.Unmarshal([]byte(body), &t) != nil {
			return
		}
		// The settings the turn ran under, not every id codex keeps.
		keep := map[string]json.RawMessage{}
		for _, k := range []string{"model", "approval_policy", "sandbox_policy", "active_permission_profile", "personality", "collaboration_mode", "cwd", "current_date", "timezone"} {
			if v, ok := t[k]; ok {
				keep[k] = v
			}
		}
		raw, _ := json.Marshal(keep)
		f.put(KindEnvironment, "settings for this turn", "", pretty(raw), nil)
	}
}

func (f *fed) codexMessage(text string) {
	m := wholeTag.FindStringSubmatch(text)
	if m == nil || m[1] != m[3] {
		f.put(KindInstructions, title(text), "", text, nil)
		return
	}
	switch tag := m[1]; tag {
	case "skills_instructions":
		roots := map[string]string{}
		var rest []string
		for _, line := range strings.Split(m[2], "\n") {
			if r := skillRoot.FindStringSubmatch(line); r != nil {
				roots[r[1]] = r[2]
				rest = append(rest, line)
				continue
			}
			if s := codexSkill.FindStringSubmatch(line); s != nil {
				source := s[3]
				if root, tail, ok := strings.Cut(source, "/"); ok && roots[root] != "" {
					source = roots[root] + "/" + tail
				}
				f.put(KindSkill, s[1], source, strings.TrimSpace(line), nil)
				continue
			}
			rest = append(rest, line)
		}
		f.put(KindInstructions, "how codex uses skills", "", strings.TrimSpace(strings.Join(rest, "\n")), nil)
	case "user_instructions", "agents_md":
		f.put(KindFile, "AGENTS.md", "", text, nil)
	case "environment_context":
		f.put(KindEnvironment, "environment", "", text, nil)
	case "recommended_plugins":
		f.put(KindOther, "plugins codex recommends installing", "", text, nil)
	default:
		f.put(KindInstructions, strings.ReplaceAll(tag, "_", " "), "", text, nil)
	}
}

// ---------------------------------------------------------------------------
// agy: the protobuf of one call's input
// ---------------------------------------------------------------------------

// The field numbers were read from agy 1.2.3's records (docs/Capabilities.md):
// the request is field 1; in it, 16 is the system prompt as named sections
// (1 name, 2 text), 8 the tools (1 name, 2 description, 3 JSON schema), and
// 9.10.3.1 agy's own token count per section.
const (
	agyRequest   = 1
	agyPrompt    = 1
	agySection   = 16
	agyTool      = 8
	agyBreakdown = 9
)

func (f *fed) agy(body string) {
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return
	}
	req, ok := pbFirst(raw, agyRequest)
	if !ok {
		f.put(KindOther, "agy's record", "", "(not in the shape agy 1.2.3 wrote)", nil)
		return
	}

	tokens := agyTokens(req)
	sections := pbAll(req, agySection)
	for _, s := range sections {
		name, _ := pbString(s, 1)
		text, _ := pbString(s, 2)
		var tk *int64
		if n, ok := tokens[name]; ok {
			tk = &n
		}
		label := strings.ReplaceAll(name, "_", " ")
		switch name {
		case "user_rules":
			f.put(KindFile, label, "", text, tk)
		case "skills":
			f.put(KindSkill, label, "", text, tk)
		case "mcp_servers":
			f.put(KindServer, label, "", text, tk)
		case "subagents":
			f.put(KindAgent, label, "", text, tk)
		case "user_information":
			f.put(KindEnvironment, label, "", text, tk)
		default:
			f.put(KindInstructions, orName(label, "untitled section"), "", text, tk)
		}
	}
	if len(sections) == 0 {
		if text, ok := pbString(req, agyPrompt); ok {
			f.put(KindInstructions, "system prompt", "", text, nil)
		}
	}

	for _, t := range pbAll(req, agyTool) {
		name, _ := pbString(t, 1)
		desc, _ := pbString(t, 2)
		schema, _ := pbString(t, 3)
		text := desc
		if schema != "" {
			text += "\n\n" + pretty([]byte(schema))
		}
		f.put(KindTool, orName(name, "unnamed tool"), "", text, nil)
	}
}

// agyTokens reads agy's own count of each system-prompt section's tokens.
func agyTokens(req []byte) map[string]int64 {
	out := map[string]int64{}
	b, ok := pbFirst(req, agyBreakdown)
	if !ok {
		return out
	}
	if b, ok = pbFirst(b, 10); !ok {
		return out
	}
	for _, part := range pbAll(b, 3) {
		p, ok := pbFirst(part, 1)
		if !ok {
			continue
		}
		for _, sub := range pbAll(p, 5) {
			name, _ := pbString(sub, 1)
			if n, ok := pbVarint(sub, 3); ok && name != "" {
				out[name] = int64(n)
			}
		}
	}
	return out
}

// A protobuf reader for the wire format alone. agy publishes no schema, so
// there is nothing to generate from; this reads fields by number, which is all
// the above needs.

type pbField struct {
	num    uint64
	wire   uint64
	varint uint64
	bytes  []byte
}

var errPB = errors.New("not protobuf")

func pbVarintAt(b []byte, i int) (uint64, int, error) {
	var v uint64
	for shift := uint(0); shift < 64; shift += 7 {
		if i >= len(b) {
			return 0, 0, errPB
		}
		c := b[i]
		i++
		v |= uint64(c&0x7f) << shift
		if c < 0x80 {
			return v, i, nil
		}
	}
	return 0, 0, errPB
}

func pbFields(b []byte) ([]pbField, error) {
	var out []pbField
	for i := 0; i < len(b); {
		k, n, err := pbVarintAt(b, i)
		if err != nil {
			return nil, err
		}
		i = n
		f := pbField{num: k >> 3, wire: k & 7}
		switch f.wire {
		case 0:
			if f.varint, i, err = pbVarintAt(b, i); err != nil {
				return nil, err
			}
		case 1:
			if i+8 > len(b) {
				return nil, errPB
			}
			i += 8
		case 5:
			if i+4 > len(b) {
				return nil, errPB
			}
			i += 4
		case 2:
			l, n, err := pbVarintAt(b, i)
			if err != nil || n+int(l) > len(b) || int(l) < 0 {
				return nil, errPB
			}
			f.bytes = b[n : n+int(l)]
			i = n + int(l)
		default:
			return nil, errPB
		}
		out = append(out, f)
	}
	return out, nil
}

func pbAll(b []byte, num uint64) [][]byte {
	fields, err := pbFields(b)
	if err != nil {
		return nil
	}
	var out [][]byte
	for _, f := range fields {
		if f.num == num && f.wire == 2 {
			out = append(out, f.bytes)
		}
	}
	return out
}

func pbFirst(b []byte, num uint64) ([]byte, bool) {
	all := pbAll(b, num)
	if len(all) == 0 {
		return nil, false
	}
	return all[0], true
}

func pbString(b []byte, num uint64) (string, bool) {
	v, ok := pbFirst(b, num)
	return string(v), ok
}

func pbVarint(b []byte, num uint64) (uint64, bool) {
	fields, err := pbFields(b)
	if err != nil {
		return 0, false
	}
	for _, f := range fields {
		if f.num == num && f.wire == 0 {
			return f.varint, true
		}
	}
	return 0, false
}
