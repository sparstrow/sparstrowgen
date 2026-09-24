package sessionstore

import (
	"database/sql"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeHome points the readers at a folder of the test's own, so no test ever
// reads the stores of the computer it runs on.
func fakeHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	original := home
	home = func() (string, error) { return dir, nil }
	t.Cleanup(func() { home = original })
	t.Setenv("CODEX_HOME", "")
	return dir
}

func place(t *testing.T, path, fixture string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "agent", "testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func kinds(docs []string) map[string]int {
	out := map[string]int{}
	for _, d := range docs {
		out[d]++
	}
	return out
}

func TestClaudesSessionIsFoundByItsIdWhateverItsFolderIsCalled(t *testing.T) {
	h := fakeHome(t)
	place(t, filepath.Join(h, ".claude", "projects", "D--some-long-encoded-folder", "abc-123.jsonl"), "claude-session-context.jsonl")

	rec := Read("claude", "abc-123")
	if rec.Error != "" {
		t.Fatalf("error = %s", rec.Error)
	}
	if !strings.HasSuffix(rec.From, "abc-123.jsonl") {
		t.Errorf("from = %s", rec.From)
	}
	var ks []string
	snapshots := 0
	for _, d := range rec.Documents {
		ks = append(ks, d.Kind)
		if strings.Contains(d.Body, `"type":"prompt_snapshot"`) {
			snapshots++
		}
		// Only the attachment and its rendering: ids and times around it would
		// make every turn's copy of the same record look new.
		if strings.Contains(d.Body, `"parentUuid"`) || strings.Contains(d.Body, `"timestamp"`) {
			t.Errorf("a record kept its line's ids: %.80s", d.Body)
		}
	}
	// Two snapshots were recorded, one per call; only the last is kept, and it
	// is the one with tools.
	if snapshots != 1 {
		t.Errorf("snapshots kept = %d, want 1", snapshots)
	}
	// Messages, and attachments that are not context (the token reminder), are
	// left behind.
	if n := kinds(ks)["claude.attachment"]; n != len(rec.Documents) || n < 8 {
		t.Errorf("documents = %v", kinds(ks))
	}
	for _, d := range rec.Documents {
		if strings.Contains(d.Body, "total_tokens_reminder") {
			t.Error("the token reminder was kept")
		}
	}
}

func TestCodexsRolloutIsFoundInItsDateFoldersUnderCodexHome(t *testing.T) {
	h := fakeHome(t)
	codexHome := filepath.Join(h, "somewhere-else")
	t.Setenv("CODEX_HOME", codexHome)
	place(t, filepath.Join(codexHome, "sessions", "2026", "09", "23", "rollout-2026-09-23T23-12-34-thread-9.jsonl"), "codex-rollout-context.jsonl")

	rec := Read("codex", "thread-9")
	if rec.Error != "" {
		t.Fatalf("error = %s", rec.Error)
	}
	var ks []string
	for _, d := range rec.Documents {
		ks = append(ks, d.Kind)
	}
	k := kinds(ks)
	if k["codex.session_meta"] != 1 || k["codex.world_state"] != 1 || k["codex.turn_context"] != 1 {
		t.Errorf("documents = %v", k)
	}
	if ks[0] != "codex.session_meta" {
		t.Errorf("first = %s, want the session header", ks[0])
	}
	// Four context messages: three of codex's own and one it wrapped in tags.
	// The person's message and the assistant's answer are not context.
	if k["codex.message"] != 4 {
		t.Errorf("messages = %d, want 4", k["codex.message"])
	}
	for _, d := range rec.Documents {
		if strings.Contains(d.Body, "You are joining a conversation") {
			t.Error("the prompt the person sent was taken as context")
		}
	}
}

func TestAgysDatabaseIsCopiedAndItsLatestInputRead(t *testing.T) {
	h := fakeHome(t)
	dir := filepath.Join(h, ".gemini", "antigravity-cli", "conversations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "conv-1.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	big := []byte(strings.Repeat("x", 5000))
	for _, stmt := range []string{
		"PRAGMA journal_mode=WAL",
		"CREATE TABLE gen_metadata (idx integer PRIMARY KEY, data blob, size integer NOT NULL DEFAULT 0)",
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	// Bookkeeping first, then two calls to the model; the last is wanted.
	if _, err := db.Exec("INSERT INTO gen_metadata (idx, data) VALUES (0, ?), (1, ?), (2, ?)",
		[]byte("small"), append([]byte("first "), big...), append([]byte("latest "), big...)); err != nil {
		t.Fatal(err)
	}
	// Left open, as agy leaves it: the rows are in the WAL file, not the
	// database file, and the copy has to carry them.
	defer db.Close()

	rec := Read("agy", "conv-1")
	if rec.Error != "" {
		t.Fatalf("error = %s", rec.Error)
	}
	if len(rec.Documents) != 1 || rec.Documents[0].Kind != "agy.gen_metadata" {
		t.Fatalf("documents = %d", len(rec.Documents))
	}
	raw, _ := base64.StdEncoding.DecodeString(rec.Documents[0].Body)
	if !strings.HasPrefix(string(raw), "latest ") {
		t.Errorf("read %.12q, want the latest call", raw)
	}
}

func TestAMissingStoreIsSaidNotSilent(t *testing.T) {
	fakeHome(t)
	for provider, want := range map[string]string{
		"claude": "was not found",
		"codex":  "was not found",
		"agy":    "was not found",
		"gemini": "does not know where gemini keeps",
	} {
		rec := Read(provider, "nope")
		if !strings.Contains(rec.Error, want) || len(rec.Documents) != 0 {
			t.Errorf("%s: %+v", provider, rec)
		}
	}
	if rec := Read("claude", ""); !strings.Contains(rec.Error, "never said which session") {
		t.Errorf("no session: %+v", rec)
	}
}
