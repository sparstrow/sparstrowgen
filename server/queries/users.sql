-- name: CountUsers :one
-- Whether the app has been claimed yet. Sign-up is reachable only while this is
-- zero, which is what stops a stranger who finds the URL from making themselves
-- the owner of a machine that runs agents.
SELECT count(*) FROM users;

-- name: CreateUser :one
-- The UNIQUE constraint on email is what makes this safe against two
-- simultaneous sign-ups: both may pass the count check, only one INSERT wins,
-- and the loser gets a constraint violation rather than a second account.
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

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
