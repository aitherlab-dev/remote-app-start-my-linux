package terminal

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/sasha/remotelauncher/internal/auth"
	"nhooyr.io/websocket"
)

// resizeMsg is the JSON message the xterm.js client sends when the
// terminal viewport changes size.
type resizeMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// NewWSHandler returns an http.Handler that upgrades to a WebSocket,
// spawns a PTY session, and relays I/O between the two. The Bearer
// token is read from the "token" query parameter because the browser
// WebSocket API does not support custom headers.
func NewWSHandler(tokenStore *auth.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Authenticate via query parameter.
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		if _, ok := tokenStore.Validate(token); !ok {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// The request comes from our own SPA on the same origin.
			InsecureSkipVerify: true,
		})
		if err != nil {
			slog.Error("terminal: websocket accept", "err", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "bye")

		sess, err := NewSession()
		if err != nil {
			slog.Error("terminal: start pty", "err", err)
			conn.Close(websocket.StatusInternalError, "pty start failed")
			return
		}
		defer sess.Close()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// PTY → WebSocket (shell output).
		go func() {
			defer cancel()
			buf := make([]byte, 4096)
			for {
				n, err := sess.Read(buf)
				if err != nil {
					if err != io.EOF {
						slog.Debug("terminal: pty read", "err", err)
					}
					return
				}
				if err := conn.Write(ctx, websocket.MessageBinary, buf[:n]); err != nil {
					return
				}
			}
		}()

		// WebSocket → PTY (user input + resize commands).
		go func() {
			defer cancel()
			for {
				typ, data, err := conn.Read(ctx)
				if err != nil {
					return
				}
				// Text messages are control commands (e.g. resize).
				if typ == websocket.MessageText {
					var msg resizeMsg
					if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" {
						if msg.Cols > 0 && msg.Rows > 0 {
							_ = sess.Resize(msg.Cols, msg.Rows)
						}
					}
					continue
				}
				// Binary messages are raw terminal input.
				if _, err := sess.Write(data); err != nil {
					return
				}
			}
		}()

		// Block until one of the goroutines cancels the context.
		<-ctx.Done()
	}
}
