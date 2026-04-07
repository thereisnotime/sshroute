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

	t.Run("missing default profile", func(t *testing.T) {
		cfg := &Config{
			Hosts: map[string]HostConfig{
				"myserver": {"vpn": {Host: "10.0.0.1"}},
			},
		}
		err := Validate(cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "default") {
			t.Errorf("error should mention 'default', got: %v", err)
		}
	})

	t.Run("default profile missing host", func(t *testing.T) {
		cfg := &Config{
			Hosts: map[string]HostConfig{
				"myserver": {"default": {Port: 22}},
			},
		}
		err := Validate(cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "host") {
			t.Errorf("error should mention 'host', got: %v", err)
		}
	})

	t.Run("port out of range", func(t *testing.T) {
		cfg := &Config{
			Hosts: map[string]HostConfig{
				"myserver": {"default": {Host: "1.2.3.4", Port: 99999}},
			},
		}
		err := Validate(cfg)
		if err == nil {
			t.Fatal("expected error for out-of-range port")
		}
		if !strings.Contains(err.Error(), "port") {
			t.Errorf("error should mention 'port', got: %v", err)
		}
	})

	t.Run("route check missing match", func(t *testing.T) {
		cfg := &Config{
			Networks: map[string]NetworkDefinition{
				"vpn": {Checks: []NetworkCheck{{Type: CheckTypeRoute}}},
			},
			Hosts: map[string]HostConfig{},
		}
		err := Validate(cfg)
		if err == nil {
			t.Fatal("expected error for route check without match")
		}
	})

	t.Run("interface check missing match", func(t *testing.T) {
		cfg := &Config{
			Networks: map[string]NetworkDefinition{
				"vpn": {Checks: []NetworkCheck{{Type: CheckTypeInterface}}},
			},
			Hosts: map[string]HostConfig{},
		}
		err := Validate(cfg)
		if err == nil {
			t.Fatal("expected error for interface check without match")
		}
	})

	t.Run("ping check missing host", func(t *testing.T) {
		cfg := &Config{
			Networks: map[string]NetworkDefinition{
				"office": {Checks: []NetworkCheck{{Type: CheckTypePing}}},
			},
			Hosts: map[string]HostConfig{},
		}
		err := Validate(cfg)
		if err == nil {
			t.Fatal("expected error for ping check without host")
		}
	})

	t.Run("exec check missing command", func(t *testing.T) {
		cfg := &Config{
			Networks: map[string]NetworkDefinition{
				"corp": {Checks: []NetworkCheck{{Type: CheckTypeExec}}},
			},
			Hosts: map[string]HostConfig{},
		}
		err := Validate(cfg)
		if err == nil {
			t.Fatal("expected error for exec check without command")
		}
	})

	t.Run("unknown check type", func(t *testing.T) {
		cfg := &Config{
			Networks: map[string]NetworkDefinition{
				"vpn": {Checks: []NetworkCheck{{Type: "magic", Match: "x"}}},
			},
			Hosts: map[string]HostConfig{},
		}
		err := Validate(cfg)
		if err == nil {
			t.Fatal("expected error for unknown check type")
		}
	})

	t.Run("multiple errors reported together", func(t *testing.T) {
		cfg := &Config{
			Networks: map[string]NetworkDefinition{
				"vpn": {Checks: []NetworkCheck{{Type: CheckTypeRoute}}}, // missing match
			},
			Hosts: map[string]HostConfig{
				"a": {"vpn": {Host: "1.2.3.4"}},       // missing default
				"b": {"default": {Host: "x", Port: -1}}, // invalid port
			},
		}
		err := Validate(cfg)
		if err == nil {
			t.Fatal("expected multiple errors, got nil")
		}
	})
}
