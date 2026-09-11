-- name: ListConversations :many
SELECT * FROM conversations
ORDER BY updated_at DESC;

-- name: GetConversation :one
SELECT * FROM conversations WHERE id = $1;

-- name: CreateConversation :one
INSERT INTO conversations (title, folder, provider, model_id, model_label)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: RenameConversation :one
UPDATE conversations SET title = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetConversationArchived :one
UPDATE conversations SET archived = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetConversationProvider :one
UPDATE conversations
SET provider = $2, model_id = $3, model_label = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: AddConversationUsage :one
UPDATE conversations
SET tokens = tokens + $2, spend_ticks = spend_ticks + $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteConversation :exec
DELETE FROM conversations WHERE id = $1;

-- name: DeleteConversationEntries :exec
DELETE FROM entries WHERE conversation_id = $1;

-- name: DeleteConversationProviderSessions :exec
DELETE FROM provider_sessions WHERE conversation_id = $1;

-- name: SetConversationFolder :one
UPDATE conversations SET folder = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- Folders already in use, most recently touched first. The picker's shortcut
-- list, and the default for a new conversation — both come free from
-- conversations that already exist rather than from a preference to maintain.
-- name: RecentFolders :many
SELECT folder, max(updated_at) AS last_used
FROM conversations
GROUP BY folder
ORDER BY last_used DESC
LIMIT $1;
