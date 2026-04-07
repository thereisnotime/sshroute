// Package ssh provides SSH parameter resolution and execution for sshroute.
package ssh

import "github.com/thereisnotime/sshroute/internal/config"

// Resolve merges the default SSHParams for alias with any network-specific overrides.
// Returns an error if alias is not found in cfg or has no "default" profile.
func Resolve(cfg *config.Config, alias, network string) (config.SSHParams, error) {
	return config.SSHParams{}, nil // implemented by A3
}
