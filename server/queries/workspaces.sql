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
