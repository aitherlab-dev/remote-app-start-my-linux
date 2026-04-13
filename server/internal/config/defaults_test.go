package config

import (
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	c := Defaults()

	if got, want := c.Server.ListenAddr, ":8443"; got != want {
		t.Errorf("Server.ListenAddr = %q, want %q", got, want)
	}
	if got, want := c.Server.ReadHeaderTimeout, 5*time.Second; got != want {
		t.Errorf("Server.ReadHeaderTimeout = %s, want %s", got, want)
	}
	if got, want := c.Server.ReadTimeout, 30*time.Second; got != want {
		t.Errorf("Server.ReadTimeout = %s, want %s", got, want)
	}
	if got, want := c.Server.WriteTimeout, 30*time.Second; got != want {
		t.Errorf("Server.WriteTimeout = %s, want %s", got, want)
	}
	if got, want := c.Server.IdleTimeout, 120*time.Second; got != want {
		t.Errorf("Server.IdleTimeout = %s, want %s", got, want)
	}
	if got, want := c.Server.ShutdownGrace, 10*time.Second; got != want {
		t.Errorf("Server.ShutdownGrace = %s, want %s", got, want)
	}
	if got, want := c.Launcher.CleanupPeriod, 5*time.Second; got != want {
		t.Errorf("Launcher.CleanupPeriod = %s, want %s", got, want)
	}
	if got, want := c.Auth.PINTTL, 10*time.Minute; got != want {
		t.Errorf("Auth.PINTTL = %s, want %s", got, want)
	}
	if got, want := c.Auth.RateLimitPerIP, 5; got != want {
		t.Errorf("Auth.RateLimitPerIP = %d, want %d", got, want)
	}
	if got, want := c.Auth.RateLimitGlobal, 20; got != want {
		t.Errorf("Auth.RateLimitGlobal = %d, want %d", got, want)
	}
	if got, want := c.Auth.RateLimitWindow, 10*time.Minute; got != want {
		t.Errorf("Auth.RateLimitWindow = %s, want %s", got, want)
	}
	if got, want := c.Paths.CertDir, ""; got != want {
		t.Errorf("Paths.CertDir = %q, want %q", got, want)
	}
	if got, want := c.Paths.ConfigFile, ""; got != want {
		t.Errorf("Paths.ConfigFile = %q, want %q", got, want)
	}
	if got, want := c.Logging.Level, "info"; got != want {
		t.Errorf("Logging.Level = %q, want %q", got, want)
	}
	if got, want := c.Logging.Format, "text"; got != want {
		t.Errorf("Logging.Format = %q, want %q", got, want)
	}
	if got, want := c.IconTheme, ""; got != want {
		t.Errorf("IconTheme = %q, want %q", got, want)
	}

	if err := c.Validate(); err != nil {
		t.Errorf("Defaults().Validate() = %v, want nil", err)
	}
}
