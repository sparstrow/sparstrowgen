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
	// And every computer the account already has, for the same reason a newly
	// paired one goes into every workspace (00018): a workspace that cannot run
	// anything the moment it is made is a workspace somebody has to go and
	// repair before using. Taking a computer out is the deliberate act.
	if err := q.AddEveryMachineToWorkspace(ctx, db.AddEveryMachineToWorkspaceParams{
		WorkspaceID: row.ID, UserID: owner,
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

// ---------------------------------------------------------------------------
// which computers a workspace may use
// ---------------------------------------------------------------------------

/* A computer belongs to the ACCOUNT and is offered to workspaces (00018). The
owner asked for both directions: "I need to choose which machine needs added to
that workspace or vice versa whick workspace needs to added to the machines."

Assignment is what a turn is routed by. Without that it would be a label, and a
label that looks like a setting is worse than no setting at all. */

// WorkspaceMachines is the computers this workspace may use.
//
// Empty is a real answer and a deliberate one — somebody took them all out —
// and the surfaces say so rather than quietly falling back to every computer
// the account has. A fallback would make the setting a suggestion.
func (s *Store) WorkspaceMachines(ctx context.Context, userID, workspaceID string) ([]Machine, error) {
	owner, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("account id: %w", err)
	}
	workspace, err := parseUUID(workspaceID)
	if err != nil {
		return nil, ErrNoWorkspace
	}
	rows, err := s.q.ListWorkspaceMachines(ctx, db.ListWorkspaceMachinesParams{
		WorkspaceID: workspace, UserID: owner,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Machine, 0, len(rows))
	for _, row := range rows {
		out = append(out, machineFrom(row))
	}
	return out, nil
}

// MachineWorkspaces is the same fact from the computer's end: which of this
// account's workspaces offer it.
func (s *Store) MachineWorkspaces(ctx context.Context, userID, machineID string) ([]string, error) {
	owner, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("account id: %w", err)
	}
	machine, err := parseUUID(machineID)
	if err != nil {
		return nil, ErrPairingUnavailable
	}
	rows, err := s.q.ListMachineWorkspaces(ctx, db.ListMachineWorkspacesParams{
		MachineID: machine, UserID: owner,
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, id := range rows {
		out = append(out, uuidToString(id))
	}
	return out, nil
}

// SetWorkspaceMachine adds or removes one assignment.
//
// Both statements check membership AND ownership of the computer, so neither id
// on its own is permission. Removing the last one is allowed: a workspace with
// no computer is a legitimate thing to want — reading old transcripts without
// being able to start anything — and refusing it here would be this code
// deciding what somebody's workspace is for.
func (s *Store) SetWorkspaceMachine(ctx context.Context, userID, workspaceID, machineID string, assigned bool) error {
	owner, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("account id: %w", err)
	}
	workspace, err := parseUUID(workspaceID)
	if err != nil {
		return ErrNoWorkspace
	}
	machine, err := parseUUID(machineID)
	if err != nil {
		return ErrPairingUnavailable
	}
	if assigned {
		return s.q.AddWorkspaceMachine(ctx, db.AddWorkspaceMachineParams{
			WorkspaceID: workspace, MachineID: machine, UserID: owner,
		})
	}
	return s.q.RemoveWorkspaceMachine(ctx, db.RemoveWorkspaceMachineParams{
		WorkspaceID: workspace, MachineID: machine, UserID: owner,
	})
}

// OfferMachineEverywhere puts a newly approved computer into every workspace
// the account has.
//
// A computer that has just been paired and is offered nowhere is a computer
// that cannot run anything, and somebody who just connected one has no reason
// to expect a second step in a settings page they have never opened. Narrowing
// is the deliberate act; widening is the default.
func (s *Store) OfferMachineEverywhere(ctx context.Context, userID, machineID string) error {
	owner, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("account id: %w", err)
	}
	machine, err := parseUUID(machineID)
	if err != nil {
		return ErrPairingUnavailable
	}
	return s.q.AddMachineToEveryWorkspace(ctx, db.AddMachineToEveryWorkspaceParams{
		MachineID: machine, UserID: owner,
	})
}
