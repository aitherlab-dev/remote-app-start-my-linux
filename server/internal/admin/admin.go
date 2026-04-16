// Package admin mounts the admin SPA on the public TLS port (8443)
// so it can be accessed from the Android WebView. The loopback admin
// server on port 17843 keeps running as-is — this package provides
// the same UI and API endpoints under /admin/ with token-or-cookie
// authentication.
package admin

import (
	"bytes"
	"io/fs"
	"net/http"

	"github.com/sasha/remotelauncher/internal/auth"
	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/httpapi"
	"github.com/sasha/remotelauncher/internal/icons"
	"github.com/sasha/remotelauncher/internal/shortcuts"
	"github.com/sasha/remotelauncher/internal/visibility"
	"github.com/sasha/remotelauncher/internal/web"
)

// Deps collects the collaborators needed to mount the admin UI on
// the public TLS port. The same Store instances are shared with
// the loopback admin server so changes made from the phone are
// immediately visible in both places.
type Deps struct {
	TokenStore      *auth.Store
	Catalog         *catalog.Catalog
	Finder          *icons.Finder
	Visibility      *visibility.Store
	Shortcuts       *shortcuts.Store
	DefaultTerminal string
}

const cookieName = "rl_session"
const cookiePath = "/"

// NewHandler returns an http.Handler that serves the admin SPA and
// API at the /admin/ prefix. Every request is guarded by
// token-or-cookie auth: the Android WebView sends a Bearer header
// on the initial page load, and the middleware sets a session cookie
// so subsequent fetch() calls are authenticated automatically.
func NewHandler(d Deps) http.Handler {
	mux := http.NewServeMux()

	wrap := func(h http.Handler) http.Handler {
		return auth.RequireTokenOrCookie(d.TokenStore, cookieName, cookiePath, h)
	}

	mux.Handle("GET /admin/", wrap(newSPAHandler()))
	mux.Handle("GET /admin/api/apps", wrap(web.NewAppsHandler(d.Catalog, d.Visibility, d.Shortcuts)))
	mux.Handle("GET /admin/api/apps/{id}/icon", wrap(httpapi.NewIconsHandler(d.Catalog, d.Finder)))
	mux.Handle("GET /admin/api/shortcuts", wrap(web.NewShortcutsHandler(d.Shortcuts, d.DefaultTerminal)))
	mux.Handle("PUT /admin/api/shortcuts", wrap(web.NewUpdateShortcutsHandler(d.Shortcuts)))
	mux.Handle("PUT /admin/api/visibility", wrap(web.NewVisibilityHandler(d.Visibility)))

	return mux
}

// newSPAHandler returns a handler that serves the embedded admin
// index.html with a <base href="/admin/"> tag injected so relative
// fetch() calls resolve against the correct prefix.
func newSPAHandler() http.HandlerFunc {
	raw, err := fs.ReadFile(web.StaticFS(), "static/index.html")
	if err != nil {
		panic("admin: read embedded admin index.html: " + err.Error())
	}

	patched := bytes.Replace(
		raw,
		[]byte(`<meta charset="utf-8" />`),
		[]byte(`<meta charset="utf-8" /><base href="/admin/">`),
		1,
	)

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(patched)
	}
}
