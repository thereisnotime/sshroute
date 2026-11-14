// Package config — validator checks the config structure for correctness.
package config

import (
	"errors"
	"fmt"
)

// Validate returns a joined error describing every problem found in cfg.
// It checks:
//   - Every host has a "default" profile with at least a Host field set.
//   - Port (when non-zero) is in the range 1–65535 for every profile.
//   - Every NetworkCheck has a known Type (route/interface/ping/exec).
//   - route and interface checks have a non-empty Match field.
//   - ping checks have a non-empty Host field.
//   - exec checks have a non-empty Command field.
//   - cache.ttl and cache.lock_timeout (when set) parse as durations.
//   - cache.scope (when set) is "network" or "alias".
func Validate(cfg *Config) error {
	var errs []error

	for networkName, def := range cfg.Networks {
		for i, check := range def.Checks {
			prefix := fmt.Sprintf("network %q check[%d]", networkName, i)
			if err := validateNetworkCheck(prefix, check); err != nil {
				errs = append(errs, err)
			}
		}
	}

	for hostName, hostCfg := range cfg.Hosts {
		errs = append(errs, validateHost(hostName, hostCfg)...)
	}

	errs = append(errs, validateCache(cfg.Cache)...)

	return errors.Join(errs...)
}

func validateNetworkCheck(prefix string, check NetworkCheck) error {
	switch check.Type {
	case CheckTypeRoute:
		if check.Match == "" {
			return fmt.Errorf("%s: route check requires a non-empty match field", prefix)
		}
	case CheckTypeInterface:
		if check.Match == "" {
			return fmt.Errorf("%s: interface check requires a non-empty match field", prefix)
		}
	case CheckTypePing:
		if check.Host == "" {
			return fmt.Errorf("%s: ping check requires a non-empty host field", prefix)
		}
	case CheckTypeExec:
		if check.Command == "" {
			return fmt.Errorf("%s: exec check requires a non-empty command field", prefix)
		}
	case "":
		return fmt.Errorf("%s: type field is required", prefix)
	default:
		return fmt.Errorf("%s: unknown type %q (must be route, interface, ping, or exec)", prefix, check.Type)
	}
	return nil
}

func validateHost(hostName string, hostCfg HostConfig) []error {
	var errs []error

	defaultProfile, hasDefault := hostCfg["default"]
	if !hasDefault {
		errs = append(errs, fmt.Errorf("host %q: missing required \"default\" profile", hostName))
	} else if defaultProfile.Host == "" {
		errs = append(errs, fmt.Errorf("host %q: default profile must have a non-empty host field", hostName))
	}

	for profileName, params := range hostCfg {
		if params.Port != 0 && (params.Port < 1 || params.Port > 65535) {
			errs = append(errs, fmt.Errorf("host %q profile %q: port %d is out of range (must be 1–65535)", hostName, profileName, params.Port))
		}
	}

	return errs
}

func validateCache(c CacheConfig) []error {
	var errs []error
	if c.TTL != "" {
		if _, err := ParseDuration(c.TTL); err != nil {
			errs = append(errs, fmt.Errorf("cache.ttl: %w", err))
		}
	}
	if c.LockTimeout != "" {
		if _, err := ParseDuration(c.LockTimeout); err != nil {
			errs = append(errs, fmt.Errorf("cache.lock_timeout: %w", err))
		}
	}
	switch c.Scope {
	case "", CacheScopeNetwork, CacheScopeAlias:
	default:
		errs = append(errs, fmt.Errorf("cache.scope: unknown scope %q (must be network or alias)", c.Scope))
	}
	return errs
}
