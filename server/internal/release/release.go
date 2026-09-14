// Package release is a daemon release as an installed computer sees it: a
// manifest naming the newest version, where its installer is published, and
// that installer's SHA-256 (docs/Decisions.md D-035).
//
// Trust is our GitHub releases over HTTPS plus that checksum, as Multica does
// it: a manifest must name an installer on the site it was read from, and a
// download that does not match its SHA-256 is refused. There is no signing key
// to keep or lose. One definition, used by both the release build that writes
// the manifest and the daemon that reads it, so the two cannot drift.
package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Manifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// ErrUntrusted means a manifest or a download failed its checks. Nothing that
// fails them is ever used.
var ErrUntrusted = errors.New("the update could not be verified")

// ParseVersion reads "major.minor.patch", with or without a leading "v".
func ParseVersion(v string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("%q is not a version", v)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, fmt.Errorf("%q is not a version", v)
		}
		out[i] = n
	}
	return out, nil
}

// Newer reports whether candidate is a later version than current.
func Newer(candidate, current string) (bool, error) {
	c, err := ParseVersion(candidate)
	if err != nil {
		return false, err
	}
	cur, err := ParseVersion(current)
	if err != nil {
		return false, err
	}
	for i := range c {
		if c[i] != cur[i] {
			return c[i] > cur[i], nil
		}
	}
	return false, nil
}

func (m Manifest) validate() error {
	if _, err := ParseVersion(m.Version); err != nil {
		return err
	}
	if !strings.HasPrefix(m.URL, "https://") {
		return fmt.Errorf("the installer URL must be https: %q", m.URL)
	}
	if b, err := hex.DecodeString(m.SHA256); err != nil || len(b) != sha256.Size {
		return fmt.Errorf("%q is not a SHA-256", m.SHA256)
	}
	return nil
}

// Encode returns the manifest exactly as published.
func Encode(m Manifest) ([]byte, error) {
	m.SHA256 = strings.ToLower(m.SHA256)
	if err := m.validate(); err != nil {
		return nil, err
	}
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

// Parse reads a manifest fetched from source. It is refused unless it is well
// formed and names an https installer on the same site as source, so a manifest
// can never send a computer to download from anywhere else. Fields it does not
// know are ignored: a newer release may add some, and this copy must still read it.
func Parse(body []byte, source string) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return Manifest{}, ErrUntrusted
	}
	if err := m.validate(); err != nil {
		return Manifest{}, ErrUntrusted
	}
	if !sameSite(m.URL, source) {
		return Manifest{}, ErrUntrusted
	}
	m.SHA256 = strings.ToLower(m.SHA256)
	return m, nil
}

func sameSite(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return ua.Scheme == "https" && ub.Scheme == "https" && ua.Host != "" && strings.EqualFold(ua.Host, ub.Host)
}

// FileSHA256 is the lower-case hex SHA-256 of a file.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
