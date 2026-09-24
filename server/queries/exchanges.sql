-- name: RecordExchange :exec
-- Once per turn. A daemon that sent it twice (it never should) changes nothing.
INSERT INTO exchanges (
    entry_id, conversation_id, program, args, cwd, resume_session_id, prompt, stdin, launched
) VALUES (
    @entry_id, @conversation_id, @program, @args, @cwd, @resume_session_id, @prompt, @stdin, @launched
)
ON CONFLICT (entry_id) DO NOTHING;

-- name: AppendExchangeLines :exec
-- One batch in one statement: the four arrays are stepped through in lockstep,
-- which is what several unnests in one select list do. A repeated seq is a
-- batch already stored, and is skipped rather than failing the ones around it,
-- and only the lines actually stored move the record's totals.
WITH stored AS (
    INSERT INTO exchange_lines (entry_id, seq, at_ms, stream, body)
    SELECT @entry_id::uuid,
           unnest(@seqs::integer[]),
           unnest(@at_ms::bigint[]),
           unnest(@streams::text[]),
           unnest(@bodies::text[])
    ON CONFLICT (entry_id, seq) DO NOTHING
    RETURNING at_ms, octet_length(body) AS bytes
)
UPDATE exchanges
SET line_count = line_count + (SELECT count(*) FROM stored),
    byte_count = byte_count + COALESCE((SELECT sum(bytes) FROM stored), 0),
    last_at_ms = GREATEST(last_at_ms, COALESCE((SELECT max(at_ms) FROM stored), 0))
WHERE exchanges.entry_id = @entry_id::uuid;

-- name: SetExchangeDropped :exec
-- The daemon sends running totals, so this sets rather than adds.
UPDATE exchanges
SET dropped_lines = @dropped_lines, dropped_bytes = @dropped_bytes
WHERE entry_id = @entry_id;

-- name: AgentEntryFor :one
-- An agent entry, only if it is in a conversation this account can see. The
-- ownership check for reading a record.
SELECT e.id, e.conversation_id, COALESCE(e.provider, '')::text AS provider
FROM entries e
JOIN conversations c ON c.id = e.conversation_id
WHERE e.id = @id
  AND e.role = 'agent'
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = c.workspace_id AND m.user_id = @user_id
  );

-- name: ListExchangeSummaries :many
-- How big each turn's record in one conversation is, without reading any of it.
SELECT entry_id, launched, line_count, byte_count, last_at_ms, dropped_lines, dropped_bytes
FROM exchanges
WHERE conversation_id = $1;

-- name: GetExchange :one
SELECT * FROM exchanges WHERE entry_id = $1;

-- name: ListExchangeLines :many
SELECT * FROM exchange_lines WHERE entry_id = $1 ORDER BY seq;

-- name: DeleteConversationExchangeLines :exec
DELETE FROM exchange_lines
WHERE entry_id IN (SELECT entry_id FROM exchanges WHERE conversation_id = $1);

-- name: DeleteConversationExchanges :exec
DELETE FROM exchanges WHERE conversation_id = $1;

-- name: DeleteEveryExchangeLine :exec
-- Tests only, with the other DeleteEvery* in users.sql.
DELETE FROM exchange_lines;

-- name: DeleteEveryExchange :exec
DELETE FROM exchanges;
