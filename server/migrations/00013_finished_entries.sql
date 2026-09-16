-- +goose Up
-- When an agent's answer actually arrived.
--
-- An agent entry is created empty at the START of a turn, so a refresh mid-turn
-- shows what has streamed in so far rather than nothing. That made created_at
-- the moment the turn was launched, and the transcript showed it as the time of
-- the reply — so a turn that took ninety seconds appeared to have been answered
-- in the same minute the question was asked (docs/Bugs.md B-35).
--
-- Both times are kept rather than created_at being overwritten: when a turn
-- started and when it answered are different facts, and how long an agent took
-- is worth being able to ask later.
--
-- Null while a turn is still running, and null forever on the entries that were
-- written before this column existed — which is why it is nullable rather than
-- defaulted to now(): a default would silently claim every historical turn
-- finished at migration time.
ALTER TABLE entries ADD COLUMN finished_at timestamptz;

-- +goose Down
ALTER TABLE entries DROP COLUMN finished_at;
