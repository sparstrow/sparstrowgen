// Package sessionstore reads what an agent CLI keeps, in its own files on this
// computer, about the context it fed its model: its instructions, tools,
// skills and rules (docs/Capabilities.md, "What each agent loaded, word for
// word").
//
// None of the three CLIs prints any of this. All three write it down, each in
// its own private format, so this is reading files that are not an interface:
// any CLI update may move or reshape them. Everything here therefore fails into
// a sentence saying what could not be read, never into silence.
//
// It selects and does not interpret. The records are handed on as found, and
// the server reads them into pieces (agent.FedContext), so a better reading
// improves turns already recorded. Daemon only: it is the process on the
// computer where the files are.
package sessionstore

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // agy keeps each conversation as a SQLite database

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// home is where the CLIs keep their stores. A variable so tests can point it
// at a folder of their own.
var home = os.UserHomeDir

// budget bounds one turn's records. A long claude session repeats little, but
// a session file is not ours to trust with a size; past this the record says
// it stopped rather than growing without end.
const budget = 8 << 20

// Read returns the context records for one session, read just after one of its
// turns ended. Never fails outright: what went wrong is in Error.
func Read(provider, sessionID string) (rec protocol.ContextRecord) {
	defer func() {
		// A malformed file must cost the record, never the turn.
		if r := recover(); r != nil {
			rec = protocol.ContextRecord{Error: fmt.Sprintf("reading %s's own record failed: %v", provider, r)}
		}
	}()
	if sessionID == "" {
		return protocol.ContextRecord{Error: provider + " never said which session it used, so its own record could not be found"}
	}
	switch provider {
	case "claude":
		return claude(sessionID)
	case "codex":
		return codex(sessionID)
	case "agy":
		return agy(sessionID)
	default:
		return protocol.ContextRecord{Error: "this sparstrowgen does not know where " + provider + " keeps its own record"}
	}
}

// collector keeps documents in the order found, once each, within the budget.
type collector struct {
	docs  []protocol.ContextDocument
	seen  map[[32]byte]bool
	size  int
	full  bool
	label string
}

func newCollector(label string) *collector {
	return &collector{seen: map[[32]byte]bool{}, label: label}
}

func (c *collector) add(kind, body string) {
	sum := sha256.Sum256([]byte(kind + "\x00" + body))
	if c.seen[sum] {
		return
	}
	if c.size+len(body) > budget {
		c.full = true
		return
	}
	c.seen[sum] = true
	c.size += len(body)
	c.docs = append(c.docs, protocol.ContextDocument{Kind: kind, Body: body})
}

func (c *collector) record(from string) protocol.ContextRecord {
	rec := protocol.ContextRecord{From: from, Documents: c.docs}
	if c.full {
		rec.Error = fmt.Sprintf("%s's own record was larger than %d MB; the rest was not kept", c.label, budget>>20)
	}
	return rec
}

