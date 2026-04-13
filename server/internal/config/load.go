package config

import (
	"os"
	"path/filepath"
)

// DefaultConfigPath returns the XDG-preferred location the server will
// look for config.toml in when no --config flag is supplied. It does
// not check whether the file actually exists — that is LoadFile's job
// and a missing file is not an error.
func DefaultConfigPath() string {
	return defaultConfigPathFrom(os.Getenv, os.UserHomeDir)
}

func defaultConfigPathFrom(getenv func(string) string, userHome func() (string, error)) string {
	base := getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := userHome()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "remotelauncher", "config.toml")
}

// Load runs the full merge chain in one call: defaults → file → env →
// flags → Validate. It is the entry point main.go wires up during
// startup. The --config flag, if present in args, overrides the XDG
// default before the file layer is read.
func Load(args []string) (*Config, error) {
	overrides, err := parseFlags(args, os.Stderr)
	if err != nil {
		return nil, err
	}

	cfg := Defaults()

	path := DefaultConfigPath()
	if overrides.configPath != nil {
		path = *overrides.configPath
	}
	if err := LoadFile(&cfg, path); err != nil {
		return nil, err
	}

	if err := cfg.ApplyEnv(); err != nil {
		return nil, err
	}

	cfg.applyFlagOverrides(overrides)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
