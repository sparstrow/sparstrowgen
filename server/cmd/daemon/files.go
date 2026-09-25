package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// Files in a conversation, on this computer (D-056).
//
// Each conversation has a folder of its own, <chats>/<conversation id>, with
// uploads/ for what the owner sent and outputs/ for what an agent made. The
// agents are given it alongside the project folder. It is outside the project
// on purpose: a picture pasted into a chat is not part of the codebase, and a
// file an agent writes INTO the project stays where the agent put it.

// chatFiles moves a conversation's files between the server and this computer.
type chatFiles struct {
	// chats is the folder holding every conversation's folder.
	chats string
	// api is the server's HTTP base; token is read when needed, because the
	// credential can change between connections.
	api    string
	token  func() string
	client *http.Client
	// home is the user's home folder, where codex and agy keep the pictures
	// they generate. A variable so tests use a scratch one.
	home func() (string, error)
	log  *slog.Logger
}

func newChatFiles(log *slog.Logger, token func() string) (*chatFiles, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	return &chatFiles{
		chats: filepath.Join(dir, "chats"), api: serverAPI(), token: token,
		client: &http.Client{Timeout: 5 * time.Minute}, home: os.UserHomeDir, log: log,
	}, nil
}

// folder is one conversation's folder. The id comes from the server, so it is
// checked to be only an id before it becomes part of a path.
func (c *chatFiles) folder(conversationID string) (string, error) {
	if conversationID == "" || strings.ContainsAny(conversationID, `/\:.`) {
		return "", fmt.Errorf("not a conversation id: %q", conversationID)
	}
	return filepath.Join(c.chats, conversationID), nil
}

func subfolder(origin string) string {
	if origin == protocol.FileOutput {
		return "outputs"
	}
	return "uploads"
}

// preparedFiles is what a turn's prompt and command line need about its files.
type preparedFiles struct {
	// Folder is the conversation's folder, given to claude and agy.
	Folder  string
	Uploads string
	Outputs string
	// Attached are the files sent with this message, in the order sent.
	Attached []preparedFile
	// Known are the outputs already recorded, so collecting after the turn
	// does not send them again.
	Known map[string]bool
}

type preparedFile struct {
	Path      string
	Name      string
	MediaType string
	Size      int64
}

// prepare makes sure every file the turn lists is in the conversation's folder,
// downloading what is missing. A file sent with this message that cannot be
// fetched fails the turn: the agent would answer about something it never saw.
// An older one only warns.
func (c *chatFiles) prepare(ctx context.Context, t protocol.RunTurn) (preparedFiles, error) {
	var p preparedFiles
	folder, err := c.folder(t.ConversationID)
	if err != nil {
		return p, err
	}
	p.Folder, p.Uploads, p.Outputs = folder, filepath.Join(folder, "uploads"), filepath.Join(folder, "outputs")
	p.Known = map[string]bool{}
	for _, dir := range []string{p.Uploads, p.Outputs} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return p, fmt.Errorf("could not make this conversation's folder on this computer: %w", err)
		}
	}
	for _, f := range t.Files {
		name := filepath.Base(filepath.FromSlash(f.Name))
		if name != f.Name || name == "." || name == ".." {
			c.log.Warn("skipped a file with an unusable name", "file", f.ID, "name", f.Name)
			continue
		}
		dest := filepath.Join(folder, subfolder(f.Origin), name)
		if f.Origin == protocol.FileOutput {
			p.Known[strings.ToLower(name)] = true
		}
		if info, err := os.Stat(dest); err != nil || info.Size() != f.Size {
			if err := c.download(ctx, f.ID, dest); err != nil {
				if f.Attached {
					return p, fmt.Errorf("could not get %s onto this computer: %w", f.Name, err)
				}
				c.log.Warn("an earlier file could not be fetched", "file", f.Name, "err", err)
				continue
			}
		}
		if f.Attached {
			p.Attached = append(p.Attached, preparedFile{Path: dest, Name: name, MediaType: f.MediaType, Size: f.Size})
		}
	}
	return p, nil
}

