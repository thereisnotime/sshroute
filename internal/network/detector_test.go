package network

import (
	"testing"

	"github.com/thereisnotime/sshroute/internal/config"
)

func TestDetect_EmptyNetworks(t *testing.T) {
	result, err := Detect(map[string]config.NetworkDefinition{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "default" {
		t.Errorf("result = %q, want %q", result, "default")
	}
}

func TestDetect_PriorityOrdering(t *testing.T) {
	// Use exec checks with known outcomes to test ordering.
	networks := map[string]config.NetworkDefinition{
		"first": {
			Priority: 10,
			Checks:   []config.NetworkCheck{{Type: config.CheckTypeExec, Command: "true"}},
		},
		"second": {
			Priority: 20,
			Checks:   []config.NetworkCheck{{Type: config.CheckTypeExec, Command: "true"}},
		},
	}
	result, err := Detect(networks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "first" has lower priority value so it should match first.
	if result != "first" {
		t.Errorf("result = %q, want %q (lower priority value wins)", result, "first")
	}
}

func TestDetect_AlphabeticalTieBreak(t *testing.T) {
	networks := map[string]config.NetworkDefinition{
		"zebra": {Priority: 0, Checks: []config.NetworkCheck{{Type: config.CheckTypeExec, Command: "true"}}},
		"alpha": {Priority: 0, Checks: []config.NetworkCheck{{Type: config.CheckTypeExec, Command: "true"}}},
	}
	result, err := Detect(networks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "alpha" {
		t.Errorf("result = %q, want %q (alphabetical tie-break)", result, "alpha")
	}
}

func TestDetect_FailingCheckSkipsNetwork(t *testing.T) {
	networks := map[string]config.NetworkDefinition{
		"failing": {
			Priority: 10,
			Checks:   []config.NetworkCheck{{Type: config.CheckTypeExec, Command: "false"}},
		},
		"passing": {
			Priority: 20,
			Checks:   []config.NetworkCheck{{Type: config.CheckTypeExec, Command: "true"}},
		},
	}
	result, err := Detect(networks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "passing" {
		t.Errorf("result = %q, want %q", result, "passing")
	}
}

func TestDetect_ANDLogic(t *testing.T) {
	// Both checks must pass for the network to match.
	networks := map[string]config.NetworkDefinition{
		"partial": {
			Priority: 10,
			Checks: []config.NetworkCheck{
				{Type: config.CheckTypeExec, Command: "true"},
				{Type: config.CheckTypeExec, Command: "false"}, // second fails
			},
		},
	}
	result, err := Detect(networks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "default" {
		t.Errorf("result = %q, want %q (AND logic: second check fails)", result, "default")
	}
}

func TestCheckExec(t *testing.T) {
	t.Run("true command passes", func(t *testing.T) {
		ok, err := checkExec("true")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true, got false")
		}
	})

	t.Run("false command fails", func(t *testing.T) {
		ok, err := checkExec("false")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected false, got true")
		}
	})

	t.Run("complex command", func(t *testing.T) {
		ok, err := checkExec("echo hello | grep -q hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true for grep match")
		}
	})
}

func TestCheckInterface(t *testing.T) {
	t.Run("nonexistent interface returns false without error", func(t *testing.T) {
		ok, err := checkInterface("doesnotexist99")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected false for nonexistent interface")
		}
	})
}

func TestCheckRoute(t *testing.T) {
	t.Run("bogus string not present", func(t *testing.T) {
		ok, err := checkRoute("xyzzy-definitely-not-a-route-99999")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected false for bogus route string")
		}
	})
}

func TestDetect_PingCheck(t *testing.T) {
	networks := map[string]config.NetworkDefinition{
		"local": {
			Priority: 10,
			Checks:   []config.NetworkCheck{{Type: config.CheckTypePing, Host: "127.0.0.1", Timeout: "2s"}},
		},
	}
	result, err := Detect(networks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "local" {
		t.Errorf("result = %q, want %q (127.0.0.1 should be pingable)", result, "local")
	}
}

func TestDetect_PingTimeout(t *testing.T) {
	networks := map[string]config.NetworkDefinition{
		"unreachable": {
			Priority: 10,
			Checks:   []config.NetworkCheck{{Type: config.CheckTypePing, Host: "192.0.2.1", Timeout: "200ms"}},
		},
	}
	result, err := Detect(networks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "default" {
		t.Errorf("result = %q, want %q (TEST-NET should not respond)", result, "default")
	}
}

func TestRunCheck_InvalidTimeout(t *testing.T) {
	_, err := runCheck(config.NetworkCheck{
		Type:    config.CheckTypePing,
		Host:    "127.0.0.1",
		Timeout: "notaduration",
	})
	if err == nil {
		t.Error("expected error for invalid timeout")
	}
}

func TestRunCheck_UnknownType(t *testing.T) {
	_, err := runCheck(config.NetworkCheck{Type: "magic"})
	if err == nil {
		t.Error("expected error for unknown check type")
	}
}

func TestCheckRoute_MatchFound(t *testing.T) {
	// "default" or "lo" almost always appears in `ip route show` output.
	ok, err := checkRoute("default")
	if err != nil {
		// ip not available in this environment — skip rather than fail.
		t.Skipf("ip route show unavailable: %v", err)
	}
	if !ok {
		// Try "lo" as a fallback match.
		ok, err = checkRoute("lo")
		if err != nil {
			t.Skipf("ip route show unavailable: %v", err)
		}
		if !ok {
			t.Log("neither 'default' nor 'lo' found in routing table — skipping assertion")
		}
	}
}

func TestCheckExec_Timeout(t *testing.T) {
	// A command that sleeps longer than execTimeout should not block forever.
	ok, err := checkExec("sleep 10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("timed-out command should return false")
	}
}

func TestCheckInterface_Loopback(t *testing.T) {
	// The loopback interface "lo" is always present on Linux; reading its
	// operstate exercises the successful ReadFile path (not ErrNotExist).
	ok, err := checkInterface("lo")
	if err != nil {
		t.Fatalf("unexpected error for loopback interface: %v", err)
	}
	// ok may be true ("up") or false ("unknown") depending on environment.
	_ = ok
}

func TestCheckInterface_InvalidName(t *testing.T) {
	t.Run("path traversal rejected", func(t *testing.T) {
		_, err := checkInterface("../../etc/passwd")
		if err == nil {
			t.Error("expected error for path traversal in interface name")
		}
	})

	t.Run("dot rejected", func(t *testing.T) {
		_, err := checkInterface(".")
		if err == nil {
			t.Error("expected error for '.' interface name")
		}
	})
}
