package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/network"
	outfmt "github.com/thereisnotime/sshroute/internal/output"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

// HostRow is the display struct for a single host entry.
type HostRow struct {
	Alias   string `json:"alias"   yaml:"alias"   table:"ALIAS"`
	Network string `json:"network" yaml:"network" table:"NETWORK"`
	Host    string `json:"host"    yaml:"host"    table:"HOST"`
	Port    int    `json:"port"    yaml:"port"    table:"PORT"`
	User    string `json:"user"    yaml:"user"    table:"USER"`
	Key     string `json:"key"     yaml:"key"     table:"KEY"`
	Jump    string `json:"jump"    yaml:"jump"    table:"JUMP"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured hosts",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	detectedNetwork, err := network.Detect(cfg.Networks)
	if err != nil {
		// Non-fatal — we still want to list hosts; just show "unknown" for active network.
		slog.Debug("list: network detection error", "error", err)
		detectedNetwork = "unknown"
	}
	slog.Debug("list: detected network", "network", detectedNetwork)

	// Collect and sort aliases.
	aliases := make([]string, 0, len(cfg.Hosts))
	for alias := range cfg.Hosts {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	rows := make([]HostRow, 0, len(aliases))
	for _, alias := range aliases {
		params, err := ssh.Resolve(cfg, alias, detectedNetwork)
		if err != nil {
			slog.Debug("list: resolve error", "alias", alias, "error", err)
			continue
		}
		rows = append(rows, HostRow{
			Alias:   alias,
			Network: detectedNetwork,
			Host:    params.Host,
			Port:    params.Port,
			User:    params.User,
			Key:     params.Key,
			Jump:    params.Jump,
		})
	}

	formatter := outfmt.New(output)
	if err := formatter.Format(os.Stdout, rows); err != nil {
		return fmt.Errorf("rendering output: %w", err)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
}
