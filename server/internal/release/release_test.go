package release

import (
	"errors"
	"strings"
	"testing"
)

const (
	someSHA = "4587e57db82cbb1d2b1e0c2f6220ba9f4544e4c031a019e2c1b430be1250ed36"
	source  = "https://github.com/sparstrow/sparstrowgen/releases/latest/download/sparstrowgen-update.json"
)

func TestAPublishedManifestReadsBack(t *testing.T) {
	want := Manifest{
		Version: "0.2.2",
		URL:     "https://github.com/sparstrow/sparstrowgen/releases/download/daemon-v0.2.2/sparstrowgen-setup.exe",
		SHA256:  strings.ToUpper(someSHA),
	}
	body, err := Encode(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(body, source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Version != "0.2.2" || got.URL != want.URL || got.SHA256 != someSHA {
		t.Errorf("manifest = %+v", got)
	}
}

// The whole point: a manifest is believed only if it names an installer on the
// site it came from, with a real version and checksum.
func TestAManifestPointingElsewhereOrMalformedIsRefused(t *testing.T) {
	for name, body := range map[string]string{
		"another site":      `{"version":"0.2.2","url":"https://example.test/sparstrowgen-setup.exe","sha256":"` + someSHA + `"}`,
		"a look-alike host": `{"version":"0.2.2","url":"https://github.com.example.test/setup.exe","sha256":"` + someSHA + `"}`,
		"plain http":        `{"version":"0.2.2","url":"http://github.com/sparstrow/sparstrowgen/setup.exe","sha256":"` + someSHA + `"}`,
		"no checksum":       `{"version":"0.2.2","url":"https://github.com/sparstrow/sparstrowgen/setup.exe","sha256":"abc"}`,
		"no version":        `{"version":"latest","url":"https://github.com/sparstrow/sparstrowgen/setup.exe","sha256":"` + someSHA + `"}`,
		"not a manifest":    `<html>not found</html>`,
		"an empty answer":   ``,
	} {
		if _, err := Parse([]byte(body), source); !errors.Is(err, ErrUntrusted) {
			t.Errorf("%s: err = %v, want ErrUntrusted", name, err)
		}
	}
	good := `{"version":"0.2.2","url":"https://github.com/sparstrow/sparstrowgen/setup.exe","sha256":"` + someSHA + `"}`
	if _, err := Parse([]byte(good), "http://github.com/update.json"); !errors.Is(err, ErrUntrusted) {
		t.Errorf("read over plain http: err = %v, want ErrUntrusted", err)
	}
}

// An older daemon reads a newer release's manifest.
func TestFieldsANewerReleaseAddsAreIgnored(t *testing.T) {
	body := `{"version":"0.3.0","url":"https://github.com/sparstrow/sparstrowgen/setup.exe","sha256":"` + someSHA + `","notes":"later"}`
	if m, err := Parse([]byte(body), source); err != nil || m.Version != "0.3.0" {
		t.Errorf("manifest = %+v, err = %v", m, err)
	}
}

func TestOnlyAnHTTPSInstallerWithARealChecksumCanBePublished(t *testing.T) {
	for _, m := range []Manifest{
		{Version: "0.2.1", URL: "http://example.test/a.exe", SHA256: someSHA},
		{Version: "0.2.1", URL: "https://example.test/a.exe", SHA256: "abc"},
		{Version: "latest", URL: "https://example.test/a.exe", SHA256: someSHA},
	} {
		if _, err := Encode(m); err == nil {
			t.Errorf("wrote an invalid manifest %+v", m)
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
