package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sasha/remotelauncher/internal/auth"
	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/desktop"
	"github.com/sasha/remotelauncher/internal/icons"
)

// testBearerToken is the plaintext token seeded into every test router
// by newRouterFor. Tests that exercise protected endpoints can build
// requests with authedRequest, which attaches the matching Bearer
// header. The string is opaque; the test suite only cares that its
// SHA-256 hash matches the entry newRouterFor inserts into the store.
const testBearerToken = "test-bearer-token"

// authedRequest is the test-only equivalent of httptest.NewRequest
// that attaches the standard Bearer header matching testBearerToken.
// Every post-S4.2a test against a protected endpoint should build its
// request through this helper so the auth middleware lets it through.
func authedRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Authorization", "Bearer "+testBearerToken)
	return req
}

// fakePINProvider is the PINProvider test double for pair_test.go. It
// returns a fixed PIN and keeps a configurable Consume outcome so
// tests can simulate both the fresh and the already-used session.
type fakePINProvider struct {
	pin       string
	consumeOK bool
	consumed  int
}

func (f *fakePINProvider) Current() string {
	return f.pin
}

func (f *fakePINProvider) Consume() bool {
	f.consumed++
	return f.consumeOK
}

// fakeTokenIssuer is the TokenIssuer test double for pair_test.go. It
// records the last label it was asked to mint a token for, and either
// returns the canned plaintext or the canned error.
type fakeTokenIssuer struct {
	token string
	err   error
	label string
	calls int
}

func (f *fakeTokenIssuer) Issue(label string) (string, error) {
	f.calls++
	f.label = label
	return f.token, f.err
}

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
//
// The returned router is seeded with a token store that already knows
// testBearerToken, so tests can hit protected endpoints through
// authedRequest without having to run the /api/pair flow.
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
	store := auth.NewStore()
	now := time.Now().UTC()
	store.Add(auth.TokenInfo{
		Hash:        auth.HashToken(testBearerToken),
		DeviceLabel: "test",
		CreatedAt:   now,
		LastSeen:    now,
	})
	return NewRouter(RouterDeps{
		Version:     "dev",
		StartedAt:   time.Now().Add(-time.Second),
		Catalog:     c,
		Finder:      finder,
		Launcher:    l,
		Alive:       alive,
		Fingerprint: testFingerprint,
		TokenStore:  store,
		PINProvider: &fakePINProvider{pin: "000000", consumeOK: true},
		TokenIssuer: &fakeTokenIssuer{token: "unused-token"},
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
