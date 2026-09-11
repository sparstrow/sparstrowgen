-- +goose Up
-- A conversation with no name has no name. It used to have the literal string
-- "Untitled conversation" in the column, which reads as a title everywhere it
-- is handled — search matched it, a rename compared against it, and nothing
-- could tell "nobody has named this" from "somebody named it that".
--
-- NULL says the true thing, and it is what makes automatic naming safe: the
-- name is written exactly once, guarded by `WHERE title IS NULL`, so a name the
-- owner typed is never overwritten by one derived from a message, and two
-- concurrent sends cannot both win.
--
-- The placeholder moves to the surface, where it belongs — the sidebar shows
-- "Untitled conversation" in muted type, which is a thing being described
-- rather than a thing being named.
ALTER TABLE conversations
    ALTER COLUMN title DROP DEFAULT,
    ALTER COLUMN title DROP NOT NULL;

-- Existing rows carrying the placeholder were never named by anyone. Saying so
-- lets them be named by the next message sent, rather than being stuck with a
-- label that only looked like a title.
UPDATE conversations SET title = NULL WHERE title = 'Untitled conversation';

-- +goose Down
UPDATE conversations SET title = 'Untitled conversation' WHERE title IS NULL;
ALTER TABLE conversations
    ALTER COLUMN title SET NOT NULL,
    ALTER COLUMN title SET DEFAULT 'Untitled conversation';
