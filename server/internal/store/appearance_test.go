package store

import (
	"context"
	"testing"
)

// What an account saves is what it reads back — including in the session, which
// is what the app paints from. (That a NEW account starts on the defaults is in
// the API tests, whose accounts are created fresh; this store account is shared
// between runs, so it is put back as it was instead.)
func TestAnAccountKeepsTheAppearanceItChose(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := storeOwner(t, s)
	before, err := s.Appearance(ctx, user.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	t.Cleanup(func() { _, _ = s.SetAppearance(context.Background(), user.ID, before) })

	want := Appearance{Mode: "dark", Surface: "slate", Accent: "teal"}
	saved, err := s.SetAppearance(ctx, user.ID, want)
	if err != nil || saved != want {
		t.Fatalf("save = %+v, %v; want %+v", saved, err, want)
	}
	if got, _ := s.Appearance(ctx, user.ID); got != want {
		t.Errorf("read back = %+v, want %+v", got, want)
	}
	// The session is where the app learns the appearance, so it must carry it.
	token, _, err := s.StartSession(ctx, user.ID, "test", "127.0.0.1")
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	t.Cleanup(func() { _ = s.EndSession(context.Background(), token) })
	fromSession, ok, err := s.SessionUser(ctx, token)
	if err != nil || !ok {
		t.Fatalf("session user: %v %v", ok, err)
	}
	if fromSession.Appearance != want {
		t.Errorf("session appearance = %+v, want %+v", fromSession.Appearance, want)
	}
}

func TestAnAppearanceThisVersionDoesNotOfferIsRefused(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := storeOwner(t, s)

	for _, bad := range []Appearance{
		{Mode: "midnight", Surface: "paper", Accent: "amber"},
		{Mode: "dark", Surface: "velvet", Accent: "amber"},
		{Mode: "dark", Surface: "paper", Accent: "chartreuse"},
	} {
		if _, err := s.SetAppearance(ctx, user.ID, bad); err == nil {
			t.Errorf("%+v was saved; it should have been refused", bad)
		}
	}
}

// A choice saved by a newer version must not leave the app unreadable: each
// part falls back on its own, so one unknown name does not discard the others.
func TestAnUnreadableSavedChoiceFallsBackPartByPart(t *testing.T) {
	got := appearanceOf("dusk", "slate", "teal")
	want := Appearance{Mode: DefaultAppearance.Mode, Surface: "slate", Accent: "teal"}
	if got != want {
		t.Errorf("appearanceOf = %+v, want %+v", got, want)
	}
}
