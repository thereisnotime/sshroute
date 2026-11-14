package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
	"github.com/thereisnotime/sshroute/internal/network"
	outfmt "github.com/thereisnotime/sshroute/internal/output"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

var (
	resolveNetwork  string
	resolveFallback bool
)

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

// ResolveOrderRow is one profile of the fallback order shown by `resolve --fallback`.
type ResolveOrderRow struct {
	Order    int    `json:"order"     yaml:"order"     table:"#"`
	Profile  string `json:"profile"   yaml:"profile"   table:"PROFILE"`
	Source   string `json:"source"    yaml:"source"    table:"SOURCE"`
	CachedAt string `json:"cached_at" yaml:"cached_at" table:"CACHED AT"`
	Host     string `json:"host"      yaml:"host"      table:"HOST"`
	Port     int    `json:"port"      yaml:"port"      table:"PORT"`
	User     string `json:"user"      yaml:"user"      table:"USER"`
	Jump     string `json:"jump"      yaml:"jump"      table:"JUMP"`
	Command  string `json:"command"   yaml:"command"   table:"COMMAND"`
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

	if resolveFallback {
		return resolveOrder(cfg, alias, net)
	}

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

// resolveOrder prints the fallback order `connect --fallback` would try,
// including where a cached route came from.
func resolveOrder(cfg *config.Config, alias, net string) error {
	plan := planFallback(cfg, alias, net, cacheOptions(cfg))
	rows := make([]ResolveOrderRow, 0, len(plan.Order))
	for _, e := range plan.Order {
		params, err := ssh.Resolve(cfg, alias, e.Profile)
		if err != nil {
			slog.Debug("resolve: skipping profile", "profile", e.Profile, "error", err)
			continue
		}
		row := ResolveOrderRow{
			Order:   len(rows) + 1,
			Profile: e.Profile,
			Source:  e.Source,
			Host:    params.Host,
			Port:    params.Port,
			User:    params.User,
			Jump:    params.Jump,
			Command: strings.Join(ssh.BuildArgv(params, ssh.ParsedArgs{}), " "),
		}
		if e.Source == ssh.SourceCache {
			row.CachedAt = plan.Lookup.Route.SucceededAt.Format(time.RFC3339)
		}
		rows = append(rows, row)
	}
	if output != outfmt.FormatJSON && output != outfmt.FormatYAML {
		fmt.Fprintf(os.Stderr, "network: %s\ncache: %s\n", net, plan.cacheSummary())
	}
	return outfmt.New(output).Format(os.Stdout, rows)
}

func init() {
	rootCmd.AddCommand(resolveCmd)
	resolveCmd.Flags().BoolVar(&resolveFallback, "fallback", false, "show the full fallback order connect --fallback would try, cached route first")
	resolveCmd.Flags().BoolVar(&noCache, "no-cache", false, "ignore the route cache when building the fallback order (also SSHROUTE_CACHE=0)")
	resolveCmd.Flags().StringVar(&resolveNetwork, "network", "", "network profile to resolve against (default: auto-detect)")
}
