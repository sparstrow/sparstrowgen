package devstack

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

/* These use a registry in a temporary directory, never the machine's own: the
   real one holds the assignments the owner's checkout and every running agent
   are using, and a test that rewrote it would move their ports underneath
   them. */

func useTempRegistry(t *testing.T) {
	t.Helper()
	t.Setenv("SPARSTROWGEN_DEVSTACKS", filepath.Join(t.TempDir(), "stacks.json"))
}

// mainCheckout is a directory shaped like the repository's own working tree:
// .git is a directory.
func mainCheckout(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// linkedWorktree is shaped like a git worktree: .git is a file naming the real
// git directory. This is the difference the allocator reads.
func linkedWorktree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: D:/repo/.git/worktrees/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The owner's own checkout keeps the ports it has always had, so every document
// that names 5433, 8080 or 3000 stays true and his local database — which
// Compose keeps under the project name — is still the one that opens.
func TestTheMainCheckoutKeepsTodaysStack(t *testing.T) {
	useTempRegistry(t)

	s, err := Reserve(mainCheckout(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.Slot != 0 {
		t.Errorf("slot = %d, want the main checkout on 0", s.Slot)
	}
	if s.DBPort != 5433 || s.APIPort != 8080 || s.WebPort != 3000 {
		t.Errorf("ports = %d/%d/%d, want 5433/8080/3000", s.DBPort, s.APIPort, s.WebPort)
	}
	if s.Project() != "sparstrowgen" {
		t.Errorf("project = %q, want the name his existing database volume is under", s.Project())
	}
}

// An agent's worktree must not take slot 0 even when it asks first. If it did,
// the owner's next `make db` would start an empty database on another port,
// which from his side is indistinguishable from losing his local data.
func TestAWorktreeNeverTakesTheMainCheckoutsSlot(t *testing.T) {
	useTempRegistry(t)

	agent, err := Reserve(linkedWorktree(t))
	if err != nil {
		t.Fatal(err)
	}
	if agent.Slot == 0 {
		t.Fatalf("a linked worktree took slot 0")
	}
	owner, err := Reserve(mainCheckout(t))
	if err != nil {
		t.Fatal(err)
	}
	if owner.Slot != 0 {
		t.Errorf("the main checkout got slot %d after a worktree reserved first", owner.Slot)
	}
}

// Reserving is what the scripts do every time they start, so it has to answer
// the same thing rather than hand out a new slot per run.
func TestReservingTwiceIsTheSameStack(t *testing.T) {
	useTempRegistry(t)
	dir := linkedWorktree(t)

	first, err := Reserve(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Reserve(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Errorf("second reservation = %+v, want the same as %+v", second, first)
	}
}

// The whole point: nothing two agents run can land in the same place.
func TestTwoWorktreesShareNothing(t *testing.T) {
	useTempRegistry(t)

	a, err := Reserve(linkedWorktree(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Reserve(linkedWorktree(t))
	if err != nil {
		t.Fatal(err)
	}
	if a.DBPort == b.DBPort || a.APIPort == b.APIPort || a.WebPort == b.WebPort {
		t.Errorf("ports collide: %+v and %+v", a, b)
	}
	if a.Project() == b.Project() {
		t.Errorf("both stacks are Compose project %q, so they share a database volume", a.Project())
	}
	if a.Token() == b.Token() {
		t.Errorf("both stacks use daemon token %q", a.Token())
	}
	homeA, err := a.Home()
	if err != nil {
		t.Fatal(err)
	}
	homeB, err := b.Home()
	if err != nil {
		t.Fatal(err)
	}
	if homeA == homeB {
		t.Errorf("both daemons keep their credential in %s, so they are one computer to the server", homeA)
	}
}

// A deleted worktree gives its slot back. Without this the registry would only
// ever grow, and a machine that had run fifty agents could not start a stack.
func TestReleasingAWorktreeFreesItsSlot(t *testing.T) {
	useTempRegistry(t)
	dir := linkedWorktree(t)

	first, err := Reserve(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := Release(dir); err != nil {
		t.Fatal(err)
	}
	again, err := Reserve(linkedWorktree(t))
	if err != nil {
		t.Fatal(err)
	}
	if again.Slot != first.Slot {
		t.Errorf("slot %d was not reused after it was released (got %d)", first.Slot, again.Slot)
	}
}

// Releasing something that was never reserved is not a failure — the caller
// wanted it gone, and it is gone.
func TestReleasingAnUnknownWorktreeIsFine(t *testing.T) {
	useTempRegistry(t)
	if err := Release(linkedWorktree(t)); err != nil {
		t.Errorf("Release on an unreserved worktree: %v", err)
	}
}

// A slot whose ports are busy is skipped, because the machine runs software
// this repository knows nothing about. Deriving the ports from the slot alone
// would hand out a port nothing can listen on.
func TestABusyPortIsSkipped(t *testing.T) {
	useTempRegistry(t)

	// Hold slot 1's API port the way any unrelated program would.
	held, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", apiBase+1))
	if err != nil {
		t.Skipf("could not hold port %d to run this test: %v", apiBase+1, err)
	}
	defer held.Close()

	s, err := Reserve(linkedWorktree(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.Slot == 1 {
		t.Errorf("slot 1 was assigned although port %d is in use", apiBase+1)
	}
}

// Two agents starting at the same moment is the normal case here, not an
// unlucky one: this is what the lock around the registry is for.
func TestReservingFromTwoPlacesAtOnceGivesTwoStacks(t *testing.T) {
	useTempRegistry(t)

	dirs := []string{linkedWorktree(t), linkedWorktree(t), linkedWorktree(t), linkedWorktree(t)}
	stacks := make([]Stack, len(dirs))
	errs := make([]error, len(dirs))
	var wg sync.WaitGroup
	for i, dir := range dirs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stacks[i], errs[i] = Reserve(dir)
		}()
	}
	wg.Wait()

	seen := map[int]string{}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("reserve %d: %v", i, err)
		}
		if other, clash := seen[stacks[i].Slot]; clash {
			t.Errorf("slot %d was handed to both %s and %s", stacks[i].Slot, other, dirs[i])
		}
		seen[stacks[i].Slot] = dirs[i]
	}
}

// The environment is the product: a script that exports this set has a whole
// stack, with nothing left pointing at another worktree's.
func TestTheEnvironmentCarriesTheAssignment(t *testing.T) {
	useTempRegistry(t)

	s, err := Reserve(linkedWorktree(t))
	if err != nil {
		t.Fatal(err)
	}
	vars, err := s.Env()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, v := range vars {
		got[v.Key] = v.Value
	}
	want := map[string]string{
		"ADDR":                 fmt.Sprintf(":%d", s.APIPort),
		"PORT":                 fmt.Sprintf("%d", s.WebPort),
		"SERVER_WS":            fmt.Sprintf("ws://localhost:%d/daemon", s.APIPort),
		"SERVER_API":           fmt.Sprintf("http://localhost:%d", s.APIPort),
		"NEXT_PUBLIC_API_URL":  fmt.Sprintf("http://localhost:%d", s.APIPort),
		"WEB_ORIGIN":           fmt.Sprintf("http://localhost:%d", s.WebPort),
		"COMPOSE_PROJECT_NAME": s.Project(),
		"DAEMON_TOKEN":         s.Token(),
		"DATABASE_URL":         s.DatabaseURL(),
		"GOOSE_DBSTRING":       s.DatabaseURL(),
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %q, want %q", key, got[key], value)
		}
	}
	if got["SPARSTROWGEN_HOME"] == "" {
		t.Error("SPARSTROWGEN_HOME is empty, so this daemon would share the shared development credential")
	}
}
