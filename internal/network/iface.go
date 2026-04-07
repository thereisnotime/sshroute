// Package network — iface.go implements the "interface" check type.
package network

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// checkInterface returns true if the named network interface exists and its
// operstate file reports "up". A missing interface is not an error — it simply
// returns false.
func checkInterface(name string) (bool, error) {
	// Reject names that would escape the /sys/class/net/ directory.
	if strings.ContainsAny(name, "/\\") || name == ".." || name == "." {
		return false, fmt.Errorf("invalid interface name: %q", name)
	}
	path := "/sys/class/net/" + name + "/operstate"
	data, err := os.ReadFile(path) //nolint:gosec // G304: path is constructed from a validated interface name under a fixed sysfs prefix
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read operstate for %q: %w", name, err)
	}
	return strings.TrimSpace(string(data)) == "up", nil
}
