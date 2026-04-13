// Package auth implements PIN generation, token issuing and in-memory
// token storage for the remotelauncher pairing flow.
//
// The pairing protocol is intentionally tiny: on startup the server
// prints a fresh six-digit PIN to stdout, the client sends it to
// POST /api/pair and receives a 32-byte bearer token. Only the SHA-256
// hash of the token is kept on the server; the plaintext is shown to
// the client exactly once. Subsequent requests authenticate via
// Authorization: Bearer <token>.
package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// pinUpperBound is 10^6, the exclusive upper bound for a 6-digit PIN.
var pinUpperBound = big.NewInt(1_000_000)

// GeneratePIN returns a 6-digit cryptographically random PIN encoded as
// a decimal string. The result always has exactly six characters; small
// numbers are padded with leading zeros (e.g. 42 -> "000042").
func GeneratePIN() (string, error) {
	n, err := rand.Int(rand.Reader, pinUpperBound)
	if err != nil {
		return "", fmt.Errorf("auth: generate pin: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// PINSession holds the current pairing PIN with one-shot semantics and
// an optional expiration window. A PINSession is safe for concurrent
// use.
type PINSession struct {
	mu        sync.Mutex
	pin       string
	consumed  bool
	expiresAt time.Time
}

// NewPINSession creates a session with a fresh 6-digit PIN valid for
// the given ttl. A ttl of 0 or less disables expiration, which is the
// right choice for MVP where the server holds the PIN for its entire
// lifetime until the first successful pairing.
func NewPINSession(ttl time.Duration) (*PINSession, error) {
	pin, err := GeneratePIN()
	if err != nil {
		return nil, err
	}
	s := &PINSession{pin: pin}
	if ttl > 0 {
		s.expiresAt = time.Now().Add(ttl)
	}
	return s, nil
}

// Current returns the PIN regardless of its consumed or expired state.
// It is meant for the /api/pair handler's constant-time comparison,
// which then calls Consume to enforce the state transition.
func (s *PINSession) Current() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pin
}

// Consume marks the PIN as used and reports whether the transition
// succeeded. A second Consume call, or any call after the PIN has
// expired, returns false and leaves no state behind for replay.
func (s *PINSession) Consume() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.consumed {
		return false
	}
	if !s.expiresAt.IsZero() && time.Now().After(s.expiresAt) {
		return false
	}
	s.consumed = true
	return true
}

// String renders the session in a form suitable for logging. The PIN
// is plainly visible: the design intent is that the server operator
// (who sees the log) must see the PIN to type it into the phone.
func (s *PINSession) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("PINSession{pin=%s}", s.pin)
}
