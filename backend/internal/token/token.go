// Package token generates and hashes opaque random tokens (sessions, email links).
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const byteLength = 32

// New returns a random URL-safe token and its hash. Only the hash may be stored;
// the raw token is handed to the client exactly once.
func New() (raw, hash string, err error) {
	buf := make([]byte, byteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, Hash(raw), nil
}

// Hash returns the hex-encoded SHA-256 of a raw token. The tokens carry 256 bits of
// entropy, so a fast hash is sufficient to protect them at rest.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
