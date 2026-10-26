package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argon2id parameters follow the OWASP password storage recommendation
// (m=19 MiB, t=2, p=1). Hashes are stored in PHC string format so the
// parameters can be raised later and old hashes still verify.
const (
	argon2Memory  uint32 = 19 * 1024
	argon2Time    uint32 = 2
	argon2Threads uint8  = 1
	argon2KeyLen  uint32 = 32
	argon2SaltLen        = 16

	argon2Prefix = "$argon2id$"
)

// newPasswordHash returns a PHC-encoded argon2id hash with a random salt.
func newPasswordHash(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("%sv=%d$m=%d,t=%d,p=%d$%s$%s", argon2Prefix, argon2.Version,
		argon2Memory, argon2Time, argon2Threads, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

// verifyPassword checks password against either an argon2id PHC hash or a
// legacy salted SHA-256 hash (see hashPassword). salt is only used for the
// legacy format.
func verifyPassword(password, salt, expectedHash string) bool {
	if strings.HasPrefix(expectedHash, argon2Prefix) {
		return verifyArgon2id(password, expectedHash)
	}
	computed := hashPassword(password, salt)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(expectedHash)) == 1
}

// passwordNeedsRehash reports whether a stored hash should be replaced with
// a fresh argon2id hash on the next successful login.
func passwordNeedsRehash(hash string) bool {
	if !strings.HasPrefix(hash, argon2Prefix) {
		return true
	}
	m, t, p, _, _, ok := parseArgon2id(hash)
	return !ok || m != argon2Memory || t != argon2Time || p != argon2Threads
}

func verifyArgon2id(password, hash string) bool {
	m, t, p, salt, key, ok := parseArgon2id(hash)
	if !ok {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(key)))
	return subtle.ConstantTimeCompare(got, key) == 1
}

func parseArgon2id(hash string) (m, t uint32, p uint8, salt, key []byte, ok bool) {
	// $argon2id$v=19$m=..,t=..,p=..$salt$key
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return
	}
	var v int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &v); err != nil || v != argon2.Version {
		return
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || m == 0 || t == 0 || p == 0 {
		return
	}
	enc := base64.RawStdEncoding
	var err error
	if salt, err = enc.DecodeString(parts[4]); err != nil || len(salt) == 0 {
		return
	}
	if key, err = enc.DecodeString(parts[5]); err != nil || len(key) == 0 {
		return
	}
	return m, t, p, salt, key, true
}
