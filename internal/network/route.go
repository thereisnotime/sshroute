// Package network — route.go implements the "route" check type.
package network

import (
	"fmt"
	"os/exec"
	"strings"
)

// checkRoute returns true if match appears anywhere in the output of
// `ip route show`. A non-zero exit from ip is treated as a hard error.
func checkRoute(match string) (bool, error) {
	out, err := exec.Command("ip", "route", "show").Output()
	if err != nil {
		return false, fmt.Errorf("ip route show: %w", err)
	}
	return strings.Contains(string(out), match), nil
}
