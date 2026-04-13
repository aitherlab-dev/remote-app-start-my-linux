package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatusHandler_OK(t *testing.T) {
	cat := newTestCatalog(t, map[string]string{
		"one.desktop": desktopEntry("One", "one", "", "", ""),
	})

	startedAt := time.Now().Add(-3 * time.Second)
	h := NewStatusHandler("v1.2.3", startedAt, cat)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
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

	var got StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Version != "v1.2.3" {
		t.Errorf("Version = %q, want v1.2.3", got.Version)
	}
	if !got.StartedAt.Equal(startedAt) {
		t.Errorf("StartedAt = %v, want %v", got.StartedAt, startedAt)
	}
	if got.UptimeSec < 0 {
		t.Errorf("UptimeSec = %d, want ≥0", got.UptimeSec)
	}
	if got.UptimeSec < 2 {
		t.Errorf("UptimeSec = %d, want ≥2 (startedAt was 3s ago)", got.UptimeSec)
	}
	if got.AppsCount != 1 {
		t.Errorf("AppsCount = %d, want 1", got.AppsCount)
	}
}

func TestStatusHandler_UptimeMonotonic(t *testing.T) {
	cat := newTestCatalog(t, nil)
	startedAt := time.Now().Add(-1 * time.Second)
	h := NewStatusHandler("dev", startedAt, cat)

	decode := func() StatusResponse {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		var s StatusResponse
		if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return s
	}

	first := decode()
	time.Sleep(1100 * time.Millisecond)
	second := decode()

	if second.UptimeSec < first.UptimeSec {
		t.Errorf("uptime went backwards: first=%d second=%d", first.UptimeSec, second.UptimeSec)
	}
	if second.UptimeSec == first.UptimeSec {
		t.Errorf("uptime did not advance between calls: %d", second.UptimeSec)
	}
}
