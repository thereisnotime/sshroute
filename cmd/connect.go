package cmd

import "github.com/spf13/cobra"

var connectCmd = &cobra.Command{
	Use:   "connect <host>",
	Short: "Connect to a configured host using the active network profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runConnect,
}

func runConnect(cmd *cobra.Command, args []string) error {
	return nil // implemented by A4
}

func init() {
	rootCmd.AddCommand(connectCmd)
}
