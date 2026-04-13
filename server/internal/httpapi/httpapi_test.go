package httpapi

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/desktop"
	"github.com/sasha/remotelauncher/internal/icons"
)

// fakeLauncher is the AppLauncher test double used across launch /
// router tests. It records the entry it was called with and returns
// the configured pid/err pair.
type fakeLauncher struct {
	pid    int
	err    error
	called desktop.Entry
	calls  int
}

func (f *fakeLauncher) Launch(e desktop.Entry) (int, error) {
	f.called = e
	f.calls++
	return f.pid, f.err
}

// fakeAlive is the AliveChecker test double. It reports true for any
// id present in the alive set and false otherwise.
type fakeAlive struct {
	alive map[string]bool
}

func (f *fakeAlive) Alive(id string) bool {
	if f == nil || f.alive == nil {
		return false
	}
	return f.alive[id]
}

// newRouterFor builds a router with sensible test defaults. Pass nil
// for collaborators that should be left out of the test (the router
// will still wire them up, but the handler endpoint won't be hit).
func newRouterFor(t *testing.T, c *catalog.Catalog, finder *icons.Finder, l AppLauncher, alive AliveChecker) http.Handler {
	t.Helper()
	if finder == nil {
		finder = icons.New([]string{t.TempDir()}, "hicolor")
	}
	if l == nil {
		l = &fakeLauncher{}
	}
	if alive == nil {
		alive = &fakeAlive{}
	}
	return NewRouter(RouterDeps{
		Version:     "dev",
		StartedAt:   time.Now().Add(-time.Second),
		Catalog:     c,
		Finder:      finder,
		Launcher:    l,
		Alive:       alive,
		Fingerprint: testFingerprint,
	})
}

const desktopTemplate = `[Desktop Entry]
Type=Application
Name=%s
Exec=/bin/%s
Comment=%s
Icon=%s
Categories=%s;
`

// newTestCatalog builds an isolated Catalog rooted in t.TempDir. files
// maps a file name (relative to the temp dir) to its raw contents so
// tests can craft exactly the .desktop corpus they need.
func newTestCatalog(t *testing.T, files map[string]string) *catalog.Catalog {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}
	c := catalog.New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("catalog load: %v", err)
	}
	return c
}

// desktopEntry is a convenience wrapper around desktopTemplate.
func desktopEntry(name, exec, comment, icon, category string) string {
	return fmt.Sprintf(desktopTemplate, name, exec, comment, icon, category)
}
