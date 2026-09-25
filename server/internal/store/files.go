package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sparstrow/sparstrowgen/server/internal/db"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// A conversation's files (D-056). The rows say what the file is and which
// message or turn it belongs to; the bytes sit in a table of their own.

// ErrFileNotFound is a file that does not exist for this account, answered the
// same way whether it belongs to someone else or never existed.
var ErrFileNotFound = errors.New("that file does not exist")

// ErrFileTooLarge is a file over protocol.MaxFileBytes.
var ErrFileTooLarge = fmt.Errorf("a file can be at most %d MB", protocol.MaxFileBytes>>20)

// File is a conversation file with the checksum only the daemon needs.
type File struct {
	protocol.ConversationFile
	Sha256 string
}

func toFile(f db.ConversationFile) File {
	return File{
		ConversationFile: protocol.ConversationFile{
			ID:             uuidToString(f.ID),
			ConversationID: uuidToString(f.ConversationID),
			EntryID:        uuidToString(f.EntryID),
			Origin:         f.Origin,
			Name:           f.Name,
			MediaType:      f.MediaType,
			Size:           f.Size,
			CreatedAt:      f.CreatedAt.Time.UTC().Format(time.RFC3339),
		},
		Sha256: f.Sha256,
	}
}

// fileName makes a name safe to be a file's name on any computer: no folder
// part, nothing Windows refuses, not empty, not endless.
func fileName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	name = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsControl(r), strings.ContainsRune(`<>:"/\|?*`, r):
			return '_'
		}
		return r
	}, name)
	// Windows drops trailing dots and spaces itself, which would make the name
	// on disk differ from the one recorded.
	name = strings.TrimRight(strings.TrimSpace(name), ". ")
	if name == "" || name == "." || name == ".." {
		name = "file"
	}
	if r := []rune(name); len(r) > 150 {
		ext := path.Ext(name)
		if len([]rune(ext)) > 20 {
			ext = ""
		}
		name = string(r[:150-len([]rune(ext))]) + ext
	}
	// Names Windows reserves for devices, with or without an extension.
	stem := strings.ToUpper(strings.TrimSuffix(name, path.Ext(name)))
	switch stem {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		name = "_" + name
	}
	return name
}

