package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/network"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

var fallback bool

var connectCmd = &cobra.Command{
	Use:   "connect <host>",
	Short: "Connect to a configured host using the active network profile",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runConnect,
}

func runConnect(cmd *cobra.Command, args []string) error {
	alias := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, ok := cfg.Hosts[alias]; !ok {
		return fmt.Errorf("host %q not found in config", alias)
	}

	detectedNetwork, err := network.Detect(cfg.Networks)
	if err != nil {
		return fmt.Errorf("detecting network: %w", err)
	}
	slog.Debug("connect: detected network", "network", detectedNetwork)

	parsed := ssh.ParsedArgs{Remaining: args[1:]}

	if !fallback {
		params, err := ssh.Resolve(cfg, alias, detectedNetwork)
		if err != nil {
			return fmt.Errorf("resolving params: %w", err)
		}
		slog.Debug("connect: resolved params", "host", params.Host, "port", params.Port, "user", params.User)
		argv := ssh.BuildArgv(params, parsed)
		if dryRun {
			ssh.DryRun(argv)
			return nil
		}
		if err := ssh.Exec(argv); err != nil {
			fmt.Fprintf(os.Stderr, "sshroute: exec error: %v\n", err)
			return err
		}
		return nil
	}

	// Fallback mode: try profiles in priority order, retry only on connection failure (exit 255).
	order := ssh.ResolveOrder(cfg, alias, detectedNetwork)
	for i, profileName := range order {
		params, err := ssh.Resolve(cfg, alias, profileName)
		if err != nil {
			slog.Debug("connect: fallback resolve error, skipping", "profile", profileName, "error", err)
			continue
		}
		slog.Debug("connect: fallback trying profile", "profile", profileName, "host", params.Host, "attempt", i+1)
		argv := ssh.BuildArgv(params, parsed)
		if dryRun {
			ssh.DryRun(argv)
			continue
		}
		code, err := ssh.Run(argv)
		if err != nil {
			return fmt.Errorf("running ssh: %w", err)
		}
		if code == 0 {
			return nil
		}
		if code != ssh.SSHConnectFailure {
			// Non-connection failure (auth, remote command) — don't retry.
			os.Exit(code)
		}
		fmt.Fprintf(os.Stderr, "sshroute: profile %q failed (connection error), trying next\n", profileName)
	}

	return fmt.Errorf("all profiles exhausted for host %q", alias)
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().BoolVar(&fallback, "fallback", false, "try all profiles in priority order on connection failure")
}
