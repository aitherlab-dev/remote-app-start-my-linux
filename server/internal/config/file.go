package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// rawConfig mirrors Config but every field is a pointer so the TOML
// decoder can tell "key missing" apart from "key set to zero value".
// Only non-nil fields are propagated into the real Config, preserving
// the merge invariant: a layer that does not mention a key must not
// clobber earlier layers.
//
// Durations live as *string so the TOML file can carry human-readable
// values like "30s" or "5m" instead of raw nanosecond integers.
type rawConfig struct {
	Server    *rawServer   `toml:"server"`
	Launcher  *rawLauncher `toml:"launcher"`
	Auth      *rawAuth     `toml:"auth"`
	Paths     *rawPaths    `toml:"paths"`
	Logging   *rawLogging  `toml:"logging"`
	IconTheme *string      `toml:"icon_theme"`
}

type rawServer struct {
	ListenAddr        *string `toml:"listen_addr"`
	ReadHeaderTimeout *string `toml:"read_header_timeout"`
	ReadTimeout       *string `toml:"read_timeout"`
	WriteTimeout      *string `toml:"write_timeout"`
	IdleTimeout       *string `toml:"idle_timeout"`
	ShutdownGrace     *string `toml:"shutdown_grace"`
}

type rawLauncher struct {
	CleanupPeriod *string `toml:"cleanup_period"`
}

type rawAuth struct {
	PINTTL          *string `toml:"pin_ttl"`
	RateLimitPerIP  *int    `toml:"rate_limit_per_ip"`
	RateLimitGlobal *int    `toml:"rate_limit_global"`
	RateLimitWindow *string `toml:"rate_limit_window"`
}

type rawPaths struct {
	CertDir *string `toml:"cert_dir"`
}

type rawLogging struct {
	Level  *string `toml:"level"`
	Format *string `toml:"format"`
}

// LoadFile merges the TOML file at path into cfg. A missing file is
// not an error: the server has always been usable without any config
// file, and that behaviour is preserved. Only explicitly-set keys
// overwrite fields of cfg; absent keys leave the previous layer alone.
func LoadFile(cfg *Config, path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	var raw rawConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if err := raw.applyTo(cfg); err != nil {
		return fmt.Errorf("apply %s: %w", path, err)
	}
	cfg.Paths.ConfigFile = path
	return nil
}

func (r *rawConfig) applyTo(c *Config) error {
	if r.Server != nil {
		if err := r.Server.applyTo(&c.Server); err != nil {
			return err
		}
	}
	if r.Launcher != nil {
		if err := r.Launcher.applyTo(&c.Launcher); err != nil {
			return err
		}
	}
	if r.Auth != nil {
		if err := r.Auth.applyTo(&c.Auth); err != nil {
			return err
		}
	}
	if r.Paths != nil {
		r.Paths.applyTo(&c.Paths)
	}
	if r.Logging != nil {
		r.Logging.applyTo(&c.Logging)
	}
	if r.IconTheme != nil {
		c.IconTheme = *r.IconTheme
	}
	return nil
}

func (r *rawServer) applyTo(c *ServerConfig) error {
	if r.ListenAddr != nil {
		c.ListenAddr = *r.ListenAddr
	}
	if err := setDuration(&c.ReadHeaderTimeout, r.ReadHeaderTimeout, "server.read_header_timeout"); err != nil {
		return err
	}
	if err := setDuration(&c.ReadTimeout, r.ReadTimeout, "server.read_timeout"); err != nil {
		return err
	}
	if err := setDuration(&c.WriteTimeout, r.WriteTimeout, "server.write_timeout"); err != nil {
		return err
	}
	if err := setDuration(&c.IdleTimeout, r.IdleTimeout, "server.idle_timeout"); err != nil {
		return err
	}
	if err := setDuration(&c.ShutdownGrace, r.ShutdownGrace, "server.shutdown_grace"); err != nil {
		return err
	}
	return nil
}

func (r *rawLauncher) applyTo(c *LauncherConfig) error {
	return setDuration(&c.CleanupPeriod, r.CleanupPeriod, "launcher.cleanup_period")
}

func (r *rawAuth) applyTo(c *AuthConfig) error {
	if err := setDuration(&c.PINTTL, r.PINTTL, "auth.pin_ttl"); err != nil {
		return err
	}
	if r.RateLimitPerIP != nil {
		c.RateLimitPerIP = *r.RateLimitPerIP
	}
	if r.RateLimitGlobal != nil {
		c.RateLimitGlobal = *r.RateLimitGlobal
	}
	if err := setDuration(&c.RateLimitWindow, r.RateLimitWindow, "auth.rate_limit_window"); err != nil {
		return err
	}
	return nil
}

func (r *rawPaths) applyTo(c *PathsConfig) {
	if r.CertDir != nil {
		c.CertDir = *r.CertDir
	}
}

func (r *rawLogging) applyTo(c *LoggingConfig) {
	if r.Level != nil {
		c.Level = *r.Level
	}
	if r.Format != nil {
		c.Format = *r.Format
	}
}

// setDuration parses s as a time.Duration and writes it to dst when
// non-nil. A blank string is treated as an explicit empty value and
// surfaces as a parse error, since every duration the config exposes
// must be positive (Validate also catches it, but we prefer an early,
// pointed error message).
func setDuration(dst *time.Duration, s *string, key string) error {
	if s == nil {
		return nil
	}
	d, err := time.ParseDuration(*s)
	if err != nil {
		return fmt.Errorf("%s: parse duration %q: %w", key, *s, err)
	}
	*dst = d
	return nil
}
