package cmd

import (
	"sort"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
)

// completeAliases is a ValidArgsFunction that suggests configured host aliases.
func completeAliases(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	aliases := make([]string, 0, len(cfg.Hosts))
	for alias := range cfg.Hosts {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases, cobra.ShellCompDirectiveNoFileComp
}
