// Package devstack gives each worktree its own local stack: its own database
// port and volume, its own server and web ports, its own daemon home and token.
//
// WORKFLOW.md requires this before `develop` opens to parallel work, and says
// why in one line: manually reusing the repository defaults is not isolation.
// Two agents running `make db` today get one container named sparstrowgen-db on
// one host port with one volume, so the second one either fails to start or
// quietly joins the first one's database. Their daemons are worse: a
// development build keeps its credential in one shared `sparstrowgen-dev`
// directory, so pairing one worktree's daemon can delete the other's
// credential.
//
// Assignments are recorded rather than derived, because a port is only free if
// nothing else on the machine holds it — including software that has nothing to
// do with this repository. The registry lives in the user's data directory, not
// in the checkout: its whole job is to stop two checkouts colliding, which a
// file inside one of them could not do.
package devstack

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Where each service starts counting. Slot 0 is exactly today's stack — 5433,
// 8080, 3000 — so the owner's own checkout keeps the ports every document and
// habit already names, and nothing about his machine changes.
const (
	dbBase  = 5433
	apiBase = 8080
	webBase = 3000

	// Enough for far more agents than will ever run at once, and small enough
	// that a runaway loop fails instead of scanning every port on the machine.
	maxSlots = 50
)

// Stack is one worktree's reserved local environment.
//
// Only what was ASSIGNED is stored. Everything else — the compose project, the
// daemon token, the database URL — is derived from the slot, so a registry
// nobody edits cannot disagree with the values the scripts actually use.
type Stack struct {
	Slot     int       `json:"slot"`
	Worktree string    `json:"worktree"`
	DBPort   int       `json:"dbPort"`
	APIPort  int       `json:"apiPort"`
	WebPort  int       `json:"webPort"`
	Reserved time.Time `json:"reserved"`
}

// Project is the Compose project name, which is what separates one stack's
// database volume from another's.
//
// Slot 0 keeps the bare name, because that is the project the owner's existing
// local database already lives under (Compose defaults it to the directory
// name). Renaming it would not delete his data, but it would hide it, which
// reads exactly like deleting it.
func (s Stack) Project() string {
	if s.Slot == 0 {
		return "sparstrowgen"
	}
	return fmt.Sprintf("sparstrowgen-%d", s.Slot)
}

// Token is the development daemon token for this stack.
//
// NOT A SECRET — it is in a public repository and only ever unlocks a server on
// localhost, the same as the one in scripts/dev.ps1 it replaces. Slot 0 keeps
// that exact string so the owner's checkout is unchanged in every respect.
func (s Stack) Token() string {
	if s.Slot == 0 {
		return "dev-daemon-token-not-a-secret-0123456789"
	}
	return fmt.Sprintf("dev-daemon-token-not-a-secret-slot-%d", s.Slot)
}

// Home is SPARSTROWGEN_HOME for this stack's daemon: where it keeps the
// credential that identifies it as a computer.
//
// This is the assignment that matters most. Without it every development daemon
// on the machine shares one directory, so two worktrees are one computer as far
// as the server is concerned, and a refused pairing in either deletes the
// credential both were using.
func (s Stack) Home() (string, error) {
	base, err := dataRoot()
	if err != nil {
		return "", err
	}
	if s.Slot == 0 {
		// What a development build already uses (cmd/daemon's dataDir).
		return filepath.Join(base, "sparstrowgen-dev"), nil
	}
	return filepath.Join(base, fmt.Sprintf("sparstrowgen-dev-%d", s.Slot)), nil
}

// DatabaseURL is the development connection string. The credentials are the
// ones compose.dev.yaml starts Postgres with, and are not secrets either.
func (s Stack) DatabaseURL() string {
	return fmt.Sprintf("postgres://sparstrowgen:sparstrowgen@localhost:%d/sparstrowgen?sslmode=disable", s.DBPort)
}

