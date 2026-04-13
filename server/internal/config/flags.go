package config

import (
	"flag"
	"fmt"
	"io"
	"time"
)

// flagOverrides records values that were explicitly set on the command
// line. Any field that remains nil was not typed by the operator and
// so must not clobber the preceding merge layer (defaults → file →
// env). Pointer semantics make the "was it set?" check trivial via
// flag.FlagSet.Visit.
type flagOverrides struct {
	configPath        *string
	listenAddr        *string
	readHeaderTimeout *time.Duration
	readTimeout       *time.Duration
	writeTimeout      *time.Duration
	idleTimeout       *time.Duration
	shutdownGrace     *time.Duration
	cleanupPeriod     *time.Duration
	pinTTL            *time.Duration
	rateLimitPerIP    *int
	rateLimitGlobal   *int
	rateLimitWindow   *time.Duration
	certDir           *string
	iconTheme         *string
	logLevel          *string
	logFormat         *string
}

// ApplyFlags parses args as command-line flags and applies only the
// ones that were explicitly set to c. A missing flag leaves the
// corresponding field untouched — that's what allows the merge chain
// (defaults → file → env → flags) to be monotone. errOut receives any
// usage messages the underlying flag.FlagSet would print.
func (c *Config) ApplyFlags(args []string) error {
	overrides, err := parseFlags(args, io.Discard)
	if err != nil {
		return err
	}
	c.applyFlagOverrides(overrides)
	return nil
}

// parseFlags builds a fresh FlagSet each call so there is no hidden
// global state between tests. The FlagSet is in ContinueOnError mode,
// which makes flag.Parse surface -h / unknown flags as regular errors
// rather than os.Exit(2).
func parseFlags(args []string, errOut io.Writer) (flagOverrides, error) {
	fs := flag.NewFlagSet("remotelauncher", flag.ContinueOnError)
	fs.SetOutput(errOut)

	var (
		configPath        string
		listenAddr        string
		readHeaderTimeout time.Duration
		readTimeout       time.Duration
		writeTimeout      time.Duration
		idleTimeout       time.Duration
		shutdownGrace     time.Duration
		cleanupPeriod     time.Duration
		pinTTL            time.Duration
		rateLimitPerIP    int
		rateLimitGlobal   int
		rateLimitWindow   time.Duration
		certDir           string
		iconTheme         string
		logLevel          string
		logFormat         string
	)

	fs.StringVar(&configPath, "config", "", "path to the TOML config file")
	fs.StringVar(&listenAddr, "listen", "", "address to bind the HTTPS server to, host:port")
	fs.DurationVar(&readHeaderTimeout, "read-header-timeout", 0, "HTTP ReadHeaderTimeout")
	fs.DurationVar(&readTimeout, "read-timeout", 0, "HTTP ReadTimeout")
	fs.DurationVar(&writeTimeout, "write-timeout", 0, "HTTP WriteTimeout")
	fs.DurationVar(&idleTimeout, "idle-timeout", 0, "HTTP IdleTimeout")
	fs.DurationVar(&shutdownGrace, "shutdown-grace", 0, "graceful shutdown window on SIGTERM/SIGINT")
	fs.DurationVar(&cleanupPeriod, "cleanup-period", 0, "how often the launcher reaps dead PIDs")
	fs.DurationVar(&pinTTL, "pin-ttl", 0, "lifetime of the pairing PIN")
	fs.IntVar(&rateLimitPerIP, "pair-rate-per-ip", 0, "max /api/pair attempts per IP per window")
	fs.IntVar(&rateLimitGlobal, "pair-rate-global", 0, "max /api/pair attempts across all clients per window")
	fs.DurationVar(&rateLimitWindow, "pair-rate-window", 0, "rate-limit window for /api/pair")
	fs.StringVar(&certDir, "cert-dir", "", "override for the TLS cert/key directory")
	fs.StringVar(&iconTheme, "icon-theme", "", "XDG icon theme name to prefer when serving icons")
	fs.StringVar(&logLevel, "log-level", "", "slog level: debug | info | warn | error")
	fs.StringVar(&logFormat, "log-format", "", "slog handler format: text | json")

	if err := fs.Parse(args); err != nil {
		return flagOverrides{}, fmt.Errorf("parse flags: %w", err)
	}

	var o flagOverrides
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "config":
			v := configPath
			o.configPath = &v
		case "listen":
			v := listenAddr
			o.listenAddr = &v
		case "read-header-timeout":
			v := readHeaderTimeout
			o.readHeaderTimeout = &v
		case "read-timeout":
			v := readTimeout
			o.readTimeout = &v
		case "write-timeout":
			v := writeTimeout
			o.writeTimeout = &v
		case "idle-timeout":
			v := idleTimeout
			o.idleTimeout = &v
		case "shutdown-grace":
			v := shutdownGrace
			o.shutdownGrace = &v
		case "cleanup-period":
			v := cleanupPeriod
			o.cleanupPeriod = &v
		case "pin-ttl":
			v := pinTTL
			o.pinTTL = &v
		case "pair-rate-per-ip":
			v := rateLimitPerIP
			o.rateLimitPerIP = &v
		case "pair-rate-global":
			v := rateLimitGlobal
			o.rateLimitGlobal = &v
		case "pair-rate-window":
			v := rateLimitWindow
			o.rateLimitWindow = &v
		case "cert-dir":
			v := certDir
			o.certDir = &v
		case "icon-theme":
			v := iconTheme
			o.iconTheme = &v
		case "log-level":
			v := logLevel
			o.logLevel = &v
		case "log-format":
			v := logFormat
			o.logFormat = &v
		}
	})
	return o, nil
}

func (c *Config) applyFlagOverrides(o flagOverrides) {
	if o.listenAddr != nil {
		c.Server.ListenAddr = *o.listenAddr
	}
	if o.readHeaderTimeout != nil {
		c.Server.ReadHeaderTimeout = *o.readHeaderTimeout
	}
	if o.readTimeout != nil {
		c.Server.ReadTimeout = *o.readTimeout
	}
	if o.writeTimeout != nil {
		c.Server.WriteTimeout = *o.writeTimeout
	}
	if o.idleTimeout != nil {
		c.Server.IdleTimeout = *o.idleTimeout
	}
	if o.shutdownGrace != nil {
		c.Server.ShutdownGrace = *o.shutdownGrace
	}
	if o.cleanupPeriod != nil {
		c.Launcher.CleanupPeriod = *o.cleanupPeriod
	}
	if o.pinTTL != nil {
		c.Auth.PINTTL = *o.pinTTL
	}
	if o.rateLimitPerIP != nil {
		c.Auth.RateLimitPerIP = *o.rateLimitPerIP
	}
	if o.rateLimitGlobal != nil {
		c.Auth.RateLimitGlobal = *o.rateLimitGlobal
	}
	if o.rateLimitWindow != nil {
		c.Auth.RateLimitWindow = *o.rateLimitWindow
	}
	if o.certDir != nil {
		c.Paths.CertDir = *o.certDir
	}
	if o.iconTheme != nil {
		c.IconTheme = *o.iconTheme
	}
	if o.logLevel != nil {
		c.Logging.Level = *o.logLevel
	}
	if o.logFormat != nil {
		c.Logging.Format = *o.logFormat
	}
}
