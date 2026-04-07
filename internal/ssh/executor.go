// Package ssh — executor.go builds the final SSH argv and execs the real ssh binary.
package ssh

import "github.com/thereisnotime/sshroute/internal/config"

// RealSSH is the absolute path to the system SSH binary.
// Using absolute path prevents infinite recursion in shadow mode.
const RealSSH = "/usr/bin/ssh"

// BuildArgv constructs the final argv slice from resolved params and remaining args.
func BuildArgv(params config.SSHParams, parsed ParsedArgs) []string { return nil } // implemented by A3

// Exec replaces the current process with ssh using the given argv.
// Never returns on success.
func Exec(argv []string) error { return nil } // implemented by A3

// DryRun prints the resolved command without executing it.
func DryRun(argv []string) { } // implemented by A3
