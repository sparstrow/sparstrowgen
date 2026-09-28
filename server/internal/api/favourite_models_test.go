package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* The models starred in the agent and model picker. They belong to the account,
like appearance, so they follow the person to every browser, and they are one
account's alone. */

type starBody struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Starred  bool   `json:"starred"`
}

func (r *rig) star(provider, model string, starred bool) (int, []store.FavouriteModel) {
	r.t.Helper()
	res := r.post("/api/favourite-models", starBody{provider, model, starred})
	var favs []store.FavouriteModel
	if res.StatusCode == http.StatusOK {
		decodeInto(r.t, res, &favs)
	}
	return res.StatusCode, favs
}

func (r *rig) favourites() []store.FavouriteModel {
	r.t.Helper()
	res := r.get("/api/favourite-models")
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("read favourites: %s", res.Status)
	}
	var favs []store.FavouriteModel
	decodeInto(r.t, res, &favs)
	return favs
}

func TestStarredModelsAreKeptInTheOrderTheyWereStarred(t *testing.T) {
	r := newRig(t)

	if got := r.favourites(); len(got) != 0 {
		t.Fatalf("a new account has %v starred, want none — and an empty list, not null", got)
	}

	r.star("claude", "claude-opus-5-5", true)
	r.star("agy", "gemini-3.8-flash-high", true)
	// Starring it again keeps its place rather than moving it to the end.
	_, favs := r.star("claude", "claude-opus-5-5", true)
	want := []store.FavouriteModel{{Provider: "claude", Model: "claude-opus-5-5"}, {Provider: "agy", Model: "gemini-3.8-flash-high"}}
	if len(favs) != 2 || favs[0] != want[0] || favs[1] != want[1] {
		t.Fatalf("after starring = %v, want %v", favs, want)
	}

	_, favs = r.star("claude", "claude-opus-5-5", false)
	if len(favs) != 1 || favs[0] != want[1] {
		t.Fatalf("after unstarring = %v, want only %v", favs, want[1])
	}
	// Unstarring one that is not starred is not an error: the click landed.
	if code, _ := r.star("codex", "gpt-5.6-sol", false); code != http.StatusOK {
		t.Errorf("unstarring a model never starred: %d, want 200", code)
	}
	if got := r.favourites(); len(got) != 1 || got[0] != want[1] {
		t.Errorf("read back = %v, want only %v", got, want[1])
	}
}

// A model id the server has never heard of is kept: a newer daemon may offer
// one. An empty or absurd one is refused.
func TestAnyReportedModelCanBeStarredButNotNothing(t *testing.T) {
	r := newRig(t)
	if code, favs := r.star("claude", "claude-fable-5-1[1m]", true); code != http.StatusOK || len(favs) != 1 {
		t.Fatalf("an id with a context tag: %d %v", code, favs)
	}
	long := make([]byte, 201)
	for i := range long {
		long[i] = 'x'
	}
	for _, bad := range []starBody{{"", "gpt-5.6-sol", true}, {"codex", "", true}, {"codex", " gpt-5.6-sol", true}, {"codex", string(long), true}} {
		if code, _ := r.star(bad.Provider, bad.Model, bad.Starred); code != http.StatusBadRequest {
			t.Errorf("%q / %q: %d, want 400", bad.Provider, bad.Model, code)
		}
	}
}

func TestEveryTabIsToldWhenAStarChanges(t *testing.T) {
	r := newRig(t)
	otherTab := r.watch()
	r.star("codex", "gpt-5.6-sol", true)
	otherTab.awaitEvent(t, "favourite_models")
}

func TestOneAccountsStarsAreItsOwn(t *testing.T) {
	r := newRig(t)
	r.star("claude", "claude-opus-5-5", true)
	other := r.secondAccount()
	if got := other.favourites(); len(got) != 0 {
		t.Fatalf("the second account sees %v", got)
	}
	other.star("codex", "gpt-5.6-sol", true)
	if got := r.favourites(); len(got) != 1 || got[0].Provider != "claude" {
		t.Errorf("the owner's list became %v", got)
	}
}

// The list has a ceiling, and at the ceiling a star already given still lands.
func TestStarsStopAtTheCeiling(t *testing.T) {
	r := newRig(t)
	for i := 0; i < store.MaxFavouriteModels; i++ {
		if code, _ := r.star("agy", fmt.Sprintf("model-%03d", i), true); code != http.StatusOK {
			t.Fatalf("star %d: %d", i, code)
		}
	}
	if code, _ := r.star("agy", "one-too-many", true); code != http.StatusBadRequest {
		t.Errorf("past the ceiling: %d, want 400", code)
	}
	if code, favs := r.star("agy", "model-000", true); code != http.StatusOK || len(favs) != store.MaxFavouriteModels {
		t.Errorf("re-starring at the ceiling: %d with %d, want 200 with %d", code, len(favs), store.MaxFavouriteModels)
	}
}
