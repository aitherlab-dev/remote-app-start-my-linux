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
	"github.com/sasha/remotelauncher/internal/config"
	"github.com/sasha/remotelauncher/internal/httpapi"
	"github.com/sasha/remotelauncher/internal/icons"
	"github.com/sasha/remotelauncher/internal/launcher"
	"github.com/sasha/remotelauncher/internal/tlsutil"
)

// Version is set via -ldflags "-X main.Version=<tag>" at release build
// time and defaults to "dev" for local builds.
var Version = "dev"

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
	if err := run(os.Args[1:]); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load(args)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.Paths.ConfigFile != "" {
		slog.Info("config loaded", "path", cfg.Paths.ConfigFile)
	}

	startedAt := time.Now()

	cat := catalog.New(nil)
	loaded, scanErrors, loadErr := cat.Load()
	if loadErr != nil {
		return loadErr
	}
	slog.Info("catalog loaded", "count", loaded, "scan_errors", len(scanErrors))
	for _, se := range scanErrors {
		slog.Warn("scan error", "path", se.Path, "err", se.Err)
	}

	finder := icons.New(nil, cfg.IconTheme)

	tracker := launcher.NewTracker()
	laun := launcher.New(tracker)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// CleanupLoop is bound to the same signal context as the HTTP
	// server so it stops as soon as SIGTERM/SIGINT arrives.
	go tracker.CleanupLoop(ctx, cfg.Launcher.CleanupPeriod)

	certDir, err := resolveCertDir(cfg.Paths.CertDir)
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

	pinSession, err := auth.NewPINSession(cfg.Auth.PINTTL)
	if err != nil {
		return fmt.Errorf("create pin session: %w", err)
	}
	tokenStore := auth.NewStore()
	tokensPath := filepath.Join(certDir, "tokens.json")
	if err := tokenStore.Load(tokensPath); err != nil {
		return fmt.Errorf("load tokens: %w", err)
	}
	slog.Info("tokens loaded", "count", tokenStore.Count(), "path", tokensPath)
	tokenStore.SetPersistPath(tokensPath, func(err error) {
		slog.Error("persist tokens", "err", err, "path", tokensPath)
	})

	// The PIN is printed to stdout on its own so a human operator
	// running the server in a foreground terminal can read it and type
	// it into the phone; slog still logs it structurally so journalctl
	// captures the same value. Printing twice is deliberate. The exact
	// format ("Pairing PIN: NNNNNN") is parsed by the integration test
	// and must not change without updating cmd/remotelauncher/integration_test.go.
	fmt.Fprintf(os.Stdout, "\nPairing PIN: %s (valid for %s)\n\n", pinSession.Current(), cfg.Auth.PINTTL)
	slog.Info("pairing pin generated", "pin", pinSession.Current(), "valid_for", cfg.Auth.PINTTL)

	pairLimiter := auth.NewRateLimiter(cfg.Auth.RateLimitPerIP, cfg.Auth.RateLimitGlobal, cfg.Auth.RateLimitWindow)

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
		RateLimiter: pairLimiter,
	})

	srv := &http.Server{
		Addr:              cfg.Server.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("https server starting", "addr", cfg.Server.ListenAddr)
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownGrace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	slog.Info("https server stopped")
	return nil
}

// resolveCertDir returns the directory that holds the server's TLS
// material. A non-empty override from the config layer wins; otherwise
// we fall back to $XDG_CONFIG_HOME/remotelauncher (or ~/.config/...),
// matching the XDG Base Directory Specification.
func resolveCertDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
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
