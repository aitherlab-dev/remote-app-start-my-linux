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
	"io/fs"
	"net/http"

	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/httpapi"
	"github.com/sasha/remotelauncher/internal/icons"
	"github.com/sasha/remotelauncher/internal/shortcuts"
	"github.com/sasha/remotelauncher/internal/visibility"
)

//go:embed static
var staticFS embed.FS

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

	return mux
}
