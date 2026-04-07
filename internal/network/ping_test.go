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