// Var is one environment variable of a stack, in the order a reader wants them.
type Var struct{ Key, Value string }

// Env is everything a shell needs to run this stack: the whole set, so no
// caller has to remember which variable belongs to which process.
func (s Stack) Env() ([]Var, error) {
	home, err := s.Home()
	if err != nil {
		return nil, err
	}
	api := fmt.Sprintf("http://localhost:%d", s.APIPort)
	return []Var{
		{"SPARSTROWGEN_SLOT", fmt.Sprintf("%d", s.Slot)},
		{"COMPOSE_PROJECT_NAME", s.Project()},
		{"SPARSTROWGEN_DB_PORT", fmt.Sprintf("%d", s.DBPort)},
		{"DATABASE_URL", s.DatabaseURL()},
		// goose reads its own two variables rather than DATABASE_URL.
		{"GOOSE_DRIVER", "postgres"},
		{"GOOSE_DBSTRING", s.DatabaseURL()},
		{"ADDR", fmt.Sprintf(":%d", s.APIPort)},
		{"PORT", fmt.Sprintf("%d", s.WebPort)},
		{"WEB_ORIGIN", fmt.Sprintf("http://localhost:%d", s.WebPort)},
		{"NEXT_PUBLIC_API_URL", api},
		{"SERVER_API", api},
		{"SERVER_WS", fmt.Sprintf("ws://localhost:%d/daemon", s.APIPort)},
		{"DAEMON_TOKEN", s.Token()},
		{"SPARSTROWGEN_HOME", home},
		{"SESSION_SECURE", "false"},
	}, nil
}

// ---------------------------------------------------------------------------
// the registry
// ---------------------------------------------------------------------------

// Reserve returns this worktree's stack, assigning one the first time it asks.
//
// It is safe to call from two agents at once, and it is meant to be called
// every time rather than once: the answer is stable, so scripts can start with
// it instead of remembering it.
func Reserve(worktree string) (Stack, error) {
	worktree, err := clean(worktree)
	if err != nil {
		return Stack{}, err
	}
	var out Stack
	err = withRegistry(func(reg *registry) error {
		if existing, ok := reg.find(worktree); ok {
			out = existing
			return nil
		}
		s, err := reg.assign(worktree)
		if err != nil {
			return err
		}
		out = s
		reg.Stacks = append(reg.Stacks, s)
		return nil
	})
	return out, err
}

// Release gives a worktree's slot back, for when it is deleted. Releasing one
// that was never reserved is not an error: the point is that it is gone.
func Release(worktree string) error {
	worktree, err := clean(worktree)
	if err != nil {
		return err
	}
	return withRegistry(func(reg *registry) error {
		kept := reg.Stacks[:0]
		for _, s := range reg.Stacks {
			if s.Worktree != worktree {
				kept = append(kept, s)
			}
		}
		reg.Stacks = kept
		return nil
	})
}

