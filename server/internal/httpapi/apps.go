package httpapi

import (
	"net/http"

	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/shortcuts"
)

// AliveChecker is the small surface httpapi needs to decorate AppInfo
// with the current process state. It is satisfied by *launcher.Tracker
// in production and by fakes in tests so the HTTP layer can be
// exercised without spawning processes.
type AliveChecker interface {
	Alive(id string) bool
}

// VisibilityChecker reports whether an application id has been
// marked hidden by the operator. It is satisfied by
// *visibility.Store in production and by fakes in tests so the HTTP
// layer can be exercised without touching the filesystem. A nil
// checker disables filtering entirely.
type VisibilityChecker interface {
	IsHidden(id string) bool
}

// NewAppsHandler returns an http.HandlerFunc that serves the current
// application list as a JSON array of catalog.AppInfo. The catalog
// itself never tracks process state, so this handler walks the list
// and stamps each AppInfo.Running via the AliveChecker before
// serialising. alive may be nil — in that case Running stays false
// for every entry.
//
// The visibility argument filters the list: any app whose id is
// reported as hidden is dropped before serialisation, so the phone
// never sees entries the operator has masked in the admin UI. A nil
// visibility checker disables filtering (all apps pass through).
//
// User-defined shortcuts from the ShortcutProvider are merged into
// the list before the visibility filter so the operator can also
// hide a custom shortcut from the phone if they want to. A nil
// provider disables the feature entirely.
func NewAppsHandler(
	c *catalog.Catalog,
	alive AliveChecker,
	visibility VisibilityChecker,
	provider ShortcutProvider,
) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		list := c.List()
		if list == nil {
			list = []catalog.AppInfo{}
		}
		if provider != nil {
			for _, sc := range provider.List() {
				list = append(list, catalog.AppInfo{
					ID:      shortcuts.PrefixedID(sc.ID),
					Name:    sc.Name,
					Comment: sc.Command,
					Icon:    "",
				})
			}
		}
		if visibility != nil {
			filtered := list[:0]
			for _, a := range list {
				if visibility.IsHidden(a.ID) {
					continue
				}
				filtered = append(filtered, a)
			}
			list = filtered
		}
		if alive != nil {
			for i := range list {
				list[i].Running = alive.Alive(list[i].ID)
			}
		}
		writeJSON(w, http.StatusOK, list)
	}
}
