package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/sasha/remotelauncher/internal/config"
)

// buildLogger turns the validated Logging section of the config into a
// concrete *slog.Logger writing to os.Stderr. config.Validate() has
// already rejected any value that would fall through to the default
// branches below, so they only guard against a misuse at the Go call
// site — not operator input.
func buildLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLogLevel(cfg.Logging.Level)}
	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(cfg.Logging.Format)) {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	return slog.New(handler)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
