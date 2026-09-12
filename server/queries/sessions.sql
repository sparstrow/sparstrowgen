-- name: CreateSession :one
INSERT INTO sessions (token_hash, user_id, expires_at, user_agent, ip)
VALUES ($1, $2, $3, $4, $5)
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
-- name: DeleteUserSessions :execrows
DELETE FROM sessions WHERE user_id = $1;

-- Signing out every OTHER device, which is what changing a password should do:
-- the point of the change is usually that somebody else may hold a session, and
-- ending your own at the same time would log you out of the screen you just
-- used to fix it.
-- name: DeleteOtherUserSessions :execrows
DELETE FROM sessions
WHERE user_id = $1 AND token_hash <> sqlc.arg(keep_token_hash);

-- name: DeleteDeadSessions :execrows
DELETE FROM sessions
WHERE expires_at < now() OR last_seen_at < sqlc.arg(idle_since);

-- name: ListUserSessions :many
SELECT * FROM sessions WHERE user_id = $1 ORDER BY last_seen_at DESC;
