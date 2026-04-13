package httpapi

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/sasha/remotelauncher/internal/auth"
)

// NewRateLimitMiddleware returns middleware that consults limiter for
// every incoming request and answers 429 JSON + Retry-After header
// when the budget is exhausted. Only /api/pair is wrapped in the
// router, so this middleware never sees other endpoints.
//
// The client IP is extracted from r.RemoteAddr, then overridden by the
// first entry of X-Forwarded-For when that header is present. The
// honest value of X-Forwarded-For depends on the deployment model
// (direct TCP vs. reverse proxy); the project doc leaves this choice
// to the operator, so we honour the header unconditionally and trust
// the operator to drop or rewrite it if needed.
func NewRateLimitMiddleware(limiter *auth.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			ok, retryAfter := limiter.Allow(ip)
			if !ok {
				seconds := int(math.Ceil(retryAfter.Seconds()))
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP returns the best guess for the caller's address. It starts
// with the host portion of r.RemoteAddr (which SplitHostPort extracts
// for us) and then, if the client presented X-Forwarded-For, takes
// the first comma-separated entry from that header. RemoteAddr is a
// sensible fallback when SplitHostPort fails (e.g. a httptest request
// with no port).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if comma := strings.IndexByte(xff, ','); comma >= 0 {
			xff = xff[:comma]
		}
		xff = strings.TrimSpace(xff)
		if xff != "" {
			host = xff
		}
	}
	return host
}
