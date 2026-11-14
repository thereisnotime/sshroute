// Package config defines the sshroute configuration structures and constants.
package config

// CheckType represents the kind of network detection check.
type CheckType string

const (
	CheckTypeRoute     CheckType = "route"
	CheckTypeInterface CheckType = "interface"
	CheckTypePing      CheckType = "ping"
	CheckTypeExec      CheckType = "exec"
)

// NetworkCheck is a single detection rule within a network definition.
type NetworkCheck struct {
	Type    CheckType `yaml:"type"`
	Match   string    `yaml:"match,omitempty"`   // route: subnet string; interface: iface name
	Host    string    `yaml:"host,omitempty"`    // ping: target host
	Timeout string    `yaml:"timeout,omitempty"` // ping: duration string e.g. "1s"
	Command string    `yaml:"command,omitempty"` // exec: shell command
}

// SSHParams holds the resolved SSH connection parameters for one profile.
type SSHParams struct {
	Host         string            `yaml:"host,omitempty"`
	Port         int               `yaml:"port,omitempty"`
	User         string            `yaml:"user,omitempty"`
	Key          string            `yaml:"key,omitempty"`
	Jump         string            `yaml:"jump,omitempty"`
	Comment      string            `yaml:"comment,omitempty"`
	Tags         []string          `yaml:"tags,omitempty"`
	Options      map[string]string `yaml:"options,omitempty"`
	ResolvedJump *SSHParams        `yaml:"-"`
}

// HostConfig maps network profile names (including "default") to SSHParams.
// The "default" key is required and used as a fallback.
type HostConfig map[string]SSHParams

// NetworkDefinition groups the detection checks for one named network together
// with an optional priority that controls evaluation order.
// Lower priority values are evaluated first. The default is 0.
type NetworkDefinition struct {
	Priority int            `yaml:"priority,omitempty"`
	Comment  string         `yaml:"comment,omitempty"`
	Tags     []string       `yaml:"tags,omitempty"`
	Checks   []NetworkCheck `yaml:"checks"`
}

// Config is the top-level sshroute configuration structure.
type Config struct {
	Networks  map[string]NetworkDefinition `yaml:"networks"`
	Hosts     map[string]HostConfig        `yaml:"hosts"`
	SSHBinary string                       `yaml:"ssh_binary,omitempty"`
	Cache     CacheConfig                  `yaml:"cache,omitempty"`
}

// Route cache scopes. See CacheConfig.Scope.
const (
	CacheScopeNetwork = "network"
	CacheScopeAlias   = "alias"
)

// CacheConfig controls the fallback route cache. Every field is optional; the
// zero value means "use the default" (cache on, default path, 7d TTL, network scope).
type CacheConfig struct {
	// Enabled turns the cache on or off. nil means on.
	Enabled *bool `yaml:"enabled,omitempty"`
	// Path is the state file location. Empty means the platform default.
	Path string `yaml:"path,omitempty"`
	// TTL is how long a successful route is trusted, e.g. "7d", "12h". "0" never expires.
	TTL string `yaml:"ttl,omitempty"`
	// Scope is "network" (key by alias + detected network) or "alias" (key by alias only).
	Scope string `yaml:"scope,omitempty"`
	// LockTimeout bounds how long a write waits for the state file lock before giving up.
	LockTimeout string `yaml:"lock_timeout,omitempty"`
}
