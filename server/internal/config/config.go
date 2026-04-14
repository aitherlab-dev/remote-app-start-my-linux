// Package config centralises all tunables the server binary reads at
// startup. Values come from four sources, in increasing priority:
// compiled-in defaults, an optional TOML file, environment variables
// (REMOTELAUNCHER_*), and command-line flags. Only sources that
// explicitly set a value override earlier layers — missing keys or
// blank env vars are left alone.
package config

import "time"

// Config holds every runtime knob the server honours. Fields are kept
// grouped by subsystem so the TOML file reads like a table-per-area
// document and main.go picks them up without having to thread dozens
// of arguments through its constructors.
type Config struct {
	Server    ServerConfig
	Launcher  LauncherConfig
	Auth      AuthConfig
	Web       WebConfig
	Paths     PathsConfig
	Logging   LoggingConfig
	IconTheme string
}

// ServerConfig mirrors the fields of net/http.Server that we expose.
// ListenAddr takes host:port form (":8443" to bind all interfaces).
type ServerConfig struct {
	ListenAddr        string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownGrace     time.Duration
}

// LauncherConfig configures the process-tracking subsystem.
// CleanupPeriod is how often the tracker reaps dead PIDs.
type LauncherConfig struct {
	CleanupPeriod time.Duration
}

// AuthConfig holds every knob the pairing and rate-limiting layer
// honours. Rate limits are applied to the /api/pair endpoint only;
// the rest of the API is guarded by Bearer tokens.
type AuthConfig struct {
	PINTTL          time.Duration
	RateLimitPerIP  int
	RateLimitGlobal int
	RateLimitWindow time.Duration
}

// WebConfig configures the local admin UI served on a separate HTTP
// listener. The UI is plain HTTP (no TLS) and only binds to a loopback
// address so a browser on the same machine as the server can open it
// without the self-signed certificate warning that the main :8443
// endpoint produces. ListenAddr is validated to refuse any non-loopback
// host when Enabled is true — exposing the admin UI over the network
// is a footgun the config layer refuses outright.
type WebConfig struct {
	Enabled    bool
	ListenAddr string
}

// PathsConfig lets operators relocate filesystem state away from
// $XDG_CONFIG_HOME. CertDir blank means "use the XDG default",
// matching the historic behaviour of the binary. ConfigFile is the
// path the loader actually read (empty if no file was loaded) and is
// populated for logging/inspection only — it is not honoured as an
// input.
type PathsConfig struct {
	CertDir    string
	ConfigFile string
}

// LoggingConfig is pre-wired for S5.2. The binary already uses slog
// with a text handler at INFO; the fields here give the config file a
// place to hold the values without the binary reading them yet.
type LoggingConfig struct {
	Level  string
	Format string
}
