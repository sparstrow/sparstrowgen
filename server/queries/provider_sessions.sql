-- name: ListProviderSessions :many
SELECT * FROM provider_sessions WHERE conversation_id = $1;

-- name: GetProviderSession :one
SELECT * FROM provider_sessions WHERE conversation_id = $1 AND provider = $2;

-- name: UpsertProviderSeen :one
-- seen_seq only ever moves forward. A replay that lands out of order must not
-- rewind it, or the next switch would re-send entries the provider already has.
INSERT INTO provider_sessions (conversation_id, provider, seen_seq, session_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (conversation_id, provider) DO UPDATE
SET seen_seq   = GREATEST(provider_sessions.seen_seq, EXCLUDED.seen_seq),
    session_id = COALESCE(EXCLUDED.session_id, provider_sessions.session_id),
    updated_at = now()
RETURNING *;
