package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON_OK(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, struct {
		Name string `json:"name"`
	}{Name: "x"})

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if got, want := w.Body.String(), "{\"name\":\"x\"}\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestWriteJSON_NilPayload(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, nil)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if got, want := w.Body.String(), "null\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestWriteError_Format(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusNotFound, "not_found", "msg")

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", got)
	}
	if got, want := w.Body.String(), "{\"error\":{\"code\":\"not_found\",\"message\":\"msg\"}}\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}
