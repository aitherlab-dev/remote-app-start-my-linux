package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newOKHandler(seen *TokenInfo) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if info, ok := TokenFromContext(r.Context()); ok && seen != nil {
			*seen = info
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireToken_MissingHeader(t *testing.T) {
	store := NewStore()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)

	RequireToken(store, newOKHandler(nil)).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type=%q, want application/json", ct)
	}
	if !strings.Contains(rr.Body.String(), "missing") {
		t.Fatalf("body=%q does not contain 'missing'", rr.Body.String())
	}
}

func TestRequireToken_InvalidScheme(t *testing.T) {
	store := NewStore()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	RequireToken(store, newOKHandler(nil)).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid authorization header") {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestRequireToken_BearerWithoutToken(t *testing.T) {
	store := NewStore()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	req.Header.Set("Authorization", "Bearer ")

	RequireToken(store, newOKHandler(nil)).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid authorization header") {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestRequireToken_UnknownToken(t *testing.T) {
	store := NewStore()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")

	RequireToken(store, newOKHandler(nil)).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid token") {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestRequireToken_ValidToken(t *testing.T) {
	store := NewStore()
	plaintext, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	store.Add(info)

	var seen TokenInfo
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	req.Header.Set("Authorization", "Bearer "+plaintext)

	RequireToken(store, newOKHandler(&seen)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
	if seen.Hash != info.Hash {
		t.Fatalf("TokenFromContext hash=%q, want %q", seen.Hash, info.Hash)
	}
	if seen.DeviceLabel != "phone" {
		t.Fatalf("TokenFromContext label=%q, want %q", seen.DeviceLabel, "phone")
	}
}

func TestTokenFromContext_NoValue(t *testing.T) {
	if _, ok := TokenFromContext(context.Background()); ok {
		t.Fatal("TokenFromContext returned ok=true for empty ctx")
	}
}
