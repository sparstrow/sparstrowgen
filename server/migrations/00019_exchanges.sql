-- +goose Up
-- Each agent turn's record: what the daemon handed the CLI, and every line the
-- CLI printed back (docs/specs/2026-09-23-raw-exchange.md).
--
-- Keyed by the agent entry, because a turn IS its agent entry (the turn id the
-- daemon is given is that entry's id). One row per turn for what was sent, and
-- one row per printed line: lines arrive a batch at a time while the turn runs,
-- and appending rows is cheap where rewriting one ever-growing value is not.
--
-- The environment the CLI ran with is deliberately not here. It carries the
-- sign-in token, and this record exists to be read back.
--
-- No cascades (D-009): deleting a conversation deletes these explicitly, lines
-- first, in the same transaction as its entries.
CREATE TABLE exchanges (
    entry_id          uuid        PRIMARY KEY REFERENCES entries (id),
    -- Redundant with the entry's, and kept so a conversation's records can be
    -- deleted without a join through entries.
    conversation_id   uuid        NOT NULL REFERENCES conversations (id),
    -- The resolved executable, or just the provider's name when it never started.
    program           text        NOT NULL,
    args              text[]      NOT NULL DEFAULT '{}',
    cwd               text        NOT NULL DEFAULT '',
    resume_session_id text        NOT NULL DEFAULT '',
    -- The prompt as the agent reads it, catch-up included.
    prompt            text        NOT NULL DEFAULT '',
    -- The bytes written to stdin, which for claude and agy is that prompt in an
    -- envelope.
    stdin             text        NOT NULL DEFAULT '',
    -- False when the daemon refused the turn before starting the CLI.
    launched          boolean     NOT NULL,
    -- What reached the daemon after the per-turn budget was spent and was not
    -- kept. Zero almost always; the record says so when it is not.
    dropped_lines     bigint      NOT NULL DEFAULT 0,
    dropped_bytes     bigint      NOT NULL DEFAULT 0,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX exchanges_conversation_idx ON exchanges (conversation_id);

CREATE TABLE exchange_lines (
    entry_id uuid    NOT NULL REFERENCES exchanges (entry_id),
    -- Order of arrival at the daemon, from 1.
    seq      integer NOT NULL,
    -- Milliseconds since the CLI started.
    at_ms    bigint  NOT NULL,
    stream   text    NOT NULL CHECK (stream IN ('stdout', 'stderr')),
    body     text    NOT NULL,
    PRIMARY KEY (entry_id, seq)
);

-- +goose Down
DROP TABLE exchange_lines;
DROP TABLE exchanges;
