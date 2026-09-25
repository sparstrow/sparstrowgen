-- +goose Up
-- The files of a conversation: what the owner uploads into a message, and what
-- an agent makes for him in a turn (docs/specs/2026-09-11-files-into-a-conversation.md).
--
-- The computer keeps them in the chat's own folder, where the agents read and
-- write them. The server keeps them too: the database is the transcript of
-- record, a picture in the chat should still show while the computer is off,
-- and a conversation that moves to another computer can have them written
-- there again (D-056).
--
-- Bytes live in their own table, so listing a conversation's files never reads
-- them (the lesson of user_avatars, D-048).
--
-- No cascades (D-009): deleting a conversation deletes these explicitly.
CREATE TABLE conversation_files (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id uuid        NOT NULL REFERENCES conversations (id),
    -- The user message it was sent with, or the agent turn that made it. Null
    -- while an upload waits in the message box.
    entry_id        uuid        REFERENCES entries (id),
    origin          text        NOT NULL CHECK (origin IN ('upload', 'output')),
    -- Unique within its folder on the computer, so it is also the file's name
    -- there: a clash is renamed "name (2).ext" when the row is made.
    name            text        NOT NULL,
    -- Sniffed from the bytes by the server, never taken from the browser.
    media_type      text        NOT NULL,
    size            bigint      NOT NULL,
    sha256          text        NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (conversation_id, origin, name)
);

CREATE INDEX conversation_files_entry_idx ON conversation_files (entry_id);

CREATE TABLE conversation_file_contents (
    file_id uuid  PRIMARY KEY REFERENCES conversation_files (id),
    content bytea NOT NULL
);

-- +goose Down
DROP TABLE conversation_file_contents;
DROP TABLE conversation_files;
