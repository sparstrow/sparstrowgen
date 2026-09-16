package api

import (
	"net/http"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* Appearance through the API a browser uses (spec
2026-09-12-appearance-preferences): it belongs to the account, it rides on the
session so the first paint is right, and it is one account's alone. */

type sessionSeen struct {
	SignedIn   bool             `json:"signedIn"`
	Email      string           `json:"email"`
	Appearance store.Appearance `json:"appearance"`
}

func (r *rig) session() sessionSeen {
	r.t.Helper()
	res := r.get("/api/auth/session")
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("session: %s", res.Status)
	}
	var seen sessionSeen
	decodeInto(r.t, res, &seen)
	return seen
}

func TestTheSessionCarriesTheAppearanceSoTheFirstPaintIsRight(t *testing.T) {
	r := newRig(t)

	if got := r.session().Appearance; got != store.DefaultAppearance {
		t.Fatalf("a new account's session = %+v, want %+v", got, store.DefaultAppearance)
	}

	want := store.Appearance{Mode: "light", Surface: "soft", Accent: "violet"}
	res := r.post("/api/appearance", want)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("save: %s", res.Status)
	}
	var saved store.Appearance
	decodeInto(t, res, &saved)
	if saved != want {
		t.Fatalf("save answered %+v, want %+v", saved, want)
	}

	// The next browser to open the app is told, without asking separately.
	if got := r.session().Appearance; got != want {
		t.Errorf("session after saving = %+v, want %+v", got, want)
	}
	res = r.get("/api/appearance")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("read back: %s", res.Status)
	}
	var read store.Appearance
	decodeInto(t, res, &read)
	if read != want {
		t.Errorf("read back = %+v, want %+v", read, want)
	}
}

// The only writer is this app's own settings screen, so a name it does not
// offer is a bug rather than a preference — and a refused save must leave the
// account exactly as it was.
func TestAnAppearanceTheAppDoesNotOfferIsRefusedAndChangesNothing(t *testing.T) {
	r := newRig(t)
	before := r.session().Appearance

	for _, bad := range []map[string]string{
		{"mode": "midnight", "surface": "paper", "accent": "amber"},
		{"mode": "dark", "surface": "velvet", "accent": "amber"},
		{"mode": "dark", "surface": "paper", "accent": "chartreuse"},
		{"mode": "dark", "surface": "paper"},
	} {
		if res := r.post("/api/appearance", bad); res.StatusCode != http.StatusBadRequest {
			t.Errorf("%v was answered %s, want 400", bad, res.Status)
		}
	}
	if after := r.session().Appearance; after != before {
		t.Errorf("a refused save changed the account: %+v, was %+v", after, before)
	}
}

// Appearance is per account, like everything else (D-031).
func TestOneAccountsAppearanceIsNotAnothers(t *testing.T) {
	r := newRig(t)
	if res := r.post("/api/appearance", store.Appearance{Mode: "dark", Surface: "mono", Accent: "teal"}); res.StatusCode != http.StatusOK {
		t.Fatalf("save: %s", res.Status)
	}

	if res := r.stranger().get("/api/appearance"); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("a signed-out browser reading appearance = %s, want 401", res.Status)
	}
	other := r.secondAccount()
	if got := other.session().Appearance; got != store.DefaultAppearance {
		t.Errorf("a second account sees %+v, want its own defaults %+v", got, store.DefaultAppearance)
	}
	if got := r.session().Appearance; got.Surface != "mono" {
		t.Errorf("the first account's appearance changed to %+v", got)
	}
}
