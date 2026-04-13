package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/sasha/remotelauncher/internal/config"
)

func TestBuildLogger_Levels(t *testing.T) {
	cases := []struct {
		level    string
		want     slog.Level
		enabled  []slog.Level
		disabled []slog.Level
	}{
		{
			level:    "debug",
			want:     slog.LevelDebug,
			enabled:  []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError},
			disabled: nil,
		},
		{
			level:    "INFO",
			want:     slog.LevelInfo,
			enabled:  []slog.Level{slog.LevelInfo, slog.LevelWarn, slog.LevelError},
			disabled: []slog.Level{slog.LevelDebug},
		},
		{
			level:    " Warn ",
			want:     slog.LevelWarn,
			enabled:  []slog.Level{slog.LevelWarn, slog.LevelError},
			disabled: []slog.Level{slog.LevelDebug, slog.LevelInfo},
		},
		{
			level:    "error",
			want:     slog.LevelError,
			enabled:  []slog.Level{slog.LevelError},
			disabled: []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.level, func(t *testing.T) {
			cfg := config.Defaults()
			cfg.Logging.Level = tc.level
			cfg.Logging.Format = "text"
			logger := buildLogger(&cfg)
			if logger == nil {
				t.Fatalf("buildLogger returned nil")
			}
			h := logger.Handler()
			for _, lvl := range tc.enabled {
				if !h.Enabled(context.Background(), lvl) {
					t.Errorf("level %q: handler should accept %v", tc.level, lvl)
				}
			}
			for _, lvl := range tc.disabled {
				if h.Enabled(context.Background(), lvl) {
					t.Errorf("level %q: handler should reject %v", tc.level, lvl)
				}
			}
		})
	}
}

func TestBuildLogger_JSONFormat(t *testing.T) {
	// Swap stderr for a buffer by going through the handler directly:
	// buildLogger writes to os.Stderr, but handler types are detectable
	// by emitting a record and checking that the output is valid JSON.
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	h := slog.NewJSONHandler(&buf, opts)
	logger := slog.New(h)
	logger.Info("hello", "k", "v")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("json handler output not valid JSON: %v\noutput=%s", err, buf.String())
	}

	// Sanity-check the buildLogger-selected handler type for each
	// format by constructing configs and comparing against expected
	// Go types via a small interface assertion.
	cfg := config.Defaults()
	cfg.Logging.Format = "json"
	jsonLogger := buildLogger(&cfg)
	if _, ok := jsonLogger.Handler().(*slog.JSONHandler); !ok {
		t.Errorf("format=json: expected *slog.JSONHandler, got %T", jsonLogger.Handler())
	}

	cfg.Logging.Format = "text"
	textLogger := buildLogger(&cfg)
	if _, ok := textLogger.Handler().(*slog.TextHandler); !ok {
		t.Errorf("format=text: expected *slog.TextHandler, got %T", textLogger.Handler())
	}

	cfg.Logging.Format = "TEXT"
	upperTextLogger := buildLogger(&cfg)
	if _, ok := upperTextLogger.Handler().(*slog.TextHandler); !ok {
		t.Errorf("format=TEXT: expected *slog.TextHandler, got %T", upperTextLogger.Handler())
	}

	cfg.Logging.Format = "Json"
	mixedJSONLogger := buildLogger(&cfg)
	if _, ok := mixedJSONLogger.Handler().(*slog.JSONHandler); !ok {
		t.Errorf("format=Json: expected *slog.JSONHandler, got %T", mixedJSONLogger.Handler())
	}
}

func TestBuildLogger_DefaultsPreserveInfoText(t *testing.T) {
	cfg := config.Defaults()
	logger := buildLogger(&cfg)
	h := logger.Handler()
	if _, ok := h.(*slog.TextHandler); !ok {
		t.Errorf("defaults: expected *slog.TextHandler, got %T", h)
	}
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Errorf("defaults: debug must be filtered out")
	}
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Errorf("defaults: info must be allowed")
	}
	// Basic consistency check: level string matches the one Defaults returns.
	if !strings.EqualFold(cfg.Logging.Level, "info") {
		t.Errorf("defaults level drifted from info: %q", cfg.Logging.Level)
	}
	if !strings.EqualFold(cfg.Logging.Format, "text") {
		t.Errorf("defaults format drifted from text: %q", cfg.Logging.Format)
	}
}
