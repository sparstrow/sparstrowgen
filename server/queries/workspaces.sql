-- A workspace is a separate area of work inside an account, and an account
-- reaches one by being a member of it (docs/Decisions.md D-050). Every
-- statement here joins that membership, for the same reason every conversation
-- statement does: an id is not permission.

-- The switcher's list, and the answer to "does this account have a workspace at
-- all" that first-run setup asks.
--
-- Ordered by name rather than by when it was made, because this is a list
-- somebody reads to find one of two or three things, and creation order is not
-- how anyone remembers them.
-- name: ListWorkspaces :many
SELECT w.*, m.role
FROM workspaces w
JOIN workspace_members m ON m.workspace_id = w.id
WHERE m.user_id = @user_id
ORDER BY w.name, w.created_at;

-- name: GetWorkspace :one
SELECT w.*, m.role
FROM workspaces w
JOIN workspace_members m ON m.workspace_id = w.id
WHERE w.id = @id AND m.user_id = @user_id;

-- name: CreateWorkspace :one
INSERT INTO workspaces (name) VALUES (@name) RETURNING *;

-- name: AddWorkspaceMember :exec
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES (@workspace_id, @user_id, @role);

-- Renaming is the only edit a workspace has. What it holds is edited through
-- the things it holds.
-- name: RenameWorkspace :one
UPDATE workspaces SET name = @name, updated_at = now()
WHERE id = @id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = workspaces.id AND m.user_id = @user_id AND m.role = 'owner'
  )
RETURNING *;

-- Which computers a workspace may use (00018). Approved and unrevoked only,
-- the same rule ListMachines applies: a computer waiting for approval belongs
-- to a pairing in progress, not to anybody's list.
--
-- Membership is joined here as everywhere else: this answers for an account
-- that is in the workspace, and for nobody else.
-- name: ListWorkspaceMachines :many
SELECT m.*
FROM machines m
JOIN workspace_machines wm ON wm.machine_id = m.id
WHERE wm.workspace_id = @workspace_id
  AND m.approved_at IS NOT NULL AND m.revoked_at IS NULL
  AND EXISTS (
      SELECT 1 FROM workspace_members mem
      WHERE mem.workspace_id = wm.workspace_id AND mem.user_id = @user_id
  )
ORDER BY m.created_at DESC;

-- The same fact from the other end: which workspaces offer this computer. The
-- owner asked for both directions in as many words.
-- name: ListMachineWorkspaces :many
SELECT w.id
FROM workspaces w
JOIN workspace_machines wm ON wm.workspace_id = w.id
WHERE wm.machine_id = @machine_id
  AND EXISTS (
      SELECT 1 FROM workspace_members mem
      WHERE mem.workspace_id = w.id AND mem.user_id = @user_id
  )
ORDER BY w.name;

-- Assigning. The SELECT is the permission check: the account has to be in the
-- workspace AND own the computer, so neither id alone is enough.
-- name: AddWorkspaceMachine :exec
INSERT INTO workspace_machines (workspace_id, machine_id)
SELECT @workspace_id, @machine_id
WHERE EXISTS (
    SELECT 1 FROM workspace_members mem
    WHERE mem.workspace_id = @workspace_id AND mem.user_id = @user_id
) AND EXISTS (
    SELECT 1 FROM machines m
    WHERE m.id = @machine_id AND m.user_id = @user_id
      AND m.approved_at IS NOT NULL AND m.revoked_at IS NULL
)
ON CONFLICT DO NOTHING;

-- name: RemoveWorkspaceMachine :exec
DELETE FROM workspace_machines
WHERE workspace_machines.workspace_id = @workspace_id
  AND workspace_machines.machine_id = @machine_id
  AND EXISTS (
      SELECT 1 FROM workspace_members mem
      WHERE mem.workspace_id = workspace_machines.workspace_id AND mem.user_id = @user_id
  );

-- Every workspace a new computer should appear in: all of this account's.
--
-- A computer that has just been paired and is offered nowhere would be a
-- computer that cannot run anything, and the person who just connected it has
-- no reason to expect a second step. Narrowing is the deliberate act, not
-- widening (00018's own reasoning).
-- name: AddMachineToEveryWorkspace :exec
INSERT INTO workspace_machines (workspace_id, machine_id)
SELECT mem.workspace_id, @machine_id
FROM workspace_members mem
WHERE mem.user_id = @user_id
ON CONFLICT DO NOTHING;

-- And its mirror: a workspace made after a computer was paired still gets it.
-- name: AddEveryMachineToWorkspace :exec
INSERT INTO workspace_machines (workspace_id, machine_id)
SELECT @workspace_id, m.id
FROM machines m
WHERE m.user_id = @user_id AND m.approved_at IS NOT NULL AND m.revoked_at IS NULL
ON CONFLICT DO NOTHING;
