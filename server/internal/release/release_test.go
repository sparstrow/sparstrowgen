package release

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

const someSHA = "4587e57db82cbb1d2b1e0c2f6220ba9f4544e4c031a019e2c1b430be1250ed36"

func testKey(t *testing.T) (string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "signing.key")
	pub, err := GenerateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, pub
}

func TestASignedManifestVerifies(t *testing.T) {
	path, pub := testKey(t)
	key, err := LoadKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if PublicKey(key) != pub {
		t.Fatal("the loaded key's public half differs from the generated one")
	}
	want := Manifest{Version: "0.2.1", URL: "https://example.test/sparstrowgen-setup.exe", SHA256: strings.ToUpper(someSHA)}
	body, sig, err := Sign(key, want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Verify(body, sig, pub)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Version != "0.2.1" || got.URL != want.URL || got.SHA256 != someSHA {
		t.Errorf("verified manifest = %+v", got)
	}
}

// The whole point: a manifest that was changed after signing, or signed by
// anyone else, is never believed.
func TestAnAlteredOrForeignManifestIsRefused(t *testing.T) {
	path, pub := testKey(t)
	key, _ := LoadKey(path)
	body, sig, err := Sign(key, Manifest{Version: "0.2.1", URL: "https://example.test/a.exe", SHA256: someSHA})
	if err != nil {
		t.Fatal(err)
	}

	tampered := bytes.Replace(body, []byte("0.2.1"), []byte("9.9.9"), 1)
	if _, err := Verify(tampered, sig, pub); !errors.Is(err, ErrUntrusted) {
		t.Errorf("a changed manifest: err = %v, want ErrUntrusted", err)
	}

	otherPath, _ := testKey(t)
	other, _ := LoadKey(otherPath)
	_, foreignSig, _ := Sign(other, Manifest{Version: "0.2.1", URL: "https://example.test/a.exe", SHA256: someSHA})
	if _, err := Verify(body, foreignSig, pub); !errors.Is(err, ErrUntrusted) {
		t.Errorf("another key's signature: err = %v, want ErrUntrusted", err)
	}

	if _, err := Verify(body, []byte("not base64!"), pub); !errors.Is(err, ErrUntrusted) {
		t.Errorf("a garbage signature: err = %v, want ErrUntrusted", err)
	}
	if _, err := Verify(body, sig, ""); err == nil {
		t.Error("a build with no key verified a manifest")
	}
}

func TestOnlyAnHTTPSInstallerWithARealChecksumCanBeSigned(t *testing.T) {
	path, _ := testKey(t)
	key, _ := LoadKey(path)
	for _, m := range []Manifest{
		{Version: "0.2.1", URL: "http://example.test/a.exe", SHA256: someSHA},
		{Version: "0.2.1", URL: "https://example.test/a.exe", SHA256: "abc"},
		{Version: "latest", URL: "https://example.test/a.exe", SHA256: someSHA},
	} {
		if _, _, err := Sign(key, m); err == nil {
			t.Errorf("signed an invalid manifest %+v", m)
		}
	}
}

func TestNewerComparesEachPartAsANumber(t *testing.T) {
	for _, c := range []struct {
		candidate, current string
		want               bool
	}{
		{"0.2.1", "0.2.0", true},
		{"0.10.0", "0.9.9", true},
		{"v1.0.0", "0.99.99", true},
		{"0.2.0", "0.2.0", false},
		{"0.1.9", "0.2.0", false},
	} {
		got, err := Newer(c.candidate, c.current)
		if err != nil || got != c.want {
			t.Errorf("Newer(%s, %s) = %v, %v; want %v", c.candidate, c.current, got, err, c.want)
		}
	}
	if _, err := Newer("0.2", "0.2.0"); err == nil {
		t.Error("accepted a two-part version")
	}
}

func TestGenerateKeyNeverReplacesAnExistingKey(t *testing.T) {
	path, pub := testKey(t)
	if _, err := GenerateKey(path); err == nil {
		t.Fatal("replaced an existing signing key")
	}
	key, err := LoadKey(path)
	if err != nil || PublicKey(key) != pub {
		t.Errorf("the original key changed: %v", err)
	}
}
