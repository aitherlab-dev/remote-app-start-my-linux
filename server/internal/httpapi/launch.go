package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/desktop"
	"github.com/sasha/remotelauncher/internal/launcher"
	"github.com/sasha/remotelauncher/internal/shortcuts"
)

// AppLauncher is the small surface httpapi needs from the launcher
// package. It is satisfied by *launcher.Launcher in production and by
// fakes in tests, so the HTTP layer never has to spawn real processes
// to be exercised.
type AppLauncher interface {
	Launch(entry desktop.Entry) (int, error)
	LaunchCommand(id, terminal, defaultTerminal, cwd, command string) (int, error)
}

// ShortcutProvider is the tiny read slice of shortcuts.Store that
// the HTTP layer needs for /api/apps and /api/apps/{id}/launch. A
// nil provider disables the whole custom-shortcut feature — clients
// see only real .desktop apps and a launch on a "custom:" id yields
// 404.
type ShortcutProvider interface {
	List() []shortcuts.Shortcut
	Get(id string) (shortcuts.Shortcut, bool)
}

// launchResponse is the JSON body returned on a successful launch.
type launchResponse struct {
	Status string `json:"status"`
	PID    int    `json:"pid"`
}

// NewLaunchHandler returns an http.HandlerFunc that starts the
// application identified by {id} and answers with its PID.
//
// Path: POST /api/apps/{id}/launch
//
// An id starting with the shortcuts.IDPrefix ("custom:") is resolved
// against the shortcut store instead of the .desktop catalog and is
// spawned through Launcher.LaunchCommand inside the configured
// terminal emulator.
//
// Status codes:
//   - 200 — process started, body `{"status":"launched","pid":N}`
//   - 400 — entry exists but has no Exec command, or the launcher
//     reports launcher.ErrEmptyExec / launcher.ErrNoTerminalEmulator /
//     ErrUnknownTerminal
//   - 404 — id is not in the catalog
//   - 500 — any other launcher failure (details are logged via slog,
//     the HTTP body only carries a generic "internal error" so we
//     never leak filesystem paths or argv tokens to the client).
func NewLaunchHandler(
	c *catalog.Catalog,
	l AppLauncher,
	provider ShortcutProvider,
	defaultTerminal string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if bare, isCustom := shortcuts.NormalizeID(id); isCustom {
			if provider == nil {
				writeError(w, http.StatusNotFound, "not_found", "shortcut not found")
				return
			}
			sc, ok := provider.Get(bare)
			if !ok {
				writeError(w, http.StatusNotFound, "not_found", "shortcut not found")
				return
			}
			pid, err := l.LaunchCommand(id, sc.Terminal, defaultTerminal, sc.Cwd, sc.Command)
			if err != nil {
				switch {
				case errors.Is(err, launcher.ErrEmptyExec):
					writeError(w, http.StatusBadRequest, "bad_request", "shortcut has no command")
				case errors.Is(err, launcher.ErrNoTerminalEmulator),
					errors.Is(err, launcher.ErrUnknownTerminal):
					writeError(w, http.StatusBadRequest, "bad_request", err.Error())
				default:
					slog.Error("shortcut launch failed", "id", id, "err", err)
					writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
				}
				return
			}
			writeJSON(w, http.StatusOK, launchResponse{Status: "launched", PID: pid})
			return
		}

		entry, ok := c.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "app not found")
			return
		}
		if entry.Exec == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "app has no exec command")
			return
		}

		pid, err := l.Launch(entry)
		if err != nil {
			switch {
			case errors.Is(err, launcher.ErrEmptyExec):
				writeError(w, http.StatusBadRequest, "bad_request", "app has no exec command")
			case errors.Is(err, launcher.ErrNoTerminalEmulator):
				writeError(w, http.StatusBadRequest, "bad_request", "no terminal emulator available")
			default:
				slog.Error("launch failed", "id", id, "err", err)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			}
			return
		}

		writeJSON(w, http.StatusOK, launchResponse{Status: "launched", PID: pid})
	}
}
