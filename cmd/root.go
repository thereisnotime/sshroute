// Package cmd contains all Cobra CLI commands for sshroute.
package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	cfgFile  string
	output   string
	verbose  bool
	dryRun   bool
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
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/sshroute/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "output format: table|json|yaml")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print resolved SSH command without executing")
}

// runShadowMode handles transparent SSH interception.
// Implemented by A4 — stub for now.
func runShadowMode() {}
