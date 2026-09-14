package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sparstrow/sparstrowgen/server/internal/db"
)

// Machine is safe to return to the browser. Credentials never cross this
// package after creation.
type Machine struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Approved   bool       `json:"approved"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty"`
}

type Pairing struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	MachineID string    `json:"machineId,omitempty"`
	ExpiresAt time.Time `json:"expiresAt"`
}

var ErrPairingUnavailable = errors.New("that pairing request is no longer available")

func machineFrom(row db.Machine) Machine {
	m := Machine{ID: uuidToString(row.ID), Name: row.DisplayName, Approved: row.ApprovedAt.Valid, CreatedAt: row.CreatedAt.Time}
	if row.LastSeenAt.Valid {
		t := row.LastSeenAt.Time
		m.LastSeenAt = &t
	}
	return m
}

func pairingFrom(row db.MachinePairing) Pairing {
	return Pairing{ID: uuidToString(row.ID), Status: row.Status, MachineID: uuidToString(row.MachineID), ExpiresAt: row.ExpiresAt.Time}
}

func (s *Store) CreatePairing(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) (Pairing, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return Pairing{}, err
	}
	row, err := s.q.CreateMachinePairing(ctx, db.CreateMachinePairingParams{UserID: u, TokenHash: tokenHash, ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true}})
	if err != nil {
		return Pairing{}, err
	}
	return pairingFrom(row), nil
}

func (s *Store) Pairing(ctx context.Context, userID, pairingID string) (Pairing, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return Pairing{}, err
	}
	p, err := parseUUID(pairingID)
	if err != nil {
		return Pairing{}, ErrPairingUnavailable
	}
	row, err := s.q.GetMachinePairingForUser(ctx, db.GetMachinePairingForUserParams{Column1: p, Column2: u})
	if errors.Is(err, pgx.ErrNoRows) {
		return Pairing{}, ErrPairingUnavailable
	}
	if err != nil {
		return Pairing{}, err
	}
	return pairingFrom(row), nil
}

// ClaimPairing atomically spends an opaque launch request. A second daemon with
// the same request loses the update race.
//
// currentHash is the credential the claiming computer already holds, if any.
// When it belongs to an approved computer of the SAME account, the request
// resolves to that computer: nothing new is created and no credential is
// issued, so adding a computer that is already connected cannot duplicate it.
func (s *Store) ClaimPairing(ctx context.Context, tokenHash, credentialHash, currentHash []byte, name string) (pairing Pairing, machineID string, alreadyPaired bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Pairing{}, "", false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	// There is no owner to know until the claim succeeds, so create only after
	// looking up the pairing inside the transaction.
	var userID pgtype.UUID
	if err := tx.QueryRow(ctx, "SELECT user_id FROM machine_pairings WHERE token_hash=$1 AND status='pending' AND expires_at > now() FOR UPDATE", tokenHash).Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Pairing{}, "", false, ErrPairingUnavailable
		}
		return Pairing{}, "", false, err
	}
	if len(currentHash) > 0 {
		existing, err := q.ApprovedMachineForUserCredential(ctx, db.ApprovedMachineForUserCredentialParams{CredentialHash: currentHash, UserID: userID})
		switch {
		case err == nil:
			p, err := q.MarkPairingAlreadyPaired(ctx, db.MarkPairingAlreadyPairedParams{TokenHash: tokenHash, MachineID: existing.ID})
			if err != nil {
				return Pairing{}, "", false, err
			}
			if err := tx.Commit(ctx); err != nil {
				return Pairing{}, "", false, err
			}
			return pairingFrom(p), uuidToString(existing.ID), true, nil
		case !errors.Is(err, pgx.ErrNoRows):
			return Pairing{}, "", false, err
		}
	}
	m, err := q.CreateMachine(ctx, db.CreateMachineParams{UserID: userID, DisplayName: name, CredentialHash: credentialHash})
	if err != nil {
		return Pairing{}, "", false, err
	}
	p, err := q.ClaimMachinePairing(ctx, db.ClaimMachinePairingParams{TokenHash: tokenHash, MachineID: m.ID})
	if err != nil {
		return Pairing{}, "", false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Pairing{}, "", false, err
	}
	return pairingFrom(p), uuidToString(m.ID), false, nil
}

