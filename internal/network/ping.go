// Package network — ping.go implements the "ping" check type.
package network

import "time"

// checkPing returns true if the given host responds to an ICMP echo within timeout.
func checkPing(host string, timeout time.Duration) (bool, error) { return false, nil } // implemented by A2
