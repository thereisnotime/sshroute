// Package ssh — fallback.go builds the cache-aware try-order and runs it.
package ssh

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/thereisnotime/sshroute/internal/config"
)

// Why a profile sits where it does in the fallback order.
const (
	SourceCache    = "cache"    // last route that worked, from the route cache
	SourceDetected = "detected" // profile named after the detected network
	SourcePriority = "priority" // remaining profiles, by network priority then name
	SourceDefault  = "default"  // the default profile, always last
)

// OrderEntry is one profile in the fallback order together with the reason it
// is there.
type OrderEntry struct {
	Profile string
	Source  string
}

// ResolveOrderPreferred returns ResolveOrder with preferred moved to the front
// and tagged SourceCache. If preferred is empty or not a profile of alias, the
// order is unchanged. Each profile appears exactly once.
func ResolveOrderPreferred(cfg *config.Config, alias, detectedNetwork, preferred string) []OrderEntry {
	base := ResolveOrder(cfg, alias, detectedNetwork)
	entries := make([]OrderEntry, 0, len(base))
	found := false
	if preferred != "" {
		for _, p := range base {
			if p == preferred {
				found = true
				break
			}
		}
	}
	if found {
		entries = append(entries, OrderEntry{Profile: preferred, Source: SourceCache})
	}
	for _, p := range base {
		if found && p == preferred {
			continue
		}
		src := SourcePriority
		switch p {
		case "default":
			src = SourceDefault
		case detectedNetwork:
			src = SourceDetected
		}
		entries = append(entries, OrderEntry{Profile: p, Source: src})
	}
	return entries
}

// Attempt is the result of trying one profile.
type Attempt struct {
	Profile string
	Code    int
}

// Runner runs one ssh argv and returns its exit code. Run and RunContext fit.
type Runner func(argv []string) (int, error)

// FallbackStatus is where progress messages ("profile X failed, trying next")
// go. Tests swap it out.
var FallbackStatus io.Writer = os.Stderr

// RunFallback tries each profile in order, moving on only when ssh exits with
// SSHConnectFailure (255). It returns the first non-255 exit code (0 included),
// or 255 when every profile failed to connect, plus every attempt made so the
// caller can record the outcome. Profiles that fail to resolve are skipped. A
// runner error stops the walk and is returned with the attempts so far.
func RunFallback(cfg *config.Config, alias string, order []OrderEntry, parsed ParsedArgs, run Runner) (int, []Attempt, error) {
	var attempts []Attempt
	for i, e := range order {
		params, err := Resolve(cfg, alias, e.Profile)
		if err != nil {
			slog.Debug("fallback: resolve error, skipping", "profile", e.Profile, "error", err)
			continue
		}
		slog.Debug("fallback: trying profile", "profile", e.Profile, "source", e.Source, "host", params.Host, "attempt", i+1)
		code, err := run(BuildArgv(params, parsed))
		if err != nil {
			return -1, attempts, fmt.Errorf("running ssh: %w", err)
		}
		attempts = append(attempts, Attempt{Profile: e.Profile, Code: code})
		if code != SSHConnectFailure {
			return code, attempts, nil
		}
		_, _ = fmt.Fprintf(FallbackStatus, "sshroute: profile %q failed (connection error), trying next\n", e.Profile)
	}
	return SSHConnectFailure, attempts, nil
}
