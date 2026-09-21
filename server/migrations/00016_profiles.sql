-- +goose Up
-- Who a person is, as distinct from how they sign in: a name to be called by
-- and a line about themselves. Per account, like appearance (00012), because it
-- follows the person rather than the browser.
--
-- Empty-string defaults rather than NULL. There is no difference here between
-- "not set" and "set to nothing" — both mean fall back to the email address —
-- and a column that can be NULL or '' invites code that checks only one.
ALTER TABLE users
    ADD COLUMN display_name text NOT NULL DEFAULT '',
    ADD COLUMN bio          text NOT NULL DEFAULT '';

-- The avatar is a separate table, not a column on users, and that is the point
-- of it. `users` is read on EVERY authenticated request to resolve the session,
-- with SELECT *; putting image bytes on that row would drag them through every
-- one of those reads for something almost nothing needs.
--
-- Bytes in Postgres rather than object storage. There is no bucket, no S3
-- credential and no storage service in this system today, and adding one for
-- avatars would mean infrastructure to provision, a secret to rotate and a
-- lifecycle to maintain — against the owner's standing "no manual upkeep" rule
-- (D-035). These are small, capped square images, one per account, and being
-- in the database means they are in the database's backup rather than needing
-- their own. If a bucket arrives for another reason, this moves behind the
-- same store interface.
CREATE TABLE user_avatars (
    user_id      uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    content_type text NOT NULL,
    bytes        bytea NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_avatars;
ALTER TABLE users
    DROP COLUMN bio,
    DROP COLUMN display_name;
