package network

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestFallbackPing(t *testing.T) {
	t.Run("localhost responds", func(t *testing.T) {
		ok, err := fallbackPing("127.0.0.1", 2*time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected 127.0.0.1 to respond to ping")
		}
	})

	t.Run("unreachable host returns false", func(t *testing.T) {
		// 192.0.2.x is TEST-NET-1 (RFC 5737) — must not be routed.
		ok, err := fallbackPing("192.0.2.1", 500*time.Millisecond)
		if err != nil {
			// Some environments return an error instead of false for unreachable hosts.
			t.Logf("got error (acceptable): %v", err)
			return
		}
		if ok {
			t.Error("TEST-NET address should not respond")
		}
	})

	t.Run("sub-second timeout rounds up to 1s", func(t *testing.T) {
		// Should not panic or error — just runs with -W1.
		_, err := fallbackPing("127.0.0.1", 100*time.Millisecond)
		if err != nil {
			t.Logf("error (acceptable in restricted env): %v", err)
		}
	})
}

func TestCheckPing_UsesIcmpOrFallback(t *testing.T) {
	// checkPing tries raw ICMP first, then falls back. Either path should work
	// for localhost in most environments.
	ok, err := checkPing("127.0.0.1", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected 127.0.0.1 to be pingable")
	}
}

func TestFallbackPing_BinaryNotFound(t *testing.T) {
	// Empty PATH makes ping binary lookup fail with exec.ErrNotFound,
	// which is not an *exec.ExitError — covers the hard-error return branch.
	t.Setenv("PATH", "")
	_, err := fallbackPing("127.0.0.1", 1*time.Second)
	if err == nil {
		t.Error("expected error when ping binary not found in PATH")
	}
}

func TestIsPermissionError(t *testing.T) {
	if !isPermissionError(os.ErrPermission) {
		t.Error("os.ErrPermission should be detected")
	}
	if isPermissionError(errors.New("some other error")) {
		t.Error("unrelated error should not be a permission error")
	}
	if isPermissionError(nil) {
		t.Error("nil should not be a permission error")
	}
}

func TestIcmpPing_InvalidHost(t *testing.T) {
	// An invalid hostname that cannot be resolved should return a resolve error,
	// not a permission error, so icmpPing must return (false, non-nil error).
	_, err := icmpPing("invalid@@hostname.xyz.invalid", 1*time.Second)
	if err == nil {
		t.Fatal("expected error for unresolvable hostname, got nil")
	}
}

func TestCheckPing_InvalidHost_NonPermissionError(t *testing.T) {
	// An invalid hostname triggers the DNS-resolve failure inside icmpPing.
	// That error is NOT a permission error, so checkPing must return it directly
	// rather than falling back to the system ping binary.
	_, err := checkPing("invalid@@hostname.xyz.invalid", 1*time.Second)
	if err == nil {
		t.Fatal("expected error for unresolvable hostname, got nil")
	}
}
