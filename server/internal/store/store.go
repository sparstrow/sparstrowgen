// Package store turns database rows into the shapes the browser and daemon
// speak, and back. Nothing above it touches pgtype.
package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sparstrow/sparstrowgen/server/internal/db"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

type Store struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: db.New(pool)}
}

// ---------------------------------------------------------------------------
// conversions
// ---------------------------------------------------------------------------

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// relative renders a timestamp the way the sidebar shows it. Deliberately
// coarse: "2m" and "1h" are what tell conversations apart at a glance, and a
// precise timestamp there would be noise.
func relative(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/24/365))
	}
}

func toConversation(c db.Conversation) protocol.Conversation {
	return protocol.Conversation{
		ID:       uuidToString(c.ID),
		Title:    c.Title,
		Folder:   c.Folder,
		Updated:  relative(c.UpdatedAt.Time),
		Provider: c.Provider,
		Model:    protocol.Model{ID: c.ModelID, Label: c.ModelLabel},
		SpendUsd: protocol.TicksToUSD(c.SpendTicks),
		Tokens:   c.Tokens,
		Archived: c.Archived,
		Entries:  []protocol.Entry{},
		SeenBy:   map[string]int32{},
	}
}

func toEntry(e db.Entry) protocol.Entry {
	out := protocol.Entry{
		ID:   uuidToString(e.ID),
		Role: e.Role,
		Seq:  e.Seq,
		At:   e.CreatedAt.Time.Local().Format("15:04"),
	}
	model := &protocol.Model{ID: str(e.ModelID), Label: str(e.ModelLabel)}
	switch e.Role {
	case "user":
		out.Text = e.Body
	case "agent":
		out.Text = e.Body
		out.Provider = str(e.Provider)
		out.Model = model
		out.Failure = str(e.Failure)
		// Usage is omitted entirely until the turn reports some. An agent
		// message still streaming has none, and showing "0 tokens" would be a
		// statement we cannot support.
		if e.Tokens != nil && *e.Tokens > 0 {
			u := &protocol.Usage{Tokens: *e.Tokens}
			// Only ever set for a provider that states real currency. Zero
			// ticks means "not reported", never "free".
			if e.SpendTicks != nil && *e.SpendTicks > 0 {
				usd := protocol.TicksToUSD(*e.SpendTicks)
				u.Usd = &usd
			}
			out.Usage = u
		}
	case "replay":
		out.To = str(e.Provider)
		out.ToModel = model
		if e.MessagesReplayed != nil {
			out.MessagesReplayed = *e.MessagesReplayed
		}
		if e.Tokens != nil {
			out.Tokens = *e.Tokens
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// conversations
// ---------------------------------------------------------------------------

// List returns every conversation, archived included. The client decides what
// to show: the archive is a filter, not a separate store, which is what lets
// search reach into it.
//
// A non-empty query searches titles, folders AND message bodies in Postgres.
// Searching titles alone would miss the ones that most need finding, since an
// unnamed conversation stays "Untitled conversation" until somebody renames it.
func (s *Store) List(ctx context.Context, query string) ([]protocol.Conversation, error) {
	query = strings.TrimSpace(query)
	if query != "" {
		return s.search(ctx, query)
	}
	rows, err := s.q.ListConversations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.Conversation, 0, len(rows))
	for _, r := range rows {
		c := toConversation(r)
		seen, err := s.seenBy(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		c.SeenBy = seen
		out = append(out, c)
	}
	return out, nil
}

// excerptPad is how much context surrounds a match. Enough to recognise the
// sentence, short enough for one line in a 288px sidebar.
const excerptPad = 34

func (s *Store) search(ctx context.Context, query string) ([]protocol.Conversation, error) {
	rows, err := s.q.SearchConversations(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.Conversation, 0, len(rows))
	for _, r := range rows {
		c := toConversation(r.Conversation)
		seen, err := s.seenBy(ctx, r.Conversation.ID)
		if err != nil {
			return nil, err
		}
		c.SeenBy = seen
		// A title match needs no excerpt; the query only returns one when the
		// hit was in a body.
		if r.Excerpt != "" && !strings.Contains(strings.ToLower(c.Title), strings.ToLower(query)) {
			c.Excerpt = excerptAround(r.Excerpt, query)
		}
		out = append(out, c)
	}
	return out, nil
}

// excerptAround trims a matching message down to the part worth reading.
func excerptAround(body, query string) string {
	at := strings.Index(strings.ToLower(body), strings.ToLower(query))
	if at < 0 {
		at = 0
	}
	start := at - excerptPad
	if start < 0 {
		start = 0
	}
	end := at + len(query) + excerptPad
	if end > len(body) {
		end = len(body)
	}
	out := strings.TrimSpace(body[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(body) {
		out += "…"
	}
	return out
}

// Get returns one conversation with its full transcript.
func (s *Store) Get(ctx context.Context, id string) (protocol.Conversation, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return protocol.Conversation{}, err
	}
	row, err := s.q.GetConversation(ctx, uid)
	if err != nil {
		return protocol.Conversation{}, err
	}
	c := toConversation(row)
	entries, err := s.q.ListEntries(ctx, uid)
	if err != nil {
		return protocol.Conversation{}, err
	}
	for _, e := range entries {
		c.Entries = append(c.Entries, toEntry(e))
	}
	if c.SeenBy, err = s.seenBy(ctx, uid); err != nil {
		return protocol.Conversation{}, err
	}
	return c, nil
}

func (s *Store) seenBy(ctx context.Context, id pgtype.UUID) (map[string]int32, error) {
	sessions, err := s.q.ListProviderSessions(ctx, id)
	if err != nil {
		return nil, err
	}
	out := map[string]int32{}
	for _, ps := range sessions {
		out[ps.Provider] = ps.SeenSeq
	}
	return out, nil
}

func (s *Store) Create(ctx context.Context, folder, provider string, model protocol.Model) (protocol.Conversation, error) {
	row, err := s.q.CreateConversation(ctx, db.CreateConversationParams{
		Title:      "Untitled conversation",
		Folder:     folder,
		Provider:   provider,
		ModelID:    model.ID,
		ModelLabel: model.Label,
	})
	if err != nil {
		return protocol.Conversation{}, err
	}
	return toConversation(row), nil
}

func (s *Store) Rename(ctx context.Context, id, title string) (protocol.Conversation, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return protocol.Conversation{}, err
	}
	row, err := s.q.RenameConversation(ctx, db.RenameConversationParams{ID: uid, Title: title})
	if err != nil {
		return protocol.Conversation{}, err
	}
	return toConversation(row), nil
}

func (s *Store) SetArchived(ctx context.Context, id string, archived bool) (protocol.Conversation, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return protocol.Conversation{}, err
	}
	row, err := s.q.SetConversationArchived(ctx, db.SetConversationArchivedParams{ID: uid, Archived: archived})
	if err != nil {
		return protocol.Conversation{}, err
	}
	return toConversation(row), nil
}

// SetFolder moves a conversation to another working directory, and drops every
// provider session with it.
//
// Allowed at any point, including mid-conversation, because the moment the
// folder is discovered to be wrong is usually after an answer about the wrong
// codebase (docs/Bugs.md B-3) — refusing then would refuse exactly when it
// matters.
//
// The sessions have to go. claude keys its sessions by project directory, so a
// resume id recorded in the old folder is simply not found from the new one —
// verified: `--resume` from a different cwd answers "No conversation found with
// session ID", and every later turn fails in about a second. Beyond that
// mechanical fact, a session built somewhere else is reasoning about the wrong
// tree, which is true of all three CLIs whether or not they refuse the resume.
//
// Dropping them resets seen_seq, so the next message replays the transcript
// into a fresh session — the same path a provider switch already uses, and the
// reason the picker warns that moving costs a catch-up.
//
// What the transcript does NOT record is that earlier answers came from
// somewhere else; see docs/Later.md L-11.
func (s *Store) SetFolder(ctx context.Context, id, folder string) (protocol.Conversation, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return protocol.Conversation{}, err
	}
	// Nothing to invalidate when the folder has not actually moved — and a
	// no-op must not cost a replay.
	current, err := s.q.GetConversation(ctx, uid)
	if err != nil {
		return protocol.Conversation{}, err
	}
	if current.Folder == folder {
		return toConversation(current), nil
	}
	row, err := s.q.SetConversationFolder(ctx, db.SetConversationFolderParams{ID: uid, Folder: folder})
	if err != nil {
		return protocol.Conversation{}, err
	}
	if err := s.q.DeleteConversationProviderSessions(ctx, uid); err != nil {
		return protocol.Conversation{}, err
	}
	return toConversation(row), nil
}

// RecentFolders is the picker's shortcut list and the default for a new
// conversation. Derived from conversations that already exist, so there is no
// separate list to keep in step with reality.
func (s *Store) RecentFolders(ctx context.Context, limit int32) ([]string, error) {
	rows, err := s.q.RecentFolders(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Folder)
	}
	return out, nil
}

