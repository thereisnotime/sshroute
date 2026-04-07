// Package network provides network/VPN detection for sshroute.
package network

import "github.com/thereisnotime/sshroute/internal/config"

// Detect runs checks for each configured network in order and returns the
// name of the first active network, or "default" if none match.
func Detect(networks map[string][]config.NetworkCheck) (string, error) {
	return "default", nil // implemented by A2
}
