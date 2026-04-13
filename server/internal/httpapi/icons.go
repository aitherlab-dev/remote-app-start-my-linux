package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/icons"
)

const (
	defaultIconSize = 64
	minIconSize     = 16
	maxIconSize     = 512

	iconCacheControl = "public, max-age=3600"
)

// NewIconsHandler returns an http.HandlerFunc that serves the icon
// bytes of a catalog application.
//
// Path: GET /api/apps/{id}/icon?size=N
//
// The id is read via r.PathValue and looked up in the catalog to get
// the raw Icon= field from the desktop entry; that value is then
// resolved to an absolute file by icons.Finder. size defaults to 64
// and is clamped to [16, 512]; a non-integer size yields a 400. A
// missing app, an app with an empty Icon field, or a finder miss all
// produce distinct 404 JSON bodies so the client can tell the cases
// apart. Internal errors (open/stat failures) are logged and returned
// as 500 with a generic message to avoid leaking filesystem details.
func NewIconsHandler(c *catalog.Catalog, f *icons.Finder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		size := defaultIconSize
		if raw := r.URL.Query().Get("size"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", "size must be integer")
				return
			}
			size = clampInt(n, minIconSize, maxIconSize)
		}

		entry, ok := c.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "app not found")
			return
		}
		if entry.Icon == "" {
			writeError(w, http.StatusNotFound, "not_found", "app has no icon")
			return
		}

		path, format, err := f.Find(entry.Icon, size)
		if err != nil {
			if errors.Is(err, icons.ErrIconNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "icon file not found")
				return
			}
			slog.Error("icon finder error", "id", id, "icon", entry.Icon, "err", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to locate icon")
			return
		}

		file, err := os.Open(path)
		if err != nil {
			slog.Error("icon open failed", "id", id, "path", path, "err", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to open icon file")
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			slog.Error("icon stat failed", "id", id, "path", path, "err", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to stat icon file")
			return
		}

		// Cache-Control and Content-Type must be set before ServeContent
		// commits the response headers via WriteHeader.
		w.Header().Set("Content-Type", contentTypeForFormat(format))
		w.Header().Set("Cache-Control", iconCacheControl)

		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func contentTypeForFormat(format string) string {
	switch format {
	case icons.FormatPNG:
		return "image/png"
	case icons.FormatSVG:
		return "image/svg+xml"
	case icons.FormatXPM:
		return "image/x-xpixmap"
	default:
		return "application/octet-stream"
	}
}