func (c *chatFiles) request(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token())
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		defer res.Body.Close()
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&e)
		if e.Error == "" {
			e.Error = res.Status
		}
		return nil, errors.New(e.Error)
	}
	return res, nil
}

// download writes one file from the server, whole or not at all.
func (c *chatFiles) download(ctx context.Context, fileID, dest string) error {
	res, err := c.request(ctx, http.MethodGet, c.api+"/daemon/files/"+url.PathEscape(fileID), nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".download-*")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, io.LimitReader(res.Body, protocol.MaxFileBytes+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	_ = os.Remove(dest)
	return os.Rename(tmp.Name(), dest)
}

// ---------------------------------------------------------------------------
// the prompt
// ---------------------------------------------------------------------------

// textInline is how much of a text file codex is given in its prompt, since it
// cannot open files itself under its sandbox (docs/Capabilities.md).
const textInline = 100 << 10

func isPicture(mediaType string) bool {
	switch mediaType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	}
	return false
}

func isText(mediaType string) bool {
	return strings.HasPrefix(mediaType, "text/") || mediaType == "application/json" ||
		mediaType == "application/xml" || strings.HasSuffix(mediaType, "+json")
}

// filesPrompt is what follows the owner's message when files came with it, and
// the codex pictures to attach. Each agent is told what it can actually use:
// claude and agy read the folder; codex sees pictures through --image, is given
// small text files in the prompt, and is told plainly about anything else.
func filesPrompt(provider string, p preparedFiles) (string, []string) {
	var b strings.Builder
	var images []string
	if len(p.Attached) > 0 {
		if provider == "codex" {
			var others []preparedFile
			for _, f := range p.Attached {
				if isPicture(f.MediaType) {
					images = append(images, f.Path)
				} else {
					others = append(others, f)
				}
			}
			if len(images) > 0 {
				fmt.Fprintf(&b, "\n\n%d picture(s) sent with this message are attached.", len(images))
			}
			for _, f := range others {
				content, ok := inlineText(f)
				if ok {
					fmt.Fprintf(&b, "\n\nThe file %s was sent with this message. Its contents:\n```\n%s\n```", f.Name, content)
				} else {
					fmt.Fprintf(&b, "\n\nThe file %s was sent with this message, but it cannot be opened with the permissions you have here. If the answer depends on it, say so.", f.Name)
				}
			}
		} else {
			b.WriteString("\n\nFiles sent with this message:")
			for _, f := range p.Attached {
				fmt.Fprintf(&b, "\n- %s", f.Path)
			}
		}
	}
	// Where a file made for the owner should go. Not for codex, whose sandbox
	// cannot write there; the pictures it generates are collected anyway.
	if provider != "codex" && p.Outputs != "" {
		fmt.Fprintf(&b, "\n\nIf you make a file for me that is not part of the project (a picture, a document, an export), save it in %s so it appears in our chat. Files that belong in the project go where they normally would.", p.Outputs)
	}
	return b.String(), images
}

func inlineText(f preparedFile) (string, bool) {
	if !isText(f.MediaType) || f.Size > textInline {
		return "", false
	}
	b, err := os.ReadFile(f.Path)
	if err != nil || !utf8.Valid(b) {
		return "", false
	}
	return strings.TrimRight(string(b), "\n"), true
}

// replayFiles is the note a catch-up gives for the files one earlier message
// carried, so the provider joining knows they exist and where.
func replayFiles(e protocol.ReplayEntry, p preparedFiles) string {
	if len(e.Files) == 0 || p.Folder == "" {
		return ""
	}
	var names []string
	for _, f := range e.Files {
		names = append(names, filepath.Join(p.Folder, subfolder(f.Origin), f.Name))
	}
	verb := "sent with this message"
	if e.Role == "agent" {
		verb = "made in this turn"
	}
	return fmt.Sprintf("\n(Files %s: %s)", verb, strings.Join(names, ", "))
}

