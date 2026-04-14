package web

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sasha/remotelauncher/internal/launcher"
	"github.com/sasha/remotelauncher/internal/shortcuts"
)

// shortcutsPayload is the PUT body shape. The client posts the
// entire desired list; the server replaces the set atomically. The
// read endpoint returns the same shape plus the list of supported
// terminal emulators so the web UI can populate the dropdown
// without hard-coding names that go out of sync with the launcher.
type shortcutsPayload struct {
	Shortcuts []shortcuts.Shortcut `json:"shortcuts"`
}

type shortcutsResponse struct {
	Shortcuts          []shortcuts.Shortcut `json:"shortcuts"`
	SupportedTerminals []string             `json:"supported_terminals"`
	DefaultTerminal    string               `json:"default_terminal"`
}

// NewShortcutsHandler returns GET /api/shortcuts for the admin UI.
// It lists the current shortcut set alongside metadata the web form
// needs (whitelist of terminals, server-side default terminal).
func NewShortcutsHandler(store *shortcuts.Store, defaultTerminal string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if store == nil {
			writeError(w, http.StatusInternalServerError, "not_configured", "shortcut store not wired")
			return
		}
		writeJSON(w, http.StatusOK, shortcutsResponse{
			Shortcuts:          store.List(),
			SupportedTerminals: launcher.SupportedTerminals(),
			DefaultTerminal:    defaultTerminal,
		})
	}
}

// NewUpdateShortcutsHandler returns PUT /api/shortcuts. It replaces
// the entire shortcut set with whatever the client posted.
// Validation is delegated to shortcuts.Store.Replace — a bad body
// surfaces as 400 so the web UI can show the exact error in a toast.
func NewUpdateShortcutsHandler(store *shortcuts.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeError(w, http.StatusInternalServerError, "not_configured", "shortcut store not wired")
			return
		}
		var body shortcutsPayload
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body: "+err.Error())
			return
		}
		if err := store.Replace(body.Shortcuts); err != nil {
			// Validation errors read nicely for the user; anything
			// else points at a filesystem problem we should log but
			// not leak verbatim.
			slog.Error("web: replace shortcuts", "err", err)
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"shortcuts": store.List(),
			"count":     store.Count(),
		})
	}
}
