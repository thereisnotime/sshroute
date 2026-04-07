package cmd

import "github.com/spf13/cobra"

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Show or test network detection",
	RunE:  runNetwork,
}

var networkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured networks and their detection rules",
	RunE:  runNetworkList,
}

var networkTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test whether a specific network is currently active",
	Args:  cobra.ExactArgs(1),
	RunE:  runNetworkTest,
}

func runNetwork(cmd *cobra.Command, args []string) error    { return nil } // implemented by A4
func runNetworkList(cmd *cobra.Command, args []string) error { return nil } // implemented by A4
func runNetworkTest(cmd *cobra.Command, args []string) error { return nil } // implemented by A4

func init() {
	networkCmd.AddCommand(networkListCmd)
	networkCmd.AddCommand(networkTestCmd)
	rootCmd.AddCommand(networkCmd)
}
