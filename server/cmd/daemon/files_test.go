package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// fakeServer stands in for the server's file routes. It answers only a
// computer presenting the right credential.
func fakeServer(t *testing.T, files map[string][]byte, got *[]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer machine-cred" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"this machine is not authorised"}`))
			return
		}
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/daemon/files/"):
			b, ok := files[strings.TrimPrefix(r.URL.Path, "/daemon/files/")]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"that file does not exist"}`))
				return
			}
			_, _ = w.Write(b)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/files"):
			name := r.URL.Query().Get("name")
			_, _ = io.ReadAll(r.Body)
			*got = append(*got, name)
			kept := name
			if name == "clash.png" {
				kept = "clash (2).png"
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(protocol.ConversationFile{Name: kept})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testFiles(t *testing.T, srv *httptest.Server) *chatFiles {
	t.Helper()
	home := t.TempDir()
	return &chatFiles{
		chats: filepath.Join(t.TempDir(), "chats"), api: srv.URL,
		token: func() string { return "machine-cred" }, client: srv.Client(),
		home: func() (string, error) { return home, nil }, log: quietLog(),
	}
}

func TestATurnsFilesAreFetchedIntoTheConversationsFolder(t *testing.T) {
	var sent []string
	srv := fakeServer(t, map[string][]byte{"f1": []byte("picture"), "f2": []byte("order,qty\n"), "f3": []byte("made earlier")}, &sent)
	c := testFiles(t, srv)

	turn := protocol.RunTurn{ConversationID: "c1", Provider: "claude", Files: []protocol.TurnFile{
		{ID: "f1", Origin: protocol.FileUpload, Name: "shot.png", Size: 7, MediaType: "image/png", Attached: true},
		{ID: "f2", Origin: protocol.FileUpload, Name: "orders.csv", Size: 10, MediaType: "text/csv"},
		{ID: "f3", Origin: protocol.FileOutput, Name: "made.png", Size: 12, MediaType: "image/png"},
	}}
	p, err := c.prepare(context.Background(), turn)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(c.chats, "c1", "uploads", "shot.png")); string(b) != "picture" {
		t.Errorf("the upload on disk: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(c.chats, "c1", "outputs", "made.png")); string(b) != "made earlier" {
		t.Errorf("the earlier output on disk: %q", b)
	}
	if len(p.Attached) != 1 || p.Attached[0].Name != "shot.png" || !p.Known["made.png"] {
		t.Errorf("prepared: %+v", p)
	}

	// Already there with the right size: not fetched again, even if the
	// server has lost it.
	c.api = "http://127.0.0.1:1"
	if _, err := c.prepare(context.Background(), turn); err != nil {
		t.Errorf("a second turn fetched what it already had: %v", err)
	}
}

func TestAFileSentWithTheMessageThatCannotBeFetchedFailsTheTurn(t *testing.T) {
	var sent []string
	c := testFiles(t, fakeServer(t, map[string][]byte{}, &sent))
	_, err := c.prepare(context.Background(), protocol.RunTurn{ConversationID: "c1", Files: []protocol.TurnFile{
		{ID: "gone", Origin: protocol.FileUpload, Name: "gone.pdf", Size: 3, Attached: true},
	}})
	if err == nil || !strings.Contains(err.Error(), "gone.pdf") {
		t.Fatalf("want a failure naming the file, got %v", err)
	}
	// An older file that cannot be fetched only warns.
	if _, err := c.prepare(context.Background(), protocol.RunTurn{ConversationID: "c1", Files: []protocol.TurnFile{
		{ID: "gone", Origin: protocol.FileUpload, Name: "old.pdf", Size: 3},
	}}); err != nil {
		t.Errorf("an earlier file stopped the turn: %v", err)
	}
	if _, err := c.prepare(context.Background(), protocol.RunTurn{ConversationID: `..\evil`}); err == nil {
		t.Error("a conversation id with a path in it was accepted")
	}
}

func TestEachAgentIsToldWhatItCanUse(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	p := preparedFiles{Folder: dir, Outputs: filepath.Join(dir, "outputs"), Attached: []preparedFile{
		{Path: write("shot.png", "png"), Name: "shot.png", MediaType: "image/png", Size: 3},
		{Path: write("orders.csv", "order,qty\nPO-1,17\n"), Name: "orders.csv", MediaType: "text/csv", Size: 18},
		{Path: write("guide.pdf", "%PDF"), Name: "guide.pdf", MediaType: "application/pdf", Size: 4},
	}}

	claude, images := filesPrompt("claude", p)
	if len(images) != 0 || !strings.Contains(claude, p.Attached[0].Path) || !strings.Contains(claude, p.Attached[2].Path) ||
		!strings.Contains(claude, p.Outputs) {
		t.Errorf("claude was told %q, pictures %v", claude, images)
	}

	codex, images := filesPrompt("codex", p)
	if len(images) != 1 || images[0] != p.Attached[0].Path {
		t.Errorf("codex's pictures: %v", images)
	}
	if !strings.Contains(codex, "PO-1,17") || !strings.Contains(codex, "guide.pdf was sent with this message, but it cannot be opened") {
		t.Errorf("codex was told %q", codex)
	}
	if strings.Contains(codex, p.Outputs) {
		t.Error("codex was told to save where its sandbox cannot write")
	}

	if note, _ := filesPrompt("agy", preparedFiles{}); note != "" {
		t.Errorf("a turn with no folder got %q", note)
	}
}

func TestACatchUpSaysWhichFilesEachMessageCarried(t *testing.T) {
	p := preparedFiles{Folder: filepath.Join("C:", "chats", "c1")}
	turn := protocol.RunTurn{Provider: "agy", Prompt: "and now?", Replay: []protocol.ReplayEntry{
		{Role: "user", Text: "look at this", Files: []protocol.ReplayFile{{Origin: protocol.FileUpload, Name: "shot.png"}}},
		{Role: "agent", Provider: "codex", Text: "here", Files: []protocol.ReplayFile{{Origin: protocol.FileOutput, Name: "flow.png"}}},
	}}
	got, _ := buildPrompt(turn, p)
	if !strings.Contains(got, "(Files sent with this message: "+filepath.Join(p.Folder, "uploads", "shot.png")+")") ||
		!strings.Contains(got, "(Files made in this turn: "+filepath.Join(p.Folder, "outputs", "flow.png")+")") {
		t.Errorf("catch-up: %s", got)
	}
}

func TestWhatTheAgentMadeIsCollectedAndSent(t *testing.T) {
	var sent []string
	c := testFiles(t, fakeServer(t, nil, &sent))
	p, err := c.prepare(context.Background(), protocol.RunTurn{ConversationID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	home, _ := c.home()
	t.Setenv("CODEX_HOME", "")
	gen := filepath.Join(home, ".codex", "generated_images", "thread-1")
	if err := os.MkdirAll(gen, 0o755); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	old := filepath.Join(gen, "exec-old.png")
	_ = os.WriteFile(old, []byte("old"), 0o644)
	_ = os.Chtimes(old, started.Add(-time.Hour), started.Add(-time.Hour))
	_ = os.WriteFile(filepath.Join(gen, "exec-new.png"), []byte("new"), 0o644)
	_ = os.WriteFile(filepath.Join(gen, "notes.txt"), []byte("not a picture"), 0o644)
	_ = os.WriteFile(filepath.Join(p.Outputs, "report.pdf"), []byte("%PDF"), 0o644)
	_ = os.WriteFile(filepath.Join(p.Outputs, "clash.png"), []byte("x"), 0o644)
	p.Known["already.png"] = true
	_ = os.WriteFile(filepath.Join(p.Outputs, "already.png"), []byte("x"), 0o644)

	found := c.collect(p, "codex", "thread-1", started)
	var names []string
	for _, f := range found {
		names = append(names, filepath.Base(f))
	}
	want := "exec-new.png clash.png report.pdf"
	if strings.Join(names, " ") != want {
		t.Fatalf("collected %v, want %s", names, want)
	}
	if _, err := os.Stat(filepath.Join(p.Outputs, "exec-new.png")); err != nil {
		t.Error("the generated picture was not copied into outputs")
	}
	for _, f := range found {
		if err := c.send(context.Background(), "turn-1", f); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(sent, " ") != want {
		t.Errorf("sent %v", sent)
	}
	// Kept under another name by the server: renamed on disk to match.
	if _, err := os.Stat(filepath.Join(p.Outputs, "clash (2).png")); err != nil {
		t.Error("the file was not renamed to the name it was kept as")
	}
}

func TestTheWorkingFolderCannotBeLeft(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "edi", "fabrikam"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("# hi"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644)
	_ = os.WriteFile(filepath.Join(filepath.Dir(root), "secret.txt"), []byte("no"), 0o644)
	big := filepath.Join(root, "big.bin")
	if f, err := os.Create(big); err == nil {
		_ = f.Truncate(protocol.MaxFolderFileBytes + 1)
		_ = f.Close()
	}

	l := folderListing(root, "")
	var names []string
	for _, e := range l.Entries {
		names = append(names, e.Kind+":"+e.Name)
	}
	if l.Error != "" || strings.Join(names, " ") != "dir:edi file:b.txt file:big.bin file:README.md" {
		t.Errorf("listing: %v %q", names, l.Error)
	}
	if l := folderListing(root, "edi/"); l.Path != "edi" || len(l.Entries) != 1 {
		t.Errorf("a subfolder: %+v", l)
	}

	if f := folderFile(root, "README.md"); f.Error != "" || f.Content != base64.StdEncoding.EncodeToString([]byte("# hi")) {
		t.Errorf("reading a file: %+v", f)
	}
	if f := folderFile(root, "big.bin"); !f.TooLarge || f.Content != "" {
		t.Errorf("a file too large to show: %+v", f)
	}
	for _, escape := range []string{"../secret.txt", `..\secret.txt`, "edi/../../secret.txt", filepath.Join(filepath.Dir(root), "secret.txt"), "C:/Windows/win.ini"} {
		if f := folderFile(root, escape); f.Error == "" || f.Content != "" {
			t.Errorf("%q was read: %+v", escape, f)
		}
	}
	// ".." that stays inside is fine; it is only leaving that is refused.
	if f := folderFile(root, "edi/../b.txt"); f.Error != "" {
		t.Errorf("a path that stays inside was refused: %q", f.Error)
	}
	if l := folderListing("", ""); l.Error == "" {
		t.Error("a listing with no root was answered")
	}
}
