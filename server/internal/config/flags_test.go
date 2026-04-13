package config

import (
	"strings"
	"testing"
	"time"
)

func TestApplyFlags_NoArgs(t *testing.T) {
	cfg := Defaults()
	if err := cfg.ApplyFlags(nil); err != nil {
		t.Fatalf("ApplyFlags nil: %v", err)
	}
	if cfg.Server.ListenAddr != ":8443" {
		t.Errorf("ListenAddr = %q, defaults should be untouched", cfg.Server.ListenAddr)
	}
}

func TestApplyFlags_AllArgs(t *testing.T) {
	args := []string{
		"--config", "/tmp/rl.toml",
		"--listen", "0.0.0.0:1111",
		"--read-header-timeout", "8s",
		"--read-timeout", "9s",
		"--write-timeout", "10s",
		"--idle-timeout", "11s",
		"--shutdown-grace", "12s",
		"--cleanup-period", "13s",
		"--pin-ttl", "14m",
		"--pair-rate-per-ip", "100",
		"--pair-rate-global", "1000",
		"--pair-rate-window", "3m",
		"--cert-dir", "/srv/cert",
		"--icon-theme", "Tango",
		"--log-level", "error",
		"--log-format", "json",
	}
	cfg := Defaults()
	if err := cfg.ApplyFlags(args); err != nil {
		t.Fatalf("ApplyFlags: %v", err)
	}
	// --config does not mutate the Config — Load() handles it — but
	// the parser must have accepted it without blowing up.
	if cfg.Server.ListenAddr != "0.0.0.0:1111" {
		t.Errorf("ListenAddr = %q", cfg.Server.ListenAddr)
	}
	if cfg.Server.ReadHeaderTimeout != 8*time.Second {
		t.Errorf("ReadHeaderTimeout = %s", cfg.Server.ReadHeaderTimeout)
	}
	if cfg.Server.ReadTimeout != 9*time.Second {
		t.Errorf("ReadTimeout = %s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout = %s", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 11*time.Second {
		t.Errorf("IdleTimeout = %s", cfg.Server.IdleTimeout)
	}
	if cfg.Server.ShutdownGrace != 12*time.Second {
		t.Errorf("ShutdownGrace = %s", cfg.Server.ShutdownGrace)
	}
	if cfg.Launcher.CleanupPeriod != 13*time.Second {
		t.Errorf("CleanupPeriod = %s", cfg.Launcher.CleanupPeriod)
	}
	if cfg.Auth.PINTTL != 14*time.Minute {
		t.Errorf("PINTTL = %s", cfg.Auth.PINTTL)
	}
	if cfg.Auth.RateLimitPerIP != 100 {
		t.Errorf("RateLimitPerIP = %d", cfg.Auth.RateLimitPerIP)
	}
	if cfg.Auth.RateLimitGlobal != 1000 {
		t.Errorf("RateLimitGlobal = %d", cfg.Auth.RateLimitGlobal)
	}
	if cfg.Auth.RateLimitWindow != 3*time.Minute {
		t.Errorf("RateLimitWindow = %s", cfg.Auth.RateLimitWindow)
	}
	if cfg.Paths.CertDir != "/srv/cert" {
		t.Errorf("CertDir = %q", cfg.Paths.CertDir)
	}
	if cfg.IconTheme != "Tango" {
		t.Errorf("IconTheme = %q", cfg.IconTheme)
	}
	if cfg.Logging.Level != "error" {
		t.Errorf("Logging.Level = %q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Logging.Format = %q", cfg.Logging.Format)
	}
}

func TestApplyFlags_UnknownFlag(t *testing.T) {
	cfg := Defaults()
	err := cfg.ApplyFlags([]string{"--no-such-flag"})
	if err == nil {
		t.Fatal("want error for unknown flag")
	}
	if !strings.Contains(err.Error(), "parse flags") {
		t.Errorf("error = %v, should mention parse", err)
	}
}

func TestApplyFlags_OnlyOneFlag(t *testing.T) {
	cfg := Defaults()
	if err := cfg.ApplyFlags([]string{"--listen", ":9000"}); err != nil {
		t.Fatalf("ApplyFlags: %v", err)
	}
	if cfg.Server.ListenAddr != ":9000" {
		t.Errorf("ListenAddr = %q", cfg.Server.ListenAddr)
	}
	// Fields that were not touched must keep their defaults.
	if cfg.Auth.PINTTL != 10*time.Minute {
		t.Errorf("PINTTL = %s, want default 10m", cfg.Auth.PINTTL)
	}
	if cfg.Paths.CertDir != "" {
		t.Errorf("CertDir = %q, want empty", cfg.Paths.CertDir)
	}
}
