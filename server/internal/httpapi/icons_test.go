package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/icons"
)

// newTestFinder builds an icons.Finder rooted at a fresh temp dir and
// seeds it with files keyed by their path relative to the hicolor theme
// directory (e.g. "48x48/apps/firefox.png"). An empty files map yields
// a finder that never hits anything.
func newTestFinder(t *testing.T, files map[string][]byte) *icons.Finder {
	t.Helper()
	base := t.TempDir()
	themeDir := filepath.Join(base, "hicolor")
	for name, content := range files {
		full := filepath.Join(themeDir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatalf("write %q: %v", full, err)
		}
	}
	return icons.New([]string{base}, "hicolor")
}

// newTestRouterWith wires an arbitrary catalog and finder through the
// real router so tests exercise the handler exactly the way a client
// would hit it, including r.PathValue resolution.
func newTestRouterWith(t *testing.T, cat *catalog.Catalog, finder *icons.Finder) http.Handler {
	t.Helper()
	return NewRouter("dev", time.Now().Add(-time.Second), cat, finder)
}

func TestIconsHandler_FoundPNG(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "firefox", "Network"),
	})
	pngBody := []byte("\x89PNG-test")
	finder := newTestFinder(t, map[string][]byte{
		"48x48/apps/firefox.png": pngBody,
	})
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/firefox/icon?size=48", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=3600" {
		t.Errorf("Cache-Control = %q, want public, max-age=3600", cc)
	}
	if got := w.Body.Bytes(); string(got) != string(pngBody) {
		t.Errorf("body = %q, want %q", got, pngBody)
	}
}

func TestIconsHandler_FoundSVG(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "firefox", "Network"),
	})
	svg := []byte("<svg></svg>")
	finder := newTestFinder(t, map[string][]byte{
		"scalable/apps/firefox.svg": svg,
	})
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/firefox/icon?size=128", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=3600" {
		t.Errorf("Cache-Control = %q, want public, max-age=3600", cc)
	}
	if got := w.Body.Bytes(); string(got) != string(svg) {
		t.Errorf("body = %q, want %q", got, svg)
	}
}

func TestIconsHandler_DefaultSize(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "firefox", ""),
	})
	// Exact match for the default size (64) must win.
	finder := newTestFinder(t, map[string][]byte{
		"64x64/apps/firefox.png": []byte("64-bytes"),
	})
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/firefox/icon", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != "64-bytes" {
		t.Errorf("body = %q, want 64-bytes (default size must be 64)", got)
	}
}

func TestIconsHandler_SizeClampedHigh(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "firefox", ""),
	})
	// Only a 512 raster exists — if clamp produces anything ≠ 512 the
	// nearest-above/nearest-below waterfall still picks it, but exact
	// match is only hit when the caller asked for 512. We feed only the
	// 512 exact bucket, so a clamp to 512 yields an exact-size hit.
	finder := newTestFinder(t, map[string][]byte{
		"512x512/apps/firefox.png": []byte("512"),
	})
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/firefox/icon?size=9999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != "512" {
		t.Errorf("body = %q, want 512 (size must clamp to %d)", got, maxIconSize)
	}
}

func TestIconsHandler_SizeClampedLow(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "firefox", ""),
	})
	finder := newTestFinder(t, map[string][]byte{
		"16x16/apps/firefox.png": []byte("16"),
	})
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/firefox/icon?size=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != "16" {
		t.Errorf("body = %q, want 16 (size must clamp to %d)", got, minIconSize)
	}
}

func TestIconsHandler_BadSize(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "firefox", ""),
	})
	finder := newTestFinder(t, nil)
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/firefox/icon?size=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "bad_request" {
		t.Errorf("error.code = %q, want bad_request", body.Error.Code)
	}
}

func TestIconsHandler_AppNotFound(t *testing.T) {
	cat := newTestCatalog(t, nil)
	finder := newTestFinder(t, nil)
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/nonexistent/icon", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body=%s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("error.code = %q, want not_found", body.Error.Code)
	}
	if body.Error.Message != "app not found" {
		t.Errorf("error.message = %q, want app not found", body.Error.Message)
	}
}

func TestIconsHandler_AppHasNoIcon(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"noicon.desktop": desktopEntry("NoIcon", "noicon", "", "", ""),
	})
	finder := newTestFinder(t, nil)
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/noicon/icon", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body=%s)", w.Code, w.Body.String())
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("error.code = %q, want not_found", body.Error.Code)
	}
	if !strings.Contains(body.Error.Message, "no icon") {
		t.Errorf("error.message = %q, want to contain 'no icon'", body.Error.Message)
	}
}

func TestIconsHandler_IconFileNotFound(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "firefox", ""),
	})
	// Empty finder — no candidates, no pixmap fallback.
	finder := newTestFinder(t, nil)
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/firefox/icon", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body=%s)", w.Code, w.Body.String())
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("error.code = %q, want not_found", body.Error.Code)
	}
	if !strings.Contains(body.Error.Message, "icon file not found") {
		t.Errorf("error.message = %q, want 'icon file not found'", body.Error.Message)
	}
}

// TestRouter_IconsEndpoint asserts that /api/apps/{id}/icon is actually
// handled by NewIconsHandler and not by the wrapNotFoundJSON fallback.
// It feeds an empty catalog, so the handler must answer with its own
// "app not found" message — distinct from the router's "route not found".
func TestRouter_IconsEndpoint(t *testing.T) {
	cat := newTestCatalog(t, nil)
	finder := newTestFinder(t, nil)
	r := newTestRouterWith(t, cat, finder)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/whatever/icon", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body=%s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Message != "app not found" {
		t.Errorf("error.message = %q, want 'app not found' (handler, not router)", body.Error.Message)
	}
}
