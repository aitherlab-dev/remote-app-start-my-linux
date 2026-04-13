package httpapi

import (
	"net/http"

	"github.com/sasha/remotelauncher/internal/catalog"
)

// NewAppsHandler returns an http.HandlerFunc that serves the current
// application list as a JSON array of catalog.AppInfo. The handler
// relies on catalog.Catalog.List to return execution-safe DTOs (no
// Exec / TryExec / Path leakage).
func NewAppsHandler(c *catalog.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		list := c.List()
		if list == nil {
			list = []catalog.AppInfo{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}
