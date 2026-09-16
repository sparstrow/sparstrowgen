-- +goose Up
-- How the app looks, per account rather than per browser: the same choice
-- follows a person to every browser they sign in to (spec
-- 2026-09-12-appearance-preferences). Text rather than enums, because a newer
-- app version may offer a name this one does not, and the reader falls back to
-- a supported choice instead of failing.
--
-- The defaults are the first-release defaults: follow the computer, Paper
-- surface, Amber accent.
ALTER TABLE users
    ADD COLUMN appearance_mode    text NOT NULL DEFAULT 'system',
    ADD COLUMN appearance_surface text NOT NULL DEFAULT 'paper',
    ADD COLUMN appearance_accent  text NOT NULL DEFAULT 'amber';

-- +goose Down
ALTER TABLE users
    DROP COLUMN appearance_accent,
    DROP COLUMN appearance_surface,
    DROP COLUMN appearance_mode;
