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

// RequireTokenOrCookie wraps next and enforces authentication via
// either a Bearer header (checked first) or a named cookie (fallback).
// When the Bearer header is present and valid the middleware also sets
// the cookie in the response so that subsequent same-origin requests
// from a browser (e.g. fetch() calls from an SPA loaded inside a
// WebView) are authenticated automatically.
//
// The cookie is scoped to cookiePath, marked Secure, HttpOnly, and
// SameSite=Strict. This makes it safe to use on the TLS-only public
// port where the admin UI is served.
func RequireTokenOrCookie(store *Store, cookieName, cookiePath string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var plaintext string

		// 1. Try Bearer header.
		if header := r.Header.Get("Authorization"); strings.HasPrefix(header, bearerPrefix) {
			plaintext = strings.TrimPrefix(header, bearerPrefix)
		}

		// 2. Fallback: cookie.
		if plaintext == "" {
			if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
				plaintext = c.Value
			}
		}

		if plaintext == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token or session cookie")
			return
		}

		info, ok := store.Validate(plaintext)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}

		// Set / refresh the cookie so browser-initiated requests work.
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    plaintext,
			Path:     cookiePath,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		ctx := context.WithValue(r.Context(), tokenContextKey, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
