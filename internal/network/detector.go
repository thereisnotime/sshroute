// Package network provides network/VPN detection for sshroute.
package network

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/thereisnotime/sshroute/internal/config"
)

const defaultPingTimeout = 2 * time.Second

// Detect runs checks for each configured network in the order they appear in
// the map. All checks for a network must pass (AND logic). Returns the name of
// the first matching network, or "default" if none match.
func Detect(networks map[string][]config.NetworkCheck) (string, error) {
	for name, checks := range networks {
		matched, err := runChecks(name, checks)
		if err != nil {
			return "", fmt.Errorf("network %q: %w", name, err)
		}
		if matched {
			slog.Debug("network matched", "network", name)
			return name, nil
		}
	}
	slog.Debug("no network matched, using default")
	return "default", nil
}

// runChecks executes every check in the slice for one network. Returns true
// only when every check passes.
func runChecks(name string, checks []config.NetworkCheck) (bool, error) {
	for i, c := range checks {
		ok, err := runCheck(c)
		if err != nil {
			return false, fmt.Errorf("check[%d] type=%s: %w", i, c.Type, err)
		}
		slog.Debug("check result",
			"network", name,
			"index", i,
			"type", c.Type,
			"passed", ok,
		)
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// runCheck dispatches a single NetworkCheck to the appropriate implementation.
func runCheck(c config.NetworkCheck) (bool, error) {
	switch c.Type {
	case config.CheckTypeRoute:
		return checkRoute(c.Match)

	case config.CheckTypeInterface:
		return checkInterface(c.Match)

	case config.CheckTypePing:
		timeout := defaultPingTimeout
		if c.Timeout != "" {
			d, err := time.ParseDuration(c.Timeout)
			if err != nil {
				return false, fmt.Errorf("invalid timeout %q: %w", c.Timeout, err)
			}
			timeout = d
		}
		return checkPing(c.Host, timeout)

	case config.CheckTypeExec:
		return checkExec(c.Command)

	default:
		return false, fmt.Errorf("unknown check type %q", c.Type)
	}
}
