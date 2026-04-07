package cmd

import "github.com/spf13/cobra"

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured hosts",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	return nil // implemented by A4
}

func init() {
	rootCmd.AddCommand(listCmd)
}
