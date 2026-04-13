package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
)

// PINProvider is the narrow surface the pair handler needs to consult
// the current pairing PIN. It is satisfied by *auth.PINSession in
// production and by fakes in tests, which keeps httpapi independent of
// the auth package's concrete types and avoids an import cycle.
type PINProvider interface {
	Current() string
	Consume() bool
}

// TokenIssuer is the narrow surface the pair handler needs to mint a
// fresh bearer token for a paired device. In production it is a small
// adapter around *auth.Store that calls auth.IssueToken and stores the
// resulting TokenInfo; tests pass a fake that records the label and
// returns a canned plaintext.
type TokenIssuer interface {
	Issue(label string) (plaintext string, err error)
}

type pairRequest struct {
	PIN         string `json:"pin"`
	DeviceLabel string `json:"device_label"`
}

type pairResponse struct {
	Token string `json:"token"`
}

// NewPairHandler returns the POST /api/pair handler. On success it
// consumes the one-shot PIN session and mints a bearer token which is
// returned to the client. The plaintext token is never stored by the
// server — only its hash, inside TokenIssuer.Issue.
func NewPairHandler(p PINProvider, t TokenIssuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req pairRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
			return
		}
		if req.PIN == "" || req.DeviceLabel == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "pin and device_label required")
			return
		}

		// ConstantTimeCompare mitigates timing attacks by refusing to
		// short-circuit on the first mismatching byte. It requires the
		// inputs to be the same length, so a different length also
		// reaches the "invalid pin" branch.
		current := p.Current()
		if subtle.ConstantTimeCompare([]byte(req.PIN), []byte(current)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid pin")
			return
		}
		if !p.Consume() {
			writeError(w, http.StatusUnauthorized, "unauthorized", "pin no longer valid")
			return
		}

		plaintext, err := t.Issue(req.DeviceLabel)
		if err != nil {
			slog.Error("issue token", "err", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}
		writeJSON(w, http.StatusOK, pairResponse{Token: plaintext})
	}
}
