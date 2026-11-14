package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
	outfmt "github.com/thereisnotime/sshroute/internal/output"
	"github.com/thereisnotime/sshroute/internal/routecache"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

// noCache is bound to --no-cache on every command that builds a fallback order.
var noCache bool

// cacheClearLockTimeout is the minimum lock wait for the interactive `cache clear`,
// which can afford to wait longer than a connect does.
const cacheClearLockTimeout = 5 * time.Second

// cacheNow is the clock used for cache lookups; tests override it.
var cacheNow = time.Now

// cacheOptions resolves the effective cache settings for cfg. An invalid cache
// section disables the cache rather than failing the connection.
func cacheOptions(cfg *config.Config) routecache.Options {
	o, err := routecache.ResolveOptions(cfg.Cache, noCache, os.Getenv)
	if err != nil {
		slog.Debug("cache: invalid settings, cache disabled", "error", err)
		o.Enabled = false
		o.Disabled = "invalid cache settings: " + err.Error()
	}
	return o
}

// fallbackPlan is the fallback try-order for one alias plus how the cache shaped it.
type fallbackPlan struct {
	Order   []ssh.OrderEntry
	Lookup  routecache.Lookup
	Opts    routecache.Options
	Network string
}

// planFallback looks the alias up in the route cache (when enabled) and builds
// the try-order with a fresh cached profile first.
func planFallback(cfg *config.Config, alias, network string, opts routecache.Options) fallbackPlan {
	plan := fallbackPlan{Opts: opts, Network: network}
	preferred := ""
	if opts.Enabled {
		st, err := routecache.NewStore(opts).Load()
		if err != nil {
			slog.Debug("cache: ignoring unreadable state file", "path", opts.Path, "error", err)
		}
		host := cfg.Hosts[alias]
		exists := func(p string) bool { _, ok := host[p]; return ok }
		plan.Lookup = st.Lookup(alias, network, opts.Scope, opts.TTL, cacheNow(), exists)
		if plan.Lookup.Hit {
			preferred = plan.Lookup.Route.Profile
			slog.Debug("cache: hit", "alias", alias, "key", plan.Lookup.Key, "profile", preferred, "succeeded_at", plan.Lookup.Route.SucceededAt)
		} else {
			slog.Debug("cache: miss", "alias", alias, "key", plan.Lookup.Key, "reason", plan.Lookup.Reason)
		}
	} else {
		slog.Debug("cache: disabled", "reason", opts.Disabled)
	}
	plan.Order = ssh.ResolveOrderPreferred(cfg, alias, network, preferred)
	return plan
}

// cacheSummary is a one-line description of the cache decision.
func (p fallbackPlan) cacheSummary() string {
	switch {
	case !p.Opts.Enabled:
		return "disabled (" + p.Opts.Disabled + ")"
	case p.Lookup.Hit:
		return fmt.Sprintf("hit [%s scope, key %s]: %s", p.Opts.Scope, p.Lookup.Key, p.Lookup.Reason)
	default:
		return fmt.Sprintf("miss [%s scope, key %s]: %s", p.Opts.Scope, p.Lookup.Key, p.Lookup.Reason)
	}
}

// orderSummary renders the order as "a (cache), b (detected), default (default)".
func (p fallbackPlan) orderSummary() string {
	parts := make([]string, len(p.Order))
	for i, e := range p.Order {
		parts[i] = fmt.Sprintf("%s (%s)", e.Profile, e.Source)
	}
	return strings.Join(parts, ", ")
}

// printDryRunPlan writes the cache decision and order ahead of the dry-run commands.
func printDryRunPlan(w io.Writer, p fallbackPlan) {
	_, _ = fmt.Fprintf(w, "[dry-run] cache: %s\n", p.cacheSummary())
	_, _ = fmt.Fprintf(w, "[dry-run] order: %s\n", p.orderSummary())
}

// outcomeFromAttempts turns fallback attempts into a cache outcome: every 255 is a
// failed profile; the attempt that ended the walk with any other exit code
// connected. A negative code (ssh killed by a signal) proves nothing either way.
func outcomeFromAttempts(attempts []ssh.Attempt) routecache.Outcome {
	var o routecache.Outcome
	for _, a := range attempts {
		switch {
		case a.Code == ssh.SSHConnectFailure:
			o.Failed = append(o.Failed, a.Profile)
		case a.Code >= 0:
			o.Succeeded = a.Profile
		}
	}
	return o
}

// recordFallback stores the outcome of a fallback walk. Failures to write are
// logged and otherwise ignored: the cache must never break a connection.
func recordFallback(opts routecache.Options, alias, network string, attempts []ssh.Attempt) {
	if !opts.Enabled {
		return
	}
	o := outcomeFromAttempts(attempts)
	if o.Empty() {
		return
	}
	if err := routecache.NewStore(opts).Record(alias, network, o); err != nil {
		slog.Debug("cache: write skipped", "path", opts.Path, "error", err)
		return
	}
	slog.Debug("cache: recorded", "alias", alias, "network", network, "succeeded", o.Succeeded, "failed", o.Failed)
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Inspect or clear the fallback route cache",
	Long: "The route cache remembers which profile last connected in --fallback mode " +
		"so the next fallback run tries it first. See `sshroute cache show`.",
}

