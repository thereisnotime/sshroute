package config

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Run("valid config passes", func(t *testing.T) {
		cfg := &Config{
			Networks: map[string]NetworkDefinition{
				"vpn": {
					Priority: 10,
					Checks: []NetworkCheck{
						{Type: CheckTypeInterface, Match: "tun0"},
					},
				},
			},
			Hosts: map[string]HostConfig{
				"myserver": {
					"default": {Host: "1.2.3.4", Port: 22},
				},
			},
		}
		if err := Validate(cfg); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	errCases := []struct {
		name    string
		cfg     *Config
		wantMsg string
	}{
		{
			name: "missing default profile",
			cfg: &Config{
				Hosts: map[string]HostConfig{
					"myserver": {"vpn": {Host: "10.0.0.1"}},
				},
			},
			wantMsg: "default",
		},
		{
			name: "default profile missing host",
			cfg: &Config{
				Hosts: map[string]HostConfig{
					"myserver": {"default": {Port: 22}},
				},
			},
			wantMsg: "host",
		},
		{
			name: "port out of range",
			cfg: &Config{
				Hosts: map[string]HostConfig{
					"myserver": {"default": {Host: "1.2.3.4", Port: 99999}},
				},
			},
			wantMsg: "port",
		},
		{
			name: "route check missing match",
			cfg: &Config{
				Networks: map[string]NetworkDefinition{
					"vpn": {Checks: []NetworkCheck{{Type: CheckTypeRoute}}},
				},
				Hosts: map[string]HostConfig{},
			},
		},
		{
			name: "interface check missing match",
			cfg: &Config{
				Networks: map[string]NetworkDefinition{
					"vpn": {Checks: []NetworkCheck{{Type: CheckTypeInterface}}},
				},
				Hosts: map[string]HostConfig{},
			},
		},
		{
			name: "ping check missing host",
			cfg: &Config{
				Networks: map[string]NetworkDefinition{
					"office": {Checks: []NetworkCheck{{Type: CheckTypePing}}},
				},
				Hosts: map[string]HostConfig{},
			},
		},
		{
			name: "exec check missing command",
			cfg: &Config{
				Networks: map[string]NetworkDefinition{
					"corp": {Checks: []NetworkCheck{{Type: CheckTypeExec}}},
				},
				Hosts: map[string]HostConfig{},
			},
		},
		{
			name: "empty check type",
			cfg: &Config{
				Networks: map[string]NetworkDefinition{
					"vpn": {Checks: []NetworkCheck{{Type: ""}}},
				},
				Hosts: map[string]HostConfig{},
			},
			wantMsg: "type field is required",
		},
		{
			name: "unknown check type",
			cfg: &Config{
				Networks: map[string]NetworkDefinition{
					"vpn": {Checks: []NetworkCheck{{Type: "magic", Match: "x"}}},
				},
				Hosts: map[string]HostConfig{},
			},
		},
		{
			name: "multiple errors reported together",
			cfg: &Config{
				Networks: map[string]NetworkDefinition{
					"vpn": {Checks: []NetworkCheck{{Type: CheckTypeRoute}}}, // missing match
				},
				Hosts: map[string]HostConfig{
					"a": {"vpn": {Host: "1.2.3.4"}},        // missing default
					"b": {"default": {Host: "x", Port: -1}}, // invalid port
				},
			},
		},
	}

	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tc.wantMsg != "" && !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error = %q, want message containing %q", err.Error(), tc.wantMsg)
			}
		})
	}
}
