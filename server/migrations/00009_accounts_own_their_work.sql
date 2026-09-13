-- +goose Up
-- More than one person, each with their own work.
--
-- Until now the deployment had exactly one account (00008), so "every
-- conversation" and "the owner's conversations" were the same set and nothing
-- needed to say whose a row was. Invitation-only registration ends that, and a
-- second account must not see, search or send to the first account's work
-- (docs/KnownGaps.md G-27).
--
-- A conversation belongs to an ACCOUNT, not to a workspace. Workspaces are
-- planned as areas a person creates inside their own account; they will group
-- conversations without changing who owns them (docs/Decisions.md D-031).

-- The existing conversations belong to the one account that exists. Refused
-- outright when that is not a single, unambiguous answer: guessing an owner for
-- somebody's transcripts is not a migration's decision to make.
-- +goose StatementBegin
DO $$
DECLARE
    accounts      integer;
    conversations integer;
BEGIN
    SELECT count(*) INTO accounts FROM users;
    SELECT count(*) INTO conversations FROM conversations;
    IF conversations > 0 AND accounts <> 1 THEN
        RAISE EXCEPTION
            'there are % conversations and % accounts; exactly one account is needed to own them',
            conversations, accounts;
    END IF;
END
$$;
-- +goose StatementEnd

-- No ON DELETE CASCADE: deleting an account must not silently take its
-- transcripts with it (D-009). An account with conversations cannot be deleted
-- until they are dealt with deliberately.
ALTER TABLE conversations ADD COLUMN user_id uuid REFERENCES users (id);
UPDATE conversations SET user_id = (SELECT id FROM users LIMIT 1) WHERE user_id IS NULL;
ALTER TABLE conversations ALTER COLUMN user_id SET NOT NULL;

-- Every list is one account's list now, so the account leads the index.
DROP INDEX conversations_updated_idx;
CREATE INDEX conversations_user_updated_idx ON conversations (user_id, archived, updated_at DESC);

-- Any number of people may have accounts. Who may create one is decided by the
-- invitation list, before a row is ever written.
DROP INDEX users_only_one;

-- A link sent by email: finishing an account, or choosing a new password.
--
-- Stored as the SHA-256 of the token, like sessions (00006): the email holds the
-- only copy of the token, and a dump of this table opens nothing.
--
-- used_at and superseded_at are separate from expiry, and from each other,
-- because the page a link opens says which happened — "already used" and "a
-- newer link was sent" send a person to different next steps.
CREATE TABLE email_links (
    token_hash    bytea       PRIMARY KEY,
    kind          text        NOT NULL CHECK (kind IN ('verify', 'reset')),
    email         citext      NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    used_at       timestamptz,
    superseded_at timestamptz
);

-- Finding the live links for one address, to supersede them.
CREATE INDEX email_links_email_idx ON email_links (email, kind);

-- Somebody who was not invited asked to be. One row per address however often
-- it asks, so the owner is told once rather than flooded.
CREATE TABLE access_requests (
    email              citext      PRIMARY KEY,
    first_requested_at timestamptz NOT NULL DEFAULT now(),
    last_requested_at  timestamptz NOT NULL DEFAULT now(),
    times_requested    integer     NOT NULL DEFAULT 1
);

-- +goose Down
DROP TABLE access_requests;
DROP TABLE email_links;
-- Fails if a second account exists, which is correct: going back to one account
-- cannot choose which person to delete.
CREATE UNIQUE INDEX users_only_one ON users ((true));
DROP INDEX conversations_user_updated_idx;
CREATE INDEX conversations_updated_idx ON conversations (archived, updated_at DESC);
ALTER TABLE conversations DROP COLUMN user_id;
