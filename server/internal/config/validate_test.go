package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidate_Defaults(t *testing.T) {
	c := Defaults()
	if err := c.Validate(); err != nil {
		t.Fatalf("Defaults should validate: %v", err)
	}
}

func TestValidate_Rejects(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantSub string
	}{
		{
			name:    "empty listen_addr",
			mutate:  func(c *Config) { c.Server.ListenAddr = "   " },
			wantSub: "listen_addr",
		},
		{
			name:    "zero read_header_timeout",
			mutate:  func(c *Config) { c.Server.ReadHeaderTimeout = 0 },
			wantSub: "read_header_timeout",
		},
		{
			name:    "negative read_timeout",
			mutate:  func(c *Config) { c.Server.ReadTimeout = -1 * time.Second },
			wantSub: "read_timeout",
		},
		{
			name:    "zero write_timeout",
			mutate:  func(c *Config) { c.Server.WriteTimeout = 0 },
			wantSub: "write_timeout",
		},
		{
			name:    "zero idle_timeout",
			mutate:  func(c *Config) { c.Server.IdleTimeout = 0 },
			wantSub: "idle_timeout",
		},
		{
			name:    "zero shutdown_grace",
			mutate:  func(c *Config) { c.Server.ShutdownGrace = 0 },
			wantSub: "shutdown_grace",
		},
		{
			name:    "zero cleanup_period",
			mutate:  func(c *Config) { c.Launcher.CleanupPeriod = 0 },
			wantSub: "cleanup_period",
		},
		{
			name:    "zero pin_ttl",
			mutate:  func(c *Config) { c.Auth.PINTTL = 0 },
			wantSub: "pin_ttl",
		},
		{
			name:    "zero rate_limit_window",
			mutate:  func(c *Config) { c.Auth.RateLimitWindow = 0 },
			wantSub: "rate_limit_window",
		},
		{
			name:    "zero rate_limit_per_ip",
			mutate:  func(c *Config) { c.Auth.RateLimitPerIP = 0 },
			wantSub: "rate_limit_per_ip",
		},
		{
			name:    "negative rate_limit_per_ip",
			mutate:  func(c *Config) { c.Auth.RateLimitPerIP = -1 },
			wantSub: "rate_limit_per_ip",
		},
		{
			name:    "zero rate_limit_global",
			mutate:  func(c *Config) { c.Auth.RateLimitGlobal = 0 },
			wantSub: "rate_limit_global",
		},
		{
			name: "global below per-ip",
			mutate: func(c *Config) {
				c.Auth.RateLimitPerIP = 50
				c.Auth.RateLimitGlobal = 5
			},
			wantSub: ">= auth.rate_limit_per_ip",
		},
		{
			name:    "invalid log level",
			mutate:  func(c *Config) { c.Logging.Level = "trace" },
			wantSub: "logging.level",
		},
		{
			name:    "invalid log format",
			mutate:  func(c *Config) { c.Logging.Format = "xml" },
			wantSub: "logging.format",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := Defaults()
			tc.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("Validate() error = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}

func TestValidate_LogLevelCaseInsensitive(t *testing.T) {
	c := Defaults()
	for _, lvl := range []string{"DEBUG", "Info", "WARN", "error"} {
		c.Logging.Level = lvl
		if err := c.Validate(); err != nil {
			t.Errorf("level %q should be accepted: %v", lvl, err)
		}
	}
}

func TestValidate_LogFormatCaseInsensitive(t *testing.T) {
	c := Defaults()
	for _, f := range []string{"TEXT", "Json"} {
		c.Logging.Format = f
		if err := c.Validate(); err != nil {
			t.Errorf("format %q should be accepted: %v", f, err)
		}
	}
}
