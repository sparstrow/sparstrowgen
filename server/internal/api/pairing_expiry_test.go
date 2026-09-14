package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/testdb"
)

// A computer whose pairing expired before anyone approved it can never be
// approved, so it must be told to stop rather than wait forever.
func TestAnExpiredUnapprovedComputerIsToldToStop(t *testing.T) {
	r := newRig(t)
	_, request := r.startPairing()
	status, out := r.claimWith(request, "LATE-PC", "")
	if status != http.StatusOK {
		t.Fatalf("claim: %d", status)
	}
	if _, err := testdb.Pool(t).Exec(context.Background(),
		"UPDATE machine_pairings SET expires_at = now() - interval '1 minute' WHERE machine_id = $1", out.MachineID); err != nil {
		t.Fatal(err)
	}
	if conn, code := r.dialPaired(out.Credential); conn != nil || code != http.StatusForbidden {
		t.Fatalf("an expired, unapproved computer dialled: conn %v, status %d; want 403", conn != nil, code)
	}
}
