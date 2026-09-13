-- Every statement a person's request can reach names the ACCOUNT as well as the
-- conversation. An id alone is not permission — ids travel in URLs, screenshots
-- and logs — and somebody else's conversation answers exactly like one that
-- never existed: no rows (docs/KnownGaps.md G-27).
--
-- The exceptions are the daemon's path (usage, and the entry and provider-session
-- queries in their own files). A turn is only ever started by postMessage after
-- the conversation was fetched for its owner, and the turn carries that owner.

-- name: ListConversations :many
SELECT * FROM conversations
WHERE user_id = $1
ORDER BY updated_at DESC;

-- name: GetConversation :one
SELECT * FROM conversations WHERE id = $1 AND user_id = $2;

-- Deleting takes the row first, so its entries cannot be removed for a
-- conversation the caller does not own. Children go before the parent because
-- the foreign keys do not cascade (D-009).
-- name: LockConversation :one
SELECT id FROM conversations WHERE id = $1 AND user_id = $2 FOR UPDATE;

-- A new conversation has no name. It gets one from the first thing said in it,
-- or from the owner typing one — never from a default that only looks like a
-- title.
-- name: CreateConversation :one
INSERT INTO conversations (user_id, folder, provider, model_id, model_label)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: RenameConversation :one
UPDATE conversations SET title = $3, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- Naming, as opposed to renaming: this only ever fills a blank. The guard is in
-- the statement rather than in Go so that a name the owner typed can never be
-- overwritten by one derived from a message, whatever order the two arrive in.
-- No rows means it already had a name, which is an outcome and not an error.
-- name: NameConversation :one
UPDATE conversations SET title = $3
WHERE id = $1 AND user_id = $2 AND title IS NULL
RETURNING *;

-- name: SetConversationArchived :one
UPDATE conversations SET archived = $3, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: SetConversationProvider :one
UPDATE conversations
SET provider = $3, model_id = $4, model_label = $5, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- The daemon's path: a finished turn reporting what it cost.
-- name: AddConversationUsage :one
UPDATE conversations
SET tokens = tokens + $2, spend_ticks = spend_ticks + $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteConversation :exec
DELETE FROM conversations WHERE id = $1 AND user_id = $2;

-- name: DeleteConversationEntries :exec
DELETE FROM entries WHERE conversation_id = $1;

-- name: DeleteConversationProviderSessions :exec
DELETE FROM provider_sessions WHERE conversation_id = $1;

-- name: SetConversationFolder :one
UPDATE conversations SET folder = $3, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- Folders already in use, most recently touched first. The picker's shortcut
-- list, and the default for a new conversation — both come free from
-- conversations that already exist rather than from a preference to maintain.
-- One account's folders only: another person's paths say where their work is.
-- name: RecentFolders :many
SELECT folder, max(updated_at) AS last_used
FROM conversations
WHERE user_id = $1
GROUP BY folder
ORDER BY last_used DESC
LIMIT $2;
