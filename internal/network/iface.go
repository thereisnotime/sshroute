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
	path := "/sys/class/net/" + name + "/operstate"
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read operstate for %q: %w", name, err)
	}
	return strings.TrimSpace(string(data)) == "up", nil
}
