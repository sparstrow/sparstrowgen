package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/sparstrow/sparstrowgen/server/internal/db"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// ErrNoTurn is an agent turn this account cannot see: another account's, one
// that never existed, or an entry that is not a turn. One answer for all three,
// for the same reason as ErrNotFound.
var ErrNoTurn = errors.New("that turn does not exist")

// storable makes printed text fit a Postgres text column, which refuses one
// byte: NUL. A CLI that prints binary would otherwise fail its whole batch and
// lose the lines around it, so the NUL becomes U+FFFD, the same character JSON
// already turned any invalid UTF-8 into on the way here.
func storable(s string) string {
	return strings.ReplaceAll(s, "\x00", "\uFFFD")
}

// RecordExchangeSent stores what a turn handed its CLI. Called from the daemon
// path, for a turn the server itself started, so it takes no account.
func (s *Store) RecordExchangeSent(ctx context.Context, conversationID, entryID string, sent protocol.ExchangeSent) error {
	conv, err := parseUUID(conversationID)
	if err != nil {
		return err
	}
	entry, err := parseUUID(entryID)
	if err != nil {
		return err
	}
	args := sent.Args
	if args == nil {
		args = []string{}
	}
	return s.q.RecordExchange(ctx, db.RecordExchangeParams{
		EntryID: entry, ConversationID: conv,
		Program: sent.Program, Args: args, Cwd: sent.Cwd,
		ResumeSessionID: sent.ResumeSessionID,
		Prompt:          storable(sent.Prompt), Stdin: storable(sent.Stdin), Launched: sent.Launched,
	})
}

// AppendExchangeLines stores one batch of a turn's lines, and the running count
// of what was not kept, when the daemon sent one.
func (s *Store) AppendExchangeLines(ctx context.Context, entryID string, lines []protocol.ExchangeLine, dropped *protocol.ExchangeDropped) error {
	entry, err := parseUUID(entryID)
	if err != nil {
		return err
	}
	if len(lines) > 0 {
		p := db.AppendExchangeLinesParams{EntryID: entry}
		for _, l := range lines {
			p.Seqs = append(p.Seqs, l.Seq)
			p.AtMs = append(p.AtMs, l.AtMs)
			p.Streams = append(p.Streams, l.Stream)
			p.Bodies = append(p.Bodies, storable(l.Text))
		}
		if err := s.q.AppendExchangeLines(ctx, p); err != nil {
			return err
		}
	}
	if dropped != nil {
		return s.q.SetExchangeDropped(ctx, db.SetExchangeDroppedParams{
			EntryID: entry, DroppedLines: dropped.Lines, DroppedBytes: dropped.Bytes,
		})
	}
	return nil
}

// Exchange reads one turn's record for its owner, and the provider that ran
// it, which is what the record has to be read with. A turn with no record is
// an answer (Recorded false), not an error.
func (s *Store) Exchange(ctx context.Context, userID, entryID string) (protocol.Exchange, string, error) {
	owner, err := parseUUID(userID)
	if err != nil {
		return protocol.Exchange{}, "", err
	}
	id, err := parseUUID(entryID)
	if err != nil {
		return protocol.Exchange{}, "", ErrNoTurn
	}
	turn, err := s.q.AgentEntryFor(ctx, db.AgentEntryForParams{ID: id, UserID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return protocol.Exchange{}, "", ErrNoTurn
	}
	if err != nil {
		return protocol.Exchange{}, "", err
	}

	out := protocol.Exchange{EntryID: entryID, Lines: []protocol.ExchangeLine{}}
	row, err := s.q.GetExchange(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, turn.Provider, nil
	}
	if err != nil {
		return protocol.Exchange{}, "", err
	}
	out.Recorded = true
	out.Sent = &protocol.ExchangeSent{
		Program: row.Program, Args: row.Args, Cwd: row.Cwd, ResumeSessionID: row.ResumeSessionID,
		Prompt: row.Prompt, Stdin: row.Stdin, Launched: row.Launched,
	}
	if row.DroppedLines > 0 || row.DroppedBytes > 0 {
		out.Dropped = &protocol.ExchangeDropped{Lines: row.DroppedLines, Bytes: row.DroppedBytes}
	}
	lines, err := s.q.ListExchangeLines(ctx, id)
	if err != nil {
		return protocol.Exchange{}, "", err
	}
	for _, l := range lines {
		out.Lines = append(out.Lines, protocol.ExchangeLine{Seq: l.Seq, AtMs: l.AtMs, Stream: l.Stream, Text: l.Body})
	}
	return out, turn.Provider, nil
}

// ExchangeSummaries says how big each turn's record in one of this account's
// conversations is.
func (s *Store) ExchangeSummaries(ctx context.Context, userID, conversationID string) ([]protocol.ExchangeSummary, error) {
	owner, uid, err := owned(userID, conversationID)
	if err != nil {
		return nil, err
	}
	if _, err := s.q.GetConversation(ctx, db.GetConversationParams{ID: uid, UserID: owner}); err != nil {
		return nil, missing(err)
	}
	rows, err := s.q.ListExchangeSummaries(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.ExchangeSummary, 0, len(rows))
	for _, r := range rows {
		sum := protocol.ExchangeSummary{
			EntryID: uuidToString(r.EntryID), Launched: r.Launched,
			Lines: r.LineCount, Bytes: r.ByteCount, LastAtMs: r.LastAtMs,
		}
		if r.DroppedLines > 0 || r.DroppedBytes > 0 {
			sum.Dropped = &protocol.ExchangeDropped{Lines: r.DroppedLines, Bytes: r.DroppedBytes}
		}
		out = append(out, sum)
	}
	return out, nil
}
