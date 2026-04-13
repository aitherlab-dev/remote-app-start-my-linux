package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// tokenBytes is the raw entropy length of an issued bearer token.
// 32 bytes = 256 bits, in line with NIST recommendations for opaque
// session tokens.
const tokenBytes = 32

// TokenInfo describes an issued token without carrying the plaintext.
// The hash is the only thing the server persists; the plaintext is
// shown to the client exactly once on /api/pair and then forgotten.
type TokenInfo struct {
	Hash        string    `json:"hash"`
	DeviceLabel string    `json:"device_label"`
	CreatedAt   time.Time `json:"created_at"`
	LastSeen    time.Time `json:"last_seen"`
}

// IssueToken generates a fresh 32-byte cryptographic token. It returns
// the base64url-encoded plaintext (no padding) and a TokenInfo that
// contains only the SHA-256 hash of that plaintext. Callers pair the
// two: send the plaintext to the client, keep the TokenInfo in the
// Store.
func IssueToken(label string) (plaintext string, info TokenInfo, err error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", TokenInfo{}, fmt.Errorf("auth: read random token: %w", err)
	}
	plaintext = base64.RawURLEncoding.EncodeToString(buf)
	now := time.Now().UTC()
	info = TokenInfo{
		Hash:        HashToken(plaintext),
		DeviceLabel: label,
		CreatedAt:   now,
		LastSeen:    now,
	}
	return plaintext, info, nil
}

// HashToken returns the lowercase SHA-256 hex of the plaintext token.
// It is deterministic: repeated calls with the same plaintext always
// produce the same hash, which is what Store uses as the lookup key.
func HashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
