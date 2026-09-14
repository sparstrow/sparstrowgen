package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/sparstrow/sparstrowgen/server/internal/db"
)

// SetAutomaticUpdates records whether one of the account's computers may
// install updates by itself (spec US3).
func (s *Store) SetAutomaticUpdates(ctx context.Context, userID, id string, enabled bool) (Machine, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return Machine{}, err
	}
	m, err := parseUUID(id)
	if err != nil {
		return Machine{}, ErrPairingUnavailable
	}
	row, err := s.q.SetMachineAutomaticUpdates(ctx, db.SetMachineAutomaticUpdatesParams{Column1: m, Column2: u, AutomaticUpdates: enabled})
	if errors.Is(err, pgx.ErrNoRows) {
		return Machine{}, ErrPairingUnavailable
	}
	if err != nil {
		return Machine{}, err
	}
	return machineFrom(row), nil
}

// RecordMachineVersion keeps the version a computer reported, so it can still be
// shown while that computer is offline.
func (s *Store) RecordMachineVersion(ctx context.Context, machineID, version string) error {
	m, err := parseUUID(machineID)
	if err != nil {
		return err
	}
	return s.q.RecordMachineVersion(ctx, db.RecordMachineVersionParams{ID: m, DaemonVersion: &version})
}
