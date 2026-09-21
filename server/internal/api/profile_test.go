package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* The profile a person sets about themselves: a name to be called by, a line
about themselves, and a picture. It belongs to the account rather than the
browser, like appearance, and it is one account's alone. */

// rawPNG is the smallest valid PNG: a 1x1 image. Enough to be stored and
// served back, and small enough to sit in the test rather than a fixture file.
var rawPNG = []byte{
	0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89,
	0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
	0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01,
	0x0d, 0x0a, 0x2d, 0xb4,
	0x00, 0x00, 0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

func (r *rig) profile() store.Profile {
	r.t.Helper()
	res := r.get("/api/profile")
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("get profile: %s", res.Status)
	}
	var p store.Profile
	decodeInto(r.t, res, &p)
	return p
}

// putAvatar sends raw bytes with a media type, which is how the browser sends
// one: there is exactly one file and no fields beside it.
func (r *rig) putAvatar(mediaType string, body []byte) *http.Response {
	r.t.Helper()
	req, err := http.NewRequest(http.MethodPost, r.http.URL+"/api/profile/avatar", bytes.NewReader(body))
	if err != nil {
		r.t.Fatal(err)
	}
	req.Header.Set("Content-Type", mediaType)
	req.Header.Set("Origin", testOrigin)
	res, err := r.client.Do(req)
	if err != nil {
		r.t.Fatalf("put avatar: %v", err)
	}
	r.t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func TestAProfileIsSavedAndReadBack(t *testing.T) {
	r := newRig(t)

	if got := r.profile(); got.DisplayName != "" || got.Bio != "" || got.AvatarUpdatedAt != nil {
		t.Fatalf("a new account's profile = %+v, want it empty", got)
	}

	res := r.post("/api/profile", map[string]any{"displayName": "Srihari", "bio": "Business systems analyst."})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("save: %s", res.Status)
	}
	got := r.profile()
	if got.DisplayName != "Srihari" || got.Bio != "Business systems analyst." {
		t.Fatalf("profile = %+v, want the saved name and bio", got)
	}
}

func TestSurroundingSpaceIsNotANameOrABio(t *testing.T) {
	r := newRig(t)

	// A name of only spaces is not a name. Storing it would give the account a
	// blank label everywhere instead of falling back to the email address.
	res := r.post("/api/profile", map[string]any{"displayName": "   ", "bio": "  hello  "})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("save: %s", res.Status)
	}
	got := r.profile()
	if got.DisplayName != "" {
		t.Fatalf("display name = %q, want it empty so the email is used instead", got.DisplayName)
	}
	if got.Bio != "hello" {
		t.Fatalf("bio = %q, want it trimmed", got.Bio)
	}
}

func TestANameOrBioThatIsTooLongIsRefusedAndNothingIsSaved(t *testing.T) {
	r := newRig(t)

	if res := r.post("/api/profile", map[string]any{"displayName": "Srihari", "bio": "kept"}); res.StatusCode != http.StatusOK {
		t.Fatalf("first save: %s", res.Status)
	}

	for _, c := range []struct {
		what string
		body map[string]any
	}{
		{"name", map[string]any{"displayName": strings.Repeat("x", store.MaxDisplayName+1), "bio": "kept"}},
		{"bio", map[string]any{"displayName": "Srihari", "bio": strings.Repeat("x", store.MaxBio+1)}},
	} {
		res := r.post("/api/profile", c.body)
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s too long = %s, want 400", c.what, res.Status)
		}
	}

	// The whole save is one statement, so a refused one leaves the earlier
	// values exactly as they were rather than half-applying.
	if got := r.profile(); got.DisplayName != "Srihari" || got.Bio != "kept" {
		t.Fatalf("after refusals the profile = %+v, want the first save intact", got)
	}
}

// A limit counted in bytes would cut a name with accents or non-Latin script
// shorter than the same-length name in ASCII, which is a different rule for
// different people's names.
func TestTheNameLimitCountsCharactersNotBytes(t *testing.T) {
	r := newRig(t)

	name := strings.Repeat("é", store.MaxDisplayName) // twice that many bytes
	if res := r.post("/api/profile", map[string]any{"displayName": name, "bio": ""}); res.StatusCode != http.StatusOK {
		t.Fatalf("a name of %d characters = %s, want it accepted", store.MaxDisplayName, res.Status)
	}
	if got := r.profile().DisplayName; got != name {
		t.Fatalf("stored name has %d characters, want %d", len([]rune(got)), store.MaxDisplayName)
	}
}

