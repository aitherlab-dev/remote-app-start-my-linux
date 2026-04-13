package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFile_MissingFileIsNotAnError(t *testing.T) {
	cfg := Defaults()
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")
	if err := LoadFile(&cfg, path); err != nil {
		t.Fatalf("LoadFile missing = %v, want nil", err)
	}
	// Defaults must remain intact.
	if cfg.Server.ListenAddr != ":8443" {
		t.Errorf("ListenAddr = %q, want untouched", cfg.Server.ListenAddr)
	}
	if cfg.Paths.ConfigFile != "" {
		t.Errorf("ConfigFile = %q, want empty", cfg.Paths.ConfigFile)
	}
}

func TestLoadFile_EmptyPathIsNoOp(t *testing.T) {
	cfg := Defaults()
	if err := LoadFile(&cfg, ""); err != nil {
		t.Fatalf("LoadFile empty path = %v, want nil", err)
	}
	if cfg.Paths.ConfigFile != "" {
		t.Errorf("ConfigFile = %q, want empty", cfg.Paths.ConfigFile)
	}
}

func TestLoadFile_FullTOML(t *testing.T) {
	path := writeTempFile(t, `
icon_theme = "breeze"

[server]
listen_addr         = "127.0.0.1:9999"
read_header_timeout = "7s"
read_timeout        = "41s"
write_timeout       = "42s"
idle_timeout        = "3m"
shutdown_grace      = "15s"

[launcher]
cleanup_period = "12s"

[auth]
pin_ttl             = "15m"
rate_limit_per_ip   = 9
rate_limit_global   = 99
rate_limit_window   = "5m"

[paths]
cert_dir = "/var/lib/rl"

[logging]
level  = "debug"
format = "json"
`)

	cfg := Defaults()
	if err := LoadFile(&cfg, path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}

	if cfg.Server.ListenAddr != "127.0.0.1:9999" {
		t.Errorf("ListenAddr = %q", cfg.Server.ListenAddr)
	}
	if cfg.Server.ReadHeaderTimeout != 7*time.Second {
		t.Errorf("ReadHeaderTimeout = %s", cfg.Server.ReadHeaderTimeout)
	}
	if cfg.Server.ReadTimeout != 41*time.Second {
		t.Errorf("ReadTimeout = %s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 42*time.Second {
		t.Errorf("WriteTimeout = %s", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 3*time.Minute {
		t.Errorf("IdleTimeout = %s", cfg.Server.IdleTimeout)
	}
	if cfg.Server.ShutdownGrace != 15*time.Second {
		t.Errorf("ShutdownGrace = %s", cfg.Server.ShutdownGrace)
	}
	if cfg.Launcher.CleanupPeriod != 12*time.Second {
		t.Errorf("CleanupPeriod = %s", cfg.Launcher.CleanupPeriod)
	}
	if cfg.Auth.PINTTL != 15*time.Minute {
		t.Errorf("PINTTL = %s", cfg.Auth.PINTTL)
	}
	if cfg.Auth.RateLimitPerIP != 9 {
		t.Errorf("RateLimitPerIP = %d", cfg.Auth.RateLimitPerIP)
	}
	if cfg.Auth.RateLimitGlobal != 99 {
		t.Errorf("RateLimitGlobal = %d", cfg.Auth.RateLimitGlobal)
	}
	if cfg.Auth.RateLimitWindow != 5*time.Minute {
		t.Errorf("RateLimitWindow = %s", cfg.Auth.RateLimitWindow)
	}
	if cfg.Paths.CertDir != "/var/lib/rl" {
		t.Errorf("CertDir = %q", cfg.Paths.CertDir)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Level = %q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Format = %q", cfg.Logging.Format)
	}
	if cfg.IconTheme != "breeze" {
		t.Errorf("IconTheme = %q", cfg.IconTheme)
	}
	if cfg.Paths.ConfigFile != path {
		t.Errorf("Paths.ConfigFile = %q, want %q", cfg.Paths.ConfigFile, path)
	}
}

func TestLoadFile_PartialTOMLDoesNotClobberDefaults(t *testing.T) {
	path := writeTempFile(t, `
[server]
listen_addr = ":1234"

[auth]
rate_limit_per_ip = 42
`)

	cfg := Defaults()
	if err := LoadFile(&cfg, path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if cfg.Server.ListenAddr != ":1234" {
		t.Errorf("listen_addr override failed: %q", cfg.Server.ListenAddr)
	}
	// Untouched fields retain defaults.
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %s, want default 30s", cfg.Server.ReadTimeout)
	}
	if cfg.Auth.PINTTL != 10*time.Minute {
		t.Errorf("PINTTL = %s, want default 10m", cfg.Auth.PINTTL)
	}
	if cfg.Auth.RateLimitPerIP != 42 {
		t.Errorf("RateLimitPerIP = %d, want 42", cfg.Auth.RateLimitPerIP)
	}
	if cfg.Auth.RateLimitGlobal != 20 {
		t.Errorf("RateLimitGlobal = %d, want default 20", cfg.Auth.RateLimitGlobal)
	}
}

func TestLoadFile_InvalidTOML(t *testing.T) {
	path := writeTempFile(t, "this is = not valid = toml\n[[[")
	cfg := Defaults()
	err := LoadFile(&cfg, path)
	if err == nil {
		t.Fatal("LoadFile invalid TOML, want error")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error = %v, want to mention parsing", err)
	}
}

func TestLoadFile_InvalidDuration(t *testing.T) {
	path := writeTempFile(t, `
[server]
read_timeout = "not-a-duration"
`)
	cfg := Defaults()
	err := LoadFile(&cfg, path)
	if err == nil {
		t.Fatal("LoadFile invalid duration, want error")
	}
	if !strings.Contains(err.Error(), "read_timeout") {
		t.Errorf("error = %v, should mention read_timeout", err)
	}
}

func TestLoadFile_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`[server]`), 0o000); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg := Defaults()
	err := LoadFile(&cfg, path)
	// Root can read anything; skip if the permission bit is ignored by
	// the test environment.
	if err == nil {
		t.Skip("filesystem ignores 0o000 (likely running as root); nothing to check")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("error = %v, want to mention read", err)
	}
}

// writeTempFile drops content into a fresh temp file and returns the
// absolute path. It centralises the pattern so individual tests stay
// focused on what they are asserting.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return path
}
