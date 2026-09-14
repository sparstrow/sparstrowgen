-- +goose Up
-- A paired credential belongs to one computer and one account. It is hashed at
-- rest exactly like a session token: the daemon keeps the only usable copy.
CREATE TABLE machines (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    display_name    text        NOT NULL,
    credential_hash bytea       NOT NULL UNIQUE,
    approved_at     timestamptz,
    revoked_at      timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_seen_at    timestamptz
);

-- Every browser list and ownership check leads with the account.
CREATE INDEX machines_user_active_idx ON machines (user_id, created_at DESC)
WHERE revoked_at IS NULL;

-- The opaque request passed to the local daemon. A browser can only create and
-- approve its own requests; possessing this token is the daemon's proof that
-- it was launched from that request.
CREATE TABLE machine_pairings (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash      bytea       NOT NULL UNIQUE,
    status          text        NOT NULL CHECK (status IN ('pending', 'claimed', 'approved', 'rejected')),
    machine_id      uuid        REFERENCES machines (id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    expires_at      timestamptz NOT NULL,
    claimed_at      timestamptz,
    decided_at      timestamptz
);

CREATE INDEX machine_pairings_user_created_idx ON machine_pairings (user_id, created_at DESC);

-- +goose Down
DROP TABLE machine_pairings;
DROP TABLE machines;