// List is every reserved stack, lowest slot first.
func List() ([]Stack, error) {
	var out []Stack
	err := withRegistry(func(reg *registry) error {
		out = append(out, reg.Stacks...)
		sort.Slice(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
		return nil
	})
	return out, err
}

type registry struct {
	Stacks []Stack `json:"stacks"`
}

func (r *registry) find(worktree string) (Stack, bool) {
	for _, s := range r.Stacks {
		if s.Worktree == worktree {
			return s, true
		}
	}
	return Stack{}, false
}

// assign picks the lowest slot whose three ports are both unclaimed by another
// stack and actually free on this machine right now.
//
// Slot 0 is held for the main checkout. A linked worktree taking it would put
// the owner's own stack on different ports the next time he ran it, which looks
// from his side like his local database emptying itself.
func (r *registry) assign(worktree string) (Stack, error) {
	taken := map[int]bool{}
	for _, s := range r.Stacks {
		taken[s.DBPort], taken[s.APIPort], taken[s.WebPort] = true, true, true
	}
	first := 0
	if !isMainCheckout(worktree) {
		first = 1
	}
	for slot := first; slot < maxSlots; slot++ {
		if r.slotUsed(slot) {
			continue
		}
		db, api, web := dbBase+slot, apiBase+slot, webBase+slot
		if taken[db] || taken[api] || taken[web] {
			continue
		}
		if !free(db) || !free(api) || !free(web) {
			continue
		}
		return Stack{
			Slot: slot, Worktree: worktree,
			DBPort: db, APIPort: api, WebPort: web,
			Reserved: time.Now().UTC().Truncate(time.Second),
		}, nil
	}
	return Stack{}, fmt.Errorf("no free local stack: %d slots are in use or their ports are busy", maxSlots)
}

func (r *registry) slotUsed(slot int) bool {
	for _, s := range r.Stacks {
		if s.Slot == slot {
			return true
		}
	}
	return false
}

// free reports whether this machine will let a server listen on a port now.
// Probed rather than assumed, because the machine runs plenty of software this
// repository knows nothing about — 5433 was itself chosen to dodge a local
// Postgres on 5432.
func free(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

// isMainCheckout is true for the repository's own working tree and false for a
// linked worktree, which git marks by making .git a FILE holding a gitdir line
// rather than the directory it is in a normal checkout. Read off the filesystem
// rather than by running git, so reserving a stack never depends on a binary
// being on PATH.
func isMainCheckout(worktree string) bool {
	info, err := os.Stat(filepath.Join(worktree, ".git"))
	return err == nil && info.IsDir()
}

func clean(worktree string) (string, error) {
	if worktree == "" {
		return "", errors.New("which worktree? give it a path")
	}
	abs, err := filepath.Abs(worktree)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if runtime.GOOS == "windows" {
		// D:\x and d:\X are one directory, and would otherwise be two entries
		// holding two slots for the same checkout.
		abs = strings.ToLower(abs)
	}
	return abs, nil
}

// ---------------------------------------------------------------------------
// the file
// ---------------------------------------------------------------------------

// RegistryPath is where the assignments are recorded. SPARSTROWGEN_DEVSTACKS
// overrides it, which is how the tests avoid touching the real one.
func RegistryPath() (string, error) {
	if v := os.Getenv("SPARSTROWGEN_DEVSTACKS"); v != "" {
		return v, nil
	}
	base, err := dataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sparstrowgen-dev", "stacks.json"), nil
}

func dataRoot() (string, error) {
	if runtime.GOOS == "windows" {
		return os.UserCacheDir() // %LOCALAPPDATA%, matching cmd/daemon's dataDir
	}
	return os.UserConfigDir()
}

// withRegistry reads the registry, hands it to fn, and writes it back if fn
// changed it — all while holding a lock, because two agents starting at the
// same moment is the normal case this exists for, not an unlucky one.
func withRegistry(fn func(*registry) error) error {
	path, err := RegistryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	unlock, err := lock(path + ".lock")
	if err != nil {
		return err
	}
	defer unlock()

	before, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	reg := &registry{}
	if len(before) > 0 {
		if err := json.Unmarshal(before, reg); err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
	}
	if err := fn(reg); err != nil {
		return err
	}
	after, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	after = append(after, '\n')
	if string(after) == string(before) {
		return nil
	}
	// Written beside the file and renamed, so a process killed mid-write leaves
	// the previous registry intact rather than a truncated one that would cost
	// every worktree its assignment.
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(tmp, after, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// lockStale is how long a lock file may exist before it is assumed to belong to
// a process that died holding it. Nothing here does real work under the lock —
// reading a small file, probing three ports — so a lock older than this is not
// somebody being slow.
const lockStale = 30 * time.Second

func lock(path string) (func(), error) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > lockStale {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("another sparstrowgen stack is being reserved and did not finish; if nothing is running, delete %s", path)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
