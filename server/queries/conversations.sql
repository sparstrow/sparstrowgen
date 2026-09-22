-- Every statement a person's request can reach names the ACCOUNT as well as the
-- conversation. An id alone is not permission — ids travel in URLs, screenshots
-- and logs — and somebody else's conversation answers exactly like one that
-- never existed: no rows (docs/KnownGaps.md G-27).
--
-- Since D-050 a conversation belongs to a WORKSPACE, and an account reaches a
-- workspace by being a member of it. So "names the account" is now this clause,
-- repeated deliberately rather than hidden in a view:
--
--     EXISTS (SELECT 1 FROM workspace_members m
--             WHERE m.workspace_id = conversations.workspace_id
--               AND m.user_id = @user_id)
--
-- One statement, one round trip, and the same answer for a conversation in
-- somebody else's workspace as for one that never existed. Nothing resolves a
-- workspace from a header: the two statements that need one take it as a named
-- parameter, and the rest reach it through the row they were given.
--
-- The exceptions are the daemon's path (usage, and the entry and provider-session
-- queries in their own files). A turn is only ever started by postMessage after
-- the conversation was fetched for its owner, and the turn carries that owner.

-- name: ListConversations :many
SELECT * FROM conversations
WHERE conversations.workspace_id = @workspace_id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  )
ORDER BY updated_at DESC;

-- name: GetConversation :one
SELECT * FROM conversations
WHERE id = @id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  );

-- Deleting takes the row first, so its entries cannot be removed for a
-- conversation the caller cannot reach. Children go before the parent because
-- the foreign keys do not cascade (D-009).
-- name: LockConversation :one
SELECT id FROM conversations
WHERE id = @id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  )
FOR UPDATE;

-- A new conversation has no name. It gets one from the first thing said in it,
-- or from the owner typing one — never from a default that only looks like a
-- title.
--
-- The workspace is where it lives; user_id is who started it. In a workspace
-- with one member those are the same person, and in a shared one the second is
-- the fact worth keeping (D-050). The SELECT is what stops a conversation being
-- created in a workspace the account is not in: no membership, no row.
-- name: CreateConversation :one
INSERT INTO conversations (workspace_id, user_id, folder, provider, model_id, model_label)
SELECT @workspace_id, @user_id, @folder, @provider, @model_id, @model_label
WHERE EXISTS (
    SELECT 1 FROM workspace_members m
    WHERE m.workspace_id = @workspace_id AND m.user_id = @user_id
)
RETURNING *;

-- name: RenameConversation :one
UPDATE conversations SET title = @title, updated_at = now()
WHERE id = @id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  )
RETURNING *;

-- Naming, as opposed to renaming: this only ever fills a blank. The guard is in
-- the statement rather than in Go so that a name the owner typed can never be
-- overwritten by one derived from a message, whatever order the two arrive in.
-- No rows means it already had a name, which is an outcome and not an error.
-- name: NameConversation :one
UPDATE conversations SET title = @title
WHERE id = @id AND title IS NULL
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  )
RETURNING *;

-- name: SetConversationArchived :one
UPDATE conversations SET archived = @archived, updated_at = now()
WHERE id = @id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  )
RETURNING *;

-- name: SetConversationProvider :one
UPDATE conversations
SET provider = @provider, model_id = @model_id, model_label = @model_label, updated_at = now()
WHERE id = @id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  )
RETURNING *;

-- The daemon's path: a finished turn reporting what it cost.
-- name: AddConversationUsage :one
UPDATE conversations
SET tokens = tokens + $2, spend_ticks = spend_ticks + $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteConversation :exec
DELETE FROM conversations
WHERE id = @id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  );

-- name: DeleteConversationEntries :exec
DELETE FROM entries WHERE conversation_id = $1;

-- name: DeleteConversationProviderSessions :exec
DELETE FROM provider_sessions WHERE conversation_id = $1;

-- name: SetConversationFolder :one
UPDATE conversations SET folder = @folder, updated_at = now()
WHERE id = @id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  )
RETURNING *;

-- Folders already in use, most recently touched first. The picker's shortcut
-- list, and the default for a new conversation — both come free from
-- conversations that already exist rather than from a preference to maintain.
--
-- One workspace's folders, not one account's: the whole point of a second
-- workspace is that work's paths do not turn up while you are doing something
-- personal (D-050).
-- name: RecentFolders :many
SELECT folder, max(updated_at) AS last_used
FROM conversations
WHERE conversations.workspace_id = @workspace_id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = conversations.workspace_id AND m.user_id = @user_id
  )
GROUP BY folder
ORDER BY last_used DESC
LIMIT @lim;
