-- +goose Up
-- A workspace: a separate area of work inside an account. One for personal
-- things, one for work, and nothing bleeding between them.
--
-- This moves ownership of a conversation from the account to the workspace,
-- which docs/Decisions.md D-031 named as the migration that would be needed if
-- workspaces ever became shareable, and D-050 is where that trade is made. It
-- happens now, while there is one person's work to move, rather than later with
-- other people's transcripts in the same rows.

CREATE TABLE workspaces (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Who may reach a workspace, and as what.
--
-- Every row written today is an owner, because there is no way to invite
-- anybody yet. The table exists anyway, and D-050 says why: the alternative is
-- an owner_id column, which makes every scoping statement read `owner_id = me`
-- and turns the first invitation into a rewrite of all of them. With the join
-- here from the start, inviting somebody later is an INSERT.
--
-- No role beyond owner and member. A third one would be a guess at a permission
-- model nobody has asked for.
CREATE TABLE workspace_members (
    workspace_id uuid        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role         text        NOT NULL CHECK (role IN ('owner', 'member')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);

-- The switcher's list: every workspace one account can reach. The account leads
-- because that is the only way this is ever read.
CREATE INDEX workspace_members_user_idx ON workspace_members (user_id);

-- A workspace with nobody in it is unreachable by anything: no statement in the
-- app can find it, because every one of them joins membership. Deleting the
-- last account would leave exactly that. There is no cascade from users to
-- workspaces on purpose (D-009 — deleting an account must not silently take
-- work with it), so this is written down rather than enforced: an account with
-- workspaces cannot be deleted until they are dealt with deliberately, which is
-- the same rule conversations already have.

ALTER TABLE conversations ADD COLUMN workspace_id uuid REFERENCES workspaces (id);

-- Every existing account gets its workspace, and its conversations move into
-- it. "Personal" because that is what a single undifferentiated area of work
-- turns out to have been, and because the owner's own words for the split were
-- "one personal and one work related workspace" — so the one that already
-- exists is the personal one.
--
-- Named in full rather than left to the app to create on next sign-in: a column
-- that is NOT NULL only once some code has run is a column that is NULL in
-- production for as long as nobody signs in.
-- +goose StatementBegin
DO $$
DECLARE
    account      record;
    new_workspace uuid;
BEGIN
    FOR account IN SELECT id FROM users LOOP
        INSERT INTO workspaces (name) VALUES ('Personal') RETURNING id INTO new_workspace;
        INSERT INTO workspace_members (workspace_id, user_id, role)
            VALUES (new_workspace, account.id, 'owner');
        UPDATE conversations SET workspace_id = new_workspace WHERE user_id = account.id;
    END LOOP;
END
$$;
-- +goose StatementEnd

-- Refuses rather than inventing an owner. A conversation with no workspace
-- after the loop above means a conversation whose user_id points at no account,
-- which cannot happen through the API and is not something a migration should
-- paper over by picking somewhere to put it.
-- +goose StatementBegin
DO $$
DECLARE
    orphans integer;
BEGIN
    SELECT count(*) INTO orphans FROM conversations WHERE workspace_id IS NULL;
    IF orphans > 0 THEN
        RAISE EXCEPTION '% conversations have no account to give a workspace to', orphans;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE conversations ALTER COLUMN workspace_id SET NOT NULL;

-- conversations.user_id stays, and now means who STARTED the conversation
-- rather than who owns it. In a workspace with one member those are the same
-- person; in a shared one, "who started this" is the fact worth having, and it
-- costs nothing to keep.

-- Every list is one workspace's list now, so the workspace leads the index.
DROP INDEX conversations_user_updated_idx;
CREATE INDEX conversations_workspace_updated_idx
    ON conversations (workspace_id, archived, updated_at DESC);

-- +goose Down
DROP INDEX conversations_workspace_updated_idx;
CREATE INDEX conversations_user_updated_idx ON conversations (user_id, archived, updated_at DESC);
ALTER TABLE conversations DROP COLUMN workspace_id;
DROP TABLE workspace_members;
DROP TABLE workspaces;
