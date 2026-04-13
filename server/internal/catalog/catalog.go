// Package catalog is a concurrency-safe, in-memory view over the
// freedesktop applications discovered by the desktop package.
//
// It stores two parallel representations of the scan result:
//   - the full desktop.Entry (with Exec and friends) keyed by ID for
//     the launcher;
//   - AppInfo, a DTO stripped of execution-sensitive fields for the
//     HTTP layer.
//
// Reload scans the filesystem outside the lock and only takes a brief
// write-lock to swap the state, so readers never block on IO.
package catalog

import (
	"sort"
	"strings"
	"sync"

	"github.com/sasha/remotelauncher/internal/desktop"
)

// AppInfo is the HTTP-facing DTO describing a single application.
//
// Fields that could leak execution details or filesystem paths
// (Exec, TryExec, Path, Hidden, OnlyShowIn, NotShowIn, StartupNotify)
// are intentionally omitted — this type is what the REST API returns.
type AppInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Comment    string   `json:"comment,omitempty"`
	Icon       string   `json:"icon,omitempty"`
	Categories []string `json:"categories,omitempty"`
	NoDisplay  bool     `json:"-"`
}

// Catalog is the authoritative in-memory list of applications. It
// wraps desktop.Scan with a sync.RWMutex so Load/Reload can be called
// concurrently with List/Get/GetInfo.
type Catalog struct {
	paths []string

	mu   sync.RWMutex
	byID map[string]desktop.Entry
	list []AppInfo
}

// New returns an empty Catalog that will scan the given paths on
// Load/Reload. If paths is nil or empty the catalog falls back to
// desktop.DefaultPaths() at scan time.
//
// paths are consumed in order of increasing priority — see desktop.Scan.
func New(paths []string) *Catalog {
	return &Catalog{paths: paths}
}

// Load performs the initial scan. It is safe to call more than once
// but Reload is the idiomatic way to refresh the catalog after the
// first load.
func (c *Catalog) Load() (int, []desktop.ScanError, error) {
	return c.Reload()
}

// Reload rescans the configured paths and atomically replaces the
// stored state. The filesystem walk runs without holding any lock;
// only the final state swap is performed under the write-lock, so
// readers are blocked for microseconds at most.
func (c *Catalog) Reload() (int, []desktop.ScanError, error) {
	paths := c.paths
	if len(paths) == 0 {
		paths = desktop.DefaultPaths()
	}

	entries, scanErrs := desktop.Scan(paths)

	byID := make(map[string]desktop.Entry, len(entries))
	list := make([]AppInfo, 0, len(entries))
	for _, e := range entries {
		byID[e.ID] = e
		list = append(list, entryToAppInfo(e))
	}
	sortAppInfos(list)

	c.mu.Lock()
	c.byID = byID
	c.list = list
	c.mu.Unlock()

	return len(list), scanErrs, nil
}

// List returns a copy of the current application list, sorted by Name
// (case-insensitive) with ID as the tie-breaker. The returned slice is
// owned by the caller and may be mutated freely.
func (c *Catalog) List() []AppInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]AppInfo(nil), c.list...)
}

// Get returns the full desktop.Entry for id, including Exec. It is
// intended for the launcher; the HTTP layer must use GetInfo instead.
// The second return value is false when the id is unknown.
func (c *Catalog) Get(id string) (desktop.Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.byID[id]
	return e, ok
}

// GetInfo returns the HTTP-safe AppInfo for id. The second return
// value is false when the id is unknown.
func (c *Catalog) GetInfo(id string) (AppInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.byID[id]
	if !ok {
		return AppInfo{}, false
	}
	return entryToAppInfo(e), true
}

func entryToAppInfo(e desktop.Entry) AppInfo {
	return AppInfo{
		ID:         e.ID,
		Name:       e.Name,
		Comment:    e.Comment,
		Icon:       e.Icon,
		Categories: e.Categories,
		NoDisplay:  e.NoDisplay,
	}
}

func sortAppInfos(list []AppInfo) {
	sort.Slice(list, func(i, j int) bool {
		ni := strings.ToLower(list[i].Name)
		nj := strings.ToLower(list[j].Name)
		if ni != nj {
			return ni < nj
		}
		return list[i].ID < list[j].ID
	})
}
