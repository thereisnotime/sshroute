package ssh

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSupervise(t *testing.T) {
	t.Run("reconnects on 255 until a clean exit", func(t *testing.T) {
		codes := []int{SSHConnectFailure, SSHConnectFailure, 0}
		calls := 0
		attempt := func() (int, error) {
			c := codes[calls]
			calls++
			return c, nil
		}
		code, err := Supervise(attempt, ReconnectConfig{Delay: time.Millisecond}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != 0 {
			t.Errorf("code = %d, want 0", code)
		}
		if calls != 3 {
			t.Errorf("attempts = %d, want 3", calls)
		}
	})

	t.Run("stops immediately on a non-255 exit", func(t *testing.T) {
		calls := 0
		attempt := func() (int, error) {
			calls++
			return 42, nil
		}
		code, err := Supervise(attempt, ReconnectConfig{Delay: time.Millisecond}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != 42 {
			t.Errorf("code = %d, want 42", code)
		}
		if calls != 1 {
			t.Errorf("attempts = %d, want 1", calls)
		}
	})

	t.Run("honours MaxTries", func(t *testing.T) {
		calls := 0
		attempt := func() (int, error) {
			calls++
			return SSHConnectFailure, nil
		}
		code, err := Supervise(attempt, ReconnectConfig{Delay: time.Millisecond, MaxTries: 3}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != SSHConnectFailure {
			t.Errorf("code = %d, want %d", code, SSHConnectFailure)
		}
		if calls != 3 {
			t.Errorf("attempts = %d, want 3", calls)
		}
	})

	t.Run("stop channel breaks the loop", func(t *testing.T) {
		stop := make(chan struct{})
		close(stop)
		calls := 0
		attempt := func() (int, error) {
			calls++
			return SSHConnectFailure, nil
		}
		// Delay is long enough that the timer would never fire on its own; the closed
		// stop channel must be what ends the loop after the first drop.
		code, err := Supervise(attempt, ReconnectConfig{Delay: time.Hour}, stop)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != SSHConnectFailure {
			t.Errorf("code = %d, want %d", code, SSHConnectFailure)
		}
		if calls != 1 {
			t.Errorf("attempts = %d, want 1", calls)
		}
	})

	t.Run("propagates attempt error", func(t *testing.T) {
		wantErr := errors.New("boom")
		attempt := func() (int, error) {
			return -1, wantErr
		}
		_, err := Supervise(attempt, ReconnectConfig{Delay: time.Millisecond}, nil)
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}

func TestRunContext(t *testing.T) {
	t.Run("exit 0 on success", func(t *testing.T) {
		code, err := RunContext(context.Background(), []string{"/bin/true"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != 0 {
			t.Errorf("code = %d, want 0", code)
		}
	})

	t.Run("non-zero exit code returned", func(t *testing.T) {
		code, err := RunContext(context.Background(), []string{"/bin/false"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code == 0 {
			t.Error("expected non-zero exit code from /bin/false")
		}
	})

	t.Run("cancelled context kills the child", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		start := time.Now()
		code, err := RunContext(ctx, []string{"/bin/sleep", "5"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if elapsed := time.Since(start); elapsed > 3*time.Second {
			t.Errorf("child not cancelled promptly, took %s", elapsed)
		}
		if code == 0 {
			t.Error("expected non-zero exit code after cancellation")
		}
	})

	t.Run("nonexistent binary returns an error", func(t *testing.T) {
		if _, err := RunContext(context.Background(), []string{"/nonexistent/definitely-not-here"}); err == nil {
			t.Error("expected an error for a missing binary")
		}
	})
}
