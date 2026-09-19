-- +goose Up
-- Move the accounts that still hold the OLD default to the new one (D-039).
--
-- 00014 changed what a NEW account starts with and left existing accounts alone,
-- because a stored 'paper' / 'amber' cannot be told apart from a choice someone made
-- on purpose. The owner answered that on 2026-09-19: every account that exists today
-- is his own, so the ambiguity does not matter, and he wants monochrome everywhere.
-- Anyone who picked another surface or another accent is untouched, and so is anyone
-- who picked one of the two on its own.
--
-- The mode (light, dark, system) is not touched: it was never part of the default's
-- colour.
UPDATE users
SET appearance_surface = 'mono',
    appearance_accent  = 'neutral'
WHERE appearance_surface = 'paper'
  AND appearance_accent  = 'amber';

-- +goose Down
-- Nothing to undo. Which accounts were moved cannot be told afterwards, and putting
-- Paper and Amber back on every account that now reads Mono and Neutral would undo
-- choices made after this ran. 00014's Down restores the column defaults.
SELECT 1;
