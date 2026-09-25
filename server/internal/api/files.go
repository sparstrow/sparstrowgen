package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

// Files in a conversation (docs/specs/2026-09-11-files-into-a-conversation.md,
// D-056). The browser uploads and reads them here; a computer fetches the ones
// a turn needs and sends back what an agent made, over HTTP with its own
// credential, so a 25 MB file never sits in the websocket a turn streams on.

var (
	errFilesNeedUpdate  = errors.New("your computer's sparstrowgen is too old for files. Update it in Settings → Updates, then send again")
	errFolderNeedUpdate = errors.New("your computer's sparstrowgen is too old to show files. Update it in Settings → Updates")
	errNoFolder         = errors.New("this conversation has no working folder")
)

func (a *API) failFile(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrFileNotFound):
		a.fail(w, err, http.StatusNotFound)
	case errors.Is(err, store.ErrFileTooLarge):
		a.fail(w, err, http.StatusRequestEntityTooLarge)
	default:
		a.failConversation(w, err)
	}
}

// readFileBody reads an upload's bytes, refusing one over the limit without
// reading the rest of it.
func readFileBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, protocol.MaxFileBytes)
	b, err := io.ReadAll(r.Body)
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return nil, store.ErrFileTooLarge
	}
	return b, err
}

type filesResponse struct {
	Files []protocol.ConversationFile `json:"files"`
	// The conversation's folder on the computer its work goes to, holding
	// uploads and outputs. Empty when that computer is offline or too old to
	// say.
	Folder string `json:"folder"`
}

// listFiles is a conversation's files, newest first.
func (a *API) listFiles(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	id := chi.URLParam(r, "id")
	conv, err := a.store.Get(r.Context(), user.ID, id)
	if err != nil {
		a.failConversation(w, err)
		return
	}
	files, err := a.store.Files(r.Context(), id)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	out := filesResponse{Files: store.Public(files)}
	if allowed, err := a.machinesIn(r, conv.WorkspaceID); err == nil {
		if target, ok := a.hub.Target(user.ID, allowed); ok {
			if dir := a.hub.ChatsDir(user.ID, target.MachineID); dir != "" {
				out.Folder = chatFolder(dir, id)
			}
		}
	}
	writeJSON(w, out)
}

