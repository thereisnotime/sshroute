// Package network — exec.go implements the "exec" check type.
package network

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const execTimeout = 5 * time.Second

// checkExec runs command via `sh -c <command>` and returns true if the exit
// code is 0. Stderr is discarded so failed probes don't pollute the terminal.
func checkExec(command string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command) // #nosec G204 -- user-defined network check command; executing arbitrary shell commands is the explicit purpose of this check type
	cmd.Stderr = openDevNull()

	err := cmd.Run()
	if err == nil {
		return true, nil
	}

	// A non-zero exit is a normal "check failed" — not a hard error.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}

	// Context deadline exceeded means the command timed out.
	if ctx.Err() != nil {
		return false, nil
	}

	return false, fmt.Errorf("exec check: %w", err)
}

// openDevNull returns an *os.File pointing at /dev/null, or nil if it cannot
// be opened (in which case the OS default applies).
func openDevNull() *os.File {
	f, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	return f
}
