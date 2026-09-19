-- +goose Up
-- A new account starts monochrome: the Mono surface and the Neutral accent, so
-- colour is left to mean something (status, provider) instead of decorating. The
-- owner chose this on 2026-09-19; the other four surfaces and five accents are
-- still one choice away in Settings.
--
-- Only the DEFAULT changes. Accounts that already exist keep what they have,
-- because a stored 'paper' / 'amber' cannot be told apart from a choice someone
-- made on purpose.
ALTER TABLE users
    ALTER COLUMN appearance_surface SET DEFAULT 'mono',
    ALTER COLUMN appearance_accent  SET DEFAULT 'neutral';

-- +goose Down
ALTER TABLE users
    ALTER COLUMN appearance_surface SET DEFAULT 'paper',
    ALTER COLUMN appearance_accent  SET DEFAULT 'amber';
