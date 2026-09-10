-- +goose Up
-- The transcript of record. A conversation is provider-neutral: no agent CLI can
-- resume another's session, so if the conversation lived with a provider it could
-- never move. See docs/Decisions.md D-004.

CREATE TABLE conversations (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title       text        NOT NULL DEFAULT 'Untitled conversation',
    -- A conversation is always about somewhere: the directory its agents run in.
    folder      text        NOT NULL,
    -- Current selection. Stored as id AND label because no provider can resolve a
    -- label back from an id once it drops the model from its list (D-015).
    provider    text        NOT NULL,
    model_id    text        NOT NULL,
    model_label text        NOT NULL,
    -- Out of the list but fully intact. Distinct from deletion, which is final.
    archived    boolean     NOT NULL DEFAULT false,
    -- Money is an integer, never a float. Ticks of 1e-10 USD, so a provider's own
    -- cost figure is stored exactly as stated rather than re-derived from tokens
    -- at a rate that cannot know request-level pricing rules. Adopted from
    -- Multica's TokenUsage.CostUSDTicks.
    spend_ticks bigint      NOT NULL DEFAULT 0,
    tokens      bigint      NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX conversations_updated_idx ON conversations (archived, updated_at DESC);

-- Every entry in a transcript, in one table. role discriminates:
--   user   — what the owner typed
--   agent  — one turn from one provider
--   replay — the marker written when a provider was caught up, at the moment it
--            was actually paid for, never when it was merely selected
CREATE TABLE entries (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id uuid        NOT NULL REFERENCES conversations (id),
    -- Position in the transcript. Dense and gapless per conversation, because
    -- provider_sessions.seen_seq counts against it.
    seq             integer     NOT NULL,
    role            text        NOT NULL CHECK (role IN ('user', 'agent', 'replay')),
    body            text        NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),

    -- agent + replay
    provider        text,
    model_id        text,
    model_label     text,

    -- agent only
    tokens          bigint,
    spend_ticks     bigint,
    -- Text that arrived before the turn died is still kept in body; this says
    -- why it stopped.
    failure         text,

    -- replay only
    messages_replayed integer,

    UNIQUE (conversation_id, seq)
);

CREATE INDEX entries_conversation_idx ON entries (conversation_id, seq);

-- Full-text search over message bodies. Searching titles alone would miss the
-- conversations that most need finding, since an unnamed one is called
-- "Untitled conversation" until somebody renames it.
CREATE INDEX entries_body_fts_idx ON entries USING gin (to_tsvector('english', body));

-- How much of a conversation each provider has already been told. A switch back
-- to a provider that has seen the first four entries replays only the gap, not
-- the whole history — which is what makes switching affordable.
CREATE TABLE provider_sessions (
    conversation_id uuid        NOT NULL REFERENCES conversations (id),
    provider        text        NOT NULL,
    -- The provider's own session id, for --resume. Null until it answers once.
    session_id      text,
    seen_seq        integer     NOT NULL DEFAULT 0,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (conversation_id, provider)
);

-- +goose Down
DROP TABLE provider_sessions;
DROP TABLE entries;
DROP TABLE conversations;
