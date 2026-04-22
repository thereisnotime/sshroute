package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

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
	Comment string `json:"comment" yaml:"comment" table:"COMMENT"`
	Tags    string `json:"tags"    yaml:"tags"    table:"TAGS"`
}

var (
	listFilterTags []string
	listFilterText string
)

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
		row := HostRow{
			Alias:   alias,
			Network: detectedNetwork,
			Host:    params.Host,
			Port:    params.Port,
			User:    params.User,
			Key:     params.Key,
			Jump:    params.Jump,
			Comment: params.Comment,
			Tags:    strings.Join(params.Tags, ","),
		}
		if !matchesFilters(row, params.Tags) {
			continue
		}
		rows = append(rows, row)
	}

	formatter := outfmt.New(output)
	if err := formatter.Format(os.Stdout, rows); err != nil {
		return fmt.Errorf("rendering output: %w", err)
	}
	return nil
}

func matchesFilters(row HostRow, tags []string) bool {
	if len(listFilterTags) > 0 {
		found := false
		for _, ft := range listFilterTags {
			for _, t := range tags {
				if strings.EqualFold(t, ft) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	if listFilterText != "" {
		needle := strings.ToLower(listFilterText)
		haystack := strings.ToLower(strings.Join([]string{
			row.Alias, row.Host, row.User, row.Key, row.Jump, row.Comment, row.Tags,
		}, " "))
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}

func init() {
	listCmd.Flags().StringSliceVar(&listFilterTags, "tag", nil, "filter hosts by tag (repeatable, OR logic)")
	listCmd.Flags().StringVar(&listFilterText, "filter", "", "filter hosts by substring across all fields")
	rootCmd.AddCommand(listCmd)
}
