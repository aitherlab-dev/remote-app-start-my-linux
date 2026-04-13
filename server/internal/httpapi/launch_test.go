package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sasha/remotelauncher/internal/launcher"
)

func TestLaunchHandler_OK(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "firefox", "Network"),
	})
	fl := &fakeLauncher{pid: 12345}
	r := newRouterFor(t, cat, nil, fl, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/apps/firefox/launch", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	var got launchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != "launched" {
		t.Errorf("status = %q, want launched", got.Status)
	}
	if got.PID != 12345 {
		t.Errorf("pid = %d, want 12345", got.PID)
	}
}

func TestLaunchHandler_AppNotFound(t *testing.T) {
	cat := newTestCatalog(t, nil)
	r := newRouterFor(t, cat, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/apps/nonexistent/launch", nil)
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
	if body.Error.Message != "app not found" {
		t.Errorf("error.message = %q, want 'app not found'", body.Error.Message)
	}
}

func TestLaunchHandler_AppHasNoExec(t *testing.T) {
	// A bare entry with Type=Application but no Exec= key. The catalog
	// scanner accepts it (Exec is not strictly required to parse), and
	// the launch handler must reject it with 400.
	cat := newTestCatalog(t, map[string]string{
		"noexec.desktop": "[Desktop Entry]\nType=Application\nName=NoExec\n",
	})
	fl := &fakeLauncher{}
	r := newRouterFor(t, cat, nil, fl, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/apps/noexec/launch", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", w.Code, w.Body.String())
	}
	if fl.calls != 0 {
		t.Errorf("fakeLauncher.calls = %d, want 0 (handler must short-circuit before calling launcher)", fl.calls)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "bad_request" {
		t.Errorf("error.code = %q, want bad_request", body.Error.Code)
	}
	if !strings.Contains(body.Error.Message, "exec") {
		t.Errorf("error.message = %q, want to mention exec", body.Error.Message)
	}
}

func TestLaunchHandler_LauncherEmptyExec(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "", ""),
	})
	fl := &fakeLauncher{err: launcher.ErrEmptyExec}
	r := newRouterFor(t, cat, nil, fl, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/apps/firefox/launch", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", w.Code, w.Body.String())
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "bad_request" {
		t.Errorf("error.code = %q, want bad_request", body.Error.Code)
	}
}

func TestLaunchHandler_LauncherNoTerminal(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"vim.desktop": desktopEntry("Vim", "vim", "", "", ""),
	})
	fl := &fakeLauncher{err: launcher.ErrNoTerminalEmulator}
	r := newRouterFor(t, cat, nil, fl, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/apps/vim/launch", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", w.Code, w.Body.String())
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "bad_request" {
		t.Errorf("error.code = %q, want bad_request", body.Error.Code)
	}
	if !strings.Contains(body.Error.Message, "terminal") {
		t.Errorf("error.message = %q, want to mention terminal", body.Error.Message)
	}
}

func TestLaunchHandler_LauncherGenericError(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "", ""),
	})
	fl := &fakeLauncher{err: errors.New("boom: secret path /home/sasha leaked")}
	r := newRouterFor(t, cat, nil, fl, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/apps/firefox/launch", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body=%s)", w.Code, w.Body.String())
	}
	bodyStr := w.Body.String()
	for _, leaked := range []string{"boom", "secret", "/home/sasha"} {
		if strings.Contains(bodyStr, leaked) {
			t.Errorf("response body leaks %q from launcher error: %s", leaked, bodyStr)
		}
	}
	var body errorBody
	if err := json.NewDecoder(strings.NewReader(bodyStr)).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "internal_error" {
		t.Errorf("error.code = %q, want internal_error", body.Error.Code)
	}
}

func TestLaunchHandler_PassesEntryToLauncher(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "", ""),
	})
	fl := &fakeLauncher{pid: 42}
	r := newRouterFor(t, cat, nil, fl, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/apps/firefox/launch", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if fl.calls != 1 {
		t.Errorf("fakeLauncher.calls = %d, want 1", fl.calls)
	}
	if fl.called.ID != "firefox" {
		t.Errorf("fakeLauncher.called.ID = %q, want firefox", fl.called.ID)
	}
	if fl.called.Exec == "" {
		t.Errorf("fakeLauncher.called.Exec is empty — handler must forward the full desktop.Entry")
	}
}

// TestLaunchHandler_MethodNotAllowed verifies that ServeMux's Go 1.22+
// method-aware routing rejects GET on a POST-only path with 405 + Allow.
func TestLaunchHandler_MethodNotAllowed(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"firefox.desktop": desktopEntry("Firefox", "firefox", "", "", ""),
	})
	r := newRouterFor(t, cat, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/firefox/launch", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 (body=%s)", w.Code, w.Body.String())
	}
	if allow := w.Header().Get("Allow"); allow != "POST" {
		t.Errorf("Allow = %q, want POST", allow)
	}
}
