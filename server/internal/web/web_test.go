package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/visibility"
)

const desktopTmpl = `[Desktop Entry]
Type=Application
Name=%s
Exec=/bin/%s
Comment=%s
Icon=%s
Categories=Utility;
`

func seedCatalog(t *testing.T, files map[string]string) *catalog.Catalog {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	c := catalog.New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("catalog load: %v", err)
	}
	return c
}

func desktop(name, exec, comment, icon string) string {
	return fmt.Sprintf(desktopTmpl, name, exec, comment, icon)
}

func newStoreIn(t *testing.T) *visibility.Store {
	t.Helper()
	s := visibility.NewStore()
	if err := s.Load(filepath.Join(t.TempDir(), "visibility.json")); err != nil {
		t.Fatalf("load visibility: %v", err)
	}
	return s
}

func TestAppsHandler_ReturnsEveryAppWithHiddenFlag(t *testing.T) {
	cat := seedCatalog(t, map[string]string{
		"alpha.desktop": desktop("Alpha", "alpha", "first", "alpha-icon"),
		"beta.desktop":  desktop("Beta", "beta", "", ""),
	})
	store := newStoreIn(t)
	if err := store.SetHidden([]string{"beta"}); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}

	h := NewAppsHandler(cat, store)
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var list []AppDTO
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, want := len(list), 2; got != want {
		t.Fatalf("len = %d, want %d", got, want)
	}
	byID := map[string]AppDTO{}
	for _, a := range list {
		byID[a.ID] = a
	}
	if !byID["alpha"].HasIcon {
		t.Error("alpha.HasIcon = false, want true")
	}
	if byID["alpha"].Hidden {
		t.Error("alpha.Hidden = true, want false")
	}
	if byID["beta"].HasIcon {
		t.Error("beta.HasIcon = true, want false (empty Icon field)")
	}
	if !byID["beta"].Hidden {
		t.Error("beta.Hidden = false, want true")
	}
}

func TestAppsHandler_NilStoreNoCrash(t *testing.T) {
	cat := seedCatalog(t, map[string]string{
		"alpha.desktop": desktop("Alpha", "alpha", "", ""),
	})
	h := NewAppsHandler(cat, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestVisibilityHandler_UpdatesStore(t *testing.T) {
	store := newStoreIn(t)
	h := NewVisibilityHandler(store)

	body := bytes.NewBufferString(`{"hidden":["firefox","chromium"]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/visibility", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	got := store.Hidden()
	want := []string{"chromium", "firefox"}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Hidden = %v, want %v", got, want)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["count"].(float64) != 2 {
		t.Errorf("count = %v, want 2", resp["count"])
	}
}

func TestVisibilityHandler_OverwritesPrevious(t *testing.T) {
	store := newStoreIn(t)
	if err := store.SetHidden([]string{"old1", "old2"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := NewVisibilityHandler(store)

	body := bytes.NewBufferString(`{"hidden":["new"]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/visibility", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if store.IsHidden("old1") || store.IsHidden("old2") {
		t.Error("old entries survived overwrite")
	}
	if !store.IsHidden("new") {
		t.Error("new entry not hidden")
	}
}

func TestVisibilityHandler_BadJSON(t *testing.T) {
	store := newStoreIn(t)
	h := NewVisibilityHandler(store)

	body := bytes.NewBufferString(`{bad}`)
	req := httptest.NewRequest(http.MethodPut, "/api/visibility", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestVisibilityHandler_UnknownField(t *testing.T) {
	store := newStoreIn(t)
	h := NewVisibilityHandler(store)

	body := bytes.NewBufferString(`{"hidden":["a"],"extra":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/visibility", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 on unknown field", w.Code)
	}
}

func TestVisibilityHandler_NilStoreErrors(t *testing.T) {
	h := NewVisibilityHandler(nil)

	body := bytes.NewBufferString(`{"hidden":[]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/visibility", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestNewHandler_ServesIndex(t *testing.T) {
	cat := seedCatalog(t, map[string]string{
		"alpha.desktop": desktop("Alpha", "alpha", "", ""),
	})
	store := newStoreIn(t)
	h := NewHandler(Deps{Catalog: cat, Visibility: store})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Error("Content-Type not set")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("RemoteLauncher")) {
		t.Error("response body missing app title")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("adminApp")) {
		t.Error("response body missing alpine root component")
	}
}

func TestNewHandler_AppsEndpointWired(t *testing.T) {
	cat := seedCatalog(t, map[string]string{
		"alpha.desktop": desktop("Alpha", "alpha", "", ""),
	})
	store := newStoreIn(t)
	h := NewHandler(Deps{Catalog: cat, Visibility: store})

	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestNewHandler_VisibilityRoundTrip(t *testing.T) {
	cat := seedCatalog(t, map[string]string{
		"alpha.desktop": desktop("Alpha", "alpha", "", ""),
		"beta.desktop":  desktop("Beta", "beta", "", ""),
	})
	store := newStoreIn(t)
	h := NewHandler(Deps{Catalog: cat, Visibility: store})

	// PUT hides beta.
	put := httptest.NewRequest(http.MethodPut, "/api/visibility", bytes.NewBufferString(`{"hidden":["beta"]}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, put)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", w.Code)
	}

	// GET sees it.
	get := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, get)
	var list []AppDTO
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	byID := map[string]bool{}
	for _, a := range list {
		byID[a.ID] = a.Hidden
	}
	if byID["alpha"] {
		t.Error("alpha should not be hidden")
	}
	if !byID["beta"] {
		t.Error("beta should be hidden after PUT")
	}
}
