package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
)

var removeCmd = &cobra.Command{
	Use:     "remove <alias>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a host from the config",
	Args:    cobra.ExactArgs(1),
	RunE:    runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
	alias := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, ok := cfg.Hosts[alias]; !ok {
		return fmt.Errorf("host %q not found in config", alias)
	}

	delete(cfg.Hosts, alias)

	if err := config.Save(cfgFile, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Removed host %q\n", alias)
	return nil
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