func (s *Store) ApprovePairing(ctx context.Context, userID, pairingID string) (Machine, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return Machine{}, err
	}
	p, err := parseUUID(pairingID)
	if err != nil {
		return Machine{}, ErrPairingUnavailable
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Machine{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	pair, err := q.ApproveMachinePairing(ctx, db.ApproveMachinePairingParams{Column1: p, Column2: u})
	if errors.Is(err, pgx.ErrNoRows) {
		return Machine{}, ErrPairingUnavailable
	}
	if err != nil {
		return Machine{}, err
	}
	m, err := q.ApproveMachine(ctx, db.ApproveMachineParams{Column1: pair.MachineID, Column2: u})
	if err != nil {
		return Machine{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Machine{}, err
	}
	return machineFrom(m), nil
}

// DeclinePairing is "Not now". The request can no longer be claimed or
// approved, and a computer that already claimed it is retired, so its
// credential is refused from its next dial.
func (s *Store) DeclinePairing(ctx context.Context, userID, pairingID string) error {
	u, err := parseUUID(userID)
	if err != nil {
		return err
	}
	p, err := parseUUID(pairingID)
	if err != nil {
		return ErrPairingUnavailable
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	row, err := q.DeclineMachinePairing(ctx, db.DeclineMachinePairingParams{Column1: p, Column2: u})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPairingUnavailable
	}
	if err != nil {
		return err
	}
	if row.MachineID.Valid {
		if err := q.RevokeUnapprovedMachine(ctx, db.RevokeUnapprovedMachineParams{Column1: row.MachineID, Column2: u}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Machines(ctx context.Context, userID string) ([]Machine, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListMachines(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]Machine, 0, len(rows))
	for _, row := range rows {
		out = append(out, machineFrom(row))
	}
	return out, nil
}

func (s *Store) Machine(ctx context.Context, userID, id string) (Machine, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return Machine{}, err
	}
	m, err := parseUUID(id)
	if err != nil {
		return Machine{}, ErrPairingUnavailable
	}
	row, err := s.q.GetMachineForUser(ctx, db.GetMachineForUserParams{Column1: m, Column2: u})
	if errors.Is(err, pgx.ErrNoRows) {
		return Machine{}, ErrPairingUnavailable
	}
	if err != nil {
		return Machine{}, err
	}
	return machineFrom(row), nil
}

func (s *Store) RevokeMachine(ctx context.Context, userID, id string) (Machine, error) {
	u, err := parseUUID(userID)
	if err != nil {
		return Machine{}, err
	}
	m, err := parseUUID(id)
	if err != nil {
		return Machine{}, ErrPairingUnavailable
	}
	row, err := s.q.RevokeMachine(ctx, db.RevokeMachineParams{Column1: m, Column2: u})
	if errors.Is(err, pgx.ErrNoRows) {
		return Machine{}, ErrPairingUnavailable
	}
	if err != nil {
		return Machine{}, err
	}
	return machineFrom(row), nil
}

// MachineForCredential accepts only a fully approved, unrecalled credential.
func (s *Store) MachineForCredential(ctx context.Context, hash []byte) (Machine, string, bool, error) {
	row, err := s.q.MachineByCredential(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Machine{}, "", false, nil
	}
	if err != nil {
		return Machine{}, "", false, err
	}
	_ = s.q.TouchMachine(ctx, row.ID)
	return machineFrom(row), uuidToString(row.UserID), true, nil
}

// CredentialRefused reports whether a credential can never connect again:
// disconnected, or declined or expired before approval. An unknown credential,
// or one whose pairing can still be approved, is not refused.
func (s *Store) CredentialRefused(ctx context.Context, hash []byte) (bool, error) {
	refused, err := s.q.CredentialRefused(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return refused, nil
}
