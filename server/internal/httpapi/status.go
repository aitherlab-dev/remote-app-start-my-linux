package httpapi

import (
	"net/http"
	"time"

	"github.com/sasha/remotelauncher/internal/catalog"
)

// StatusResponse is the JSON body returned by GET /api/status.
type StatusResponse struct {
	Version         string    `json:"version"`
	StartedAt       time.Time `json:"started_at"`
	UptimeSec       int64     `json:"uptime_sec"`
	AppsCount       int       `json:"apps_count"`
	CertFingerprint string    `json:"cert_fingerprint"`
}

// NewStatusHandler returns an http.HandlerFunc that reports the
// server's version, start time, monotonic uptime in seconds, the
// number of applications currently exposed by the catalog and the
// SHA-256 fingerprint of the server's TLS certificate (used by clients
// for trust-on-first-use pinning).
func NewStatusHandler(version string, startedAt time.Time, c *catalog.Catalog, fingerprint string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		resp := StatusResponse{
			Version:         version,
			StartedAt:       startedAt,
			UptimeSec:       int64(time.Since(startedAt).Seconds()),
			AppsCount:       len(c.List()),
			CertFingerprint: fingerprint,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
