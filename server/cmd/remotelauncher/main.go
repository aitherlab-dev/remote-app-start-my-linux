package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sasha/remotelauncher/internal/auth"
	"github.com/sasha/remotelauncher/internal/catalog"
	"github.com/sasha/remotelauncher/internal/httpapi"
	"github.com/sasha/remotelauncher/internal/icons"
	"github.com/sasha/remotelauncher/internal/launcher"
	"github.com/sasha/remotelauncher/internal/tlsutil"
)

// Version is set via -ldflags "-X main.Version=<tag>" at release build
// time and defaults to "dev" for local builds.
var Version = "dev"

const (
	listenAddr           = ":8443"
	readHeaderTimeout    = 5 * time.Second
	readTimeout          = 30 * time.Second
	writeTimeout         = 30 * time.Second
	idleTimeout          = 120 * time.Second
	shutdownGraceLimit   = 10 * time.Second
	trackerCleanupPeriod = 5 * time.Second
	pinTTL               = 10 * time.Minute
)

// storeTokenIssuer is the main-package adapter that bridges the
// httpapi.TokenIssuer interface to the auth package: mint a fresh
// token, record its hash in the Store, hand the plaintext back to the
// pair handler. Keeping the adapter in main avoids any import cycle
// between httpapi and auth.
type storeTokenIssuer struct {
	store *auth.Store
}

func (i storeTokenIssuer) Issue(label string) (string, error) {
	plaintext, info, err := auth.IssueToken(label)
	if err != nil {
		return "", err
	}
	i.store.Add(info)
	return plaintext, nil
}

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

	finder := icons.New(nil, os.Getenv("REMOTELAUNCHER_ICON_THEME"))

	tracker := launcher.NewTracker()
	laun := launcher.New(tracker)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// CleanupLoop is bound to the same signal context as the HTTP
	// server so it stops as soon as SIGTERM/SIGINT arrives.
	go tracker.CleanupLoop(ctx, trackerCleanupPeriod)

	certDir, err := configCertDir()
	if err != nil {
		return fmt.Errorf("locate cert dir: %w", err)
	}
	certPath, keyPath, err := tlsutil.EnsureCert(certDir)
	if err != nil {
		return fmt.Errorf("ensure tls cert: %w", err)
	}
	fingerprint, err := tlsutil.Fingerprint(certPath)
	if err != nil {
		return fmt.Errorf("compute fingerprint: %w", err)
	}
	slog.Info("tls certificate ready", "cert", certPath, "fingerprint", fingerprint)

	pinSession, err := auth.NewPINSession(pinTTL)
	if err != nil {
		return fmt.Errorf("create pin session: %w", err)
	}
	tokenStore := auth.NewStore()

	// The PIN is printed to stdout on its own so a human operator
	// running the server in a foreground terminal can read it and type
	// it into the phone; slog still logs it structurally so journalctl
	// captures the same value. Printing twice is deliberate.
	fmt.Fprintf(os.Stdout, "\nPairing PIN: %s (valid for %s)\n\n", pinSession.Current(), pinTTL)
	slog.Info("pairing pin generated", "pin", pinSession.Current(), "valid_for", pinTTL)

	handler := httpapi.NewRouter(httpapi.RouterDeps{
		Version:     Version,
		StartedAt:   startedAt,
		Catalog:     cat,
		Finder:      finder,
		Launcher:    laun,
		Alive:       tracker,
		Fingerprint: fingerprint,
		TokenStore:  tokenStore,
		PINProvider: pinSession,
		TokenIssuer: storeTokenIssuer{store: tokenStore},
	})

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("https server starting", "addr", listenAddr)
		if err := srv.ListenAndServeTLS(certPath, keyPath); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	slog.Info("https server stopped")
	return nil
}

// configCertDir returns the directory that stores the server's TLS
// material. $XDG_CONFIG_HOME takes precedence; otherwise we fall back
// to ~/.config, matching the XDG Base Directory Specification.
func configCertDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "remotelauncher"), nil
}
