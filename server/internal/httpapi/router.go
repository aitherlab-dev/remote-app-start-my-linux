package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/sasha/remotelauncher/internal/auth"
	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/icons"
)

// RouterDeps bundles the collaborators NewRouter needs to wire up the
// REST API. Grouping them in a struct keeps the call-site readable as
// the dependency list grows (each new handler adds one more field).
type RouterDeps struct {
	Version     string
	StartedAt   time.Time
	Catalog     *catalog.Catalog
	Finder      *icons.Finder
	Launcher    AppLauncher
	Alive       AliveChecker
	Fingerprint string
	TokenStore  *auth.Store
	PINProvider PINProvider
	TokenIssuer TokenIssuer
	RateLimiter *auth.RateLimiter
}

// NewRouter builds the top-level http.Handler for the REST API.
//
// Routing uses the Go 1.22+ method-aware patterns of http.ServeMux:
// mismatched methods on a registered path are returned as 405 with an
// Allow header, unknown paths fall through to a JSON 404 via the
// wrapNotFoundJSON middleware.
//
// Auth model: /api/status and /api/pair are unauthenticated — the
// former so clients can probe the server and fingerprint the cert, the
// latter because pairing itself is the authentication. Every other
// endpoint is wrapped in auth.RequireToken and demands a valid Bearer
// token minted by a successful /api/pair.
func NewRouter(d RouterDeps) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/status", NewStatusHandler(d.Version, d.StartedAt, d.Catalog, d.Fingerprint))
	pairHandler := NewPairHandler(d.PINProvider, d.TokenIssuer)
	if d.RateLimiter != nil {
		rateLimitMw := NewRateLimitMiddleware(d.RateLimiter)
		mux.Handle("POST /api/pair", rateLimitMw(pairHandler))
	} else {
		mux.Handle("POST /api/pair", pairHandler)
	}
	mux.Handle("GET /api/apps", auth.RequireToken(d.TokenStore, NewAppsHandler(d.Catalog, d.Alive)))
	mux.Handle("GET /api/apps/{id}/icon", auth.RequireToken(d.TokenStore, NewIconsHandler(d.Catalog, d.Finder)))
	mux.Handle("POST /api/apps/{id}/launch", auth.RequireToken(d.TokenStore, NewLaunchHandler(d.Catalog, d.Launcher)))
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
		// A downstream handler that already committed a JSON
		// Content-Type is producing an intentional 404 (e.g. /icon
		// reporting "app not found"). Let it through untouched.
		// ServeMux's own NotFoundHandler uses text/plain, which is
		// how we distinguish the two.
		if ct := n.ResponseWriter.Header().Get("Content-Type"); strings.HasPrefix(ct, "application/json") {
			n.ResponseWriter.WriteHeader(code)
			return
		}
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
