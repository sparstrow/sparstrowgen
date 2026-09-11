-- +goose Up
-- Names for the conversations that already exist.
--
-- 00004 made "no name" mean NULL, which means every conversation from before it
-- is unnamed — and would stay that way until somebody sent another message into
-- it. The sidebar this feature exists to make scannable would have gone on being
-- a column of placeholders for everything already in it.
--
-- The rule here is deliberately cruder than the Go one that names conversations
-- from now on (server/internal/store/title.go): first non-blank line, its
-- markdown lead-in trimmed, cut at 60 characters. It does not need to agree with
-- that one and it is not a second copy to keep in step — it runs once, over rows
-- that already exist, and nothing ever takes this path again.
UPDATE conversations c
SET title = CASE
        WHEN length(l.line) > 60 THEN left(l.line, 60) || U&'\2026'
        ELSE l.line
    END
FROM (
    SELECT f.conversation_id,
           -- Trimmed from the LEFT only. A marker set trimmed from both ends
           -- eats the end of the line as well: "dsa tree in c++" came back
           -- from the first run of this as "dsa tree in c".
           rtrim(
               ltrim(
                   (regexp_match(f.body, '^[[:space:]]*([^[:space:]].*)$', 'n'))[1],
                   E'#>*+-_ \t'
               ),
               E' \t'
           ) AS line
    FROM (
        SELECT DISTINCT ON (e.conversation_id) e.conversation_id, e.body
        FROM entries e
        WHERE e.role = 'user'
        ORDER BY e.conversation_id, e.seq
    ) f
) l
WHERE c.id = l.conversation_id
  AND c.title IS NULL
  AND l.line IS NOT NULL
  AND l.line <> '';

-- +goose Down
-- Nothing. A derived name cannot be told apart from a typed one after the fact,
-- and guessing wrong would delete something the owner wrote. Names that came
-- from here are names now.
SELECT 1;
