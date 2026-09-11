-- name: CreateSession :one
INSERT INTO sessions (token_hash, expires_at, user_agent, ip)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- Validating and touching a session are ONE statement on purpose. Read-then-
-- write would leave a window where a session deleted in between is still
-- accepted, and the whole point of storing sessions in rows is that deleting one
-- takes effect immediately.
--
-- Two expiries, both checked here: expires_at is how old a session may get,
-- idle_since is how long it may go unused. A row that fails either is simply not
-- returned, and the caller sees the same "not a valid session" as for a token
-- that never existed.
-- name: TouchSession :one
UPDATE sessions
SET last_seen_at = now()
WHERE token_hash = $1
  AND expires_at > now()
  AND last_seen_at > sqlc.arg(idle_since)
RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- Signing out everywhere. The reason this exists is that the owner may one day
-- need it in a hurry, and "delete the rows by hand in psql" is not a thing to
-- work out under pressure.
-- name: DeleteAllSessions :execrows
DELETE FROM sessions;

-- name: DeleteDeadSessions :execrows
DELETE FROM sessions
WHERE expires_at < now() OR last_seen_at < sqlc.arg(idle_since);

-- name: ListSessions :many
SELECT * FROM sessions ORDER BY last_seen_at DESC;
