package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// contextKey is a private type used for context keys produced by this
// package. Using a dedicated type avoids collisions with keys defined
// in any other package that happens to use the same string value.
type contextKey string

const tokenContextKey contextKey = "auth.token"

const bearerPrefix = "Bearer "

// RequireToken wraps next and enforces Bearer authentication against
// store. On success the matched TokenInfo is attached to the request
// context so downstream handlers can recover it via TokenFromContext.
// Every failure path writes a canonical JSON error body matching the
// httpapi package's shape; see writeJSONError below.
func RequireToken(store *Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}
		if !strings.HasPrefix(header, bearerPrefix) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid authorization header")
			return
		}
		plaintext := strings.TrimPrefix(header, bearerPrefix)
		if plaintext == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid authorization header")
			return
		}
		info, ok := store.Validate(plaintext)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), tokenContextKey, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TokenFromContext returns the TokenInfo attached to ctx by
// RequireToken, if any. The second return value reports whether a
// token was actually present: callers that want to behave differently
// for unauthenticated paths can use it as a guard.
func TokenFromContext(ctx context.Context) (TokenInfo, bool) {
	info, ok := ctx.Value(tokenContextKey).(TokenInfo)
	return info, ok
}

// writeJSONError writes a JSON error body directly without importing
// the httpapi package. The fmt.Sprintf formatting is safe because
// code and message are always hard-coded literals in this file — no
// user-controlled bytes reach the format string.
func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"code":%q,"message":%q}}`, code, message)
}
