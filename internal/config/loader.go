// Package config — loader handles reading and writing the YAML config file.
package config

// DefaultConfigPath returns the XDG-aware config file path.
// Expands ~ to the user home directory.
func DefaultConfigPath() (string, error) { return "", nil } // implemented by A1

// Load reads and parses the config file at the given path.
func Load(path string) (*Config, error) { return &Config{}, nil } // implemented by A1

// Save writes the config to the given path, creating directories as needed.
func Save(path string, cfg *Config) error { return nil } // implemented by A1
