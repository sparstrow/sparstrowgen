-- Account access: the links sent by email, and requests from people who were
-- not invited. See migrations/00009 for why each column exists.

-- name: CreateEmailLink :one
INSERT INTO email_links (token_hash, kind, email, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- Only the newest link for an address works. Run in the same transaction as
-- the insert of the new one, so there is never a moment with two live links or
-- with none.
-- name: SupersedeEmailLinks :exec
UPDATE email_links
SET superseded_at = now()
WHERE email = $1 AND kind = $2 AND used_at IS NULL AND superseded_at IS NULL;

-- name: GetEmailLink :one
SELECT * FROM email_links WHERE token_hash = $1 AND kind = $2;

-- Spending a link. Checking it and marking it used are ONE statement, so two
-- tabs submitting the same link at once cannot both succeed: the second finds
-- used_at already set and gets no row.
-- name: UseEmailLink :one
UPDATE email_links
SET used_at = now()
WHERE token_hash = $1
  AND kind = $2
  AND used_at IS NULL
  AND superseded_at IS NULL
  AND expires_at > now()
RETURNING *;

-- Links nobody can use any more, kept for a week so a late click still gets the
-- honest "expired" rather than looking like a link that never existed.
-- name: DeleteDeadEmailLinks :execrows
DELETE FROM email_links WHERE expires_at < now() - interval '7 days';

-- Recording a request. times_requested = 1 in the result means this is the
-- first, which is the only time the owner is emailed about it.
-- name: RecordAccessRequest :one
INSERT INTO access_requests (email)
VALUES ($1)
ON CONFLICT (email) DO UPDATE
SET last_requested_at = now(),
    times_requested   = access_requests.times_requested + 1
RETURNING *;

-- An address that has since been approved and has created its account is no
-- longer waiting.
-- name: DeleteAccessRequest :exec
DELETE FROM access_requests WHERE email = $1;
