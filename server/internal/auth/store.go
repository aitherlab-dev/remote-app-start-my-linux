package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store keeps issued tokens in memory, keyed by the SHA-256 hex of the
// plaintext token. With SetPersistPath it also mirrors every Add/Revoke
// to disk so that tokens survive a server restart; without it, the
// contents are lost on exit and the client must re-pair.
type Store struct {
	mu          sync.RWMutex
	tokens      map[string]TokenInfo
	persistPath string
	errLog      func(error)
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
	if _, exists := s.tokens[info.Hash]; exists {
		s.mu.Unlock()
		return
	}
	s.tokens[info.Hash] = info
	path, errLog, snap := s.persistStateLocked()
	s.mu.Unlock()
	s.persistSnapshot(path, errLog, snap)
}

// Validate looks up the plaintext token, hashing it in constant time
// before the map lookup. When found it updates LastSeen to time.Now()
// and returns the refreshed TokenInfo. A missing token returns a zero
// TokenInfo and false.
//
// Validate deliberately does not persist the LastSeen bump: writing the
// tokens file on every authenticated request would turn a cheap call
// into a disk round-trip. LastSeen is best-effort telemetry and will be
// rewritten on the next Add/Revoke anyway.
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
	if _, exists := s.tokens[hash]; !exists {
		s.mu.Unlock()
		return
	}
	delete(s.tokens, hash)
	path, errLog, snap := s.persistStateLocked()
	s.mu.Unlock()
	s.persistSnapshot(path, errLog, snap)
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
	return s.snapshotLocked()
}

// SetPersistPath enables automatic persistence: every subsequent
// Add/Revoke will atomically rewrite `path` with the current set of
// tokens. errLog is called with any disk error so the caller can route
// it into slog; passing nil silently drops persistence errors.
//
// Callers typically pair this with Load(path) at startup: Load
// populates the in-memory state, then SetPersistPath attaches the same
// file as the write target.
func (s *Store) SetPersistPath(path string, errLog func(error)) {
	s.mu.Lock()
	s.persistPath = path
	s.errLog = errLog
	s.mu.Unlock()
}

// Load reads a tokens file from disk and replaces the Store's contents
// with it. A missing file is not an error — it is the normal state of a
// fresh install — but a present-but-malformed file is, because that
// means either user corruption or a real bug, and silently dropping
// every token on startup would be worse than failing loudly.
//
// Load does not attach the path for automatic persistence. Call
// SetPersistPath separately for that so tests can exercise Load without
// spraying disk writes.
func (s *Store) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("auth: read tokens file: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	var infos []TokenInfo
	if err := json.Unmarshal(data, &infos); err != nil {
		return fmt.Errorf("auth: parse tokens file %s: %w", path, err)
	}
	next := make(map[string]TokenInfo, len(infos))
	for _, info := range infos {
		if info.Hash == "" {
			continue
		}
		next[info.Hash] = info
	}
	s.mu.Lock()
	s.tokens = next
	s.mu.Unlock()
	return nil
}

// Save writes the current token set to the configured persist path.
// If SetPersistPath was never called it returns nil: callers who want
// an explicit save without attaching a path can call SaveTo instead.
func (s *Store) Save() error {
	s.mu.RLock()
	path := s.persistPath
	snap := s.snapshotLocked()
	s.mu.RUnlock()
	if path == "" {
		return nil
	}
	return writeTokensFile(path, snap)
}

// SaveTo writes the current token set to an explicit path, bypassing
// any persist path configured via SetPersistPath. It is used by tests
// and by main.go before the persist path is wired up.
func (s *Store) SaveTo(path string) error {
	s.mu.RLock()
	snap := s.snapshotLocked()
	s.mu.RUnlock()
	return writeTokensFile(path, snap)
}

// snapshotLocked returns a fresh slice copy of the token map. Callers
// must already hold s.mu (read or write).
func (s *Store) snapshotLocked() []TokenInfo {
	out := make([]TokenInfo, 0, len(s.tokens))
	for _, info := range s.tokens {
		out = append(out, info)
	}
	return out
}

// persistStateLocked captures the fields needed to write to disk under
// the current lock so Add/Revoke can release s.mu before touching the
// filesystem. Returning a snapshot instead of calling Save under the
// lock is what keeps us deadlock-free: Save would re-acquire s.mu as a
// reader, and Lock→RLock re-entry is not allowed by sync.RWMutex.
func (s *Store) persistStateLocked() (string, func(error), []TokenInfo) {
	if s.persistPath == "" {
		return "", nil, nil
	}
	return s.persistPath, s.errLog, s.snapshotLocked()
}

// persistSnapshot writes a pre-captured token slice to disk and routes
// any error through errLog. It runs without holding s.mu.
func (s *Store) persistSnapshot(path string, errLog func(error), snap []TokenInfo) {
	if path == "" {
		return
	}
	if err := writeTokensFile(path, snap); err != nil {
		if errLog != nil {
			errLog(err)
		}
	}
}

// writeTokensFile atomically replaces `path` with a JSON array of
// TokenInfo. It writes to a temp file in the same directory (so the
// rename is guaranteed to stay on one filesystem), fsyncs, and renames
// on top of the destination. The parent directory is created with
// 0o700 if it does not already exist, and the final file is 0o600 —
// only the user running the server can read its own tokens.
func writeTokensFile(path string, snap []TokenInfo) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("auth: create tokens dir: %w", err)
	}
	// Ensure the slice marshals as [] rather than null on empty input —
	// Load treats both as "no tokens" but [] is friendlier to humans
	// inspecting the file by hand.
	if snap == nil {
		snap = []TokenInfo{}
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("auth: marshal tokens: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".tokens-*.json.tmp")
	if err != nil {
		return fmt.Errorf("auth: create tokens temp: %w", err)
	}
	tmpPath := tmp.Name()
	// On any failure past this point we must not leave the temp file
	// behind; a deferred Remove is a no-op once Rename has moved it.
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("auth: write tokens temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("auth: chmod tokens temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("auth: fsync tokens temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("auth: close tokens temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("auth: rename tokens file: %w", err)
	}
	cleanup = false
	return nil
}
