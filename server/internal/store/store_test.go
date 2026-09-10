package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* These need a real Postgres, because the rules being tested live in SQL: seq
   is allocated inside the INSERT, and seen_seq is clamped by a GREATEST in the
   upsert. Testing them against a fake would only test the fake.

   They SKIP rather than fail when no database is reachable, so `go test ./...`
   stays useful on a machine without Docker running. Start one with
   `make db && make migrate`. */

func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://sparstrowgen:sparstrowgen@localhost:5433/sparstrowgen?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("no database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skip("no database reachable — run `make db && make migrate`")
	}
	t.Cleanup(pool.Close)
	return New(pool)
}

func newConversation(t *testing.T, s *Store) protocol.Conversation {
	t.Helper()
	ctx := context.Background()
	c, err := s.Create(ctx, "D:\\test", "codex", protocol.Model{ID: "m1", Label: "M One"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Leave nothing behind: these run against the development database.
	t.Cleanup(func() { _ = s.Delete(context.Background(), c.ID) })
	return c
}

// seq must be dense and gapless per conversation, because seen_seq counts
// against it. A gap would make a provider look further behind than it is.
func TestSeqIsAllocatedDenselyPerConversation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a := newConversation(t, s)
	b := newConversation(t, s)

	for i := 1; i <= 3; i++ {
		e, err := s.AppendUser(ctx, a.ID, "hello")
		if err != nil {
			t.Fatalf("append: %v", err)
		}
		if int(e.Seq) != i {
			t.Fatalf("seq = %d, want %d", e.Seq, i)
		}
	}

	// Numbering is per conversation, not global.
	first, err := s.AppendUser(ctx, b.ID, "hello")
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if first.Seq != 1 {
		t.Errorf("a second conversation started at seq %d, want 1", first.Seq)
	}
}

// The whole economics of switching rests on this: a provider is only ever sent
// what it has not already seen.
func TestUnseenReturnsOnlyTheGap(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	if _, err := s.AppendUser(ctx, c.ID, "first question"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendUser(ctx, c.ID, "second question"); err != nil {
		t.Fatal(err)
	}

	// A provider that has seen nothing gets everything.
	unseen, err := s.Unseen(ctx, c.ID, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(unseen) != 2 {
		t.Fatalf("a fresh provider should replay 2, got %d", len(unseen))
	}

	// After being caught up to seq 2, it gets nothing.
	if err := s.MarkSeen(ctx, c.ID, "claude", "session-1", 2); err != nil {
		t.Fatal(err)
	}
	if unseen, err = s.Unseen(ctx, c.ID, "claude"); err != nil {
		t.Fatal(err)
	}
	if len(unseen) != 0 {
		t.Fatalf("a caught-up provider should replay 0, got %d", len(unseen))
	}

	// One more message, and only that one is owed.
	if _, err := s.AppendUser(ctx, c.ID, "third question"); err != nil {
		t.Fatal(err)
	}
	if unseen, err = s.Unseen(ctx, c.ID, "codex"); err != nil {
		t.Fatal(err)
	}
	if len(unseen) != 3 {
		t.Errorf("codex has seen nothing, should replay 3, got %d", len(unseen))
	}
	if unseen, err = s.Unseen(ctx, c.ID, "claude"); err != nil {
		t.Fatal(err)
	}
	if len(unseen) != 1 || unseen[0].Text != "third question" {
		t.Errorf("claude should owe exactly the new message, got %d: %+v", len(unseen), unseen)
	}
}

// seen_seq must never go backwards. A late or out-of-order update that rewound
// it would silently re-send everything after that point, which is the expensive
// failure this design exists to avoid.
func TestSeenSeqOnlyMovesForward(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)
	for i := 0; i < 4; i++ {
		if _, err := s.AppendUser(ctx, c.ID, "msg"); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.MarkSeen(ctx, c.ID, "agy", "sess", 4); err != nil {
		t.Fatal(err)
	}
	// A stale update arrives claiming less progress.
	if err := s.MarkSeen(ctx, c.ID, "agy", "", 1); err != nil {
		t.Fatal(err)
	}

	unseen, err := s.Unseen(ctx, c.ID, "agy")
	if err != nil {
		t.Fatal(err)
	}
	if len(unseen) != 0 {
		t.Errorf("a stale update rewound seen_seq: %d entries now owed", len(unseen))
	}
	// And the session id survives an update that does not carry one.
	if got := s.ResumeID(ctx, c.ID, "agy"); got != "sess" {
		t.Errorf("resume id = %q, want it preserved across a blank update", got)
	}
}

// A replay marker is a note about the conversation, not part of it. Sending it
// to a provider would be telling the agent about our own UI.
func TestUnseenExcludesReplayMarkers(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	if _, err := s.AppendUser(ctx, c.ID, "a question"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendReplay(ctx, c.ID, "claude",
		protocol.Model{ID: "m", Label: "M"}, 1, 42); err != nil {
		t.Fatal(err)
	}

	unseen, err := s.Unseen(ctx, c.ID, "claude")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range unseen {
		if e.Role == "replay" {
			t.Fatal("a replay marker must never be replayed to a provider")
		}
	}
	if len(unseen) != 1 {
		t.Errorf("want just the user message, got %d", len(unseen))
	}
}

// Usage is omitted entirely until a turn reports some — "0 tokens" would be a
// claim we cannot support, and zero cost is not the same as free.
func TestAgentUsageOmittedUntilReported(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	placeholder, err := s.AppendAgentPlaceholder(ctx, c.ID, "codex",
		protocol.Model{ID: "m", Label: "M"})
	if err != nil {
		t.Fatal(err)
	}
	if placeholder.Usage != nil {
		t.Error("an unfinished turn must report no usage at all")
	}

	// codex reports tokens but never currency.
	done, err := s.FinishAgent(ctx, placeholder.ID, "answer", 1234, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if done.Usage == nil || done.Usage.Tokens != 1234 {
		t.Fatalf("usage = %+v, want 1234 tokens", done.Usage)
	}
	if done.Usage.Usd != nil {
		t.Error("zero spend must stay absent, not render as $0.00 — only claude reports currency")
	}

	// claude does, and it should come through as real dollars.
	withCost, err := s.FinishAgent(ctx, placeholder.ID, "answer", 1234, 363094500, "")
	if err != nil {
		t.Fatal(err)
	}
	if withCost.Usage.Usd == nil {
		t.Fatal("a reported cost must survive")
	}
	if got := *withCost.Usage.Usd; got < 0.0363 || got > 0.0364 {
		t.Errorf("usd = %v, want ~0.03630945", got)
	}
}

// A partial answer is kept when a turn dies. Deleting it would hide what went
// wrong and throw away text the owner may still want.
func TestFailedTurnKeepsItsPartialText(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	e, err := s.AppendAgentPlaceholder(ctx, c.ID, "claude", protocol.Model{ID: "m", Label: "M"})
	if err != nil {
		t.Fatal(err)
	}
	done, err := s.FinishAgent(ctx, e.ID, "got this far", 0, 0, "connection lost")
	if err != nil {
		t.Fatal(err)
	}
	if done.Text != "got this far" {
		t.Errorf("partial text = %q, want it kept", done.Text)
	}
	if done.Failure != "connection lost" {
		t.Errorf("failure = %q", done.Failure)
	}
}
