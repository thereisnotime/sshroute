package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/network"
	outfmt "github.com/thereisnotime/sshroute/internal/output"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

var resolveNetwork string

var resolveCmd = &cobra.Command{
	Use:               "resolve <alias>",
	Short:             "Show the resolved SSH parameters for a host on the active network",
	Args:              cobra.ExactArgs(1),
	RunE:              runResolve,
	ValidArgsFunction: completeAliases,
}

// ResolveRow is the display struct for resolved SSH parameters.
type ResolveRow struct {
	Alias   string `json:"alias"   yaml:"alias"   table:"ALIAS"`
	Network string `json:"network" yaml:"network" table:"NETWORK"`
	Host    string `json:"host"    yaml:"host"    table:"HOST"`
	Port    int    `json:"port"    yaml:"port"    table:"PORT"`
	User    string `json:"user"    yaml:"user"    table:"USER"`
	Key     string `json:"key"     yaml:"key"     table:"KEY"`
	Jump    string `json:"jump"    yaml:"jump"    table:"JUMP"`
	Command string `json:"command" yaml:"command" table:"COMMAND"`
}

func runResolve(cmd *cobra.Command, args []string) error {
	alias := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, ok := cfg.Hosts[alias]; !ok {
		return fmt.Errorf("host %q not found in config", alias)
	}

	net := resolveNetwork
	if net == "" {
		net, err = network.Detect(cfg.Networks)
		if err != nil {
			slog.Debug("resolve: network detection error, using default", "error", err)
			net = "default"
		}
	}
	slog.Debug("resolve: network", "network", net)

	params, err := ssh.Resolve(cfg, alias, net)
	if err != nil {
		return fmt.Errorf("resolving params: %w", err)
	}

	argv := ssh.BuildArgv(params, ssh.ParsedArgs{})
	row := ResolveRow{
		Alias:   alias,
		Network: net,
		Host:    params.Host,
		Port:    params.Port,
		User:    params.User,
		Key:     params.Key,
		Jump:    params.Jump,
		Command: strings.Join(argv, " "),
	}

	formatter := outfmt.New(output)
	return formatter.Format(os.Stdout, []ResolveRow{row})
}

func init() {
	rootCmd.AddCommand(resolveCmd)
	resolveCmd.Flags().StringVar(&resolveNetwork, "network", "", "network profile to resolve against (default: auto-detect)")
}
