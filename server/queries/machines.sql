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

-- name: CreateMachine :one
INSERT INTO machines (user_id, display_name, credential_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ApproveMachinePairing :one
UPDATE machine_pairings
SET status = 'approved', decided_at = now()
WHERE id = $1::uuid AND user_id = $2::uuid AND status = 'claimed' AND expires_at > now()
RETURNING *;

-- name: ApproveMachine :one
UPDATE machines
SET approved_at = now()
WHERE id = $1::uuid AND user_id = $2::uuid AND revoked_at IS NULL
RETURNING *;

-- name: MachineByCredential :one
SELECT * FROM machines
WHERE credential_hash = $1 AND approved_at IS NOT NULL AND revoked_at IS NULL;

-- name: MachineByCredentialAnyState :one
SELECT * FROM machines WHERE credential_hash = $1;

-- name: ListMachines :many
SELECT * FROM machines
WHERE user_id = $1::uuid AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: GetMachineForUser :one
SELECT * FROM machines WHERE id = $1::uuid AND user_id = $2::uuid AND revoked_at IS NULL;

-- name: RevokeMachine :one
UPDATE machines
SET revoked_at = now()
WHERE id = $1::uuid AND user_id = $2::uuid AND revoked_at IS NULL
RETURNING *;

-- name: TouchMachine :exec
UPDATE machines SET last_seen_at = now()
WHERE id = $1 AND revoked_at IS NULL;
