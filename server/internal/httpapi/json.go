// Package httpapi implements the REST layer of remotelauncher.
//
// It exposes a small set of stdlib-only HTTP handlers (/api/status,
// /api/apps) wired through an http.ServeMux with Go 1.22+ method
// routing patterns, plus JSON response helpers used by every handler.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// errorBody is the canonical JSON shape returned for every non-2xx
// response produced by this package.
type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeJSON writes payload as the response body with the given status
// code, setting Content-Type and Cache-Control headers consistently.
// Encoding failures are logged via slog; the response headers have
// already been committed so no error is returned to the caller.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write json response", "err", err)
	}
}

// writeError serialises an errorBody with the given code and message.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: errorPayload{Code: code, Message: message}})
}