// ---------------------------------------------------------------------------
// what the agent made
// ---------------------------------------------------------------------------

var pictureExt = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true}

// generatedFolder is where a provider keeps the pictures it generates, by
// session, or empty for one that generates none. Verified 2026-09-24: codex
// writes $CODEX_HOME/generated_images/<thread>/exec-<uuid>.png and says
// nothing about it in its stream; agy writes
// ~/.gemini/antigravity-cli/brain/<conversation>/<name>_<ms>.jpg.
func (c *chatFiles) generatedFolder(provider, sessionID string) string {
	if sessionID == "" || strings.ContainsAny(sessionID, `/\:`) || strings.Contains(sessionID, "..") {
		return ""
	}
	home, err := c.home()
	if err != nil {
		return ""
	}
	switch provider {
	case "codex":
		base := os.Getenv("CODEX_HOME")
		if base == "" {
			base = filepath.Join(home, ".codex")
		}
		return filepath.Join(base, "generated_images", sessionID)
	case "agy":
		return filepath.Join(home, ".gemini", "antigravity-cli", "brain", sessionID)
	}
	return ""
}

// collect finds what the agent made in this turn: new files in the outputs
// folder, and pictures it generated into its own folder, which are copied into
// outputs first. Files older than the turn are someone else's.
func (c *chatFiles) collect(p preparedFiles, provider, sessionID string, since time.Time) []string {
	var found []string
	seen := map[string]bool{}
	// A little slack: file times on some filesystems are coarser than ours.
	since = since.Add(-2 * time.Second)

	if gen := c.generatedFolder(provider, sessionID); gen != "" {
		entries, _ := os.ReadDir(gen)
		for _, e := range entries {
			info, err := e.Info()
			if err != nil || !info.Mode().IsRegular() || info.ModTime().Before(since) ||
				!pictureExt[strings.ToLower(filepath.Ext(e.Name()))] {
				continue
			}
			dest := freePath(p.Outputs, e.Name())
			if err := copyFile(filepath.Join(gen, e.Name()), dest); err != nil {
				c.log.Warn("could not copy a generated picture", "file", e.Name(), "err", err)
				continue
			}
			seen[strings.ToLower(filepath.Base(dest))] = true
			found = append(found, dest)
		}
	}

	entries, _ := os.ReadDir(p.Outputs)
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if seen[name] || p.Known[name] || strings.HasPrefix(e.Name(), ".download-") {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() || info.ModTime().Before(since) {
			continue
		}
		found = append(found, filepath.Join(p.Outputs, e.Name()))
	}
	return found
}

// send uploads one output of a running turn. The server may rename it to keep
// names unique; the file on disk is renamed to match, so the two never differ.
func (c *chatFiles) send(ctx context.Context, turnID, file string) error {
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	if info.Size() > protocol.MaxFileBytes {
		return fmt.Errorf("%s is %d MB, over the %d MB a conversation keeps", filepath.Base(file), info.Size()>>20, protocol.MaxFileBytes>>20)
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	res, err := c.request(ctx, http.MethodPost,
		c.api+"/daemon/turns/"+url.PathEscape(turnID)+"/files?name="+url.QueryEscape(filepath.Base(file)), f)
	_ = f.Close()
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var got protocol.ConversationFile
	if err := json.NewDecoder(res.Body).Decode(&got); err == nil && got.Name != "" && got.Name != filepath.Base(file) {
		if err := os.Rename(file, filepath.Join(filepath.Dir(file), got.Name)); err != nil {
			c.log.Warn("could not rename an output to the name it was kept as", "file", file, "as", got.Name, "err", err)
		}
	}
	return nil
}

func freePath(dir, name string) string {
	p := filepath.Join(dir, name)
	if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
		return p
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for n := 2; ; n++ {
		p = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, n, ext))
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			return p
		}
	}
}

