// Package network provides network/VPN detection for sshroute.
package network

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/thereisnotime/sshroute/internal/config"
)

const defaultPingTimeout = 2 * time.Second

// networkEntry pairs a network name with its definition for sorted iteration.
type networkEntry struct {
	name string
	def  config.NetworkDefinition
}

// Detect runs checks for each configured network in priority order (lowest
// priority value first). All checks for a network must pass (AND logic).
// Returns the name of the first matching network, or "default" if none match.
func Detect(networks map[string]config.NetworkDefinition) (string, error) {
	entries := make([]networkEntry, 0, len(networks))
	for name, def := range networks {
		entries = append(entries, networkEntry{name: name, def: def})
	}
	// Sort: lower Priority value = evaluated first. Tie-break alphabetically.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].def.Priority != entries[j].def.Priority {
			return entries[i].def.Priority < entries[j].def.Priority
		}
		return entries[i].name < entries[j].name
	})

	for _, e := range entries {
		matched, err := runChecks(e.name, e.def.Checks)
		if err != nil {
			return "", fmt.Errorf("network %q: %w", e.name, err)
		}
		if matched {
			slog.Debug("network matched", "network", e.name, "priority", e.def.Priority)
			return e.name, nil
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
