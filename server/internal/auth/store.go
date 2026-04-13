package auth

import (
	"sync"
	"time"
)

// Store keeps issued tokens in memory, keyed by the SHA-256 hex of the
// plaintext token. S4.2b will add disk persistence on top of this same
// API; for S4.2a the contents are lost on restart, which is acceptable
// — the client simply re-pairs.
type Store struct {
	mu     sync.RWMutex
	tokens map[string]TokenInfo
}

// NewStore returns an empty Store ready for concurrent use.
func NewStore() *Store {
	return &Store{tokens: make(map[string]TokenInfo)}
}

// Add inserts a TokenInfo keyed by its Hash. If the hash already
// exists the call is a no-op: we never overwrite a live token's
// timestamps with a stale copy.
func (s *Store) Add(info TokenInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tokens[info.Hash]; exists {
		return
	}
	s.tokens[info.Hash] = info
}

// Validate looks up the plaintext token, hashing it in constant time
// before the map lookup. When found it updates LastSeen to time.Now()
// and returns the refreshed TokenInfo. A missing token returns a zero
// TokenInfo and false.
func (s *Store) Validate(plaintext string) (TokenInfo, bool) {
	hash := HashToken(plaintext)
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.tokens[hash]
	if !ok {
		return TokenInfo{}, false
	}
	info.LastSeen = time.Now().UTC()
	s.tokens[hash] = info
	return info, true
}

// Revoke removes the token identified by hash. Missing hashes are
// silently ignored so callers can treat revoke as idempotent.
func (s *Store) Revoke(hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, hash)
}

// Count returns the number of tokens currently held by the Store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tokens)
}

// Snapshot returns a freshly allocated slice with a copy of every
// TokenInfo. The returned slice is independent of the Store's internal
// map — mutating it has no effect on the Store.
func (s *Store) Snapshot() []TokenInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TokenInfo, 0, len(s.tokens))
	for _, info := range s.tokens {
		out = append(out, info)
	}
	return out
}
