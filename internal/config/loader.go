// Package config — loader handles reading and writing the YAML config file.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPath returns the XDG-aware config file path for sshroute.
// Resolution order:
//  1. $SSHROUTE_CONFIG env var
//  2. $XDG_CONFIG_HOME/sshroute/config.yaml
//  3. ~/.config/sshroute/config.yaml
//
// Tildes in the result are expanded to the user home directory.
func DefaultConfigPath() (string, error) {
	if v := os.Getenv("SSHROUTE_CONFIG"); v != "" {
		expanded, err := expandTilde(v)
		if err != nil {
			return "", fmt.Errorf("expanding SSHROUTE_CONFIG: %w", err)
		}
		slog.Debug("config path from SSHROUTE_CONFIG", "path", expanded) // #nosec G115 G706 -- slog uses structured key-value pairs, not format strings; no injection risk
		return expanded, nil
	}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		expanded, err := expandTilde(xdg)
		if err != nil {
			return "", fmt.Errorf("expanding XDG_CONFIG_HOME: %w", err)
		}
		p := filepath.Join(expanded, "sshroute", "config.yaml")
		slog.Debug("config path from XDG_CONFIG_HOME", "path", p) // #nosec G115 G706 -- slog uses structured key-value pairs, not format strings; no injection risk
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	p := filepath.Join(home, ".config", "sshroute", "config.yaml")
	slog.Debug("config path from default", "path", p)
	return p, nil
}

// Load reads and parses the config file at path.
// If path is empty, DefaultConfigPath is used.
// If the file does not exist, an empty Config is returned without error.
// After parsing, Validate is called and all Key fields are tilde-expanded.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, fmt.Errorf("resolving config path: %w", err)
		}
	}

	slog.Debug("loading config", "path", path)

	data, err := os.ReadFile(path) // #nosec G304 -- path is the user's own config file, explicitly provided or resolved from trusted XDG/home locations
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("config file not found, using empty config", "path", path)
			return &Config{
				Networks: make(map[string]NetworkDefinition),
				Hosts:    make(map[string]HostConfig),
			}, nil
		}
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	// Ensure top-level maps are never nil.
	if cfg.Networks == nil {
		cfg.Networks = make(map[string]NetworkDefinition)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = make(map[string]HostConfig)
	}

	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}

	// Expand ~ in all Key fields after validation so validators see raw values.
	if err := expandKeyTildes(&cfg); err != nil {
		return nil, fmt.Errorf("expanding key paths in config %q: %w", path, err)
	}

	slog.Debug("config loaded successfully", "path", path,
		"networks", len(cfg.Networks), "hosts", len(cfg.Hosts))
	return &cfg, nil
}

// Save marshals cfg to YAML and writes it atomically to path.
// If path is empty, DefaultConfigPath is used.
// Parent directories are created with mode 0700; the file is written with mode 0600.
func Save(path string, cfg *Config) error {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("resolving config path: %w", err)
		}
	}

	slog.Debug("saving config", "path", path)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory %q: %w", dir, err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Write atomically: temp file in the same directory, then rename.
	tmp, err := os.CreateTemp(dir, ".sshroute-config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file in %q: %w", dir, err)
	}
	tmpName := tmp.Name()

	// Clean up the temp file on any error path.
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("setting permissions on temp file: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing config to temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp file to %q: %w", path, err)
	}

	committed = true
	slog.Debug("config saved successfully", "path", path)
	return nil
}

// expandTilde replaces a leading ~ with the user's home directory.
func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, path[1:]), nil
}

// expandKeyTildes expands ~ in all SSHParams.Key fields in-place.
func expandKeyTildes(cfg *Config) error {
	for hostName, hostCfg := range cfg.Hosts {
		for profileName, params := range hostCfg {
			if params.Key == "" {
				continue
			}
			expanded, err := expandTilde(params.Key)
			if err != nil {
				return fmt.Errorf("host %q profile %q key: %w", hostName, profileName, err)
			}
			params.Key = expanded
			hostCfg[profileName] = params
		}
	}
	return nil
}
