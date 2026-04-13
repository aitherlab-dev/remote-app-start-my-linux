package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/httpapi"
)

// Version is set via -ldflags "-X main.Version=<tag>" at release build
// time and defaults to "dev" for local builds.
var Version = "dev"

const (
	listenAddr         = ":8765"
	readHeaderTimeout  = 5 * time.Second
	readTimeout        = 30 * time.Second
	writeTimeout       = 30 * time.Second
	idleTimeout        = 120 * time.Second
	shutdownGraceLimit = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	startedAt := time.Now()

	cat := catalog.New(nil)
	loaded, scanErrors, err := cat.Load()
	if err != nil {
		return err
	}
	slog.Info("catalog loaded", "count", loaded, "scan_errors", len(scanErrors))
	for _, se := range scanErrors {
		slog.Warn("scan error", "path", se.Path, "err", se.Err)
	}

	handler := httpapi.NewRouter(Version, startedAt, cat)

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("http server starting", "addr", listenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGraceLimit)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	slog.Info("http server stopped")
	return nil
}
