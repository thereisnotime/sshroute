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
	Host         string     `yaml:"host,omitempty"`
	Port         int        `yaml:"port,omitempty"`
	User         string     `yaml:"user,omitempty"`
	Key          string     `yaml:"key,omitempty"`
	Jump         string     `yaml:"jump,omitempty"`
	ResolvedJump *SSHParams `yaml:"-"`
}

// HostConfig maps network profile names (including "default") to SSHParams.
// The "default" key is required and used as a fallback.
type HostConfig map[string]SSHParams

// NetworkDefinition groups the detection checks for one named network together
// with an optional priority that controls evaluation order.
// Lower priority values are evaluated first. The default is 0.
type NetworkDefinition struct {
	Priority int            `yaml:"priority,omitempty"`
	Checks   []NetworkCheck `yaml:"checks"`
}

// Config is the top-level sshroute configuration structure.
type Config struct {
	Networks map[string]NetworkDefinition `yaml:"networks"`
	Hosts    map[string]HostConfig        `yaml:"hosts"`
}
