-- +goose Up
-- Search is substring, not word-based: the owner types "quarter hour" and
-- expects the message containing it, and he types partial words too. A GIN
-- full-text index cannot serve `ILIKE '%...%'` at all — it was the wrong index
-- for the query this feature actually needs, added before the query existed.
--
-- pg_trgm indexes trigrams, which is exactly what a leading-wildcard ILIKE
-- scans for, so this one is used rather than admired.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

DROP INDEX IF EXISTS entries_body_fts_idx;

CREATE INDEX entries_body_trgm_idx ON entries USING gin (body gin_trgm_ops);
CREATE INDEX conversations_title_trgm_idx ON conversations USING gin (title gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS conversations_title_trgm_idx;
DROP INDEX IF EXISTS entries_body_trgm_idx;
CREATE INDEX entries_body_fts_idx ON entries USING gin (to_tsvector('english', body));
