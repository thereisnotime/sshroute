// Package ssh — executor.go builds the final SSH argv and execs the real ssh binary.
package ssh

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/thereisnotime/sshroute/internal/config"
)

// SSHConnectFailure is the exit code ssh returns when the connection itself fails
// (timeout, refused, unreachable). Used to distinguish retryable failures from
// remote-command exits or auth failures.
const SSHConnectFailure = 255

// RealSSH is the absolute path to the system SSH binary.
// Using absolute path prevents infinite recursion in shadow/transparent mode.
const RealSSH = "/usr/bin/ssh"

// expandTilde replaces a leading ~ with the current user's home directory.
func expandTilde(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		u, err := user.Current()
		if err != nil {
			return path
		}
		return filepath.Join(u.HomeDir, path[1:])
	}
	return path
}

// BuildArgv constructs the final argv slice from resolved params and remaining args.
//
// Order of arguments:
//
//	argv[0]  = RealSSH
//	-p <port>   (if Port != 0)
//	-i <key>    (if Key != "", ~ expanded)
//	-l <user>   (params.User preferred; parsed.User used as fallback)
//	-J <jump>   (if Jump != "")
//	<params.Host>
//	<parsed.Remaining...>
func BuildArgv(params config.SSHParams, parsed ParsedArgs) []string {
	argv := []string{RealSSH}

	if params.Port != 0 {
		argv = append(argv, "-p", fmt.Sprintf("%d", params.Port))
	}

	if params.Key != "" {
		argv = append(argv, "-i", expandTilde(params.Key))
	}

	resolvedUser := params.User
	if resolvedUser == "" {
		resolvedUser = parsed.User
	}
	if resolvedUser != "" {
		argv = append(argv, "-l", resolvedUser)
	}

	if params.ResolvedJump != nil {
		argv = append(argv, "-o", "ProxyCommand="+buildProxyCommand(params.ResolvedJump))
	} else if params.Jump != "" {
		argv = append(argv, "-J", params.Jump)
	}

	argv = append(argv, params.Host)
	argv = append(argv, parsed.Remaining...)

	return argv
}

// buildProxyCommand constructs an ssh ProxyCommand string from resolved jump params.
func buildProxyCommand(jump *config.SSHParams) string {
	parts := []string{RealSSH}
	if jump.Key != "" {
		parts = append(parts, "-i", expandTilde(jump.Key))
	}
	if jump.Port != 0 {
		parts = append(parts, "-p", fmt.Sprintf("%d", jump.Port))
	}
	if jump.ResolvedJump != nil {
		nested := buildProxyCommand(jump.ResolvedJump)
		parts = append(parts, "-o", fmt.Sprintf("ProxyCommand=%s", nested))
	} else if jump.Jump != "" {
		parts = append(parts, "-J", jump.Jump)
	}
	parts = append(parts, "-W", "%h:%p")
	host := jump.Host
	if jump.User != "" {
		host = jump.User + "@" + host
	}
	parts = append(parts, host)
	return strings.Join(parts, " ")
}

// Exec replaces the current process with ssh using the given argv.
// On success this function never returns; the current process is replaced entirely.
func Exec(argv []string) error {
	if err := syscall.Exec(RealSSH, argv, os.Environ()); err != nil { // #nosec G204 -- RealSSH is a compile-time constant (/usr/bin/ssh)
		return fmt.Errorf("exec %s: %w", RealSSH, err)
	}
	// Unreachable — syscall.Exec replaces the process image.
	return nil
}

// Run executes ssh as a subprocess (keeping sshroute alive) and returns its exit code.
// stdin/stdout/stderr are inherited from the parent. Use this for fallback retry logic.
func Run(argv []string) (int, error) {
	cmd := exec.Command(argv[0], argv[1:]...) // #nosec G204 -- argv[0] is always RealSSH, a compile-time constant
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

// DryRun prints the resolved command to stdout without executing it.
func DryRun(argv []string) {
	fmt.Println("[dry-run] " + strings.Join(argv, " "))
}
