// Package release is a daemon release as an installed computer sees it: a
// manifest naming the newest version and its installer, and a detached
// signature proving the manifest came from us.
//
// The installer itself is not signed here. Its SHA-256 is inside the signed
// manifest, so a download that does not match is refused (docs/Decisions.md
// D-034). One definition, used by both the release tool that signs and the
// daemon that verifies, so the two cannot drift.
package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Manifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// ErrUntrusted means the signature or the content did not verify. Nothing that
// fails verification is ever used.
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

// Sign returns the manifest body exactly as published, and its signature as
// base64 text.
func Sign(key ed25519.PrivateKey, m Manifest) (body, signature []byte, err error) {
	m.SHA256 = strings.ToLower(m.SHA256)
	if err := m.validate(); err != nil {
		return nil, nil, err
	}
	body, err = json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	body = append(body, '\n')
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(key, body))
	return body, []byte(sig + "\n"), nil
}

// Verify checks the signature over the exact bytes received before reading a
// single field from them.
func Verify(body, signature []byte, publicKey string) (Manifest, error) {
	pub, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return Manifest{}, errors.New("this build has no usable update key")
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signature)))
	if err != nil || !ed25519.Verify(ed25519.PublicKey(pub), body, sig) {
		return Manifest{}, ErrUntrusted
	}
	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return Manifest{}, ErrUntrusted
	}
	if err := m.validate(); err != nil {
		return Manifest{}, ErrUntrusted
	}
	m.SHA256 = strings.ToLower(m.SHA256)
	return m, nil
}

// GenerateKey writes a new signing key and returns its public half. It refuses
// to replace an existing key: every installed computer trusts the public half,
// and losing it means they can no longer update themselves.
func GenerateKey(path string) (public string, err error) {
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists; refusing to replace the signing key installed computers trust", path)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	seed := base64.StdEncoding.EncodeToString(priv.Seed())
	if err := os.WriteFile(path, []byte(seed+"\n"), 0o600); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

func LoadKey(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("%s is not a signing key", path)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

func PublicKey(key ed25519.PrivateKey) string {
	return base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey))
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