func TestAPictureIsStoredAndServedBack(t *testing.T) {
	r := newRig(t)

	if res := r.putAvatar("image/png", rawPNG); res.StatusCode != http.StatusOK {
		t.Fatalf("upload: %s", res.Status)
	}
	if r.profile().AvatarUpdatedAt == nil {
		t.Fatal("the profile does not report a picture after one was uploaded")
	}

	res := r.get("/api/profile/avatar")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("fetch: %s", res.Status)
	}
	if got := res.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("content type = %q, want image/png", got)
	}
	// A shared cache handing one account's picture to the next request would be
	// a leak; the header has to say private.
	if got := res.Header.Get("Cache-Control"); !strings.Contains(got, "private") {
		t.Fatalf("cache-control = %q, want it private", got)
	}
	if got := res.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("x-content-type-options = %q, want nosniff", got)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, rawPNG) {
		t.Fatalf("served %d bytes, want the %d that were uploaded", len(body), len(rawPNG))
	}
}

// An SVG is a document that can carry script. Served back from our own origin
// it would run there, so the accepted list is closed rather than "image/*".
func TestAnSvgIsNotAPicture(t *testing.T) {
	r := newRig(t)

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	if res := r.putAvatar("image/svg+xml", svg); res.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("svg upload = %s, want 415", res.Status)
	}
	if r.profile().AvatarUpdatedAt != nil {
		t.Fatal("a refused upload still left a picture on the account")
	}
}

func TestAPictureLargerThanTheLimitIsRefused(t *testing.T) {
	r := newRig(t)

	res := r.putAvatar("image/png", bytes.Repeat([]byte{0}, store.MaxAvatarBytes+1))
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized upload = %s, want 413", res.Status)
	}
	if r.profile().AvatarUpdatedAt != nil {
		t.Fatal("a refused upload still left a picture on the account")
	}
}

func TestRemovingAPictureLeavesTheNameAndBio(t *testing.T) {
	r := newRig(t)

	if res := r.post("/api/profile", map[string]any{"displayName": "Srihari", "bio": "kept"}); res.StatusCode != http.StatusOK {
		t.Fatalf("save: %s", res.Status)
	}
	if res := r.putAvatar("image/png", rawPNG); res.StatusCode != http.StatusOK {
		t.Fatalf("upload: %s", res.Status)
	}
	if res := r.do(http.MethodDelete, "/api/profile/avatar", nil); res.StatusCode != http.StatusOK {
		t.Fatalf("delete: %s", res.Status)
	}

	got := r.profile()
	if got.AvatarUpdatedAt != nil {
		t.Fatal("the picture is still reported after it was removed")
	}
	if got.DisplayName != "Srihari" || got.Bio != "kept" {
		t.Fatalf("profile = %+v, want the name and bio untouched", got)
	}
	if res := r.get("/api/profile/avatar"); res.StatusCode != http.StatusNotFound {
		t.Fatalf("fetching a removed picture = %s, want 404", res.Status)
	}
}

// The same rule as every other account-scoped thing (D-031): one account's
// profile and picture are invisible to another, and the endpoint has no id in
// its path to walk.
func TestOneAccountCannotSeeAnothersProfileOrPicture(t *testing.T) {
	r := newRig(t)

	if res := r.post("/api/profile", map[string]any{"displayName": "Srihari", "bio": "private"}); res.StatusCode != http.StatusOK {
		t.Fatalf("save: %s", res.Status)
	}
	if res := r.putAvatar("image/png", rawPNG); res.StatusCode != http.StatusOK {
		t.Fatalf("upload: %s", res.Status)
	}

	other, err := r.store.CreateUser(context.Background(), testSecond, testPassword)
	if err != nil {
		t.Fatalf("create the second account: %v", err)
	}
	if other.ID == r.userID {
		t.Fatal("the second account is the first")
	}
	second := r.stranger()
	res := second.post("/api/auth/login", map[string]any{"email": testSecond, "password": testPassword})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("sign in as the second account: %s", res.Status)
	}

	if got := second.profile(); got.DisplayName != "" || got.Bio != "" || got.AvatarUpdatedAt != nil {
		t.Fatalf("the second account sees %+v, want its own empty profile", got)
	}
	if res := second.get("/api/profile/avatar"); res.StatusCode != http.StatusNotFound {
		t.Fatalf("the second account fetching a picture = %s, want 404", res.Status)
	}
}

func TestASignedOutBrowserHasNoProfile(t *testing.T) {
	r := newRig(t)

	out := r.stranger()
	for _, path := range []string{"/api/profile", "/api/profile/avatar"} {
		if res := out.get(path); res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("signed out %s = %s, want 401", path, res.Status)
		}
	}
}
