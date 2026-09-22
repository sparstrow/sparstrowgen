-- name: SearchConversations :many
-- Titles alone would miss the conversations that most need finding. A name is
-- derived from the first message or typed by hand, so it describes where a
-- conversation started and never where it ended up — and the thing you go
-- looking for is usually what was said in the middle of it.
--
-- The excerpt is the first matching message body, so a hit in a long transcript
-- is explicable rather than mysterious. A title match returns none — the reason
-- for that hit is already on screen.
--
-- One workspace's conversations, and only for an account that is in it. Both
-- filters wrap the whole match, so no OR branch can reach past either of them.
-- Searching across every workspace at once would undo the separation the
-- workspaces are for (D-050): a search in Personal must not surface work.
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
WHERE c.workspace_id = @workspace_id
  AND EXISTS (
        SELECT 1 FROM workspace_members m
        WHERE m.workspace_id = c.workspace_id AND m.user_id = @user_id
  )
  AND (
        c.title ILIKE '%' || @q::text || '%'
     OR c.folder ILIKE '%' || @q::text || '%'
     OR EXISTS (
            SELECT 1
            FROM entries e2
            WHERE e2.conversation_id = c.id
              AND e2.role <> 'replay'
              AND e2.body ILIKE '%' || @q::text || '%'
        )
  )
ORDER BY c.updated_at DESC;
