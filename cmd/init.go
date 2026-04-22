package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
)

// starterConfig is written verbatim — comments are preserved this way.
const starterConfig = `# sshroute configuration
# Documentation: https://github.com/thereisnotime/sshroute
#
# Networks are evaluated in priority order (lower value = checked first).
# All checks within a network must pass (AND logic).
# The first matching network wins; "default" is used when none match.
#
# Both networks and hosts support optional "comment" and "tags" fields.
# Use them to organize and filter: sshroute list --tag production

networks: {}
  # Uncomment and adjust the examples below:
  #
  # vpn:
  #   comment: "Corporate VPN (example)"
  #   tags: [corp, example]
  #   priority: 10
  #   checks:
  #     - type: interface
  #       match: wg0          # interface must be UP
  #     - type: route
  #       match: 10.8.0.0    # subnet must be in routing table
  #
  # office:
  #   comment: "Office LAN (example)"
  #   tags: [corp, example]
  #   priority: 20
  #   checks:
  #     - type: ping
  #       host: 192.168.1.1  # gateway must respond
  #       timeout: 500ms
  #
  # corp:
  #   priority: 30
  #   checks:
  #     - type: exec
  #       command: "dig +short +time=1 internal.corp | grep -qE '^[0-9]'"

# Host definitions.
# Every host requires a "default" profile.
# Network profiles only need to specify fields that differ from default.
# The "comment" and "tags" fields on the default profile are used for display
# and filtering (sshroute list --tag <tag> --filter <text>).

hosts: {}
  # myserver:
  #   default:
  #     host: myserver.example.com
  #     port: 22
  #     user: alice
  #     key: ~/.ssh/id_ed25519
  #     comment: "Web server (example)"
  #     tags: [production, web, example]
  #   vpn:
  #     host: 10.8.0.50
  #     port: 2222
  #     jump: bastion.vpn
  #
  # See examples/ in the repo for more patterns:
  #   basic, multi-network, wireguard-backconnect, jump-hosts
`

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter config file",
	Long:  "Creates ~/.config/sshroute/config.yaml with commented examples. Fails if the file already exists unless --force is given.",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	path := cfgFile
	if path == "" {
		var err error
		path, err = config.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("resolving config path: %w", err)
		}
	}

	if !initForce {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config already exists at %s (use --force to overwrite)", path)
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory %q: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(starterConfig), 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	fmt.Printf("Config created: %s\n", path)
	fmt.Println("Edit it with: sshroute config edit")
	return nil
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing config")
	rootCmd.AddCommand(initCmd)
}
