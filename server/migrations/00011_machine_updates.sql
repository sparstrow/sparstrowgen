-- +goose Up
-- US3. Whether a computer installs updates by itself — on until the person
-- turns it off — and the daemon version it last reported, so a computer that
-- is offline still says what it runs.
ALTER TABLE machines
    ADD COLUMN automatic_updates boolean NOT NULL DEFAULT true,
    ADD COLUMN daemon_version text;

-- +goose Down
ALTER TABLE machines
    DROP COLUMN daemon_version,
    DROP COLUMN automatic_updates;
