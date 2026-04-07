// Package cmd contains all Cobra CLI commands for sshroute.
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
	"github.com/thereisnotime/sshroute/internal/network"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

var (
	cfgFile string
	output  string
	verbose bool
	dryRun  bool
)

var rootCmd = &cobra.Command{
	Use:           "sshroute",
	Short:         "Network-aware SSH router",
	Long:          "sshroute routes SSH connections to different hosts/ports/keys based on your active network or VPN.",
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute is the main entrypoint. Detects shadow mode (called as "ssh") or runs Cobra.
func Execute() {
	if filepath.Base(os.Args[0]) == "ssh" {
		runShadowMode()
		return
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/sshroute/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "output format: table|json|yaml")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print resolved SSH command without executing")
}

// loadConfig loads the sshroute config using the global --config flag value.
func loadConfig() (*config.Config, error) {
	return config.Load(cfgFile)
}

// runShadowMode handles transparent SSH interception when the binary is invoked as "ssh".
func runShadowMode() {
	// Configure slog if verbose is set. In shadow mode we parse os.Args
	// directly, so check the environment variable as a fallback.
	if verbose || os.Getenv("SSHROUTE_VERBOSE") == "1" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	passthrough := func() {
		if err := syscall.Exec(ssh.RealSSH, append([]string{ssh.RealSSH}, os.Args[1:]...), os.Environ()); err != nil { // #nosec G204 -- RealSSH is a compile-time constant (/usr/bin/ssh); transparent passthrough is the intended behaviour
			fmt.Fprintf(os.Stderr, "sshroute: passthrough exec failed: %v\n", err)
			os.Exit(1)
		}
	}

	// 1. Load config — if it fails, passthrough transparently.
	cfg, err := loadConfig()
	if err != nil {
		slog.Debug("shadow mode: config load failed, passthrough", "error", err)
		passthrough()
		return
	}

	// 2. Parse os.Args[1:] to extract alias and remaining flags.
	parsed := ssh.ParseArgs(os.Args[1:])

	// 3. If no alias found or alias not in cfg.Hosts, passthrough.
	if parsed.Alias == "" {
		slog.Debug("shadow mode: no alias parsed, passthrough")
		passthrough()
		return
	}
	if _, ok := cfg.Hosts[parsed.Alias]; !ok {
		slog.Debug("shadow mode: alias not in config, passthrough", "alias", parsed.Alias)
		passthrough()
		return
	}

	// 4. Detect active network.
	detectedNetwork, err := network.Detect(cfg.Networks)
	if err != nil {
		slog.Debug("shadow mode: network detection error, passthrough", "error", err)
		passthrough()
		return
	}
	slog.Debug("shadow mode: detected network", "network", detectedNetwork)

	// 5. Resolve SSH params for alias + detected network.
	params, err := ssh.Resolve(cfg, parsed.Alias, detectedNetwork)
	if err != nil {
		slog.Debug("shadow mode: resolve error, passthrough", "error", err)
		passthrough()
		return
	}

	// 6. Build final argv.
	argv := ssh.BuildArgv(params, parsed)

	// 7. Dry-run: print and exit.
	if dryRun {
		ssh.DryRun(argv)
		os.Exit(0)
	}

	// 8. Exec — replace the process.
	if err := ssh.Exec(argv); err != nil {
		fmt.Fprintf(os.Stderr, "sshroute: exec error: %v\n", err)
		// 9. On exec error, fall back to real ssh with original args.
		passthrough()
	}
}
