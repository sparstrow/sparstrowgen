// Package auth is the front door: who is allowed to reach the API at all.
//
// It exists because of what is behind it. Every other endpoint in this server
// eventually causes a coding agent to run on the owner's own machine, in a
// folder the request names, with whatever tools that agent can reach. That is
// not an app with a login bolted on — the login is the only thing standing
// between a stranger and a shell.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

/* Password hashing.

argon2id, which is what you reach for when the alternative is bcrypt and the
year is not 2013: it is memory-hard, so an attacker with a GPU does not get the
enormous advantage over a server that bcrypt hands them.

The parameters below are the current OWASP baseline (19 MiB, 2 iterations, 1
degree of parallelism). They are ENCODED INTO EVERY HASH rather than read from a
constant at verify time, so raising them later does not invalidate the password
the owner already set — an old hash still verifies with its own parameters, and
only a fresh hash uses the new ones. Constants that silently lock you out of
your own deployment are not a theoretical problem. */

const (
	argonTime    = 2
	argonMemory  = 19 * 1024 // KiB
	argonThreads = 1
	argonKeyLen  = 32
	saltLen      = 16
)

// ErrBadHash means the configured hash is not one we wrote — malformed, or from
// some other tool. It is deliberately distinct from "wrong password": one is the
// deployment being misconfigured and the other is somebody guessing.
var ErrBadHash = errors.New("the stored password hash is not in a format this server understands")

// HashPassword produces the PHC-format string that goes in OWNER_PASSWORD_HASH.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating a salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches encoded.
//
// The comparison is constant-time. The timing of a byte-by-byte compare is
// measurable over a network, and leaking "your first three characters were
// right" one request at a time turns an unguessable password into a guessable
// one.
func VerifyPassword(encoded, password string) (bool, error) {
	params, salt, want, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt,
		params.time, params.memory, params.threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

// decodeHash reads the parameters back out of the hash, so a hash written under
// older settings keeps verifying under its own.
func decodeHash(encoded string) (argonParams, []byte, []byte, error) {
	var p argonParams
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, nil, nil, ErrBadHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, nil, nil, ErrBadHash
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return p, nil, nil, ErrBadHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return p, nil, nil, ErrBadHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return p, nil, nil, ErrBadHash
	}
	return p, salt, key, nil
}
