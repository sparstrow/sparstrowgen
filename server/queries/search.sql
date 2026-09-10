-- name: SearchConversations :many
-- Titles alone would miss the conversations that most need finding: an unnamed
-- one is called "Untitled conversation" until somebody renames it, and those
-- are exactly the ones you go looking for by what was said in them.
--
-- The excerpt is the first matching message body, so a hit in a long transcript
-- is explicable rather than mysterious. A title match returns none — the reason
-- for that hit is already on screen.
SELECT
    sqlc.embed(c),
    (
        SELECT e.body
        FROM entries e
        WHERE e.conversation_id = c.id
          AND e.role <> 'replay'
          AND e.body ILIKE '%' || @q::text || '%'
        ORDER BY e.seq
        LIMIT 1
    ) AS excerpt
FROM conversations c
WHERE c.title ILIKE '%' || @q::text || '%'
   OR c.folder ILIKE '%' || @q::text || '%'
   OR EXISTS (
        SELECT 1
        FROM entries e2
        WHERE e2.conversation_id = c.id
          AND e2.role <> 'replay'
          AND e2.body ILIKE '%' || @q::text || '%'
   )
ORDER BY c.updated_at DESC;