// eachLine reads a JSON-lines file of any line length. A claude prompt snapshot
// is one line of well over 100 KB.
func eachLine(path string, fn func([]byte)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			fn(line)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// ---------------------------------------------------------------------------
// claude
// ---------------------------------------------------------------------------

// claudeKinds are the attachments that are context: what claude adds to the
// model's input by itself. Left out on purpose: files the agent read or edited
// (the turn's own lines already carry them), the running token reminder
// (thousands per session, a number each time), and the desktop app's own.
var claudeKinds = map[string]bool{
	"prompt_snapshot": true, "instructions": true, "nested_memory": true,
	"skill_listing": true, "invoked_skills": true, "mcp_instructions_delta": true,
	"deferred_tools_delta": true, "agent_listing_delta": true,
	"environment": true, "session_context": true, "model": true, "date": true,
	"hook_additional_context": true, "command_permissions": true, "auto_mode": true,
}

// claude reads ~/.claude/projects/<folder>/<session>.jsonl, found by the
// session id alone: the folder's name is an encoding of the path that claude
// shortens for long paths, and a session id is unique on its own.
func claude(sessionID string) protocol.ContextRecord {
	h, err := home()
	if err != nil {
		return protocol.ContextRecord{Error: "could not find this computer's home folder: " + err.Error()}
	}
	root := filepath.Join(h, ".claude", "projects")
	matches, _ := filepath.Glob(filepath.Join(root, "*", sessionID+".jsonl"))
	if len(matches) == 0 {
		return protocol.ContextRecord{Error: fmt.Sprintf("claude's record of session %s was not found in %s", sessionID, root)}
	}
	path := matches[0]

	type line struct {
		Type       string          `json:"type"`
		Attachment json.RawMessage `json:"attachment"`
		Rendered   json.RawMessage `json:"rendered"`
	}
	type kept struct {
		kind, body string
		snapshot   bool
		tools      bool
	}
	var found []kept
	err = eachLine(path, func(b []byte) {
		// Only attachment lines are worth decoding; most of a long session is
		// messages. The key can sit anywhere in the line.
		if !bytes.Contains(b, []byte(`"attachment"`)) {
			return
		}
		var l line
		if json.Unmarshal(b, &l) != nil || l.Type != "attachment" {
			return
		}
		var a struct {
			Type  string          `json:"type"`
			Tools json.RawMessage `json:"tools"`
		}
		if json.Unmarshal(l.Attachment, &a) != nil || !claudeKinds[a.Type] {
			return
		}
		// Only the attachment and what it rendered to: the line around it holds
		// ids and times that differ every turn and would defeat storing a
		// repeated record once.
		body, _ := json.Marshal(map[string]json.RawMessage{"attachment": l.Attachment, "rendered": orNull(l.Rendered)})
		found = append(found, kept{kind: "claude.attachment", body: string(body),
			snapshot: a.Type == "prompt_snapshot", tools: len(a.Tools) > 2})
	})
	if err != nil {
		return protocol.ContextRecord{From: path, Error: "claude's record could not be read: " + err.Error()}
	}

	// A snapshot is taken before every call to the model and each is whole, so
	// only the last matters, and the last that carried tools if that was an
	// earlier one.
	last, lastTools := -1, -1
	for i, k := range found {
		if k.snapshot {
			last = i
			if k.tools {
				lastTools = i
			}
		}
	}
	c := newCollector("claude")
	for i, k := range found {
		if k.snapshot && i != last && i != lastTools {
			continue
		}
		c.add(k.kind, k.body)
	}
	if len(c.docs) == 0 {
		return protocol.ContextRecord{From: path, Error: "claude's record held none of the context it usually keeps; its format may have changed"}
	}
	return c.record(path)
}

func orNull(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("null")
	}
	return raw
}

// ---------------------------------------------------------------------------
// codex
// ---------------------------------------------------------------------------

