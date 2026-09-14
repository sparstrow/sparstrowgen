-- name: CreateMachinePairing :one
INSERT INTO machine_pairings (user_id, token_hash, status, expires_at)
VALUES ($1, $2, 'pending', $3)
RETURNING *;

-- name: GetMachinePairingForUser :one
SELECT * FROM machine_pairings WHERE id = $1::uuid AND user_id = $2::uuid;

-- name: ClaimMachinePairing :one
UPDATE machine_pairings
SET status = 'claimed', machine_id = $2, claimed_at = now()
WHERE token_hash = $1 AND status = 'pending' AND expires_at > now()
RETURNING *;

-- name: MarkPairingAlreadyPaired :one
-- The computer presenting this request is already approved for the same
-- account, so the request resolves to that computer instead of a duplicate.
UPDATE machine_pairings
SET status = 'approved', machine_id = $2, claimed_at = now(), decided_at = now()
WHERE token_hash = $1 AND status = 'pending' AND expires_at > now()
RETURNING *;

-- name: CreateMachine :one
INSERT INTO machines (user_id, display_name, credential_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ApproveMachinePairing :one
UPDATE machine_pairings
SET status = 'approved', decided_at = now()
WHERE id = $1::uuid AND user_id = $2::uuid AND status = 'claimed' AND expires_at > now()
RETURNING *;

-- name: DeclineMachinePairing :one
UPDATE machine_pairings
SET status = 'rejected', decided_at = now()
WHERE id = $1::uuid AND user_id = $2::uuid AND status IN ('pending', 'claimed')
RETURNING *;

-- name: ApproveMachine :one
UPDATE machines
SET approved_at = now()
WHERE id = $1::uuid AND user_id = $2::uuid AND revoked_at IS NULL
RETURNING *;

-- name: RevokeUnapprovedMachine :exec
UPDATE machines
SET revoked_at = now()
WHERE id = $1::uuid AND user_id = $2::uuid AND approved_at IS NULL AND revoked_at IS NULL;

-- name: MachineByCredential :one
SELECT * FROM machines
WHERE credential_hash = $1 AND approved_at IS NOT NULL AND revoked_at IS NULL;

-- name: ApprovedMachineForUserCredential :one
SELECT * FROM machines
WHERE credential_hash = $1 AND user_id = $2 AND approved_at IS NOT NULL AND revoked_at IS NULL;

-- name: CredentialRefused :one
-- A credential that can never connect: its computer was disconnected, or it
-- was never approved and can no longer be, because its pairing was declined
-- or has expired.
SELECT (m.revoked_at IS NOT NULL OR (m.approved_at IS NULL AND NOT EXISTS (
    SELECT 1 FROM machine_pairings p
    WHERE p.machine_id = m.id AND p.status = 'claimed' AND p.expires_at > now()
)))::boolean AS refused
FROM machines m
WHERE m.credential_hash = $1;

-- name: ListMachines :many
-- Only approved computers. One still waiting for approval belongs to the
-- pairing in progress, not to the account's list.
SELECT * FROM machines
WHERE user_id = $1::uuid AND approved_at IS NOT NULL AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: GetMachineForUser :one
SELECT * FROM machines
WHERE id = $1::uuid AND user_id = $2::uuid AND approved_at IS NOT NULL AND revoked_at IS NULL;

-- name: RevokeMachine :one
UPDATE machines
SET revoked_at = now()
WHERE id = $1::uuid AND user_id = $2::uuid AND revoked_at IS NULL
RETURNING *;

-- name: SetMachineAutomaticUpdates :one
UPDATE machines
SET automatic_updates = $3
WHERE id = $1::uuid AND user_id = $2::uuid AND approved_at IS NOT NULL AND revoked_at IS NULL
RETURNING *;

-- name: RecordMachineVersion :exec
UPDATE machines SET daemon_version = $2
WHERE id = $1 AND revoked_at IS NULL;

-- name: TouchMachine :exec
UPDATE machines SET last_seen_at = now()
WHERE id = $1 AND revoked_at IS NULL;