func (s *Store) SetProvider(ctx context.Context, id, provider string, model protocol.Model) error {
	uid, err := parseUUID(id)
	if err != nil {
		return err
	}
	_, err = s.q.SetConversationProvider(ctx, db.SetConversationProviderParams{
		ID: uid, Provider: provider, ModelID: model.ID, ModelLabel: model.Label,
	})
	return err
}

// Delete removes the transcript for good. Foreign keys are kept and cascades
// are banned (D-009), so the children go first, explicitly, in one transaction.
func (s *Store) Delete(ctx context.Context, id string) error {
	uid, err := parseUUID(id)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := s.q.WithTx(tx)
	if err := q.DeleteConversationEntries(ctx, uid); err != nil {
		return err
	}
	if err := q.DeleteConversationProviderSessions(ctx, uid); err != nil {
		return err
	}
	if err := q.DeleteConversation(ctx, uid); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ---------------------------------------------------------------------------
// entries
// ---------------------------------------------------------------------------

func (s *Store) AppendUser(ctx context.Context, conversationID, text string) (protocol.Entry, error) {
	uid, err := parseUUID(conversationID)
	if err != nil {
		return protocol.Entry{}, err
	}
	row, err := s.q.AppendEntry(ctx, db.AppendEntryParams{
		ConversationID: uid, Role: "user", Body: text,
	})
	if err != nil {
		return protocol.Entry{}, err
	}
	return toEntry(row), nil
}

// AppendReplay records a catch-up that has actually been paid for. It is only
// ever called at send time — selecting a provider writes nothing.
func (s *Store) AppendReplay(ctx context.Context, conversationID, provider string, model protocol.Model, messages int32, tokens int64) (protocol.Entry, error) {
	uid, err := parseUUID(conversationID)
	if err != nil {
		return protocol.Entry{}, err
	}
	row, err := s.q.AppendEntry(ctx, db.AppendEntryParams{
		ConversationID:   uid,
		Role:             "replay",
		Provider:         &provider,
		ModelID:          &model.ID,
		ModelLabel:       &model.Label,
		Tokens:           &tokens,
		MessagesReplayed: &messages,
	})
	if err != nil {
		return protocol.Entry{}, err
	}
	return toEntry(row), nil
}

// AppendAgentPlaceholder creates the empty agent entry a turn streams into, so
// a refresh mid-turn shows what has arrived rather than nothing.
func (s *Store) AppendAgentPlaceholder(ctx context.Context, conversationID, provider string, model protocol.Model) (protocol.Entry, error) {
	uid, err := parseUUID(conversationID)
	if err != nil {
		return protocol.Entry{}, err
	}
	row, err := s.q.AppendEntry(ctx, db.AppendEntryParams{
		ConversationID: uid,
		Role:           "agent",
		Provider:       &provider,
		ModelID:        &model.ID,
		ModelLabel:     &model.Label,
	})
	if err != nil {
		return protocol.Entry{}, err
	}
	return toEntry(row), nil
}

func (s *Store) AppendDelta(ctx context.Context, entryID, text string) error {
	uid, err := parseUUID(entryID)
	if err != nil {
		return err
	}
	_, err = s.q.AppendEntryBody(ctx, db.AppendEntryBodyParams{ID: uid, Column2: text})
	return err
}

func (s *Store) FinishAgent(ctx context.Context, entryID, text string, tokens, spendTicks int64, failure string) (protocol.Entry, error) {
	uid, err := parseUUID(entryID)
	if err != nil {
		return protocol.Entry{}, err
	}
	var failPtr *string
	if failure != "" {
		failPtr = &failure
	}
	row, err := s.q.FinishAgentEntry(ctx, db.FinishAgentEntryParams{
		ID: uid, Body: text, Tokens: &tokens, SpendTicks: &spendTicks, Failure: failPtr,
	})
	if err != nil {
		return protocol.Entry{}, err
	}
	return toEntry(row), nil
}

func (s *Store) AddUsage(ctx context.Context, conversationID string, tokens, spendTicks int64) (protocol.Conversation, error) {
	uid, err := parseUUID(conversationID)
	if err != nil {
		return protocol.Conversation{}, err
	}
	row, err := s.q.AddConversationUsage(ctx, db.AddConversationUsageParams{
		ID: uid, Tokens: tokens, SpendTicks: spendTicks,
	})
	if err != nil {
		return protocol.Conversation{}, err
	}
	return toConversation(row), nil
}

// ---------------------------------------------------------------------------
// provider sessions
// ---------------------------------------------------------------------------

// Unseen returns the entries a provider has not been told about, and how many
// there are. This is the replay payload, and its length is what the cost quote
// is computed from.
func (s *Store) Unseen(ctx context.Context, conversationID, provider string) ([]protocol.Entry, error) {
	uid, err := parseUUID(conversationID)
	if err != nil {
		return nil, err
	}
	seen := int32(0)
	if ps, err := s.q.GetProviderSession(ctx, db.GetProviderSessionParams{
		ConversationID: uid, Provider: provider,
	}); err == nil {
		seen = ps.SeenSeq
	}
	rows, err := s.q.ListEntriesFrom(ctx, db.ListEntriesFromParams{ConversationID: uid, Seq: seen})
	if err != nil {
		return nil, err
	}
	out := make([]protocol.Entry, 0, len(rows))
	for _, r := range rows {
		if r.Role == "replay" {
			continue // a marker about switching is not part of the conversation
		}
		out = append(out, toEntry(r))
	}
	return out, nil
}

func (s *Store) ResumeID(ctx context.Context, conversationID, provider string) string {
	uid, err := parseUUID(conversationID)
	if err != nil {
		return ""
	}
	ps, err := s.q.GetProviderSession(ctx, db.GetProviderSessionParams{
		ConversationID: uid, Provider: provider,
	})
	if err != nil {
		return ""
	}
	return str(ps.SessionID)
}

// MarkSeen moves a provider's high-water mark to the end of the transcript.
// seen_seq only ever moves forward, enforced in SQL.
func (s *Store) MarkSeen(ctx context.Context, conversationID, provider, sessionID string, seq int32) error {
	uid, err := parseUUID(conversationID)
	if err != nil {
		return err
	}
	var sess *string
	if sessionID != "" {
		sess = &sessionID
	}
	_, err = s.q.UpsertProviderSeen(ctx, db.UpsertProviderSeenParams{
		ConversationID: uid, Provider: provider, SeenSeq: seq, SessionID: sess,
	})
	return err
}

func (s *Store) LastSeq(ctx context.Context, conversationID string) (int32, error) {
	uid, err := parseUUID(conversationID)
	if err != nil {
		return 0, err
	}
	rows, err := s.q.ListEntries(ctx, uid)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[len(rows)-1].Seq, nil
}
