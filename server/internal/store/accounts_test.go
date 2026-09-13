package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// uniqueAccount creates an account with an address no other test uses, so
// tests do not have to empty the users table to get one.
func uniqueAccount(t *testing.T, s *Store) User {
	t.Helper()
	email := fmt.Sprintf("t%d@sparstrow.test", time.Now().UnixNano())
	u, err := s.CreateUser(context.Background(), email, "a-long-enough-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// Somebody else's conversation answers exactly like one that does not exist,
// through every path a request can reach.
func TestAnotherAccountsConversationIsNotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	owner := uniqueAccount(t, s)
	other := uniqueAccount(t, s)

	c, err := s.Create(ctx, owner.ID, "D:\\owner\\project", "codex", protocol.Model{ID: "m1", Label: "M One"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Delete(context.Background(), owner.ID, c.ID) })
	if _, err := s.AppendUser(ctx, c.ID, "the owner's private plan"); err != nil {
		t.Fatal(err)
	}

	checks := map[string]error{}
	_, checks["Get"] = s.Get(ctx, other.ID, c.ID)
	_, checks["Rename"] = s.Rename(ctx, other.ID, c.ID, "taken over")
	_, checks["SetArchived"] = s.SetArchived(ctx, other.ID, c.ID, true)
	_, checks["SetFolder"] = s.SetFolder(ctx, other.ID, c.ID, "D:\\elsewhere")
	_, checks["SetProvider"] = s.SetProvider(ctx, other.ID, c.ID, "claude", protocol.Model{ID: "x", Label: "X"})
	checks["Delete"] = s.Delete(ctx, other.ID, c.ID)
	_, checks["Get with a malformed id"] = s.Get(ctx, owner.ID, "not-a-uuid")
	for name, err := range checks {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s by another account: err = %v, want ErrNotFound", name, err)
		}
	}

	if _, named, err := s.NameFrom(ctx, other.ID, c.ID, "a name from somebody else"); err != nil || named {
		t.Errorf("NameFrom by another account: named=%v err=%v, want neither", named, err)
	}

	for _, query := range []string{"", "private plan", "owner"} {
		list, err := s.List(ctx, other.ID, query)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 0 {
			t.Errorf("List(%q) for another account returned %d conversations", query, len(list))
		}
	}
	if folders, err := s.RecentFolders(ctx, other.ID, 8); err != nil || len(folders) != 0 {
		t.Errorf("another account's recent folders = %v (err %v), want none", folders, err)
	}

	// And the owner's conversation is exactly as it was.
	got, err := s.Get(ctx, owner.ID, c.ID)
	if err != nil {
		t.Fatalf("the owner lost their conversation: %v", err)
	}
	if got.Title != "" || got.Archived || got.Folder != "D:\\owner\\project" || got.Provider != "codex" || len(got.Entries) != 1 {
		t.Errorf("another account changed it: %+v", got)
	}
}

func TestIssuingALinkSupersedesTheOlderOne(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	email := fmt.Sprintf("link%d@sparstrow.test", time.Now().UnixNano())

	first, err := s.IssueLink(ctx, LinkVerify, email, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.IssueLink(ctx, LinkVerify, email, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	// A reset link is a different kind, and does not supersede a confirmation.
	if _, err := s.IssueLink(ctx, LinkReset, email, time.Hour); err != nil {
		t.Fatal(err)
	}

	if st, _ := s.InspectLink(ctx, LinkVerify, first); st.Usable || st.Reason != LinkSuperseded {
		t.Errorf("the first link = %+v, want superseded", st)
	}
	if st, _ := s.InspectLink(ctx, LinkVerify, second); !st.Usable || st.Email != email {
		t.Errorf("the newest link = %+v, want usable", st)
	}
	// A token presented as the wrong kind is not a link of that kind.
	if st, _ := s.InspectLink(ctx, LinkReset, second); st.Usable {
		t.Error("a confirmation token worked as a reset link")
	}
}

// Two tabs submitting the same link at the same moment: exactly one account.
func TestALinkCanOnlyBeSpentOnce(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	email := fmt.Sprintf("once%d@sparstrow.test", time.Now().UnixNano())
	token, err := s.IssueLink(ctx, LinkVerify, email, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.UserCount(ctx)
	if err != nil {
		t.Fatal(err)
	}

	const racers = 4
	results := make([]error, racers)
	var start, done sync.WaitGroup
	start.Add(1)
	for i := 0; i < racers; i++ {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait()
			_, _, _, results[i] = s.CompleteRegistration(ctx, token, "a-long-enough-passphrase", "test", "127.0.0.1")
		}(i)
	}
	start.Done()
	done.Wait()

	won := 0
	for i, err := range results {
		switch {
		case err == nil:
			won++
		case errors.Is(err, ErrLinkUnusable):
		default:
			t.Errorf("racer %d: unexpected %v", i, err)
		}
	}
	if won != 1 {
		t.Errorf("%d of %d registrations with one link succeeded; exactly one may", won, racers)
	}
	after, err := s.UserCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after-before != 1 {
		t.Errorf("%d accounts were created from one link", after-before)
	}
}

func TestResettingAPasswordEndsEverySessionAndStartsOne(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := uniqueAccount(t, s)

	oldA, _, err := s.StartSession(ctx, user.ID, "laptop", "10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	oldB, _, err := s.StartSession(ctx, user.ID, "phone", "10.0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.IssueLink(ctx, LinkReset, user.Email, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	_, fresh, _, err := s.ResetPassword(ctx, token, "a-brand-new-passphrase", "browser", "10.0.0.3")
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	for name, old := range map[string]string{"laptop": oldA, "phone": oldB} {
		if _, ok, _ := s.SessionUser(ctx, old); ok {
			t.Errorf("the %s session survived a password reset", name)
		}
	}
	if _, ok, _ := s.SessionUser(ctx, fresh); !ok {
		t.Error("the session started by the reset does not work")
	}
	if _, _, _, err := s.SignIn(ctx, user.Email, "a-brand-new-passphrase", "", ""); err != nil {
		t.Errorf("the new password does not sign in: %v", err)
	}
	if _, _, _, err := s.ResetPassword(ctx, token, "yet-another-passphrase", "", ""); !errors.Is(err, ErrLinkUnusable) {
		t.Errorf("the reset link worked twice: %v", err)
	}
}

func TestAccessRequestsCollapseIntoOne(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	email := fmt.Sprintf("Request%d@Contoso.test", time.Now().UnixNano())

	first, err := s.RecordAccessRequest(ctx, email)
	if err != nil || !first {
		t.Fatalf("first request: first=%v err=%v", first, err)
	}
	for i := 0; i < 2; i++ {
		// The same address with different capitalisation and spacing is the same
		// request.
		again, err := s.RecordAccessRequest(ctx, "  "+email+" ")
		if err != nil {
			t.Fatal(err)
		}
		if again {
			t.Error("a repeated request was reported as the first")
		}
	}
}
