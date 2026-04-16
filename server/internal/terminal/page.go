package terminal

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFS embed.FS

// NewPageHandler returns a handler that serves the xterm.js terminal
// SPA. It reads the embedded index.html once at init and serves it
// on every request — there are no other static assets (JS/CSS are
// loaded from CDN).
func NewPageHandler() http.HandlerFunc {
	raw, err := fs.ReadFile(staticFS, "static/index.html")
	if err != nil {
		panic("terminal: read embedded index.html: " + err.Error())
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(raw)
	}
}