// freeName returns name, or "stem (2).ext" and upward, whichever is not taken.
// Compared without case, because Windows folders are.
func freeName(name string, taken []string) string {
	used := make(map[string]bool, len(taken))
	for _, t := range taken {
		used[strings.ToLower(t)] = true
	}
	if !used[strings.ToLower(name)] {
		return name
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for n := 2; ; n++ {
		c := fmt.Sprintf("%s (%d)%s", stem, n, ext)
		if !used[strings.ToLower(c)] {
			return c
		}
	}
}

// MediaType decides what a file is from its bytes, falling back to its name
// only where the bytes cannot tell (text of every kind, and office files, which
// are zip archives inside). Never trusted from the browser. Anything a browser
// would run — HTML, SVG, script — is recorded as plain text, so it is only ever
// shown as text.
func MediaType(name string, content []byte) string {
	sniffed := http.DetectContentType(content)
	if i := strings.IndexByte(sniffed, ';'); i >= 0 {
		sniffed = sniffed[:i]
	}
	byName := mime.TypeByExtension(strings.ToLower(path.Ext(name)))
	if i := strings.IndexByte(byName, ';'); i >= 0 {
		byName = byName[:i]
	}
	t := sniffed
	switch sniffed {
	case "text/plain", "application/octet-stream", "application/zip":
		if byName != "" {
			t = byName
		}
	}
	if active(t) || active(sniffed) {
		return "text/plain"
	}
	return t
}

func active(t string) bool {
	switch {
	case t == "text/html", t == "image/svg+xml", t == "application/xhtml+xml",
		strings.Contains(t, "javascript"):
		return true
	}
	return false
}

// AddFile stores one file in a conversation. entryID is empty for an upload
// waiting in the message box. The name it gets may differ from the one given:
// made safe, and made unique in its folder.
func (s *Store) AddFile(ctx context.Context, conversationID, entryID, origin, name string, content []byte) (File, error) {
	if len(content) > protocol.MaxFileBytes {
		return File{}, ErrFileTooLarge
	}
	cid, err := parseUUID(conversationID)
	if err != nil {
		return File{}, ErrNotFound
	}
	var eid pgtype.UUID
	if entryID != "" {
		if eid, err = parseUUID(entryID); err != nil {
			return File{}, fmt.Errorf("entry id: %w", err)
		}
	}
	name = fileName(name)
	sum := sha256.Sum256(content)
	mediaType := MediaType(name, content)

	// Two files with one name arriving together both find it free; the unique
	// index refuses the second, which then looks again.
	for attempt := 0; ; attempt++ {
		f, err := s.addFile(ctx, db.CreateConversationFileParams{
			ConversationID: cid, EntryID: eid, Origin: origin, MediaType: mediaType,
			Size: int64(len(content)), Sha256: hex.EncodeToString(sum[:]),
		}, name, content)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && attempt < 5 {
			continue
		}
		return f, err
	}
}

func (s *Store) addFile(ctx context.Context, p db.CreateConversationFileParams, name string, content []byte) (File, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return File{}, err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	taken, err := q.ConversationFileNames(ctx, db.ConversationFileNamesParams{
		ConversationID: p.ConversationID, Origin: p.Origin,
	})
	if err != nil {
		return File{}, err
	}
	p.Name = freeName(name, taken)
	row, err := q.CreateConversationFile(ctx, p)
	if err != nil {
		return File{}, err
	}
	if err := q.PutConversationFileContent(ctx, db.PutConversationFileContentParams{
		FileID: row.ID, Content: content,
	}); err != nil {
		return File{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return File{}, err
	}
	return toFile(row), nil
}

// Files lists a conversation's files, newest first. The caller has already
// checked the conversation is this account's.
func (s *Store) Files(ctx context.Context, conversationID string) ([]File, error) {
	cid, err := parseUUID(conversationID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.ListConversationFiles(ctx, cid)
	if err != nil {
		return nil, err
	}
	out := make([]File, 0, len(rows))
	for _, r := range rows {
		out = append(out, toFile(r))
	}
	return out, nil
}

// FileFor is one file, only if this account can see its conversation.
func (s *Store) FileFor(ctx context.Context, userID, fileID string) (File, error) {
	owner, fid, err := owned(userID, fileID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return File{}, ErrFileNotFound
		}
		return File{}, err
	}
	row, err := s.q.ConversationFileFor(ctx, db.ConversationFileForParams{ID: fid, UserID: owner})
	if err != nil {
		if errors.Is(missing(err), ErrNotFound) {
			return File{}, ErrFileNotFound
		}
		return File{}, err
	}
	return toFile(row), nil
}

// FileContent is a file's bytes. Check ownership with FileFor first.
func (s *Store) FileContent(ctx context.Context, fileID string) ([]byte, error) {
	fid, err := parseUUID(fileID)
	if err != nil {
		return nil, ErrFileNotFound
	}
	b, err := s.q.ConversationFileContent(ctx, fid)
	if err != nil {
		if errors.Is(missing(err), ErrNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	return b, nil
}

// AttachFiles gives the uploads waiting in a conversation's box to the message
// just sent. An id that is not such an upload is left out, never an error: the
// message is what matters, and it goes with the files that are there.
func (s *Store) AttachFiles(ctx context.Context, conversationID, entryID string, ids []string) ([]File, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	cid, err := parseUUID(conversationID)
	if err != nil {
		return nil, ErrNotFound
	}
	eid, err := parseUUID(entryID)
	if err != nil {
		return nil, err
	}
	var uids []pgtype.UUID
	for _, id := range ids {
		if u, err := parseUUID(id); err == nil {
			uids = append(uids, u)
		}
	}
	rows, err := s.q.AttachConversationFiles(ctx, db.AttachConversationFilesParams{
		EntryID: eid, Ids: uids, ConversationID: cid,
	})
	if err != nil {
		return nil, err
	}
	out := make([]File, 0, len(rows))
	for _, r := range rows {
		out = append(out, toFile(r))
	}
	sortOldestFirst(out)
	return out, nil
}

// DeleteUnsentFile removes an upload from the message box. One already sent is
// part of the record and is refused as not found.
func (s *Store) DeleteUnsentFile(ctx context.Context, userID, fileID string) (File, error) {
	f, err := s.FileFor(ctx, userID, fileID)
	if err != nil {
		return File{}, err
	}
	if f.EntryID != "" || f.Origin != protocol.FileUpload {
		return File{}, ErrFileNotFound
	}
	fid, _ := parseUUID(fileID)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return File{}, err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	if err := q.DeleteConversationFileContent(ctx, fid); err != nil {
		return File{}, err
	}
	n, err := q.DeleteUnsentConversationFile(ctx, fid)
	if err != nil {
		return File{}, err
	}
	if n == 0 {
		return File{}, ErrFileNotFound
	}
	return f, tx.Commit(ctx)
}

// withFiles puts each entry's files on it, oldest first, which is the order
// they were added in.
func withFiles(entries []protocol.Entry, files []File) {
	by := map[string][]protocol.ConversationFile{}
	for i := len(files) - 1; i >= 0; i-- { // files arrive newest first
		f := files[i]
		if f.EntryID != "" {
			by[f.EntryID] = append(by[f.EntryID], f.ConversationFile)
		}
	}
	for i := range entries {
		if fs := by[entries[i].ID]; len(fs) > 0 {
			entries[i].Files = fs
		}
	}
}

func sortOldestFirst(fs []File) {
	for i := 1; i < len(fs); i++ {
		for j := i; j > 0 && fs[j].CreatedAt < fs[j-1].CreatedAt; j-- {
			fs[j], fs[j-1] = fs[j-1], fs[j]
		}
	}
}

// Public returns the browser's view of files.
func Public(fs []File) []protocol.ConversationFile {
	out := make([]protocol.ConversationFile, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.ConversationFile)
	}
	return out
}
