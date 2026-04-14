package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// osLookup is the default Lookuper that forwards to os.LookupEnv.
// Tests substitute a map-backed Lookuper to avoid mutating the real
// process environment.
func osLookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

// Lookuper is the subset of os.LookupEnv the package relies on. The
// indirection exists so tests can feed a fixed environment without
// touching process-global state beyond t.Setenv where convenient.
type Lookuper func(key string) (string, bool)

// ApplyEnv overrides fields of c with values read from the process
// environment, using the REMOTELAUNCHER_ prefix. Unset variables are
// ignored; an invalid value (e.g. non-numeric rate limit) returns an
// error so typos cannot silently fall through to defaults.
func (c *Config) ApplyEnv() error {
	return c.applyEnvFrom(osLookup)
}

func (c *Config) applyEnvFrom(look Lookuper) error {
	if err := envString(look, "REMOTELAUNCHER_LISTEN_ADDR", &c.Server.ListenAddr); err != nil {
		return err
	}
	if err := envDuration(look, "REMOTELAUNCHER_READ_HEADER_TIMEOUT", &c.Server.ReadHeaderTimeout); err != nil {
		return err
	}
	if err := envDuration(look, "REMOTELAUNCHER_READ_TIMEOUT", &c.Server.ReadTimeout); err != nil {
		return err
	}
	if err := envDuration(look, "REMOTELAUNCHER_WRITE_TIMEOUT", &c.Server.WriteTimeout); err != nil {
		return err
	}
	if err := envDuration(look, "REMOTELAUNCHER_IDLE_TIMEOUT", &c.Server.IdleTimeout); err != nil {
		return err
	}
	if err := envDuration(look, "REMOTELAUNCHER_SHUTDOWN_GRACE", &c.Server.ShutdownGrace); err != nil {
		return err
	}
	if err := envDuration(look, "REMOTELAUNCHER_CLEANUP_PERIOD", &c.Launcher.CleanupPeriod); err != nil {
		return err
	}
	if err := envDuration(look, "REMOTELAUNCHER_PIN_TTL", &c.Auth.PINTTL); err != nil {
		return err
	}
	if err := envInt(look, "REMOTELAUNCHER_PAIR_RATE_PER_IP", &c.Auth.RateLimitPerIP); err != nil {
		return err
	}
	if err := envInt(look, "REMOTELAUNCHER_PAIR_RATE_GLOBAL", &c.Auth.RateLimitGlobal); err != nil {
		return err
	}
	if err := envDuration(look, "REMOTELAUNCHER_PAIR_RATE_WINDOW", &c.Auth.RateLimitWindow); err != nil {
		return err
	}
	if err := envBool(look, "REMOTELAUNCHER_WEB_ENABLED", &c.Web.Enabled); err != nil {
		return err
	}
	if err := envString(look, "REMOTELAUNCHER_WEB_LISTEN_ADDR", &c.Web.ListenAddr); err != nil {
		return err
	}
	if err := envString(look, "REMOTELAUNCHER_CERT_DIR", &c.Paths.CertDir); err != nil {
		return err
	}
	if err := envString(look, "REMOTELAUNCHER_ICON_THEME", &c.IconTheme); err != nil {
		return err
	}
	if err := envString(look, "REMOTELAUNCHER_LOG_LEVEL", &c.Logging.Level); err != nil {
		return err
	}
	if err := envString(look, "REMOTELAUNCHER_LOG_FORMAT", &c.Logging.Format); err != nil {
		return err
	}
	return nil
}

func envString(look Lookuper, key string, dst *string) error {
	v, ok := look(key)
	if !ok {
		return nil
	}
	*dst = v
	return nil
}

func envDuration(look Lookuper, key string, dst *time.Duration) error {
	v, ok := look(key)
	if !ok {
		return nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fmt.Errorf("%s: parse duration %q: %w", key, v, err)
	}
	*dst = d
	return nil
}

func envInt(look Lookuper, key string, dst *int) error {
	v, ok := look(key)
	if !ok {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%s: parse int %q: %w", key, v, err)
	}
	*dst = n
	return nil
}

func envBool(look Lookuper, key string, dst *bool) error {
	v, ok := look(key)
	if !ok {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("%s: parse bool %q: %w", key, v, err)
	}
	*dst = b
	return nil
}
