package httpapi

import (
	"net/http"
	"time"

	"github.com/sasha/remotelauncher/internal/catalog"
)

// NewRouter builds the top-level http.Handler for the REST API.
//
// Routing uses the Go 1.22+ method-aware patterns of http.ServeMux:
// mismatched methods on a registered path are returned as 405 with an
// Allow header, unknown paths fall through to a JSON 404 via the
// wrapNotFoundJSON middleware.
func NewRouter(version string, startedAt time.Time, c *catalog.Catalog) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/status", NewStatusHandler(version, startedAt, c))
	mux.Handle("GET /api/apps", NewAppsHandler(c))
	return wrapNotFoundJSON(mux)
}

// wrapNotFoundJSON rewrites plain-text 404 responses produced by
// http.ServeMux into the package's canonical JSON error body.
//
// The middleware leaves every other response (200, 405, ...) untouched
// and only intercepts the exact moment ServeMux calls WriteHeader(404).
// That keeps the common happy-path allocation-free.
func wrapNotFoundJSON(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		iw := &notFoundInterceptor{ResponseWriter: w, path: r.URL.Path}
		h.ServeHTTP(iw, r)
	})
}

// notFoundInterceptor swaps ServeMux's plaintext "404 page not found"
// for a JSON body. When WriteHeader(404) is observed it resets any
// headers the mux may have set (Content-Type text/plain, X-Content-
// Type-Options) and hands control over to writeError. Subsequent Write
// calls from the inner handler are discarded.
type notFoundInterceptor struct {
	http.ResponseWriter
	path        string
	intercepted bool
}

func (n *notFoundInterceptor) WriteHeader(code int) {
	if code == http.StatusNotFound {
		n.intercepted = true
		h := n.ResponseWriter.Header()
		for k := range h {
			h.Del(k)
		}
		writeError(n.ResponseWriter, http.StatusNotFound, "not_found", "route not found: "+n.path)
		return
	}
	n.ResponseWriter.WriteHeader(code)
}

func (n *notFoundInterceptor) Write(b []byte) (int, error) {
	if n.intercepted {
		return len(b), nil
	}
	return n.ResponseWriter.Write(b)
}
