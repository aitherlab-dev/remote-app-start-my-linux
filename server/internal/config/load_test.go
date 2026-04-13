package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfigPath_XDGWins(t *testing.T) {
	got := defaultConfigPathFrom(
		func(k string) string {
			if k == "XDG_CONFIG_HOME" {
				return "/xdg"
			}
			return ""
		},
		func() (string, error) { return "/home/me", nil },
	)
	if want := "/xdg/remotelauncher/config.toml"; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

func TestDefaultConfigPath_HomeFallback(t *testing.T) {
	got := defaultConfigPathFrom(
		func(string) string { return "" },
		func() (string, error) { return "/home/me", nil },
	)
	if want := "/home/me/.config/remotelauncher/config.toml"; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

func TestDefaultConfigPath_NoHome(t *testing.T) {
	got := defaultConfigPathFrom(
		func(string) string { return "" },
		func() (string, error) { return "", errors.New("no home") },
	)
	if got != "" {
		t.Errorf("path = %q, want empty on home error", got)
	}
}

func TestDefaultConfigPath_OSBacked(t *testing.T) {
	// Exercises the exported DefaultConfigPath() so it does not
	// drift from the injectable helper. Forces XDG_CONFIG_HOME to a
	// known value so the result is deterministic.
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-default")
	got := DefaultConfigPath()
	if want := "/tmp/xdg-default/remotelauncher/config.toml"; got != want {
		t.Errorf("DefaultConfigPath() = %q, want %q", got, want)
	}
}

// TestLoad_MergeOrder pins the precedence chain: defaults → file →
// env → flags. Each layer touches a different field so the test can
// witness them all at once, and each layer also overwrites a field
// set by the previous layer to confirm overriding works in order.
func TestLoad_MergeOrder(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.toml")
	// File sets listen to :1000, read_timeout to 10s, pin_ttl to 1m.
	body := `
[server]
listen_addr  = ":1000"
read_timeout = "10s"

[auth]
pin_ttl = "1m"
`
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	// Env overrides listen (layer priority check) and read_timeout,
	// leaves pin_ttl alone so it keeps the file value.
	t.Setenv("REMOTELAUNCHER_LISTEN_ADDR", ":2000")
	t.Setenv("REMOTELAUNCHER_READ_TIMEOUT", "20s")

	// Flag overrides listen one more time (env → flag priority) and
	// also sets cert_dir which no earlier layer touched.
	args := []string{
		"--config", cfgPath,
		"--listen", ":3000",
		"--cert-dir", "/etc/rl",
	}

	cfg, err := Load(args)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.ListenAddr != ":3000" {
		t.Errorf("ListenAddr = %q, want :3000 (flag wins)", cfg.Server.ListenAddr)
	}
	if cfg.Server.ReadTimeout != 20*time.Second {
		t.Errorf("ReadTimeout = %s, want 20s (env wins)", cfg.Server.ReadTimeout)
	}
	if cfg.Auth.PINTTL != 1*time.Minute {
		t.Errorf("PINTTL = %s, want 1m (file value survives)", cfg.Auth.PINTTL)
	}
	if cfg.Server.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout = %s, want default 30s", cfg.Server.WriteTimeout)
	}
	if cfg.Paths.CertDir != "/etc/rl" {
		t.Errorf("CertDir = %q, want /etc/rl", cfg.Paths.CertDir)
	}
	if cfg.Paths.ConfigFile != cfgPath {
		t.Errorf("ConfigFile = %q, want %q", cfg.Paths.ConfigFile, cfgPath)
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Point to a dir that has no config.toml — Load must fall back
	// to pure defaults without error.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// Guard against leaking dev-machine env vars into the test.
	for _, k := range []string{
		"REMOTELAUNCHER_LISTEN_ADDR",
		"REMOTELAUNCHER_READ_HEADER_TIMEOUT",
		"REMOTELAUNCHER_READ_TIMEOUT",
		"REMOTELAUNCHER_WRITE_TIMEOUT",
		"REMOTELAUNCHER_IDLE_TIMEOUT",
		"REMOTELAUNCHER_SHUTDOWN_GRACE",
		"REMOTELAUNCHER_CLEANUP_PERIOD",
		"REMOTELAUNCHER_PIN_TTL",
		"REMOTELAUNCHER_PAIR_RATE_PER_IP",
		"REMOTELAUNCHER_PAIR_RATE_GLOBAL",
		"REMOTELAUNCHER_PAIR_RATE_WINDOW",
		"REMOTELAUNCHER_CERT_DIR",
		"REMOTELAUNCHER_ICON_THEME",
		"REMOTELAUNCHER_LOG_LEVEL",
		"REMOTELAUNCHER_LOG_FORMAT",
	} {
		if _, ok := os.LookupEnv(k); ok {
			t.Setenv(k, "") // shadow the dev value first so Cleanup restores
			os.Unsetenv(k)
		}
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.ListenAddr != ":8443" {
		t.Errorf("ListenAddr = %q, want default :8443", cfg.Server.ListenAddr)
	}
	if cfg.Paths.ConfigFile != "" {
		t.Errorf("ConfigFile = %q, want empty (no file loaded)", cfg.Paths.ConfigFile)
	}
}

func TestLoad_FlagParseError(t *testing.T) {
	_, err := Load([]string{"--no-such-flag"})
	if err == nil {
		t.Fatal("want error for unknown flag")
	}
	if !strings.Contains(err.Error(), "parse flags") {
		t.Errorf("error = %v, should mention parse flags", err)
	}
}

func TestLoad_FileParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(path, []byte("not = valid\ntoml [[["), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Load([]string{"--config", path})
	if err == nil {
		t.Fatal("want error for invalid TOML")
	}
}

func TestLoad_ValidationError(t *testing.T) {
	_, err := Load([]string{"--listen", ""})
	if err == nil {
		t.Fatal("want error for empty listen")
	}
	if !strings.Contains(err.Error(), "listen_addr") {
		t.Errorf("error = %v, should mention listen_addr", err)
	}
}
