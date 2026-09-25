package api

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// A PNG's first bytes, enough for the server to know it is one.
var pngBytes = append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), bytes.Repeat([]byte{0}, 64)...)

func (r *rig) upload(conversationID, name string, body []byte) (*http.Response, protocol.ConversationFile) {
	r.t.Helper()
	req, err := http.NewRequest(http.MethodPost, r.http.URL+"/api/conversations/"+conversationID+"/files?name="+url.QueryEscape(name), bytes.NewReader(body))
	if err != nil {
		r.t.Fatal(err)
	}
	req.Header.Set("Origin", testOrigin)
	res, err := r.client.Do(req)
	if err != nil {
		r.t.Fatal(err)
	}
	r.t.Cleanup(func() { _ = res.Body.Close() })
	var f protocol.ConversationFile
	if res.StatusCode == http.StatusCreated {
		decodeInto(r.t, res, &f)
	}
	return res, f
}

func (r *rig) files(conversationID string) filesResponse {
	r.t.Helper()
	res := r.get("/api/conversations/" + conversationID + "/files")
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("list files: %s", res.Status)
	}
	var out filesResponse
	decodeInto(r.t, res, &out)
	return out
}

// daemonHTTP is a request made the way a computer makes one: its credential,
// no browser session.
func (r *rig) daemonHTTP(method, path, token string, body []byte) *http.Response {
	r.t.Helper()
	req, err := http.NewRequest(method, r.http.URL+path, bytes.NewReader(body))
	if err != nil {
		r.t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		r.t.Fatal(err)
	}
	r.t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func TestAnUploadIsKeptAndSentWithTheMessage(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")

	res, f := r.upload(c.ID, "nav-export-error.png", pngBytes)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("upload: %s", res.Status)
	}
	if f.Name != "nav-export-error.png" || f.MediaType != "image/png" || f.Origin != protocol.FileUpload || f.EntryID != "" || f.Size != int64(len(pngBytes)) {
		t.Fatalf("the upload came back as %+v", f)
	}
	if got := r.files(c.ID).Files; len(got) != 1 || got[0].ID != f.ID {
		t.Fatalf("the conversation lists %+v", got)
	}

	res = r.post("/api/conversations/"+c.ID+"/messages", map[string]any{
		"text": "What does this say?", "fileIds": []string{f.ID},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("send: %s", res.Status)
	}
	turn := d.nextTurn()
	if len(turn.Files) != 1 || turn.Files[0].ID != f.ID || !turn.Files[0].Attached || turn.Files[0].Sha256 == "" || turn.Files[0].MediaType != "image/png" {
		t.Fatalf("the turn carried %+v", turn.Files)
	}

	// The message now owns the file, and says so after a refresh.
	res = r.get("/api/conversations/" + c.ID)
	var conv protocol.Conversation
	decodeInto(t, res, &conv)
	var user *protocol.Entry
	for i := range conv.Entries {
		if conv.Entries[i].Role == "user" {
			user = &conv.Entries[i]
		}
	}
	if user == nil || len(user.Files) != 1 || user.Files[0].ID != f.ID || user.Files[0].EntryID != user.ID {
		t.Fatalf("the message's files after a refresh: %+v", user)
	}

	res = r.get("/api/files/" + f.ID + "/content")
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !bytes.Equal(body, pngBytes) {
		t.Fatalf("content: %s, %d bytes", res.Status, len(body))
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" || res.Header.Get("Content-Type") != "image/png" ||
		!strings.Contains(res.Header.Get("Content-Security-Policy"), "sandbox") {
		t.Errorf("served with %v", res.Header)
	}
}

func TestTwoFilesWithOneNameBothKeepTheirs(t *testing.T) {
	r := newRig(t)
	c := r.conversation("claude")
	_, a := r.upload(c.ID, "orders.csv", []byte("order,qty\n"))
	_, b := r.upload(c.ID, "Orders.CSV", []byte("order,qty\n1,2\n"))
	_, p := r.upload(c.ID, `C:\Users\someone\..\evil/../orders.csv`, []byte("x"))
	if a.Name != "orders.csv" || b.Name != "Orders (2).CSV" || p.Name != "orders (3).csv" {
		t.Errorf("names: %q, %q, %q", a.Name, b.Name, p.Name)
	}
}

func TestAnythingABrowserWouldRunIsKeptAsText(t *testing.T) {
	r := newRig(t)
	c := r.conversation("claude")
	for _, name := range []string{"page.html", "logo.svg", "run.js"} {
		_, f := r.upload(c.ID, name, []byte("<html><script>alert(1)</script></html>"))
		if f.MediaType != "text/plain" {
			t.Errorf("%s was kept as %q", name, f.MediaType)
		}
	}
}

func TestAnUploadOverTheLimitIsRefused(t *testing.T) {
	r := newRig(t)
	c := r.conversation("claude")
	res, _ := r.upload(c.ID, "scan.tiff", make([]byte, protocol.MaxFileBytes+1))
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("an oversized upload: %s, want 413", res.Status)
	}
	if got := r.files(c.ID).Files; len(got) != 0 {
		t.Errorf("it was kept anyway: %+v", got)
	}
}

func TestAnotherAccountCannotReachAFile(t *testing.T) {
	r := newRig(t)
	c := r.conversation("claude")
	_, f := r.upload(c.ID, "private.txt", []byte("mine"))

	other := r.secondAccount()
	if res := other.get("/api/files/" + f.ID + "/content"); res.StatusCode != http.StatusNotFound {
		t.Errorf("another account reading it: %s, want 404", res.Status)
	}
	if res := other.do(http.MethodDelete, "/api/files/"+f.ID, nil); res.StatusCode != http.StatusNotFound {
		t.Errorf("another account deleting it: %s, want 404", res.Status)
	}
	if res, _ := other.upload(c.ID, "theirs.txt", []byte("x")); res.StatusCode != http.StatusNotFound {
		t.Errorf("another account uploading into it: %s, want 404", res.Status)
	}
	if res := r.daemonHTTP(http.MethodGet, "/daemon/files/"+f.ID, "", nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("fetching it with no credential: %s, want 401", res.Status)
	}
}

func TestAnUnsentUploadCanBeRemovedButASentOneCannot(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")
	_, keep := r.upload(c.ID, "keep.txt", []byte("keep"))
	_, drop := r.upload(c.ID, "drop.txt", []byte("drop"))

	if res := r.do(http.MethodDelete, "/api/files/"+drop.ID, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("removing an unsent upload: %s", res.Status)
	}
	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{"text": "here", "fileIds": []string{keep.ID, drop.ID}})
	turn := d.nextTurn()
	if len(turn.Files) != 1 || turn.Files[0].ID != keep.ID {
		t.Fatalf("the turn carried %+v", turn.Files)
	}
	if res := r.do(http.MethodDelete, "/api/files/"+keep.ID, nil); res.StatusCode != http.StatusNotFound {
		t.Errorf("removing a sent file: %s, want 404", res.Status)
	}
}

