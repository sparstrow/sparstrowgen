-- name: CountUsers :one
-- How many accounts exist. Tests use it to assert a flow created exactly the
-- accounts it should have.
SELECT count(*) FROM users;

-- name: CreateUser :one
-- users_email_key is the constraint that can refuse this: the same address
-- twice. Any number of people may have accounts (migrations/00009); who may
-- create one is decided before this runs, by the invitation list.
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- Signing in, holding the row against a password change.
--
-- FOR SHARE rather than a plain read, and it matters. Verifying a password
-- takes ~50ms of argon2, and without a lock a sign-in can read the OLD hash,
-- spend that time, and insert its session AFTER a password change has deleted
-- every session — so the password the owner just revoked still produces a
-- working login. FOR SHARE does not block other sign-ins (they share it); it
-- blocks only GetUserForChange below, which is exactly the conflict.
-- name: GetUserByEmailForSignIn :one
SELECT * FROM users WHERE email = $1 FOR SHARE;

-- Changing the password, excluding everything else on this row.
--
-- FOR UPDATE waits for any sign-in holding FOR SHARE to finish, so the sessions
-- this change is about to delete include that one. It also serialises two
-- simultaneous changes, which would otherwise both verify against the same old
-- hash and one would silently overwrite the other.
-- name: GetUserForChange :one
SELECT * FROM users WHERE id = $1 FOR UPDATE;

-- Resetting a forgotten password: the same lock as GetUserForChange, found by
-- the address the reset link was sent to rather than by a session.
-- name: GetUserByEmailForChange :one
SELECT * FROM users WHERE email = $1 FOR UPDATE;

-- name: SetUserPassword :one
UPDATE users SET password_hash = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserEmail :one
UPDATE users SET email = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- Tests only, all six. A suite needs a way back to "nobody has an account",
-- and conversations reference users WITHOUT a cascade (D-009), so their rows go
-- first. Sessions go with their users by ON DELETE CASCADE. Links and access
-- requests are keyed by address rather than by account, so they are emptied
-- explicitly — a request left from an earlier run would stop the next run's
-- first request from being the first.

-- name: DeleteEveryEmailLink :exec
DELETE FROM email_links;

-- name: DeleteEveryAccessRequest :exec
DELETE FROM access_requests;

-- name: DeleteEveryEntry :exec
DELETE FROM entries;

-- name: DeleteEveryProviderSession :exec
DELETE FROM provider_sessions;

-- name: DeleteEveryConversation :exec
DELETE FROM conversations;

-- name: DeleteEveryUser :execrows
DELETE FROM users;

-- name: SetUserAppearance :one
-- The whole appearance in one statement: a save always carries all three, so a
-- tab holding stale values cannot merge half of them into the account.
UPDATE users
SET appearance_mode = $2, appearance_surface = $3, appearance_accent = $4, updated_at = now()
WHERE id = $1
RETURNING *;
