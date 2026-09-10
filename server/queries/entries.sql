-- name: ListEntries :many
SELECT * FROM entries
WHERE conversation_id = $1
ORDER BY seq;

-- name: ListEntriesFrom :many
-- The replay payload: everything a provider has not been told yet.
SELECT * FROM entries
WHERE conversation_id = $1 AND seq > $2
ORDER BY seq;

-- name: CountEntries :one
SELECT COUNT(*) FROM entries WHERE conversation_id = $1;

-- name: AppendEntry :one
-- seq is allocated inside the statement so two concurrent appends cannot pick
-- the same number. The unique constraint would catch it; this avoids the retry.
INSERT INTO entries (
    conversation_id, seq, role, body,
    provider, model_id, model_label,
    tokens, spend_ticks, failure, messages_replayed
) VALUES (
    $1,
    (SELECT COALESCE(MAX(seq), 0) + 1 FROM entries WHERE conversation_id = $1),
    $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: FinishAgentEntry :one
UPDATE entries
SET body = $2, tokens = $3, spend_ticks = $4, failure = $5
WHERE id = $1
RETURNING *;

-- name: AppendEntryBody :one
-- Streaming deltas are accumulated in place rather than kept only in memory, so
-- a refresh mid-turn does not lose what has already arrived.
UPDATE entries
SET body = body || $2::text
WHERE id = $1
RETURNING *;
