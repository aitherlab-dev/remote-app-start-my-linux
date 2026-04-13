package config

import "time"

// Defaults returns a Config populated with the values that the server
// binary used to carry as const(...) block in main.go. Any layer that
// does not explicitly override a field inherits it from here, so the
// binary launched with no config file and no flags behaves exactly as
// it did before S5.1.
func Defaults() Config {
	return Config{
		Server: ServerConfig{
			ListenAddr:        ":8443",
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
			ShutdownGrace:     10 * time.Second,
		},
		Launcher: LauncherConfig{
			CleanupPeriod: 5 * time.Second,
		},
		Auth: AuthConfig{
			PINTTL:          10 * time.Minute,
			RateLimitPerIP:  5,
			RateLimitGlobal: 20,
			RateLimitWindow: 10 * time.Minute,
		},
		Paths: PathsConfig{
			CertDir:    "",
			ConfigFile: "",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		IconTheme: "",
	}
}