// ---------------------------------------------------------------------------
// the working folder, for the pane
// ---------------------------------------------------------------------------

// maxFolderEntries keeps one listing to a size a pane can show and a socket
// message can carry.
const maxFolderEntries = 2000

// within resolves rel inside root, refusing anything that leaves it: "..", an
// absolute path, or a link pointing out. The browser chooses rel; the server
// chose root from the conversation.
func within(root, rel string) (string, error) {
	if root == "" || !filepath.IsAbs(root) {
		return "", errors.New("this conversation has no working folder")
	}
	rel = strings.ReplaceAll(rel, `\`, "/")
	if path.IsAbs(rel) || strings.Contains(rel, ":") {
		return "", errors.New("that is not inside the working folder")
	}
	clean := path.Clean("/" + rel)[1:]
	full := filepath.Join(root, filepath.FromSlash(clean))
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", errors.New("the working folder is not on this computer any more")
	}
	realFull, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", errors.New("that is not in the working folder any more")
	}
	if !inside(realRoot, realFull) {
		return "", errors.New("that is not inside the working folder")
	}
	return realFull, nil
}

func inside(root, p string) bool {
	if runtime.GOOS == "windows" {
		root, p = strings.ToLower(root), strings.ToLower(p)
	}
	if p == root {
		return true
	}
	return strings.HasPrefix(p, strings.TrimRight(root, `\/`)+string(filepath.Separator))
}

func folderListing(root, rel string) protocol.FolderListing {
	out := protocol.FolderListing{Path: strings.Trim(strings.ReplaceAll(rel, `\`, "/"), "/"), Entries: []protocol.FolderEntry{}}
	dir, err := within(root, rel)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		out.Error = "this folder could not be read: " + err.Error()
		return out
	}
	for _, e := range entries {
		if len(out.Entries) == maxFolderEntries {
			break
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		out.Entries = append(out.Entries, protocol.FolderEntry{
			Name: e.Name(), Kind: kind, Size: info.Size(), Modified: info.ModTime().UTC().Format(time.RFC3339),
		})
	}
	// Folders first, then files, each by name without case, as a file
	// explorer shows them.
	sort.SliceStable(out.Entries, func(i, j int) bool {
		a, b := out.Entries[i], out.Entries[j]
		if a.Kind != b.Kind {
			return a.Kind == "dir"
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	return out
}

func folderFile(root, rel string) protocol.FolderFile {
	out := protocol.FolderFile{Path: strings.Trim(strings.ReplaceAll(rel, `\`, "/"), "/")}
	full, err := within(root, rel)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	info, err := os.Stat(full)
	switch {
	case err != nil:
		out.Error = "that file could not be read: " + err.Error()
		return out
	case info.IsDir():
		out.Error = "that is a folder"
		return out
	}
	out.Size = info.Size()
	if info.Size() > protocol.MaxFolderFileBytes {
		out.TooLarge = true
		return out
	}
	b, err := os.ReadFile(full)
	if err != nil {
		out.Error = "that file could not be read: " + err.Error()
		return out
	}
	out.Content = base64.StdEncoding.EncodeToString(b)
	return out
}

func (d *daemon) answerFolder(msg protocol.ServerMessage) {
	reply := protocol.DaemonMessage{RequestID: msg.RequestID}
	if msg.Type == protocol.ServerListFolder {
		listing := folderListing(msg.Root, msg.Path)
		reply.Type, reply.Folder = protocol.DaemonFolderListing, &listing
	} else {
		file := folderFile(msg.Root, msg.Path)
		reply.Type, reply.File = protocol.DaemonFolderFile, &file
	}
	if err := d.send(reply); err != nil {
		d.log.Warn("folder answer not sent", "err", err, "path", msg.Path)
	}
}
