// Package config — duration.go parses durations that also accept a day suffix.
package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// day is the length of the "d" unit accepted by ParseDuration.
const day = 24 * time.Hour

// ParseDuration parses a Go duration string ("90m", "12h") and additionally accepts
// a whole or fractional number of days with a "d" suffix ("7d", "1.5d"). A bare "0"
// is zero. Negative values are rejected.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	var d time.Duration
	if strings.HasSuffix(s, "d") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "d"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		d = time.Duration(n * float64(day))
	} else {
		var err error
		d, err = time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
	}
	if d < 0 {
		return 0, fmt.Errorf("duration %q must not be negative", s)
	}
	return d, nil
}
