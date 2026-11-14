// Package routecache remembers which fallback profile last connected for each
// host, so the next fallback run can try that profile first.
//
// The cache lives in a small versioned JSON state file. Reads are lock-free (the
// file is always replaced atomically by rename); writes take an advisory lock on a
// sibling ".lock" file, re-read the current state, apply the change and rename a
// temp file into place, so concurrent sshroute processes never lose each other's
// entries. A missing, unreadable or corrupt state file is treated as empty.
package routecache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/thereisnotime/sshroute/internal/config"
)

// Version is the state file format version written by this build.
const Version = 1

// ErrUnsupportedVersion is returned when the state file was written by a newer
// sshroute with a format this build does not understand. Such a file is neither
// used nor overwritten.
var ErrUnsupportedVersion = errors.New("unsupported state file version")

// Route is one successful connection: which profile connected, on which detected
// network, and when.
type Route struct {
	Profile     string    `json:"profile"`
	Network     string    `json:"network"`
	SucceededAt time.Time `json:"succeeded_at"`
}

// Failure is the most recent connection failure (ssh exit 255) of one profile.
type Failure struct {
	Network  string    `json:"network"`
	FailedAt time.Time `json:"failed_at"`
}

// HostState holds everything recorded for one alias.
type HostState struct {
	// LastSuccess is the most recent successful route on any network. It is what
	// the "alias" scope looks up, and what dashboards should show.
	LastSuccess *Route `json:"last_success,omitempty"`
	// Networks maps a detected network name to the last route that worked on it.
	// This is what the "network" scope looks up.
	Networks map[string]Route `json:"networks,omitempty"`
	// Failures maps a profile name to its most recent connection failure.
	Failures map[string]Failure `json:"failures,omitempty"`
}

// State is the whole state file.
type State struct {
	Version int                   `json:"version"`
	Hosts   map[string]*HostState `json:"hosts"`
}

// NewState returns an empty state at the current version.
func NewState() *State {
	return &State{Version: Version, Hosts: make(map[string]*HostState)}
}

// Outcome is the result of one fallback run for an alias.
type Outcome struct {
	// Succeeded is the profile whose connection worked (ssh exited with anything
	// other than 255), or "" if none did.
	Succeeded string
	// Failed lists profiles that failed to connect (ssh exit 255), in try order.
	Failed []string
}

// Empty reports whether the outcome has nothing worth recording.
func (o Outcome) Empty() bool { return o.Succeeded == "" && len(o.Failed) == 0 }

// Apply records outcome for alias on the detected network at time now.
func (s *State) Apply(alias, network string, o Outcome, now time.Time) {
	if o.Empty() {
		return
	}
	now = now.UTC().Truncate(time.Second)
	h := s.Hosts[alias]
	if h == nil {
		h = &HostState{}
		s.Hosts[alias] = h
	}
	for _, p := range o.Failed {
		if h.Failures == nil {
			h.Failures = make(map[string]Failure)
		}
		h.Failures[p] = Failure{Network: network, FailedAt: now}
	}
	if o.Succeeded != "" {
		r := Route{Profile: o.Succeeded, Network: network, SucceededAt: now}
		if h.Networks == nil {
			h.Networks = make(map[string]Route)
		}
		h.Networks[network] = r
		h.LastSuccess = &r
	}
}

// Clear removes alias from the state, or every alias when alias is "". It reports
// whether anything was removed.
func (s *State) Clear(alias string) bool {
	if alias == "" {
		n := len(s.Hosts)
		s.Hosts = make(map[string]*HostState)
		return n > 0
	}
	if _, ok := s.Hosts[alias]; !ok {
		return false
	}
	delete(s.Hosts, alias)
	return true
}

// decode parses state file bytes. Corrupt data yields an error; callers treat
// that as an empty state.
func decode(data []byte) (*State, error) {
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}
	if st.Version > Version {
		return nil, fmt.Errorf("%w %d (this build understands %d)", ErrUnsupportedVersion, st.Version, Version)
	}
	if st.Version < 1 {
		return nil, fmt.Errorf("parsing state file: missing or invalid version %d", st.Version)
	}
	if st.Hosts == nil {
		st.Hosts = make(map[string]*HostState)
	}
	for alias, h := range st.Hosts {
		if h == nil {
			delete(st.Hosts, alias)
		}
	}
	return &st, nil
}

func encode(st *State) ([]byte, error) {
	st.Version = Version
	if st.Hosts == nil {
		st.Hosts = make(map[string]*HostState)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// readFile loads the state at path. A missing file is an empty state with no
// error. A corrupt file is an empty state plus the parse error, so callers can
// log it and carry on.
func readFile(path string) (*State, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is the user's own state file from config, env or the XDG default
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewState(), nil
		}
		return NewState(), fmt.Errorf("reading state file: %w", err)
	}
	st, err := decode(data)
	if err != nil {
		return NewState(), err
	}
	return st, nil
}

// Scope selects how cache entries are keyed for lookups.
type Scope string

// Scope values mirror config.CacheScope*.
const (
	ScopeNetwork Scope = config.CacheScopeNetwork
	ScopeAlias   Scope = config.CacheScopeAlias
)
