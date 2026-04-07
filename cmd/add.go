package cmd

import "github.com/spf13/cobra"

var (
	addHost    string
	addPort    int
	addUser    string
	addKey     string
	addJump    string
	addNetwork string
)

var addCmd = &cobra.Command{
	Use:   "add <alias>",
	Short: "Add or update a host in the config",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	return nil // implemented by A4
}

func init() {
	addCmd.Flags().StringVar(&addHost, "host", "", "target hostname or IP")
	addCmd.Flags().IntVar(&addPort, "port", 22, "SSH port")
	addCmd.Flags().StringVar(&addUser, "user", "", "SSH user")
	addCmd.Flags().StringVar(&addKey, "key", "", "path to identity file")
	addCmd.Flags().StringVar(&addJump, "jump", "", "jump host (ProxyJump)")
	addCmd.Flags().StringVar(&addNetwork, "network", "default", "network profile to set params for")
	rootCmd.AddCommand(addCmd)
}