var cacheShowCmd = &cobra.Command{
	Use:               "show [alias]",
	Short:             "Show cached routes (all hosts, or one alias)",
	Args:              cobra.MaximumNArgs(1),
	RunE:              runCacheShow,
	ValidArgsFunction: completeAliases,
}

var cacheClearCmd = &cobra.Command{
	Use:               "clear [alias]",
	Short:             "Remove cached routes (all hosts, or one alias)",
	Args:              cobra.MaximumNArgs(1),
	RunE:              runCacheClear,
	ValidArgsFunction: completeAliases,
}

var cachePathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the route cache state file path",
	Args:  cobra.NoArgs,
	RunE:  runCachePath,
}

// CacheRow is the display struct for `cache show`.
type CacheRow struct {
	Alias       string `json:"alias"        yaml:"alias"        table:"ALIAS"`
	Key         string `json:"key"          yaml:"key"          table:"KEY"`
	Profile     string `json:"profile"      yaml:"profile"      table:"PROFILE"`
	Network     string `json:"network"      yaml:"network"      table:"NETWORK"`
	SucceededAt string `json:"succeeded_at" yaml:"succeeded_at" table:"LAST SUCCESS"`
	Age         string `json:"-"            yaml:"-"            table:"AGE"`
	Status      string `json:"status"       yaml:"status"       table:"STATUS"`
}

// cacheSettings loads the config and resolves cache options for the cache
// subcommands. --no-cache does not apply to them.
func cacheSettings() (*config.Config, routecache.Options, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, routecache.Options{}, fmt.Errorf("loading config: %w", err)
	}
	o, err := routecache.ResolveOptions(cfg.Cache, false, os.Getenv)
	if err != nil {
		return nil, o, err
	}
	return cfg, o, nil
}

func runCacheShow(cmd *cobra.Command, args []string) error {
	cfg, opts, err := cacheSettings()
	if err != nil {
		return err
	}
	st, err := routecache.NewStore(opts).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sshroute: ignoring unreadable state file %s: %v\n", opts.Path, err)
	}
	filter := ""
	if len(args) == 1 {
		filter = args[0]
	}
	rows := cacheRows(cfg, st, opts, filter, cacheNow())
	if len(rows) == 0 && output != outfmt.FormatJSON && output != outfmt.FormatYAML {
		fmt.Println("no cached routes")
		return nil
	}
	return outfmt.New(output).Format(os.Stdout, rows)
}

// cacheRows flattens the state into display rows: per alias, the alias-scope
// entry ("*") first, then one row per network, sorted.
func cacheRows(cfg *config.Config, st *routecache.State, opts routecache.Options, filter string, now time.Time) []CacheRow {
	aliases := make([]string, 0, len(st.Hosts))
	for a := range st.Hosts {
		if filter == "" || a == filter {
			aliases = append(aliases, a)
		}
	}
	sort.Strings(aliases)

	rows := []CacheRow{}
	for _, a := range aliases {
		h := st.Hosts[a]
		host, known := cfg.Hosts[a]
		exists := func(p string) bool {
			if !known {
				return false
			}
			_, ok := host[p]
			return ok
		}
		row := func(key string, r routecache.Route, scope routecache.Scope) CacheRow {
			return CacheRow{
				Alias:       a,
				Key:         key,
				Profile:     r.Profile,
				Network:     r.Network,
				SucceededAt: r.SucceededAt.Format(time.RFC3339),
				Age:         now.Sub(r.SucceededAt).Truncate(time.Second).String(),
				Status:      string(routecache.RouteStatus(h, r, scope, opts.TTL, now, exists)),
			}
		}
		if h.LastSuccess != nil {
			rows = append(rows, row(routecache.AliasKey, *h.LastSuccess, routecache.ScopeAlias))
		}
		nets := make([]string, 0, len(h.Networks))
		for n := range h.Networks {
			nets = append(nets, n)
		}
		sort.Strings(nets)
		for _, n := range nets {
			rows = append(rows, row(n, h.Networks[n], routecache.ScopeNetwork))
		}
	}
	return rows
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	_, opts, err := cacheSettings()
	if err != nil {
		return err
	}
	alias := ""
	if len(args) == 1 {
		alias = args[0]
	}
	store := routecache.NewStore(opts)
	if store.LockTimeout < cacheClearLockTimeout {
		store.LockTimeout = cacheClearLockTimeout
	}
	removed := false
	if err := store.Update(func(st *routecache.State) { removed = st.Clear(alias) }); err != nil {
		return fmt.Errorf("clearing cache: %w", err)
	}
	switch {
	case alias == "":
		fmt.Println("cleared all cached routes")
	case removed:
		fmt.Printf("cleared cached routes for %q\n", alias)
	default:
		fmt.Printf("no cached routes for %q\n", alias)
	}
	return nil
}

func runCachePath(cmd *cobra.Command, args []string) error {
	_, opts, err := cacheSettings()
	if err != nil {
		return err
	}
	fmt.Println(opts.Path)
	return nil
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheShowCmd, cacheClearCmd, cachePathCmd)
}
