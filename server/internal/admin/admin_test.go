package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sasha/remotelauncher/internal/auth"
	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/icons"
	"github.com/sasha/remotelauncher/internal/shortcuts"
	"github.com/sasha/remotelauncher/internal/visibility"
)

func testDeps(t *testing.T) (Deps, string) {
	t.Helper()
	store := auth.NewStore()
	plaintext, info, err := auth.IssueToken("test-device")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	store.Add(info)

	return Deps{
		TokenStore:      store,
		Catalog:         catalog.New(nil),
		Finder:          icons.New(nil, ""),
		Visibility:      visibility.NewStore(),
		Shortcuts:       shortcuts.NewStore(),
		DefaultTerminal: "kitty",
	}, plaintext
}

func TestAdminSPA_Unauthorized(t *testing.T) {
	deps, _ := testDeps(t)
	h := NewHandler(deps)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
}

func TestAdminSPA_WithBearer(t *testing.T) {
	deps, token := testDeps(t)
	h := NewHandler(deps)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `<base href="/admin/">`) {
		t.Fatal("response missing <base href=\"/admin/\">")
	}
	if !strings.Contains(body, "RemoteLauncher") {
		t.Fatal("response missing page title")
	}

	// Must set session cookie.
	cookies := rr.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == cookieName {
			found = true
			if c.Value != token {
				t.Fatalf("cookie value=%q, want token", c.Value)
			}
			if !c.Secure {
				t.Fatal("cookie not Secure")
			}
			if !c.HttpOnly {
				t.Fatal("cookie not HttpOnly")
			}
		}
	}
	if !found {
		t.Fatal("session cookie not set")
	}
}

func TestAdminSPA_WithCookie(t *testing.T) {
	deps, token := testDeps(t)
	h := NewHandler(deps)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
}

func TestAdminAPI_Shortcuts(t *testing.T) {
	deps, token := testDeps(t)
	h := NewHandler(deps)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/shortcuts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type=%q, want application/json", ct)
	}
	if !strings.Contains(rr.Body.String(), "shortcuts") {
		t.Fatal("response missing shortcuts key")
	}
}

func TestAdminAPI_Apps(t *testing.T) {
	deps, token := testDeps(t)
	h := NewHandler(deps)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/apps", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
}
