package web

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/visibility"
)

// AppDTO is the admin-UI view of a catalog entry. Unlike the
// phone-facing catalog.AppInfo it deliberately exposes the Hidden
// flag (so the UI can render the right toggle state) and a HasIcon
// boolean the front-end uses to decide whether to request the
// /api/apps/{id}/icon endpoint at all — apps without an Icon= field
// in their .desktop file would otherwise produce a 404 on every
// card load, spamming the browser console.
type AppDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
	HasIcon bool   `json:"has_icon"`
	Hidden  bool   `json:"hidden"`
}

// NewAppsHandler returns GET /api/apps for the admin UI. Unlike the
// main API handler on :8443, this one returns every application
// regardless of visibility and stamps each entry with its current
// hidden flag so the UI can render the correct toggle state.
func NewAppsHandler(cat *catalog.Catalog, store *visibility.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		list := cat.List()
		out := make([]AppDTO, 0, len(list))
		for _, a := range list {
			dto := AppDTO{
				ID:      a.ID,
				Name:    a.Name,
				Comment: a.Comment,
				HasIcon: a.Icon != "",
			}
			if store != nil {
				dto.Hidden = store.IsHidden(a.ID)
			}
			out = append(out, dto)
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// writeJSON mirrors httpapi.writeJSON but with a no-store
// Cache-Control header — the admin UI reloads on every toggle and
// caching an old list would surface a stale visibility state as soon
// as a second tab opened.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("web: write json response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
