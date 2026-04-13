package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/sasha/remotelauncher/internal/auth"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimitMiddleware_AllowsRequest(t *testing.T) {
	limiter := auth.NewRateLimiter(5, 20, 10*time.Minute)
	mw := NewRateLimitMiddleware(limiter)
	h := mw(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/api/pair", nil)
	req.RemoteAddr = "1.2.3.4:54321"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want 200", w.Code)
	}
	if w.Header().Get("Retry-After") != "" {
		t.Errorf("Retry-After set on success: %q", w.Header().Get("Retry-After"))
	}
}

func TestRateLimitMiddleware_BlocksWith429(t *testing.T) {
	limiter := auth.NewRateLimiter(2, 20, 10*time.Minute)
	mw := NewRateLimitMiddleware(limiter)
	h := mw(okHandler())

	// Drain the per-IP budget.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/pair", nil)
		req.RemoteAddr = "9.9.9.9:10000"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("drain %d: status=%d, want 200", i+1, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/pair", nil)
	req.RemoteAddr = "9.9.9.9:10000"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d, want 429", w.Code)
	}
	retry := w.Header().Get("Retry-After")
	if retry == "" {
		t.Fatal("Retry-After header missing")
	}
	seconds, err := strconv.Atoi(retry)
	if err != nil {
		t.Fatalf("Retry-After=%q: %v", retry, err)
	}
	if seconds < 1 {
		t.Errorf("Retry-After=%d, want >= 1", seconds)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != "rate_limited" {
		t.Errorf("error.code=%q, want rate_limited", body.Error.Code)
	}
}

func TestRateLimitMiddleware_UsesXForwardedFor(t *testing.T) {
	limiter := auth.NewRateLimiter(1, 20, 10*time.Minute)
	mw := NewRateLimitMiddleware(limiter)
	h := mw(okHandler())

	// Same RemoteAddr for both requests, but the X-Forwarded-For
	// value differs — the middleware must key on the XFF header.
	req1 := httptest.NewRequest(http.MethodPost, "/api/pair", nil)
	req1.RemoteAddr = "127.0.0.1:11111"
	req1.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("req1 status=%d, want 200", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/pair", nil)
	req2.RemoteAddr = "127.0.0.1:22222"
	req2.Header.Set("X-Forwarded-For", "198.51.100.7")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("req2 status=%d, want 200 (different XFF should have its own bucket)", w2.Code)
	}

	// Second hit from the first XFF IP must be blocked.
	req3 := httptest.NewRequest(http.MethodPost, "/api/pair", nil)
	req3.RemoteAddr = "127.0.0.1:33333"
	req3.Header.Set("X-Forwarded-For", "203.0.113.10")
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("req3 status=%d, want 429 (XFF 203.0.113.10 already spent)", w3.Code)
	}
}
