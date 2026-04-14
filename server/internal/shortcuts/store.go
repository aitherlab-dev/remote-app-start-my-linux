// Package shortcuts manages user-defined custom launcher entries —
// small records that describe "run this command in that directory
// inside such terminal". They are the mechanism behind one of the
// project's main use cases: tapping a tile on the phone and having
// `claude` (or any other tool) pop open on the desktop inside the
// right project folder.
//
// Shortcuts live in shortcuts.json alongside tokens.json and
// visibility.json. On disk they look like:
//
//	{
//	  "shortcuts": [
//	    {
//	      "id": "claude-myproj",
//	      "name": "Claude: MyProj",
//	      "command": "claude",
//	      "cwd": "/home/sasha/WORK/MyProj",
//	      "terminal": "kitty",
//	      "icon": "🤖"
//	    }
//	  ]
//	}
//
// The Store is concurrency-safe and persists atomically (temp file +
// rename), exactly like the visibility.Store — crashes mid-write
// never truncate the file on disk.
package shortcuts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// IDPrefix is the namespace under which shortcut ids are surfaced on
// the REST API so they cannot collide with real .desktop app ids.
// The web UI and the HTTP handlers strip and attach it; the on-disk
// file stores ids without the prefix.
const IDPrefix = "custom:"

// Shortcut is one user-defined launcher entry. All fields are
// validated by the Store on save: id must be non-empty and unique
// within the set, name must be non-empty, command must be non-empty.
// cwd, terminal and icon are optional. A blank terminal means "use
// the server-wide default configured in config.toml".
type Shortcut struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Command  string `json:"command"`
	Cwd      string `json:"cwd,omitempty"`
	Terminal string `json:"terminal,omitempty"`
	Icon     string `json:"icon,omitempty"`
}

// persisted is the JSON shape written to disk. Wrapping the slice in
// a struct leaves room for a future version/schema field without
// breaking old files.
type persisted struct {
	Shortcuts []Shortcut `json:"shortcuts"`
}

// ErrInvalidShortcut is returned when a Shortcut fails validation.
var ErrInvalidShortcut = errors.New("invalid shortcut")

// Store is the concurrency-safe, file-backed shortcut registry.
// The zero value is not usable — always construct via NewStore().
type Store struct {
	mu    sync.RWMutex
	items map[string]Shortcut
	order []string
	path  string
}

// NewStore returns an empty in-memory shortcut store. Call Load (or
// SetPath + Load) before first use in production so Save knows
// where to put the file.
func NewStore() *Store {
	return &Store{items: make(map[string]Shortcut)}
}

// Load reads the JSON file at path into the store, replacing any
// current state. A missing or empty file is not an error — the store
// is left empty and path is remembered so that the next Save creates
// the file on disk. A malformed file is an error so that the server
// refuses to start rather than silently wiping the operator's
// shortcuts.
func (s *Store) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s.items = make(map[string]Shortcut)
			s.order = nil
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		s.items = make(map[string]Shortcut)
		s.order = nil
		return nil
	}
	var p persisted
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	items := make(map[string]Shortcut, len(p.Shortcuts))
	order := make([]string, 0, len(p.Shortcuts))
	for _, sc := range p.Shortcuts {
		if err := validate(sc); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if _, dup := items[sc.ID]; dup {
			return fmt.Errorf("%s: duplicate shortcut id %q", path, sc.ID)
		}
		items[sc.ID] = sc
		order = append(order, sc.ID)
	}
	s.items = items
	s.order = order
	return nil
}

// SetPath overrides the persistence path without touching the
// in-memory state.
func (s *Store) SetPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = path
}

// Path returns the configured on-disk location.
func (s *Store) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

// Get returns the shortcut with the given id. The second return
// value is false when the id is unknown. The id must be passed
// WITHOUT the IDPrefix.
func (s *Store) Get(id string) (Shortcut, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.items[id]
	return sc, ok
}

// List returns a sorted (insertion-order stable) snapshot of the
// current shortcut set. The returned slice is owned by the caller.
func (s *Store) List() []Shortcut {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Shortcut, 0, len(s.items))
	for _, id := range s.order {
		if sc, ok := s.items[id]; ok {
			out = append(out, sc)
		}
	}
	return out
}

// Count returns the number of shortcuts currently stored.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// Replace atomically replaces the entire shortcut set with the given
// slice and persists the new state. Validation is performed on the
// full slice before any mutation — an error leaves the old state
// intact. Duplicate ids, blank name/command, or missing id are
// rejected.
func (s *Store) Replace(list []Shortcut) error {
	next := make(map[string]Shortcut, len(list))
	order := make([]string, 0, len(list))
	for i, sc := range list {
		sc.ID = strings.TrimSpace(sc.ID)
		sc.Name = strings.TrimSpace(sc.Name)
		sc.Command = strings.TrimSpace(sc.Command)
		sc.Cwd = strings.TrimSpace(sc.Cwd)
		sc.Terminal = strings.TrimSpace(sc.Terminal)
		sc.Icon = strings.TrimSpace(sc.Icon)
		if err := validate(sc); err != nil {
			return fmt.Errorf("shortcut #%d: %w", i, err)
		}
		if _, dup := next[sc.ID]; dup {
			return fmt.Errorf("shortcut #%d: %w: duplicate id %q", i, ErrInvalidShortcut, sc.ID)
		}
		next[sc.ID] = sc
		order = append(order, sc.ID)
	}

	s.mu.Lock()
	s.items = next
	s.order = order
	path := s.path
	snapshot := make([]Shortcut, 0, len(order))
	for _, id := range order {
		snapshot = append(snapshot, next[id])
	}
	s.mu.Unlock()

	if path == "" {
		return nil
	}
	return writeAtomic(path, snapshot)
}

// validate checks a single Shortcut record. Blank fields that are
// required produce ErrInvalidShortcut; the wrapped detail explains
// which field failed so the HTTP handler can surface it.
func validate(sc Shortcut) error {
	if sc.ID == "" {
		return fmt.Errorf("%w: id must not be empty", ErrInvalidShortcut)
	}
	if strings.Contains(sc.ID, "/") || strings.Contains(sc.ID, " ") {
		return fmt.Errorf("%w: id %q must not contain spaces or slashes", ErrInvalidShortcut, sc.ID)
	}
	if sc.Name == "" {
		return fmt.Errorf("%w: name must not be empty", ErrInvalidShortcut)
	}
	if sc.Command == "" {
		return fmt.Errorf("%w: command must not be empty", ErrInvalidShortcut)
	}
	return nil
}

// writeAtomic marshals shortcuts as the persisted JSON shape and
// writes it to path via temp-file + rename so a crash mid-write
// never leaves a truncated file.
func writeAtomic(path string, list []Shortcut) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	// Sort for deterministic output when the caller does not care
	// about order — current callers always feed a concrete order,
	// but stable output is nice for diffs.
	sort.SliceStable(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	data, err := json.MarshalIndent(persisted{Shortcuts: list}, "", "  ")
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

// NormalizeID strips IDPrefix from id if present. Handlers use this
// to translate a REST path id ("custom:foo") into the on-disk id
// ("foo") the Store speaks.
func NormalizeID(id string) (string, bool) {
	if strings.HasPrefix(id, IDPrefix) {
		return strings.TrimPrefix(id, IDPrefix), true
	}
	return id, false
}

// PrefixedID returns id with the IDPrefix attached, used when
// emitting shortcut entries into /api/apps so the Android client
// and the icon handler cannot confuse them with real app ids.
func PrefixedID(id string) string { return IDPrefix + id }
