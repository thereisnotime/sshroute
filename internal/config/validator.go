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
func Validate(cfg *Config) error {
	var errs []error

	for networkName, def := range cfg.Networks {
		for i, check := range def.Checks {
			prefix := fmt.Sprintf("network %q check[%d]", networkName, i)

			switch check.Type {
			case CheckTypeRoute:
				if check.Match == "" {
					errs = append(errs, fmt.Errorf("%s: route check requires a non-empty match field", prefix))
				}
			case CheckTypeInterface:
				if check.Match == "" {
					errs = append(errs, fmt.Errorf("%s: interface check requires a non-empty match field", prefix))
				}
			case CheckTypePing:
				if check.Host == "" {
					errs = append(errs, fmt.Errorf("%s: ping check requires a non-empty host field", prefix))
				}
			case CheckTypeExec:
				if check.Command == "" {
					errs = append(errs, fmt.Errorf("%s: exec check requires a non-empty command field", prefix))
				}
			case "":
				errs = append(errs, fmt.Errorf("%s: type field is required", prefix))
			default:
				errs = append(errs, fmt.Errorf("%s: unknown type %q (must be route, interface, ping, or exec)", prefix, check.Type))
			}
		}
	}

	for hostName, hostCfg := range cfg.Hosts {
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
	}

	return errors.Join(errs...)
}
