package store

import (
	"context"
	"testing"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/testdb"
)

/* These need a real Postgres, because the rules being tested live in SQL: seq
   is allocated inside the INSERT, and seen_seq is clamped by a GREATEST in the
   upsert. Testing them against a fake would only test the fake.

   They SKIP rather than fail when no database is reachable, so `go test ./...`
   stays useful on a machine without Docker running. Start one with `make db`.

   The database is the TEST one, not the development one, and testdb creates it
   the first time. See that package for why. */

func testStore(t *testing.T) *Store {
	t.Helper()
	return New(testdb.Pool(t))
}

func newConversation(t *testing.T, s *Store) protocol.Conversation {
	t.Helper()
	ctx := context.Background()
	c, err := s.Create(ctx, storeOwner(t, s).ID, "D:\\test", "codex", protocol.Model{ID: "m1", Label: "M One"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Leave nothing behind: these run against the development database.
	t.Cleanup(func() { _ = s.Delete(context.Background(), storeOwner(t, s).ID, c.ID) })
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
	done, err := s.FinishTurn(ctx, TurnResult{ConversationID: c.ID, EntryID: placeholder.ID, Provider: "claude", Text: "answer", Tokens: 1234})
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
	withCost, err := s.FinishTurn(ctx, TurnResult{ConversationID: c.ID, EntryID: placeholder.ID, Provider: "claude", Text: "answer", Tokens: 1234, SpendTicks: 363094500})
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
// The transcript shows an agent's answer at the time it ARRIVED. The entry is
// created empty when the turn starts, so before finished_at existed a turn that
// took a minute and a half was shown at the same minute as the question that
// prompted it — which is what the owner saw with agy (docs/Bugs.md B-35).
func TestAnAgentTurnIsShownWhenItAnswered(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	e, err := s.AppendAgentPlaceholder(ctx, c.ID, "agy", protocol.Model{ID: "m", Label: "M"})
	if err != nil {
		t.Fatal(err)
	}
	// Stand in for a slow turn: the entry was opened nine minutes ago, and the
	// answer is arriving now.
	if _, err := s.pool.Exec(ctx,
		"UPDATE entries SET created_at = now() - interval '9 minutes' WHERE id = $1", e.ID); err != nil {
		t.Fatal(err)
	}
	started, err := s.Get(ctx, storeOwner(t, s).ID, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	beforeFinishing := started.Entries[len(started.Entries)-1].At

	done, err := s.FinishTurn(ctx, TurnResult{
		ConversationID: c.ID, EntryID: e.ID, Provider: "agy", Text: "Hey! How can I assist you today?",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Local().Format("15:04")
	if done.At != now {
		t.Errorf("finished turn shown at %q, want the time it answered (%q)", done.At, now)
	}
	if done.At == beforeFinishing {
		t.Errorf("finished turn still shown at %q, the time the turn started", beforeFinishing)
	}

	// And it stays that way when the conversation is read back, not only in the
	// row FinishTurn happens to return.
	after, err := s.Get(ctx, storeOwner(t, s).ID, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if last := after.Entries[len(after.Entries)-1]; last.At != now {
		t.Errorf("re-read shows %q, want %q", last.At, now)
	}
}

// A turn still running has nothing to show but when it started, and neither has
// any entry written before finished_at existed.
func TestATurnStillRunningIsShownFromWhenItStarted(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	e, err := s.AppendAgentPlaceholder(ctx, c.ID, "agy", protocol.Model{ID: "m", Label: "M"})
	if err != nil {
		t.Fatal(err)
	}
	if e.At != time.Now().Local().Format("15:04") {
		t.Errorf("running turn shown at %q, want the time it started", e.At)
	}
}

func TestFailedTurnKeepsItsPartialText(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	e, err := s.AppendAgentPlaceholder(ctx, c.ID, "claude", protocol.Model{ID: "m", Label: "M"})
	if err != nil {
		t.Fatal(err)
	}
	done, err := s.FinishTurn(ctx, TurnResult{ConversationID: c.ID, EntryID: e.ID, Provider: "claude", Text: "got this far", Failure: "connection lost"})
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

// A stopped turn is not a failed one, and the transcript has to be able to tell
// them apart: one says something went wrong, the other says the owner decided
// he had seen enough.
func TestStoppedTurnIsNotAFailure(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	e, err := s.AppendAgentPlaceholder(ctx, c.ID, "claude", protocol.Model{ID: "m", Label: "M"})
	if err != nil {
		t.Fatal(err)
	}
	done, err := s.FinishTurn(ctx, TurnResult{ConversationID: c.ID, EntryID: e.ID, Provider: "claude", Text: "half an answer", Stopped: true})
	if err != nil {
		t.Fatal(err)
	}
	if !done.Stopped {
		t.Error("stopped = false, want the stop recorded")
	}
	if done.Failure != "" {
		t.Errorf("failure = %q, want nothing: being stopped is not going wrong", done.Failure)
	}
	if done.Text != "half an answer" {
		t.Errorf("text = %q, want what arrived before the stop", done.Text)
	}

	// And it survives a reload, because that is the whole reason it is a column
	// rather than something the client remembers.
	reloaded, err := s.Get(ctx, storeOwner(t, s).ID, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, entry := range reloaded.Entries {
		if entry.ID == e.ID {
			found = true
			if !entry.Stopped {
				t.Error("stopped did not survive a reload")
			}
		}
	}
	if !found {
		t.Fatal("the stopped entry is not in the transcript")
	}
}

// A turn can be both: a CLI killed mid-answer often complains on its way out.
// Recording only one of the two would lose either the reason it ended or the
// detail of how it died.
func TestATurnCanBeStoppedAndAlsoReportAFailure(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	e, err := s.AppendAgentPlaceholder(ctx, c.ID, "claude", protocol.Model{ID: "m", Label: "M"})
	if err != nil {
		t.Fatal(err)
	}
	done, err := s.FinishTurn(ctx, TurnResult{ConversationID: c.ID, EntryID: e.ID, Provider: "claude", Failure: "signal: killed", Stopped: true})
	if err != nil {
		t.Fatal(err)
	}
	if !done.Stopped || done.Failure != "signal: killed" {
		t.Errorf("stopped = %v, failure = %q; want both kept", done.Stopped, done.Failure)
	}
}

// Finishing a turn also records that the provider has seen the whole
// transcript, and the two are one write.
//
// As two statements this was a real hazard (docs/KnownGaps.md G-18): anything
// dropping the provider's session in the gap had the drop undone by the marker
// that followed, and the next turn ran with no history at all.
func TestFinishingATurnMarksTheProviderSeenInTheSameBreath(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	if _, err := s.AppendUser(ctx, c.ID, "a question"); err != nil {
		t.Fatal(err)
	}
	e, err := s.AppendAgentPlaceholder(ctx, c.ID, "claude", protocol.Model{ID: "m", Label: "M"})
	if err != nil {
		t.Fatal(err)
	}

	unseen, err := s.Unseen(ctx, c.ID, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(unseen) == 0 {
		t.Fatal("a provider that has been told nothing should have everything unseen")
	}

	if _, err := s.FinishTurn(ctx, TurnResult{
		ConversationID: c.ID, EntryID: e.ID, Provider: "claude", Text: "an answer", Tokens: 10,
	}); err != nil {
		t.Fatal(err)
	}

	unseen, err = s.Unseen(ctx, c.ID, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(unseen) != 0 {
		t.Errorf("%d entries still unseen; finishing a turn must mark the provider caught up", len(unseen))
	}
}

// The consequence that matters: once the session is dropped, nothing re-marks it
// as seen behind our back. This is the sequence that produced B-7.
func TestDroppingASessionAfterATurnIsNotUndone(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	if _, err := s.AppendUser(ctx, c.ID, "a question"); err != nil {
		t.Fatal(err)
	}
	e, err := s.AppendAgentPlaceholder(ctx, c.ID, "claude", protocol.Model{ID: "m", Label: "M"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.FinishTurn(ctx, TurnResult{
		ConversationID: c.ID, EntryID: e.ID, Provider: "claude", Text: "an answer", Tokens: 10,
	}); err != nil {
		t.Fatal(err)
	}

	// Moving the conversation drops provider sessions, because a session built
	// in another directory is reasoning about the wrong tree.
	if _, err := s.SetFolder(ctx, storeOwner(t, s).ID, c.ID, "/projects/elsewhere"); err != nil {
		t.Fatal(err)
	}

	unseen, err := s.Unseen(ctx, c.ID, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(unseen) == 0 {
		t.Error("the session drop was undone: the next turn would run with no history")
	}
}

func TestSetFolderDropsProviderSessions(t *testing.T) {
	// The store stores a path, it never parses one, so the shape does not matter.
	const somewhereElse = "/projects/elsewhere"
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	if _, err := s.AppendUser(ctx, c.ID, "what does this repo do?"); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSeen(ctx, c.ID, "codex", "thread-abc", 1); err != nil {
		t.Fatal(err)
	}
	if got := s.ResumeID(ctx, c.ID, "codex"); got != "thread-abc" {
		t.Fatalf("resume id = %q before the move, want thread-abc", got)
	}

	moved, err := s.SetFolder(ctx, storeOwner(t, s).ID, c.ID, somewhereElse)
	if err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if moved.Folder != somewhereElse {
		t.Errorf("folder = %q", moved.Folder)
	}

	// A session recorded in the old directory is a dead pointer: claude keys
	// sessions by project directory and answers "No conversation found with
	// session ID" when resumed from anywhere else, which fails every later turn
	// in about a second.
	if got := s.ResumeID(ctx, c.ID, "codex"); got != "" {
		t.Errorf("resume id = %q after the move, want it dropped", got)
	}
	// seen_seq went with it, so the next message replays the transcript into a
	// fresh session rather than continuing one that knows a different tree.
	unseen, err := s.Unseen(ctx, c.ID, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if len(unseen) != 1 {
		t.Errorf("unseen = %d entries, want the whole transcript back (1)", len(unseen))
	}
}

func TestSetFolderToTheSamePlaceCostsNothing(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	if err := s.MarkSeen(ctx, c.ID, "codex", "thread-abc", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetFolder(ctx, storeOwner(t, s).ID, c.ID, c.Folder); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	// Re-choosing the folder a conversation is already in must not silently
	// charge a full replay on the next message.
	if got := s.ResumeID(ctx, c.ID, "codex"); got != "thread-abc" {
		t.Errorf("resume id = %q, want it untouched by a no-op move", got)
	}
}

// Naming only ever fills a blank, and the rule lives in the statement rather
// than in the caller — so a rename that lands between a send starting and its
// naming still wins. That ordering cannot be forced through the API, which is
// why it is asserted here, where NameFrom can be called directly on a
// conversation that already has a name.
func TestNamingNeverReplacesANameThatIsAlreadyThere(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	named, took, err := s.NameFrom(ctx, storeOwner(t, s).ID, c.ID, "the first thing said")
	if err != nil {
		t.Fatalf("name: %v", err)
	}
	if !took || named.Title != "the first thing said" {
		t.Fatalf("took=%v title=%q, want it named from the message", took, named.Title)
	}

	_, took, err = s.NameFrom(ctx, storeOwner(t, s).ID, c.ID, "something said later")
	if err != nil {
		t.Fatalf("second name: %v", err)
	}
	if took {
		t.Error("a conversation that already had a name was renamed by a message")
	}
	after, err := s.Get(ctx, storeOwner(t, s).ID, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Title != "the first thing said" {
		t.Errorf("title = %q, want the name it already had", after.Title)
	}
}

// A message with no words in it leaves the conversation unnamed rather than
// naming it something worse than nothing — and unnamed means the next message
// can still name it.
func TestAMessageWithNoWordsInItLeavesTheConversationUnnamed(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := newConversation(t, s)

	if _, took, err := s.NameFrom(ctx, storeOwner(t, s).ID, c.ID, "---\n***"); err != nil || took {
		t.Fatalf("took=%v err=%v, want it declined", took, err)
	}
	if after, err := s.Get(ctx, storeOwner(t, s).ID, c.ID); err != nil || after.Title != "" {
		t.Fatalf("title = %q, want it still unnamed", after.Title)
	}
	if _, took, err := s.NameFrom(ctx, storeOwner(t, s).ID, c.ID, "and now a real one"); err != nil || !took {
		t.Fatalf("took=%v err=%v, want the next message to name it", took, err)
	}
}

// storeOwner is the account the conversation tests work in. Found or made on
// demand, because users_test empties the users table between its tests.
func storeOwner(t *testing.T, s *Store) User {
	t.Helper()
	ctx := context.Background()
	const email = "store-tests@sparstrow.test"
	u, ok, err := s.UserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		return u
	}
	u, err = s.CreateUser(ctx, email, "a-long-enough-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	return u
}