// chatFolder joins with the separator the computer's own path uses, since the
// server may not run on the same kind of system.
func chatFolder(chatsDir, conversationID string) string {
	sep := "/"
	if strings.Contains(chatsDir, `\`) {
		sep = `\`
	}
	return strings.TrimRight(chatsDir, `\/`) + sep + conversationID
}

// uploadFile keeps one file for the message being written. The body is the
// bytes; the name is in the query.
func (a *API) uploadFile(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	id := chi.URLParam(r, "id")
	if _, err := a.store.Get(r.Context(), user.ID, id); err != nil {
		a.failConversation(w, err)
		return
	}
	name := r.URL.Query().Get("name")
	if strings.TrimSpace(name) == "" {
		a.fail(w, errors.New("the file needs a name"), http.StatusBadRequest)
		return
	}
	body, err := readFileBody(w, r)
	if err != nil {
		a.failFile(w, err)
		return
	}
	f, err := a.store.AddFile(r.Context(), id, "", protocol.FileUpload, name, body)
	if err != nil {
		a.failFile(w, err)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventFiles, ConversationID: id})
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, f.ConversationFile)
}

// deleteFile takes an upload back out of the message box.
func (a *API) deleteFile(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	f, err := a.store.DeleteUnsentFile(r.Context(), user.ID, chi.URLParam(r, "fileId"))
	if err != nil {
		a.failFile(w, err)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventFiles, ConversationID: f.ConversationID})
	w.WriteHeader(http.StatusNoContent)
}

// fileContent serves a file's bytes to the browser.
func (a *API) fileContent(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	f, err := a.store.FileFor(r.Context(), user.ID, chi.URLParam(r, "fileId"))
	if err != nil {
		a.failFile(w, err)
		return
	}
	b, err := a.store.FileContent(r.Context(), f.ID)
	if err != nil {
		a.failFile(w, err)
		return
	}
	serveFile(w, f.Name, f.MediaType, b, r.URL.Query().Get("download") != "")
}

// serveFile sends bytes someone uploaded or an agent made, which nobody here
// has vetted. They can be shown in the page (a picture, a PDF in the browser's
// own viewer) but can never run as part of it: the type is the one recorded,
// never sniffed again; anything active was recorded as text; and a sandbox
// policy stops a document opened directly from doing anything.
func serveFile(w http.ResponseWriter, name, mediaType string, b []byte, download bool) {
	h := w.Header()
	h.Set("Content-Type", mediaType)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'")
	h.Set("Cache-Control", "private, max-age=3600")
	disposition := "inline"
	if download {
		disposition = "attachment"
	}
	h.Set("Content-Disposition", fmt.Sprintf("%s; filename*=UTF-8''%s", disposition, url.PathEscape(name)))
	h.Set("Content-Length", fmt.Sprint(len(b)))
	_, _ = w.Write(b)
}

// ---------------------------------------------------------------------------
// the working folder, read live from the computer
// ---------------------------------------------------------------------------

// askFolder asks the computer this conversation's work goes to about its
// working folder, and answers the browser when that cannot happen.
func (a *API) askFolder(w http.ResponseWriter, r *http.Request, msgType string) (protocol.DaemonMessage, bool) {
	user, _ := userFrom(r.Context())
	conv, err := a.store.Get(r.Context(), user.ID, chi.URLParam(r, "id"))
	if err != nil {
		a.failConversation(w, err)
		return protocol.DaemonMessage{}, false
	}
	if conv.Folder == "" {
		a.fail(w, errNoFolder, http.StatusNotFound)
		return protocol.DaemonMessage{}, false
	}
	allowed, err := a.machinesIn(r, conv.WorkspaceID)
	if err != nil {
		a.failWorkspace(w, err)
		return protocol.DaemonMessage{}, false
	}
	target, ok := a.hub.Target(user.ID, allowed)
	if !ok {
		a.fail(w, errDaemonOffline, http.StatusServiceUnavailable)
		return protocol.DaemonMessage{}, false
	}
	if !a.hub.SupportsFiles(user.ID, target.MachineID) {
		a.fail(w, errFolderNeedUpdate, http.StatusConflict)
		return protocol.DaemonMessage{}, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	reply, err := a.hub.AskIn(ctx, user.ID, allowed, protocol.ServerMessage{
		Type: msgType, Root: conv.Folder, Path: r.URL.Query().Get("path"),
	})
	switch {
	case errors.Is(err, hub.ErrDaemonOffline):
		a.fail(w, errDaemonOffline, http.StatusServiceUnavailable)
		return protocol.DaemonMessage{}, false
	case errors.Is(err, context.DeadlineExceeded):
		a.fail(w, errors.New("your machine did not answer in time"), http.StatusGatewayTimeout)
		return protocol.DaemonMessage{}, false
	case err != nil:
		a.fail(w, err, http.StatusInternalServerError)
		return protocol.DaemonMessage{}, false
	}
	return reply, true
}

// listFolder is one directory of a conversation's working folder.
func (a *API) listFolder(w http.ResponseWriter, r *http.Request) {
	reply, ok := a.askFolder(w, r, protocol.ServerListFolder)
	if !ok {
		return
	}
	if reply.Folder == nil {
		a.fail(w, errors.New("the machine sent an empty folder listing"), http.StatusInternalServerError)
		return
	}
	writeJSON(w, reply.Folder)
}

// folderFile is one file of a conversation's working folder, as bytes.
func (a *API) folderFile(w http.ResponseWriter, r *http.Request) {
	reply, ok := a.askFolder(w, r, protocol.ServerReadFile)
	if !ok {
		return
	}
	f := reply.File
	switch {
	case f == nil:
		a.fail(w, errors.New("the machine sent no file"), http.StatusInternalServerError)
		return
	case f.TooLarge:
		a.fail(w, fmt.Errorf("this file is %s, too large to show here (the limit is %d MB)", sizeWords(f.Size), protocol.MaxFolderFileBytes>>20), http.StatusRequestEntityTooLarge)
		return
	case f.Error != "":
		a.fail(w, errors.New(f.Error), http.StatusNotFound)
		return
	}
	b, err := base64.StdEncoding.DecodeString(f.Content)
	if err != nil {
		a.fail(w, errors.New("the machine sent a file that could not be read"), http.StatusInternalServerError)
		return
	}
	name := f.Path
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	serveFile(w, name, store.MediaType(name, b), b, r.URL.Query().Get("download") != "")
}

func sizeWords(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%d KB", n>>10)
	}
	return fmt.Sprintf("%d bytes", n)
}

// ---------------------------------------------------------------------------
// the computer's side
// ---------------------------------------------------------------------------

// daemonAccount is the account a computer's credential belongs to, checked the
// same way the websocket checks it. machineID is empty for the legacy shared
// token (D-038).
func (a *API) daemonAccount(r *http.Request) (userID, machineID string, status int) {
	presented := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if presented == "" {
		return "", "", http.StatusUnauthorized
	}
	paired, pairedUserID, ok, err := a.store.MachineForCredential(r.Context(), auth.HashToken(presented))
	if err != nil {
		return "", "", http.StatusServiceUnavailable
	}
	if ok {
		return pairedUserID, paired.ID, 0
	}
	if a.daemonAuthorised(r) {
		owner, exists, err := a.store.UserByEmail(r.Context(), a.cfg.OwnerEmail)
		if err != nil {
			return "", "", http.StatusServiceUnavailable
		}
		if exists {
			return owner.ID, "", 0
		}
	}
	return "", "", http.StatusUnauthorized
}

// daemonFile hands a computer a file a turn needs.
func (a *API) daemonFile(w http.ResponseWriter, r *http.Request) {
	userID, _, status := a.daemonAccount(r)
	if status != 0 {
		a.fail(w, errors.New("this machine is not authorised"), status)
		return
	}
	f, err := a.store.FileFor(r.Context(), userID, chi.URLParam(r, "fileId"))
	if err != nil {
		a.failFile(w, err)
		return
	}
	b, err := a.store.FileContent(r.Context(), f.ID)
	if err != nil {
		a.failFile(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprint(len(b)))
	_, _ = w.Write(b)
}

// daemonOutput takes a file an agent made during a running turn. Only the
// account whose turn it is may add to it, and only while it runs.
func (a *API) daemonOutput(w http.ResponseWriter, r *http.Request) {
	userID, _, status := a.daemonAccount(r)
	if status != 0 {
		a.fail(w, errors.New("this machine is not authorised"), status)
		return
	}
	a.mu.Lock()
	t := a.turns[chi.URLParam(r, "turnId")]
	a.mu.Unlock()
	if t == nil || t.UserID != userID {
		a.fail(w, errTurnNotRunning, http.StatusConflict)
		return
	}
	body, err := readFileBody(w, r)
	if err != nil {
		a.failFile(w, err)
		return
	}
	f, err := a.store.AddFile(r.Context(), t.ConversationID, t.EntryID, protocol.FileOutput, r.URL.Query().Get("name"), body)
	if err != nil {
		a.failFile(w, err)
		return
	}
	a.mu.Lock()
	t.Files = append(t.Files, f.ConversationFile)
	files := append([]protocol.ConversationFile(nil), t.Files...)
	a.mu.Unlock()
	a.hub.BroadcastTo(userID, protocol.ClientEvent{
		Type: protocol.EventFiles, ConversationID: t.ConversationID, EntryID: t.EntryID, Files: files,
	})
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, f.ConversationFile)
}

// turnFiles is every file of a conversation as a turn needs them, the ones
// sent with this message marked.
func turnFiles(files []store.File, attached []store.File) []protocol.TurnFile {
	sent := map[string]bool{}
	for _, f := range attached {
		sent[f.ID] = true
	}
	out := make([]protocol.TurnFile, 0, len(files))
	for _, f := range files {
		if f.EntryID == "" && !sent[f.ID] {
			continue // still waiting in some message box
		}
		out = append(out, protocol.TurnFile{
			ID: f.ID, Origin: f.Origin, Name: f.Name, Size: f.Size, Sha256: f.Sha256,
			MediaType: f.MediaType, Attached: sent[f.ID],
		})
	}
	return out
}

func replayFiles(e protocol.Entry) []protocol.ReplayFile {
	var out []protocol.ReplayFile
	for _, f := range e.Files {
		out = append(out, protocol.ReplayFile{Origin: f.Origin, Name: f.Name})
	}
	return out
}
