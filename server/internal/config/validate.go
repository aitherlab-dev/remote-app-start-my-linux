package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate rejects obviously broken configurations — zero or negative
// timeouts, empty listen addresses, empty rate-limit buckets — before
// the binary starts wiring up networking or auth state. It is run as
// the final step of Load so that earlier layers (file, env, flags)
// can still see "unset" values without racing the check.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Server.ListenAddr) == "" {
		return fmt.Errorf("server.listen_addr must not be empty")
	}
	if err := positiveDuration("server.read_header_timeout", c.Server.ReadHeaderTimeout); err != nil {
		return err
	}
	if err := positiveDuration("server.read_timeout", c.Server.ReadTimeout); err != nil {
		return err
	}
	if err := positiveDuration("server.write_timeout", c.Server.WriteTimeout); err != nil {
		return err
	}
	if err := positiveDuration("server.idle_timeout", c.Server.IdleTimeout); err != nil {
		return err
	}
	if err := positiveDuration("server.shutdown_grace", c.Server.ShutdownGrace); err != nil {
		return err
	}
	if err := positiveDuration("launcher.cleanup_period", c.Launcher.CleanupPeriod); err != nil {
		return err
	}
	if err := positiveDuration("auth.pin_ttl", c.Auth.PINTTL); err != nil {
		return err
	}
	if err := positiveDuration("auth.rate_limit_window", c.Auth.RateLimitWindow); err != nil {
		return err
	}
	if c.Auth.RateLimitPerIP <= 0 {
		return fmt.Errorf("auth.rate_limit_per_ip must be > 0, got %d", c.Auth.RateLimitPerIP)
	}
	if c.Auth.RateLimitGlobal <= 0 {
		return fmt.Errorf("auth.rate_limit_global must be > 0, got %d", c.Auth.RateLimitGlobal)
	}
	if c.Auth.RateLimitGlobal < c.Auth.RateLimitPerIP {
		return fmt.Errorf("auth.rate_limit_global (%d) must be >= auth.rate_limit_per_ip (%d)",
			c.Auth.RateLimitGlobal, c.Auth.RateLimitPerIP)
	}
	if err := validateLogLevel(c.Logging.Level); err != nil {
		return err
	}
	if err := validateLogFormat(c.Logging.Format); err != nil {
		return err
	}
	return nil
}

func positiveDuration(key string, v time.Duration) error {
	if v <= 0 {
		return fmt.Errorf("%s must be > 0, got %s", key, v)
	}
	return nil
}

func validateLogLevel(level string) error {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("logging.level must be one of debug|info|warn|error, got %q", level)
	}
}

func validateLogFormat(format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text", "json":
		return nil
	default:
		return fmt.Errorf("logging.format must be one of text|json, got %q", format)
	}
}
