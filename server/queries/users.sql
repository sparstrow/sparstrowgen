-- name: CountUsers :one
-- Whether the app has been claimed yet. Sign-up is reachable only while this is
-- zero, which is what stops a stranger who finds the URL from making themselves
-- the owner of a machine that runs agents.
SELECT count(*) FROM users;

-- name: CreateUser :one
-- Two constraints decide this, not the caller's earlier check:
--   users_email_key  two people claiming the same address
--   users_only_one   a SECOND account at all, whatever its address
-- The second is the one that matters here. Without it, simultaneous sign-ups
-- with different emails both pass the count check and both insert
-- (migrations/00008_one_account_only.sql).
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

-- name: SetUserPassword :one
UPDATE users SET password_hash = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserEmail :one
UPDATE users SET email = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteEveryUser :execrows
-- For tests: the app can be claimed once, so a suite needs a way back to
-- unclaimed. Sessions go with it, by ON DELETE CASCADE.
DELETE FROM users;
