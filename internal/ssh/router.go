// Package ssh provides SSH parameter resolution and execution for sshroute.
package ssh

import (
	"fmt"

	"github.com/thereisnotime/sshroute/internal/config"
)

// Resolve merges the default SSHParams for alias with any network-specific overrides.
// Returns an error if alias is not found in cfg or has no "default" profile.
func Resolve(cfg *config.Config, alias, network string) (config.SSHParams, error) {
	hostConfig, ok := cfg.Hosts[alias]
	if !ok {
		return config.SSHParams{}, fmt.Errorf("resolve %q: host not found in config", alias)
	}

	defaults, ok := hostConfig["default"]
	if !ok {
		return config.SSHParams{}, fmt.Errorf("resolve %q: missing required \"default\" profile", alias)
	}

	// Start with a copy of the default profile.
	merged := defaults

	// If caller wants the default network, we're done.
	if network == "" || network == "default" {
		return merged, nil
	}

	override, ok := hostConfig[network]
	if !ok {
		// Network profile not defined for this host — fall back to default silently.
		return merged, nil
	}

	// Apply non-zero / non-empty fields from the network profile.
	if override.Host != "" {
		merged.Host = override.Host
	}
	if override.Port != 0 {
		merged.Port = override.Port
	}
	if override.User != "" {
		merged.User = override.User
	}
	if override.Key != "" {
		merged.Key = override.Key
	}
	if override.Jump != "" {
		merged.Jump = override.Jump
	}

	return merged, nil
}
