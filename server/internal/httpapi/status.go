package httpapi

import (
	"net/http"
	"time"

	"github.com/sasha/remotelauncher/internal/catalog"
)

// StatusResponse is the JSON body returned by GET /api/status.
type StatusResponse struct {
	Version   string    `json:"version"`
	StartedAt time.Time `json:"started_at"`
	UptimeSec int64     `json:"uptime_sec"`
	AppsCount int       `json:"apps_count"`
}

// NewStatusHandler returns an http.HandlerFunc that reports the
// server's version, start time, monotonic uptime in seconds and the
// number of applications currently exposed by the catalog.
func NewStatusHandler(version string, startedAt time.Time, c *catalog.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		resp := StatusResponse{
			Version:   version,
			StartedAt: startedAt,
			UptimeSec: int64(time.Since(startedAt).Seconds()),
			AppsCount: len(c.List()),
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
