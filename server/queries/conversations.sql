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
