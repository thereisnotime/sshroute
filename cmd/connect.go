package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/network"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

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

	params, err := ssh.Resolve(cfg, alias, detectedNetwork)
	if err != nil {
		return fmt.Errorf("resolving params: %w", err)
	}
	slog.Debug("connect: resolved params", "host", params.Host, "port", params.Port, "user", params.User)

	parsed := ssh.ParsedArgs{Remaining: args[1:]}
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

func init() {
	rootCmd.AddCommand(connectCmd)
}