func TestAComputerFetchesWhatATurnNeedsAndSendsBackWhatTheAgentMade(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	b := r.watch()
	c := r.conversation("codex")
	_, f := r.upload(c.ID, "evidence.png", pngBytes)
	r.post("/api/conversations/"+c.ID+"/messages", map[string]any{"text": "draw it", "fileIds": []string{f.ID}})
	turn := d.nextTurn()

	res := r.daemonHTTP(http.MethodGet, "/daemon/files/"+f.ID, testDaemonToken, nil)
	got, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !bytes.Equal(got, pngBytes) {
		t.Fatalf("the computer fetching the upload: %s, %d bytes", res.Status, len(got))
	}

	res = r.daemonHTTP(http.MethodPost, "/daemon/turns/"+turn.TurnID+"/files?name=blue_square.png", testDaemonToken, pngBytes)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("the computer sending an output: %s", res.Status)
	}
	ev := b.await("the output reaching the browser", func(ev protocol.ClientEvent) bool {
		return ev.Type == protocol.EventFiles && ev.EntryID == turn.EntryID
	})
	if len(ev.Files) != 1 || ev.Files[0].Origin != protocol.FileOutput || ev.Files[0].Name != "blue_square.png" {
		t.Fatalf("the event carried %+v", ev.Files)
	}

	d.send(protocol.DaemonMessage{Type: protocol.DaemonDone, TurnID: turn.TurnID, Full: "Here it is."})
	r.awaitEntry(c.ID, turn.EntryID, func(e protocol.Entry) bool { return e.Text == "Here it is." })
	res = r.get("/api/conversations/" + c.ID)
	var conv protocol.Conversation
	decodeInto(t, res, &conv)
	for _, e := range conv.Entries {
		if e.ID == turn.EntryID && (len(e.Files) != 1 || e.Files[0].Name != "blue_square.png") {
			t.Errorf("the answer's files after a refresh: %+v", e.Files)
		}
	}

	// A finished turn takes nothing more.
	res = r.daemonHTTP(http.MethodPost, "/daemon/turns/"+turn.TurnID+"/files?name=late.png", testDaemonToken, pngBytes)
	if res.StatusCode != http.StatusConflict {
		t.Errorf("an output for a finished turn: %s, want 409", res.Status)
	}
}

