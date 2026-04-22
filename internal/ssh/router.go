// Package ssh provides SSH parameter resolution and execution for sshroute.
package ssh

import (
	"fmt"

	"github.com/thereisnotime/sshroute/internal/config"
)

// Resolve merges the default SSHParams for alias with any network-specific overrides.
// If the resulting Jump value matches another host alias in cfg, it is resolved
// recursively and stored in ResolvedJump.
// Returns an error if alias is not found in cfg or has no "default" profile.
func Resolve(cfg *config.Config, alias, network string) (config.SSHParams, error) {
	return resolveRecursive(cfg, alias, network, nil)
}

func resolveRecursive(cfg *config.Config, alias, network string, visited map[string]bool) (config.SSHParams, error) {
	if visited == nil {
		visited = make(map[string]bool)
	}
	if visited[alias] {
		return config.SSHParams{}, fmt.Errorf("resolve %q: circular jump chain detected", alias)
	}
	visited[alias] = true

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

	// If caller wants the default network, skip override merge.
	if network != "" && network != "default" {
		if override, ok := hostConfig[network]; ok {
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
		}
	}

	// If Jump references another host alias, resolve it recursively.
	if merged.Jump != "" {
		if _, isAlias := cfg.Hosts[merged.Jump]; isAlias {
			jumpParams, err := resolveRecursive(cfg, merged.Jump, network, visited)
			if err != nil {
				return config.SSHParams{}, fmt.Errorf("resolve jump %q for %q: %w", merged.Jump, alias, err)
			}
			merged.ResolvedJump = &jumpParams
		}
	}

	return merged, nil
}
