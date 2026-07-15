package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
	"github.com/thereisnotime/sshroute/internal/network"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

var (
	fallback       bool
	reconnect      bool
	reconnectDelay time.Duration
)

var connectCmd = &cobra.Command{
	Use:               "connect <host>",
	Short:             "Connect to a configured host using the active network profile",
	Args:              cobra.MinimumNArgs(1),
	RunE:              runConnect,
	ValidArgsFunction: completeAliases,
}

func runConnect(cmd *cobra.Command, args []string) error {
	alias := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	ssh.RealSSH = ssh.ResolveSSHBinary(cfg)

	if _, ok := cfg.Hosts[alias]; !ok {
		return fmt.Errorf("host %q not found in config", alias)
	}

	parsed := ssh.ParsedArgs{Remaining: args[1:]}

	if reconnect {
		return runConnectReconnect(cfg, alias, parsed)
	}

	detectedNetwork, err := network.Detect(cfg.Networks)
	if err != nil {
		return fmt.Errorf("detecting network: %w", err)
	}
	slog.Debug("connect: detected network", "network", detectedNetwork)

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

// runConnectReconnect supervises the ssh subprocess and reconnects whenever it exits
// with a connection failure (255), re-detecting the network and re-resolving the route
// each cycle so the connection follows network changes. It composes with --fallback.
func runConnectReconnect(cfg *config.Config, alias string, parsed ssh.ParsedArgs) error {
	if dryRun {
		detectedNetwork, err := network.Detect(cfg.Networks)
		if err != nil {
			return fmt.Errorf("detecting network: %w", err)
		}
		if fallback {
			for _, profileName := range ssh.ResolveOrder(cfg, alias, detectedNetwork) {
				params, err := ssh.Resolve(cfg, alias, profileName)
				if err != nil {
					continue
				}
				ssh.DryRun(ssh.BuildArgv(params, parsed))
			}
			return nil
		}
		params, err := ssh.Resolve(cfg, alias, detectedNetwork)
		if err != nil {
			return fmt.Errorf("resolving params: %w", err)
		}
		ssh.DryRun(ssh.BuildArgv(params, parsed))
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	attempt := func() (int, error) {
		detectedNetwork, err := network.Detect(cfg.Networks)
		if err != nil {
			return -1, fmt.Errorf("detecting network: %w", err)
		}
		slog.Debug("reconnect: detected network", "network", detectedNetwork)
		return attemptOnce(ctx, cfg, alias, detectedNetwork, parsed)
	}

	code, err := ssh.Supervise(attempt, ssh.ReconnectConfig{Delay: reconnectDelay}, ctx.Done())
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		// Interrupted by SIGINT/SIGTERM — normal shutdown.
		return nil
	}
	if code != 0 {
		os.Exit(code)
	}
	return nil
}

// attemptOnce runs one connection for the detected network — a single route, or the
// fallback try-order when --fallback is set — using a context-bound subprocess so a
// signal tears the child down. It returns ssh's exit code: 0 on success, the first
// non-255 code encountered, or 255 if every profile failed to connect.
func attemptOnce(ctx context.Context, cfg *config.Config, alias, netw string, parsed ssh.ParsedArgs) (int, error) {
	if !fallback {
		params, err := ssh.Resolve(cfg, alias, netw)
		if err != nil {
			return -1, fmt.Errorf("resolving params: %w", err)
		}
		return ssh.RunContext(ctx, ssh.BuildArgv(params, parsed))
	}
	for _, profileName := range ssh.ResolveOrder(cfg, alias, netw) {
		params, err := ssh.Resolve(cfg, alias, profileName)
		if err != nil {
			slog.Debug("reconnect: fallback resolve error, skipping", "profile", profileName, "error", err)
			continue
		}
		code, err := ssh.RunContext(ctx, ssh.BuildArgv(params, parsed))
		if err != nil {
			return -1, fmt.Errorf("running ssh: %w", err)
		}
		if code != ssh.SSHConnectFailure {
			return code, nil
		}
		fmt.Fprintf(os.Stderr, "sshroute: profile %q failed (connection error), trying next\n", profileName)
	}
	return ssh.SSHConnectFailure, nil
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().BoolVar(&fallback, "fallback", false, "try all profiles in priority order on connection failure")
	connectCmd.Flags().BoolVar(&reconnect, "reconnect", false, "supervise the connection and reconnect on drop, re-detecting the route each time")
	connectCmd.Flags().DurationVar(&reconnectDelay, "reconnect-delay", 2*time.Second, "wait between reconnect attempts when --reconnect is set")
}
