package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sasha/remotelauncher/internal/catalog"
)

func TestAppsHandler_ReturnsList(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"alpha.desktop": desktopEntry("Alpha", "alpha", "A", "a-icon", "Utility"),
		"beta.desktop":  desktopEntry("Beta", "beta", "B", "b-icon", "Utility"),
	})

	h := NewAppsHandler(cat)
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", got)
	}

	body := w.Body.Bytes()

	var list []catalog.AppInfo
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}

	// Exec-related fields must not be leaked on the HTTP surface.
	lowered := bytes.ToLower(body)
	for _, forbidden := range []string{"exec", "tryexec", "path", "hidden", "onlyshowin", "notshowin", "startupnotify"} {
		if bytes.Contains(lowered, []byte(forbidden)) {
			t.Errorf("JSON body leaks forbidden token %q: %s", forbidden, body)
		}
	}
}

func TestAppsHandler_EmptyCatalog(t *testing.T) {
	cat := newTestCatalog(t, nil)

	h := NewAppsHandler(cat)
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := w.Body.String(); got != "[]\n" {
		t.Errorf("body = %q, want %q", got, "[]\n")
	}
}