// codex reads $CODEX_HOME/sessions/YYYY/MM/DD/rollout-…-<thread>.jsonl. A
// resumed thread appends to the same file, so reading it after a turn covers
// every turn so far.
func codex(threadID string) protocol.ContextRecord {
	root := os.Getenv("CODEX_HOME")
	if root == "" {
		h, err := home()
		if err != nil {
			return protocol.ContextRecord{Error: "could not find this computer's home folder: " + err.Error()}
		}
		root = filepath.Join(h, ".codex")
	}
	sessions := filepath.Join(root, "sessions")
	matches, _ := filepath.Glob(filepath.Join(sessions, "*", "*", "*", "rollout-*-"+threadID+".jsonl"))
	if len(matches) == 0 {
		return protocol.ContextRecord{Error: fmt.Sprintf("codex's record of thread %s was not found in %s", threadID, sessions)}
	}
	path := matches[0]

	c := newCollector("codex")
	var meta, world, turn string
	err := eachLine(path, func(b []byte) {
		var l struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(b, &l) != nil {
			return
		}
		switch l.Type {
		case "session_meta":
			meta = string(l.Payload)
		case "world_state":
			world = string(l.Payload)
		case "turn_context":
			turn = string(l.Payload)
		case "response_item":
			var m struct {
				Type    string `json:"type"`
				Role    string `json:"role"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			}
			if json.Unmarshal(l.Payload, &m) != nil || m.Type != "message" {
				return
			}
			// Developer messages are codex's own. A user message is context
			// when codex wrapped it in a tag of its own (<recommended_plugins>,
			// <environment_context>); the person's words are not.
			context := m.Role == "developer"
			if m.Role == "user" && len(m.Content) > 0 && strings.HasPrefix(m.Content[0].Text, "<") {
				context = true
			}
			if context {
				c.add("codex.message", string(l.Payload))
			}
		}
	})
	if err != nil {
		return protocol.ContextRecord{From: path, Error: "codex's record could not be read: " + err.Error()}
	}
	if meta == "" && len(c.docs) == 0 {
		return protocol.ContextRecord{From: path, Error: "codex's record held none of the context it usually keeps; its format may have changed"}
	}
	// The latest of each, ahead of the messages: they describe the session as
	// it stood at the end of this turn.
	head := newCollector("codex")
	for kind, body := range map[string]string{"codex.session_meta": meta, "codex.world_state": world, "codex.turn_context": turn} {
		if body != "" {
			head.add(kind, body)
		}
	}
	sortDocs(head.docs)
	for _, d := range c.docs {
		head.add(d.Kind, d.Body)
	}
	head.full = head.full || c.full
	return head.record(path)
}

// sortDocs puts the session-wide records in a fixed order, so the same session
// read twice produces the same record.
func sortDocs(docs []protocol.ContextDocument) {
	order := map[string]int{"codex.session_meta": 0, "codex.world_state": 1, "codex.turn_context": 2}
	for i := 1; i < len(docs); i++ {
		for j := i; j > 0 && order[docs[j].Kind] < order[docs[j-1].Kind]; j-- {
			docs[j], docs[j-1] = docs[j-1], docs[j]
		}
	}
}

// ---------------------------------------------------------------------------
// agy
// ---------------------------------------------------------------------------

// agy reads ~/.gemini/antigravity-cli/conversations/<conversation>.db. Its
// gen_metadata table holds, per call to the model, the whole input as a
// protobuf: the system prompt in named sections, every tool, the conversation.
// The latest call is the turn's fullest.
//
// The database is copied first and the copy opened. agy owns the file, keeps
// it in WAL mode, and a reader must never be the reason agy finds it locked.
func agy(conversationID string) protocol.ContextRecord {
	h, err := home()
	if err != nil {
		return protocol.ContextRecord{Error: "could not find this computer's home folder: " + err.Error()}
	}
	dir := filepath.Join(h, ".gemini", "antigravity-cli", "conversations")
	path := filepath.Join(dir, conversationID+".db")
	if _, err := os.Stat(path); err != nil {
		return protocol.ContextRecord{Error: fmt.Sprintf("agy's record of conversation %s was not found in %s", conversationID, dir)}
	}

	tmp, err := os.MkdirTemp("", "sparstrowgen-agy-context-*")
	if err != nil {
		return protocol.ContextRecord{From: path, Error: "could not make a place to copy agy's record: " + err.Error()}
	}
	defer os.RemoveAll(tmp)
	copyPath := filepath.Join(tmp, "conversation.db")
	for _, suffix := range []string{"", "-wal"} {
		if err := copyFile(path+suffix, copyPath+suffix); err != nil && !(suffix != "" && errors.Is(err, os.ErrNotExist)) {
			return protocol.ContextRecord{From: path, Error: "agy's record could not be copied: " + err.Error()}
		}
	}

	db, err := sql.Open("sqlite", copyPath)
	if err != nil {
		return protocol.ContextRecord{From: path, Error: "agy's record could not be opened: " + err.Error()}
	}
	defer db.Close()
	rows, err := db.Query(`SELECT data FROM gen_metadata ORDER BY idx DESC`)
	if err != nil {
		return protocol.ContextRecord{From: path, Error: "agy's record is not in the shape it had: " + err.Error()}
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return protocol.ContextRecord{From: path, Error: "agy's record could not be read: " + err.Error()}
		}
		// The first rows of a conversation are bookkeeping of a kilobyte or
		// so; a model's input is tens of kilobytes.
		if len(data) < 4096 {
			continue
		}
		c := newCollector("agy")
		c.add("agy.gen_metadata", base64.StdEncoding.EncodeToString(data))
		return c.record(path)
	}
	return protocol.ContextRecord{From: path, Error: "agy's record holds no input to the model yet"}
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
