package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
)

var (
	addHost    string
	addPort    int
	addUser    string
	addKey     string
	addJump    string
	addComment string
	addTags    []string
	addNetwork string
)

var addCmd = &cobra.Command{
	Use:   "add <alias>",
	Short: "Add or update a host in the config",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	alias := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	params := config.SSHParams{
		Host:    addHost,
		Port:    addPort,
		User:    addUser,
		Key:     addKey,
		Jump:    addJump,
		Comment: addComment,
		Tags:    addTags,
	}

	if cfg.Hosts == nil {
		cfg.Hosts = make(map[string]config.HostConfig)
	}

	if cfg.Hosts[alias] == nil {
		cfg.Hosts[alias] = make(config.HostConfig)
	}
	cfg.Hosts[alias][addNetwork] = params

	// When adding a non-default network profile, ensure a "default" profile
	// exists so the config remains valid. If the host is brand new, seed the
	// default from the same params.
	if addNetwork != "default" {
		if _, hasDefault := cfg.Hosts[alias]["default"]; !hasDefault {
			cfg.Hosts[alias]["default"] = params
		}
	}

	if err := config.Save(cfgFile, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Added/updated host %q for network %q\n", alias, addNetwork)
	return nil
}

func init() {
	addCmd.Flags().StringVar(&addHost, "host", "", "target hostname or IP")
	addCmd.Flags().IntVar(&addPort, "port", 22, "SSH port")
	addCmd.Flags().StringVar(&addUser, "user", "", "SSH user")
	addCmd.Flags().StringVar(&addKey, "key", "", "path to identity file")
	addCmd.Flags().StringVar(&addJump, "jump", "", "jump host (ProxyJump)")
	addCmd.Flags().StringVar(&addComment, "comment", "", "descriptive comment for the host")
	addCmd.Flags().StringSliceVar(&addTags, "tags", nil, "comma-separated tags for filtering")
	addCmd.Flags().StringVar(&addNetwork, "network", "default", "network profile to set params for")
	rootCmd.AddCommand(addCmd)
}
