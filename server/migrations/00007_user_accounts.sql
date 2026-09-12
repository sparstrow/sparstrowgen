-- +goose Up
-- A real account, replacing a password in an environment variable.
--
-- What was there before worked and was hostile: one argon2id hash in
-- OWNER_PASSWORD_HASH, no identity, and changing your own password meant
-- editing the deployment and redeploying. The owner's words were that it was
-- the worst part of the app, and he was right — a config file with a form in
-- front of it is not a login.
--
-- Modelled on Reference/multica-main, which is the same shape of system: users
-- carry an email, and credentials that machines use are rows that can be
-- revoked rather than secrets baked into a deployment.

-- citext so Sri@example.com and sri@example.com are the same account. Doing it
-- with lower() in every query instead means the one query that forgets creates
-- a second account nobody can explain.
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         citext      NOT NULL UNIQUE,
    -- argon2id, in the PHC format the auth package reads. Never a password.
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Sessions belong to somebody now.
--
-- Added nullable and then filled, because the column has to exist before the
-- rows can point anywhere — but there are no rows worth keeping: every session
-- that exists right now was created by the env-var password that this migration
-- retires, and the person holding one should sign in again as themselves.
DELETE FROM sessions;
ALTER TABLE sessions
    ADD COLUMN user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE;

-- Deleting the account ends every session with it, enforced by the database
-- rather than by remembering to do it in Go.
CREATE INDEX sessions_user_idx ON sessions (user_id);

-- +goose Down
ALTER TABLE sessions DROP COLUMN user_id;
DROP TABLE users;
