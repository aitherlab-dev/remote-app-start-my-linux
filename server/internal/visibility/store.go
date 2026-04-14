// Package visibility is a concurrency-safe, file-backed set of
// application IDs that the operator has chosen to hide from the
// Android client. The HTTP layer filters /api/apps through Store
// before sending the list to the phone; the web admin UI reads and
// writes the same Store so the operator can toggle visibility
// interactively.
//
// The persistence format is a tiny JSON document:
//
//	{"hidden": ["chromium", "firefox"]}
//
// Load creates an empty set when the file does not exist — this is
// the first-run case and not an error. Save writes atomically
// (temp-file + rename) so a crash mid-write never leaves a truncated
// file on disk.
package visibility

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Store holds the current set of hidden application IDs. Every
// accessor is safe to call from multiple goroutines; readers take the
// RLock and writers take the Lock, so reads never block on each other.
//
// The zero value is not usable — always obtain a Store via NewStore().
type Store struct {
	mu     sync.RWMutex
	hidden map[string]struct{}
	path   string
}

// persisted is the JSON shape written to disk. Keeping it in a
// private type lets us evolve the on-disk format (e.g. add a
// "version" field) without touching the public API.
type persisted struct {
	Hidden []string `json:"hidden"`
}

// NewStore returns an empty, in-memory visibility store. Call Load
// (or SetPath + Load) before first use in production so the store
// knows where to persist subsequent updates.
func NewStore() *Store {
	return &Store{hidden: make(map[string]struct{})}
}

// Load reads the JSON file at path into the store, replacing any
// current state. A missing file is not an error — the store is left
// empty and path is remembered so that the next Save creates the file
// on disk. A malformed file is an error: rather than silently dropping
// the operator's configuration, we refuse to start.
func (s *Store) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s.hidden = make(map[string]struct{})
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		s.hidden = make(map[string]struct{})
		return nil
	}
	var p persisted
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	next := make(map[string]struct{}, len(p.Hidden))
	for _, id := range p.Hidden {
		if id == "" {
			continue
		}
		next[id] = struct{}{}
	}
	s.hidden = next
	return nil
}

// SetPath overrides the persistence path without touching the in-memory
// state. Mostly useful for tests that want to load a snapshot from one
// location and then redirect writes elsewhere.
func (s *Store) SetPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = path
}

// Path returns the current on-disk location, or the empty string if
// Load/SetPath has never been called.
func (s *Store) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

// IsHidden reports whether id is present in the hidden set. It is the
// hot-path predicate used by /api/apps when filtering the catalog.
func (s *Store) IsHidden(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.hidden[id]
	return ok
}

// Hidden returns a sorted copy of the current hidden set. The
// returned slice is owned by the caller.
func (s *Store) Hidden() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.hidden))
	for id := range s.hidden {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Count returns the number of hidden entries. Cheaper than len(Hidden())
// because it avoids the allocation.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.hidden)
}

// SetHidden replaces the entire hidden set with ids and persists the
// new state. Duplicates and empty strings are silently dropped. If no
// path has been configured, SetHidden updates the in-memory state and
// returns nil — useful for tests, but production code always calls
// Load first.
func (s *Store) SetHidden(ids []string) error {
	next := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		next[id] = struct{}{}
	}

	s.mu.Lock()
	s.hidden = next
	path := s.path
	snapshot := sortedKeys(next)
	s.mu.Unlock()

	if path == "" {
		return nil
	}
	return writeAtomic(path, snapshot)
}

// writeAtomic marshals ids as the persisted JSON shape and writes it
// to path via temp-file + rename, guaranteeing that a crash mid-write
// never leaves a truncated file. The parent directory is created if
// needed, mirroring how the token store handles first-run state.
func writeAtomic(path string, ids []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(persisted{Hidden: ids}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %s: %w", path, err)
	}
	return nil
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
