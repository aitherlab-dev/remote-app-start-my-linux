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

	h := NewAppsHandler(cat, nil, nil)
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

	h := NewAppsHandler(cat, nil, nil)
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

func TestAppsHandler_FiltersHiddenApps(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"alpha.desktop": desktopEntry("Alpha", "alpha", "", "", ""),
		"beta.desktop":  desktopEntry("Beta", "beta", "", "", ""),
		"gamma.desktop": desktopEntry("Gamma", "gamma", "", "", ""),
	})
	vis := &fakeVisibility{hidden: map[string]bool{"beta": true}}

	h := NewAppsHandler(cat, nil, vis)
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var list []catalog.AppInfo
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, want := len(list), 2; got != want {
		t.Fatalf("len = %d, want %d", got, want)
	}
	for _, a := range list {
		if a.ID == "beta" {
			t.Errorf("hidden app %q leaked into response", a.ID)
		}
	}
}

func TestAppsHandler_FilterKeepsRunningFlag(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"alpha.desktop": desktopEntry("Alpha", "alpha", "", "", ""),
		"beta.desktop":  desktopEntry("Beta", "beta", "", "", ""),
	})
	alive := &fakeAlive{alive: map[string]bool{"alpha": true, "beta": true}}
	vis := &fakeVisibility{hidden: map[string]bool{"beta": true}}

	h := NewAppsHandler(cat, alive, vis)
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var list []catalog.AppInfo
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 || list[0].ID != "alpha" || !list[0].Running {
		t.Errorf("list = %+v, want only alpha running", list)
	}
}

func TestAppsHandler_RunningStatus(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"alpha.desktop": desktopEntry("Alpha", "alpha", "", "", ""),
		"beta.desktop":  desktopEntry("Beta", "beta", "", "", ""),
	})
	alive := &fakeAlive{alive: map[string]bool{"alpha": true}}

	h := NewAppsHandler(cat, alive, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var list []catalog.AppInfo
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	got := map[string]bool{}
	for _, a := range list {
		got[a.ID] = a.Running
	}
	if !got["alpha"] {
		t.Errorf("alpha.Running = false, want true")
	}
	if got["beta"] {
		t.Errorf("beta.Running = true, want false")
	}
}
