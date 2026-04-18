// Package web serves the local admin UI.
//
// The admin UI is a small single-page application (Tailwind + daisyUI
// + Alpine.js, all loaded from public CDNs) that lets the operator
// toggle which applications are visible to the Android client. It is
// served on a separate, plain-HTTP listener bound to a loopback
// address so a browser on the same machine can open it without TLS
// warnings. Authorisation is intentionally absent — the loopback
// binding is the security boundary; config validation refuses any
// non-loopback listen address.
//
// The package owns:
//   - an embedded static/ directory with index.html and friends;
//   - HTTP handlers for GET /api/apps, PUT /api/visibility;
//   - a thin wrapper around the existing icon handler from httpapi so
//     the UI can display app icons without duplicating the resolver.
package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"time"

	"github.com/sasha/remotelauncher/internal/auth"
	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/httpapi"
	"github.com/sasha/remotelauncher/internal/icons"
	"github.com/sasha/remotelauncher/internal/shortcuts"
	"github.com/sasha/remotelauncher/internal/visibility"
)

//go:embed static
var staticFS embed.FS

// StaticFS returns the embedded static filesystem so other packages
// (e.g. httpapi) can serve the same SPA under a different prefix
// without duplicating the embed directive.
func StaticFS() embed.FS { return staticFS }

// Deps collects the collaborators the admin UI needs. None of them
// are owned by the web package — main.go wires the same catalog /
// finder / store instances into both the HTTPS API and the admin UI
// so changes saved from the UI are immediately visible on the next
// /api/apps poll from the phone.
type Deps struct {
	Catalog         *catalog.Catalog
	Finder          *icons.Finder
	Visibility      *visibility.Store
	Shortcuts       *shortcuts.Store
	DefaultTerminal string
	PINSession      *auth.PINSession // optional, nil = no PIN endpoints
	PINTTL          time.Duration
}

// NewHandler builds the top-level http.Handler for the admin UI. It
// serves the embedded single-page app on GET /, a REST endpoint on
// GET /api/apps that returns every application (with its hidden
// flag), a PUT /api/visibility endpoint that rewrites the hidden set,
// and delegates icon bytes to the existing httpapi icon handler.
func NewHandler(d Deps) http.Handler {
	mux := http.NewServeMux()

	// Static SPA. The embed.FS root is "static", so we Sub into it and
	// hand the trimmed filesystem to http.FileServer. A bare GET "/"
	// is served as "index.html" by http.FileServer automatically.
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		// fs.Sub only fails when "static" is missing from the embed
		// — which is a compile-time invariant of this package.
		panic("web: embed static missing: " + err.Error())
	}
	mux.Handle("GET /", http.FileServer(http.FS(sub)))

	mux.Handle("GET /api/apps", NewAppsHandler(d.Catalog, d.Visibility, d.Shortcuts))
	mux.Handle("GET /api/apps/{id}/icon", httpapi.NewIconsHandler(d.Catalog, d.Finder))
	mux.Handle("PUT /api/visibility", NewVisibilityHandler(d.Visibility))
	mux.Handle("GET /api/shortcuts", NewShortcutsHandler(d.Shortcuts, d.DefaultTerminal))
	mux.Handle("PUT /api/shortcuts", NewUpdateShortcutsHandler(d.Shortcuts))

	if d.PINSession != nil {
		mux.Handle("GET /api/pin", newPINGetHandler(d.PINSession))
		mux.Handle("POST /api/pin", newPINPostHandler(d.PINSession, d.PINTTL))
	}

	return mux
}

type pinResponse struct {
	PIN       string    `json:"pin"`
	ExpiresAt time.Time `json:"expires_at"`
	Consumed  bool      `json:"consumed"`
	Expired   bool      `json:"expired"`
}

func pinJSON(ps *auth.PINSession) pinResponse {
	pin, expiresAt, consumed := ps.Status()
	expired := !expiresAt.IsZero() && time.Now().After(expiresAt)
	return pinResponse{
		PIN:       pin,
		ExpiresAt: expiresAt,
		Consumed:  consumed,
		Expired:   expired,
	}
}

func newPINGetHandler(ps *auth.PINSession) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pinJSON(ps))
	})
}

func newPINPostHandler(ps *auth.PINSession, ttl time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := ps.Regenerate(ttl); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pinJSON(ps))
	})
}
