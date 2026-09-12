// Package credentials provides the primitives behind authentication: argon2id
// password hashing and the access/refresh JWTs.
package credentials

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// ErrPasswordMismatch is returned when a password does not match the hash.
var ErrPasswordMismatch = errors.New("password does not match")

// OWASP-recommended argon2id baseline: 19 MiB, 2 iterations, 1 lane.
const (
	defaultMemory  uint32 = 19 * 1024
	defaultTime    uint32 = 2
	defaultThreads uint8  = 1
	defaultKeyLen  uint32 = 32
	saltLen               = 16
)

// HashPassword derives a PHC-encoded argon2id hash using a fresh random salt.
func HashPassword(plain string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	key := argon2.IDKey([]byte(plain), salt, defaultTime, defaultMemory, defaultThreads, defaultKeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, defaultMemory, defaultTime, defaultThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether plain matches the given PHC-encoded hash. It
// returns ErrPasswordMismatch on a wrong password and a different error if the
// hash is malformed.
func VerifyPassword(plain, encoded string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return errors.New("malformed argon2id hash")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return fmt.Errorf("parsing argon2id version: %w", err)
	}
	if version != argon2.Version {
		return fmt.Errorf("unsupported argon2id version %d", version)
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return fmt.Errorf("parsing argon2id parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return fmt.Errorf("decoding salt: %w", err)
	}
	want, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return fmt.Errorf("decoding hash: %w", err)
	}

	got := argon2.IDKey([]byte(plain), salt, time, memory, threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrPasswordMismatch
	}
	return nil
}
