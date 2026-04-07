package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	Version = "v1.2.3"
	Commit = "abc1234"
	Date = "2026-04-07"

	s := String()

	for _, want := range []string{"v1.2.3", "abc1234", "2026-04-07", "sshroute"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() = %q, missing %q", s, want)
		}
	}
}

func TestString_Defaults(t *testing.T) {
	Version = "dev"
	Commit = "none"
	Date = "unknown"

	s := String()
	if !strings.Contains(s, "dev") {
		t.Errorf("String() = %q, missing default version", s)
	}
}
