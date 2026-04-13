package config

import (
	"strings"
	"testing"
	"time"
)

// mapLookup builds a Lookuper backed by a map, matching the semantics
// of os.LookupEnv: only present keys count. It is the preferred way
// to test ApplyEnv because it avoids mutating process state.
func mapLookup(env map[string]string) Lookuper {
	return func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
}

func TestApplyEnv_NoVars(t *testing.T) {
	cfg := Defaults()
	if err := cfg.applyEnvFrom(mapLookup(nil)); err != nil {
		t.Fatalf("applyEnvFrom empty: %v", err)
	}
	if cfg.Server.ListenAddr != ":8443" {
		t.Errorf("ListenAddr = %q, defaults should be untouched", cfg.Server.ListenAddr)
	}
}

func TestApplyEnv_AllVars(t *testing.T) {
	env := map[string]string{
		"REMOTELAUNCHER_LISTEN_ADDR":         "0.0.0.0:8888",
		"REMOTELAUNCHER_READ_HEADER_TIMEOUT": "11s",
		"REMOTELAUNCHER_READ_TIMEOUT":        "22s",
		"REMOTELAUNCHER_WRITE_TIMEOUT":       "33s",
		"REMOTELAUNCHER_IDLE_TIMEOUT":        "44s",
		"REMOTELAUNCHER_SHUTDOWN_GRACE":      "55s",
		"REMOTELAUNCHER_CLEANUP_PERIOD":      "7s",
		"REMOTELAUNCHER_PIN_TTL":             "9m",
		"REMOTELAUNCHER_PAIR_RATE_PER_IP":    "13",
		"REMOTELAUNCHER_PAIR_RATE_GLOBAL":    "17",
		"REMOTELAUNCHER_PAIR_RATE_WINDOW":    "6m",
		"REMOTELAUNCHER_CERT_DIR":            "/etc/rl",
		"REMOTELAUNCHER_ICON_THEME":          "Adwaita",
		"REMOTELAUNCHER_LOG_LEVEL":           "warn",
		"REMOTELAUNCHER_LOG_FORMAT":          "json",
	}
	cfg := Defaults()
	if err := cfg.applyEnvFrom(mapLookup(env)); err != nil {
		t.Fatalf("applyEnvFrom: %v", err)
	}

	if cfg.Server.ListenAddr != "0.0.0.0:8888" {
		t.Errorf("ListenAddr = %q", cfg.Server.ListenAddr)
	}
	if cfg.Server.ReadHeaderTimeout != 11*time.Second {
		t.Errorf("ReadHeaderTimeout = %s", cfg.Server.ReadHeaderTimeout)
	}
	if cfg.Server.ReadTimeout != 22*time.Second {
		t.Errorf("ReadTimeout = %s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 33*time.Second {
		t.Errorf("WriteTimeout = %s", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 44*time.Second {
		t.Errorf("IdleTimeout = %s", cfg.Server.IdleTimeout)
	}
	if cfg.Server.ShutdownGrace != 55*time.Second {
		t.Errorf("ShutdownGrace = %s", cfg.Server.ShutdownGrace)
	}
	if cfg.Launcher.CleanupPeriod != 7*time.Second {
		t.Errorf("CleanupPeriod = %s", cfg.Launcher.CleanupPeriod)
	}
	if cfg.Auth.PINTTL != 9*time.Minute {
		t.Errorf("PINTTL = %s", cfg.Auth.PINTTL)
	}
	if cfg.Auth.RateLimitPerIP != 13 {
		t.Errorf("RateLimitPerIP = %d", cfg.Auth.RateLimitPerIP)
	}
	if cfg.Auth.RateLimitGlobal != 17 {
		t.Errorf("RateLimitGlobal = %d", cfg.Auth.RateLimitGlobal)
	}
	if cfg.Auth.RateLimitWindow != 6*time.Minute {
		t.Errorf("RateLimitWindow = %s", cfg.Auth.RateLimitWindow)
	}
	if cfg.Paths.CertDir != "/etc/rl" {
		t.Errorf("CertDir = %q", cfg.Paths.CertDir)
	}
	if cfg.IconTheme != "Adwaita" {
		t.Errorf("IconTheme = %q", cfg.IconTheme)
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level = %q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Logging.Format = %q", cfg.Logging.Format)
	}
}

func TestApplyEnv_InvalidDuration(t *testing.T) {
	cfg := Defaults()
	err := cfg.applyEnvFrom(mapLookup(map[string]string{
		"REMOTELAUNCHER_PIN_TTL": "forever",
	}))
	if err == nil {
		t.Fatal("want error for invalid duration")
	}
	if !strings.Contains(err.Error(), "PIN_TTL") {
		t.Errorf("error = %v, should mention PIN_TTL", err)
	}
}

func TestApplyEnv_InvalidInt(t *testing.T) {
	cfg := Defaults()
	err := cfg.applyEnvFrom(mapLookup(map[string]string{
		"REMOTELAUNCHER_PAIR_RATE_PER_IP": "many",
	}))
	if err == nil {
		t.Fatal("want error for invalid int")
	}
	if !strings.Contains(err.Error(), "PAIR_RATE_PER_IP") {
		t.Errorf("error = %v, should mention PAIR_RATE_PER_IP", err)
	}
}

func TestOSLookup(t *testing.T) {
	// The process-backed Lookuper is a thin wrapper but must still be
	// covered so ApplyEnv's exported path has a test.
	t.Setenv("REMOTELAUNCHER_CONFIG_TEST_VAR", "hello")
	v, ok := osLookup("REMOTELAUNCHER_CONFIG_TEST_VAR")
	if !ok || v != "hello" {
		t.Errorf("osLookup set = (%q, %v), want (hello, true)", v, ok)
	}
	if _, ok := osLookup("REMOTELAUNCHER_CONFIG_DEFINITELY_UNSET_abc123"); ok {
		t.Errorf("osLookup unset = ok, want false")
	}
}
