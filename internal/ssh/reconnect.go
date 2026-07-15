// Package ssh — reconnect.go supervises an ssh subprocess and reconnects on a dropped connection.
package ssh

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"
)

// RunContext executes ssh as a subprocess like Run, but bound to ctx: when ctx is
// cancelled (e.g. SIGINT/SIGTERM) the child is killed. It returns ssh's exit code.
func RunContext(ctx context.Context, argv []string) (int, error) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) // #nosec G204 -- argv[0] is RealSSH, resolved from trusted sources
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}
	return 0, nil
}

// ReconnectConfig controls the supervised reconnect loop.
type ReconnectConfig struct {
	Delay    time.Duration // wait between reconnect attempts
	MaxTries int           // cap on reconnects after a drop; 0 means retry forever
}

// AttemptFunc performs one full connect attempt and returns ssh's exit code. It is
// expected to re-detect the network and re-resolve the route on each call so route
// selection follows network changes across reconnects.
type AttemptFunc func() (int, error)

// Supervise runs attempt in a loop, reconnecting only when an attempt exits with
// SSHConnectFailure (255). Any other exit code — clean exit, auth failure, or a
// remote-command exit — stops the loop and is returned. When stop fires (before an
// attempt or during the inter-attempt delay) the loop returns the last exit code.
// MaxTries, if > 0, caps the number of reconnects following a drop.
func Supervise(attempt AttemptFunc, cfg ReconnectConfig, stop <-chan struct{}) (int, error) {
	tries := 0
	for {
		code, err := attempt()
		if err != nil {
			return code, err
		}
		if code != SSHConnectFailure {
			return code, nil
		}
		// Connection dropped. Stop if asked before scheduling another attempt.
		select {
		case <-stop:
			return code, nil
		default:
		}
		tries++
		if cfg.MaxTries > 0 && tries >= cfg.MaxTries {
			return code, nil
		}
		select {
		case <-stop:
			return code, nil
		case <-time.After(cfg.Delay):
		}
	}
}
