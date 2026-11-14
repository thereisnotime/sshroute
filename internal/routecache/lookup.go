package routecache

import (
	"fmt"
	"time"
)

// Status describes whether a cached route may be used.
type Status string

// Status values.
const (
	StatusFresh          Status = "fresh"           // usable
	StatusExpired        Status = "expired"         // older than the TTL
	StatusFailed         Status = "failed"          // the profile failed to connect after this success
	StatusUnknownProfile Status = "unknown-profile" // the profile no longer exists in the config
)

// Lookup is the result of asking the cache for an alias on a network.
type Lookup struct {
	Hit    bool   // true when Route should be tried first
	Key    string // what was looked up: the network name, or "*" for the alias scope
	Route  Route  // the entry found (zero when there was none)
	Status Status // status of the entry found ("" when there was none)
	Reason string // short human explanation, for -v and dry-run output
}

// AliasKey is the Lookup.Key used for the alias scope.
const AliasKey = "*"

// RouteStatus evaluates a cached route. exists reports whether a profile is still
// defined for the host; ttl <= 0 disables expiry. In the network scope only
// failures recorded on the same network invalidate the route.
func RouteStatus(h *HostState, r Route, scope Scope, ttl time.Duration, now time.Time, exists func(string) bool) Status {
	if exists != nil && !exists(r.Profile) {
		return StatusUnknownProfile
	}
	if f, ok := h.Failures[r.Profile]; ok && f.FailedAt.After(r.SucceededAt) {
		if scope == ScopeAlias || f.Network == r.Network {
			return StatusFailed
		}
	}
	if ttl > 0 && now.Sub(r.SucceededAt) > ttl {
		return StatusExpired
	}
	return StatusFresh
}

// Lookup finds the cached route for alias on the detected network.
func (s *State) Lookup(alias, network string, scope Scope, ttl time.Duration, now time.Time, exists func(string) bool) Lookup {
	key := network
	if scope == ScopeAlias {
		key = AliasKey
	}
	lk := Lookup{Key: key}
	h := s.Hosts[alias]
	if h == nil {
		lk.Reason = "no entry for host"
		return lk
	}
	var r *Route
	if scope == ScopeAlias {
		r = h.LastSuccess
	} else if nr, ok := h.Networks[network]; ok {
		r = &nr
	}
	if r == nil {
		lk.Reason = fmt.Sprintf("no entry for network %q", network)
		if scope == ScopeAlias {
			lk.Reason = "no entry for host"
		}
		return lk
	}
	lk.Route = *r
	lk.Status = RouteStatus(h, *r, scope, ttl, now, exists)
	switch lk.Status {
	case StatusFresh:
		lk.Hit = true
		lk.Reason = fmt.Sprintf("profile %q connected on network %q at %s", r.Profile, r.Network, r.SucceededAt.Format(time.RFC3339))
	case StatusExpired:
		lk.Reason = fmt.Sprintf("entry for profile %q expired (last success %s)", r.Profile, r.SucceededAt.Format(time.RFC3339))
	case StatusFailed:
		lk.Reason = fmt.Sprintf("profile %q failed to connect since its last success", r.Profile)
	case StatusUnknownProfile:
		lk.Reason = fmt.Sprintf("profile %q is no longer in the config", r.Profile)
	}
	return lk
}
