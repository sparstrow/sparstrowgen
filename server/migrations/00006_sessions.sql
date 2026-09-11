-- +goose Up
-- Logged-in sessions.
--
-- Opaque random tokens rather than signed ones (a JWT), because the thing behind
-- this login can run a coding agent on the owner's machine — and the question
-- that matters for that is not "is this token well-formed" but "should this
-- token still work right now". A signed token cannot be answered no. A row can
-- be deleted.
--
-- What is stored is the SHA-256 of the token, never the token. A dump of this
-- table, or a leaked backup, is then a list of hashes rather than a list of
-- working logins. SHA-256 and not argon2 deliberately: the input is 32 bytes of
-- cryptographic randomness, so there is no dictionary to slow an attacker down
-- to, and the lookup happens on every request.
--
-- No user_id column. There is one owner, and inventing a users table for a
-- second person who does not exist would be a speculative abstraction
-- (AGENTS.md rule 4). Adding one later is a migration, and a small one.
CREATE TABLE sessions (
    token_hash   bytea       PRIMARY KEY,
    created_at   timestamptz NOT NULL DEFAULT now(),
    -- Touched on use, so an idle session can be expired separately from an old
    -- one. A session used daily for a month is not the same risk as one
    -- abandoned in a browser a month ago.
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    -- Context for the owner reading his own session list, never for a decision:
    -- both are self-reported by the client and neither can be trusted as
    -- identity.
    user_agent   text        NOT NULL DEFAULT '',
    ip           text        NOT NULL DEFAULT ''
);

-- Sweeping expired rows, and nothing else, so it stays narrow.
CREATE INDEX sessions_expires_idx ON sessions (expires_at);

-- +goose Down
DROP TABLE sessions;
