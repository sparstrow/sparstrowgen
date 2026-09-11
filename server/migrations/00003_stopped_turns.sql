-- +goose Up
-- A turn the owner stopped is neither a success nor a failure, and the
-- difference is worth keeping. `failure` means something went wrong; this means
-- someone decided they had seen enough. Rendering the second as the first would
-- put a red alert box around a deliberate act.
--
-- A column on the agent entry rather than a new role, for two reasons. The
-- partial text lives on that entry, and a separate marker row would put the
-- note somewhere other than the thing it is about. And `entries.role` has a
-- CHECK constraint precisely so the set of roles stays small and meaningful —
-- "this turn ended early" is a property of a turn, not a fourth kind of thing
-- that can appear in a transcript.
--
-- Both can be true at once: a CLI that reports an error while being killed is
-- stopped AND failed, and the surface prefers the stop when saying why it ended.
ALTER TABLE entries ADD COLUMN stopped boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE entries DROP COLUMN stopped;
