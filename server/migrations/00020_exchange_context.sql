-- +goose Up
-- What each agent fed its model on a turn, as found in that agent's own
-- session store on the computer (docs/specs/2026-09-23-raw-exchange.md, US4;
-- docs/Capabilities.md, "What each agent loaded, word for word").
--
-- Records are stored once per conversation and turns point at them. They are
-- large (a claude prompt snapshot is over 100 KB) and nearly the same from one
-- turn to the next, so storing each turn's copy would multiply the database by
-- the number of turns for text that did not change. Once per CONVERSATION, not
-- once overall: deleting a conversation then deletes its records outright,
-- without asking whether another conversation, or another account, still
-- points at them.
--
-- No cascades (D-009): a conversation's delete removes these explicitly.
ALTER TABLE exchanges
    -- Whether the daemon tried to read the agent's store for this turn. False
    -- for turns run by a daemon from before this, which is not the same as
    -- having tried and found nothing.
    ADD COLUMN context_read  boolean NOT NULL DEFAULT false,
    -- The file it read, so the owner can open it himself.
    ADD COLUMN context_from  text    NOT NULL DEFAULT '',
    -- Why nothing, or not everything, could be read.
    ADD COLUMN context_error text    NOT NULL DEFAULT '';

CREATE TABLE context_documents (
    conversation_id uuid  NOT NULL REFERENCES conversations (id),
    -- sha256 of kind, a NUL, and body.
    hash            bytea NOT NULL,
    -- "claude.attachment", "codex.message", "agy.gen_metadata" and so on.
    kind            text  NOT NULL,
    -- As the agent recorded it: JSON, or base64 of agy's protobuf.
    body            text  NOT NULL,
    PRIMARY KEY (conversation_id, hash)
);

CREATE TABLE exchange_context (
    entry_id        uuid    NOT NULL REFERENCES exchanges (entry_id),
    -- The order the agent recorded them in, which is the order they apply:
    -- a later record can replace or remove an earlier one.
    ord             integer NOT NULL,
    conversation_id uuid    NOT NULL,
    hash            bytea   NOT NULL,
    PRIMARY KEY (entry_id, ord),
    FOREIGN KEY (conversation_id, hash) REFERENCES context_documents (conversation_id, hash)
);

CREATE INDEX exchange_context_conversation_idx ON exchange_context (conversation_id);

-- +goose Down
DROP TABLE exchange_context;
DROP TABLE context_documents;
ALTER TABLE exchanges
    DROP COLUMN context_error,
    DROP COLUMN context_from,
    DROP COLUMN context_read;
