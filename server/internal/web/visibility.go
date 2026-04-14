package web

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sasha/remotelauncher/internal/visibility"
)

// visibilityPayload is the request body for PUT /api/visibility.
// The client sends the FULL desired hidden list on every save — the
// server never performs a partial diff. Keeping the API declarative
// removes every "what if an add and a remove race each other" corner
// case at the cost of a handful of extra bytes per request.
type visibilityPayload struct {
	Hidden []string `json:"hidden"`
}

// NewVisibilityHandler returns PUT /api/visibility. It replaces the
// entire hidden set with whatever the client just posted and
// persists the new state to disk. An unset store (pre-init) or a
// malformed body surfaces as a 400/500 — the UI shows the error in a
// toast and rolls the toggle back locally.
func NewVisibilityHandler(store *visibility.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeError(w, http.StatusInternalServerError, "not_configured", "visibility store not wired")
			return
		}
		var body visibilityPayload
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body: "+err.Error())
			return
		}
		if err := store.SetHidden(body.Hidden); err != nil {
			slog.Error("web: persist visibility", "err", err)
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to save visibility")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"hidden": store.Hidden(),
			"count":  store.Count(),
		})
	}
}
