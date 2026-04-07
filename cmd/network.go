package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
	internalnetwork "github.com/thereisnotime/sshroute/internal/network"
	outfmt "github.com/thereisnotime/sshroute/internal/output"
)

// NetworkRow is the display struct for a single network entry.
type NetworkRow struct {
	Name   string `json:"name"   yaml:"name"   table:"NETWORK"`
	Type   string `json:"type"   yaml:"type"   table:"TYPE"`
	Rule   string `json:"rule"   yaml:"rule"   table:"RULE"`
	Active bool   `json:"active" yaml:"active" table:"ACTIVE"`
}

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

func runNetwork(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	detected, err := internalnetwork.Detect(cfg.Networks)
	if err != nil {
		return fmt.Errorf("detecting network: %w", err)
	}

	fmt.Println(detected)
	return nil
}

func runNetworkList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	detected, err := internalnetwork.Detect(cfg.Networks)
	if err != nil {
		slog.Debug("network list: detection error", "error", err)
		detected = "unknown"
	}

	// Sort network names for deterministic output.
	names := make([]string, 0, len(cfg.Networks))
	for name := range cfg.Networks {
		names = append(names, name)
	}
	sort.Strings(names)

	var rows []NetworkRow
	for _, name := range names {
		checks := cfg.Networks[name]
		if len(checks) == 0 {
			rows = append(rows, NetworkRow{
				Name:   name,
				Active: name == detected,
			})
			continue
		}
		for _, check := range checks {
			rows = append(rows, NetworkRow{
				Name:   name,
				Type:   string(check.Type),
				Rule:   checkRuleString(check),
				Active: name == detected,
			})
		}
	}

	formatter := outfmt.New(output)
	if err := formatter.Format(os.Stdout, rows); err != nil {
		return fmt.Errorf("rendering output: %w", err)
	}
	return nil
}

func runNetworkTest(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	checks, ok := cfg.Networks[name]
	if !ok {
		return fmt.Errorf("network %q not found in config", name)
	}

	if len(checks) == 0 {
		fmt.Printf("network %q has no checks defined\n", name)
		return nil
	}

	// Run each check individually by passing a single-check map to Detect.
	// Detect returns the network name if all checks in the slice pass, so we
	// test one check at a time using a synthetic single-entry map.
	allPassed := true
	for i, check := range checks {
		singleMap := map[string][]config.NetworkCheck{
			name: {check},
		}
		result, err := internalnetwork.Detect(singleMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  check[%d] type=%-10s rule=%-30s  ERROR: %v\n",
				i, check.Type, checkRuleString(check), err)
			allPassed = false
			continue
		}
		passed := result == name
		if !passed {
			allPassed = false
		}
		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		fmt.Printf("  check[%d] type=%-10s rule=%-30s  %s\n",
			i, check.Type, checkRuleString(check), status)
	}

	fmt.Println()
	if allPassed {
		fmt.Printf("network %q: ACTIVE\n", name)
	} else {
		fmt.Printf("network %q: NOT ACTIVE\n", name)
	}
	return nil
}

// checkRuleString returns a compact human-readable description of a single check.
func checkRuleString(check config.NetworkCheck) string {
	switch {
	case check.Match != "":
		return check.Match
	case check.Host != "":
		return check.Host
	case check.Command != "":
		if len(check.Command) > 40 {
			return check.Command[:37] + "..."
		}
		return check.Command
	}
	return ""
}

func init() {
	networkCmd.AddCommand(networkListCmd)
	networkCmd.AddCommand(networkTestCmd)
	rootCmd.AddCommand(networkCmd)
}
