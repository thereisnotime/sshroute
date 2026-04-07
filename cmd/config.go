package cmd

import "github.com/spf13/cobra"

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show config file path or open it in $EDITOR",
	RunE:  runConfig,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the config file in $EDITOR",
	RunE:  runConfigEdit,
}

func runConfig(cmd *cobra.Command, args []string) error     { return nil } // implemented by A4
func runConfigEdit(cmd *cobra.Command, args []string) error { return nil } // implemented by A4

func init() {
	configCmd.AddCommand(configEditCmd)
	rootCmd.AddCommand(configCmd)
}
