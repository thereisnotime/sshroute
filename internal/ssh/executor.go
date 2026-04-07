// Package ssh — executor.go builds the final SSH argv and execs the real ssh binary.
package ssh

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/thereisnotime/sshroute/internal/config"
)

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

	if params.Jump != "" {
		argv = append(argv, "-J", params.Jump)
	}

	argv = append(argv, params.Host)
	argv = append(argv, parsed.Remaining...)

	return argv
}

// Exec replaces the current process with ssh using the given argv.
// On success this function never returns; the current process is replaced entirely.
func Exec(argv []string) error {
	if err := syscall.Exec(RealSSH, argv, os.Environ()); err != nil { //nolint:gosec // G204: RealSSH is a compile-time constant (/usr/bin/ssh)
		return fmt.Errorf("exec %s: %w", RealSSH, err)
	}
	// Unreachable — syscall.Exec replaces the process image.
	return nil
}

// DryRun prints the resolved command to stdout without executing it.
func DryRun(argv []string) {
	fmt.Println("[dry-run] " + strings.Join(argv, " "))
}
