package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	cat := newTestCatalog(t, map[string]string{
		"one.desktop": desktopEntry("One", "one", "", "", ""),
	})
	return newRouterFor(t, cat, nil, nil, nil)
}

func TestRouter_StatusEndpoint(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	var got StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.AppsCount != 1 {
		t.Errorf("AppsCount = %d, want 1", got.AppsCount)
	}
	if got.CertFingerprint != testFingerprint {
		t.Errorf("CertFingerprint = %q, want %q (RouterDeps.Fingerprint not wired)", got.CertFingerprint, testFingerprint)
	}
}

func TestRouter_AppsEndpoint(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRouter_NotFound(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code == "" {
		t.Error("error.code is empty, want non-empty")
	}
	if body.Error.Code != "not_found" {
		t.Errorf("error.code = %q, want not_found", body.Error.Code)
	}
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Go 1.22+ ServeMux returns 405 for mismatched methods on a
	// registered path and advertises the allowed methods in the
	// Allow header. HEAD is auto-registered alongside GET.
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405 (Go 1.22+ ServeMux behaviour)", w.Code)
	}
	if allow := w.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Errorf("Allow = %q, want %q", allow, "GET, HEAD")
	}
}