func TestFilesAreNotSentToAComputerTooOldForThem(t *testing.T) {
	r := newRig(t)
	old := currentDaemon
	old.Protocol = protocol.FilesProtocol - 1
	id, _, inbox := r.updatingComputer("OLD-PC", old)
	r.awaitMachineUpdates(id, func(m machineUpdatesSeen) bool { return m.Version == "0.2.0" })

	c := r.conversation("claude")
	_, f := r.upload(c.ID, "evidence.png", pngBytes)
	res := r.post("/api/conversations/"+c.ID+"/messages", map[string]any{"text": "look", "fileIds": []string{f.ID}})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("files to a too-old computer: %s, want 409", res.Status)
	}
	var body struct {
		Error string `json:"error"`
	}
	decodeInto(t, res, &body)
	if !strings.Contains(body.Error, "too old") || !strings.Contains(body.Error, "Settings → Updates") {
		t.Errorf("the refusal does not say what to do: %q", body.Error)
	}
	if res := r.get("/api/conversations/" + c.ID + "/folder"); res.StatusCode != http.StatusConflict {
		t.Errorf("browsing the folder on a too-old computer: %s, want 409", res.Status)
	}

	// A message without files still goes: nothing about it needs the update.
	if res := r.post("/api/conversations/"+c.ID+"/messages", map[string]any{"text": "plain"}); res.StatusCode != http.StatusOK {
		t.Errorf("a plain message to the same computer: %s", res.Status)
	}
	awaitServerMessage(t, inbox, func(m protocol.ServerMessage) bool { return m.Type == protocol.ServerRunTurn })
}

func TestTheWorkingFolderIsReadFromTheComputer(t *testing.T) {
	r := newRig(t)
	d := r.connectDaemon()
	c := r.conversation("claude")

	// The computer's side: answer whatever the server asks.
	go func() {
		for msg := range d.work {
			switch msg.Type {
			case protocol.ServerListFolder:
				d.send(protocol.DaemonMessage{Type: protocol.DaemonFolderListing, RequestID: msg.RequestID, Folder: &protocol.FolderListing{
					Path: msg.Path, Entries: []protocol.FolderEntry{{Name: msg.Root, Kind: "dir"}},
				}})
			case protocol.ServerReadFile:
				f := &protocol.FolderFile{Path: msg.Path, Size: 5}
				if msg.Path == "big.bin" {
					f.Size, f.TooLarge = 9<<20, true
				} else {
					f.Content = base64.StdEncoding.EncodeToString([]byte("hello"))
				}
				d.send(protocol.DaemonMessage{Type: protocol.DaemonFolderFile, RequestID: msg.RequestID, File: f})
			}
		}
	}()

	res := r.get("/api/conversations/" + c.ID + "/folder?path=edi-mappings")
	var listing protocol.FolderListing
	decodeInto(t, res, &listing)
	// The root is the conversation's folder, decided by the server, never the
	// browser.
	if listing.Path != "edi-mappings" || len(listing.Entries) != 1 || listing.Entries[0].Name != c.Folder {
		t.Fatalf("listing: %+v", listing)
	}

	res = r.get("/api/conversations/" + c.ID + "/folder/file?path=notes.md")
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || string(body) != "hello" || res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("file: %s %q", res.Status, body)
	}
	if res := r.get("/api/conversations/" + c.ID + "/folder/file?path=big.bin"); res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("a file too large to show: %s, want 413", res.Status)
	}
}

func TestTheFolderIsUnreachableWithoutAComputer(t *testing.T) {
	r := newRig(t)
	c := r.conversation("claude")
	if res := r.get("/api/conversations/" + c.ID + "/folder"); res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("browsing with no computer: %s, want 503", res.Status)
	}
	// The chat's own files do not need the computer.
	_, f := r.upload(c.ID, "notes.txt", []byte("n"))
	if got := r.files(c.ID); len(got.Files) != 1 || got.Files[0].ID != f.ID || got.Folder != "" {
		t.Errorf("files with no computer: %+v", got)
	}

}
