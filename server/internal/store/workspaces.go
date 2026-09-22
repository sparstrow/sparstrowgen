package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/sparstrow/sparstrowgen/server/internal/db"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* Workspaces: separate areas of work inside one account (docs/Decisions.md
D-050). A conversation belongs to a workspace, and an account reaches a
workspace by being a member of it.

Membership is checked in SQL, in the same statement that does the work, rather
than by a middleware that resolves a workspace from whichever header the caller
chose to send. Multica does the latter and its own code records what that cost
(MUL-2600, an agent widening its scope by naming a different workspace). */

// MaxWorkspaceName is a label in a switcher, not prose. Long enough for "Client
// work — Contoso", short enough that the rail is not the thing it breaks.
const MaxWorkspaceName = 60

// ErrNoWorkspace is a workspace this account is not a member of, and one that
// never existed. The same answer on purpose, for the same reason ErrNotFound is
// (G-27).
var ErrNoWorkspace = errors.New("that workspace does not exist")

// ErrWorkspaceName is a name that is empty once trimmed, or too long.
var ErrWorkspaceName = fmt.Errorf("a workspace name is required, and can be at most %d characters", MaxWorkspaceName)

// The only roles there are. Everything written today is RoleOwner, because
// nothing can invite anybody yet; RoleMember is what an invitation would write,
// and the check constraint in 00017 is the authority on both.
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

func cleanWorkspaceName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxWorkspaceName {
		return "", ErrWorkspaceName
	}
	return name, nil
}

func toWorkspace(id, name, role string) protocol.Workspace {
	return protocol.Workspace{ID: id, Name: name, Role: role}
}

// Workspaces lists every workspace an account can reach.
//
// An account with none is a real state rather than an error: it is what a brand
// new account is, and what first-run setup's second step exists to end.
func (s *Store) Workspaces(ctx context.Context, userID string) ([]protocol.Workspace, error) {
	owner, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("account id: %w", err)
	}
	rows, err := s.q.ListWorkspaces(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.Workspace, 0, len(rows))
	for _, r := range rows {
		out = append(out, toWorkspace(uuidToString(r.ID), r.Name, r.Role))
	}
	return out, nil
}

// Workspace is the membership check every route that names a workspace runs
// before it reads anything. It answers ErrNoWorkspace for a workspace that is
// not this account's, which is what turns a guessed id into a plain 404.
func (s *Store) Workspace(ctx context.Context, userID, id string) (protocol.Workspace, error) {
	owner, err := parseUUID(userID)
	if err != nil {
		return protocol.Workspace{}, fmt.Errorf("account id: %w", err)
	}
	// A malformed workspace id is simply not one of this account's, exactly as
	// a malformed conversation id is not one of its conversations.
	wid, err := parseUUID(id)
	if err != nil {
		return protocol.Workspace{}, ErrNoWorkspace
	}
	row, err := s.q.GetWorkspace(ctx, db.GetWorkspaceParams{ID: wid, UserID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return protocol.Workspace{}, ErrNoWorkspace
	}
	if err != nil {
		return protocol.Workspace{}, err
	}
	return toWorkspace(uuidToString(row.ID), row.Name, row.Role), nil
}

// CreateWorkspace makes one and puts the account in it as owner.
//
// Both statements or neither: a workspace with no members is reachable by
// nothing in this system, because every statement that can find one joins
// membership. Committing the first without the second would leave a row that
// only a migration could ever see again.
func (s *Store) CreateWorkspace(ctx context.Context, userID, name string) (protocol.Workspace, error) {
	owner, err := parseUUID(userID)
	if err != nil {
		return protocol.Workspace{}, fmt.Errorf("account id: %w", err)
	}
	name, err = cleanWorkspaceName(name)
	if err != nil {
		return protocol.Workspace{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return protocol.Workspace{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)

	row, err := q.CreateWorkspace(ctx, name)
	if err != nil {
		return protocol.Workspace{}, err
	}
	if err := q.AddWorkspaceMember(ctx, db.AddWorkspaceMemberParams{
		WorkspaceID: row.ID, UserID: owner, Role: RoleOwner,
	}); err != nil {
		return protocol.Workspace{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return protocol.Workspace{}, err
	}
	return toWorkspace(uuidToString(row.ID), row.Name, RoleOwner), nil
}

// RenameWorkspace is the only edit a workspace has. Owners only — the statement
// says so, not this function, so a member cannot rename one by reaching a
// different code path.
func (s *Store) RenameWorkspace(ctx context.Context, userID, id, name string) (protocol.Workspace, error) {
	owner, err := parseUUID(userID)
	if err != nil {
		return protocol.Workspace{}, fmt.Errorf("account id: %w", err)
	}
	wid, err := parseUUID(id)
	if err != nil {
		return protocol.Workspace{}, ErrNoWorkspace
	}
	name, err = cleanWorkspaceName(name)
	if err != nil {
		return protocol.Workspace{}, err
	}
	row, err := s.q.RenameWorkspace(ctx, db.RenameWorkspaceParams{ID: wid, UserID: owner, Name: name})
	if errors.Is(err, pgx.ErrNoRows) {
		return protocol.Workspace{}, ErrNoWorkspace
	}
	if err != nil {
		return protocol.Workspace{}, err
	}
	return toWorkspace(uuidToString(row.ID), row.Name, RoleOwner), nil
}
