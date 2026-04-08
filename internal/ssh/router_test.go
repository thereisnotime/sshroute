package ssh

import (
	"testing"

	"github.com/thereisnotime/sshroute/internal/config"
)

func makeConfig(hosts map[string]config.HostConfig) *config.Config {
	return &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    hosts,
	}
}

func mustResolve(t *testing.T, cfg *config.Config, host, network string) config.SSHParams {
	t.Helper()
	params, err := Resolve(cfg, host, network)
	if err != nil {
		t.Fatalf("Resolve(%q, %q): unexpected error: %v", host, network, err)
	}
	return params
}

func TestResolve(t *testing.T) {
	cfg := makeConfig(map[string]config.HostConfig{
		"webserver": {
			"default": {Host: "1.2.3.4", Port: 22, User: "deploy", Key: "~/.ssh/id_ed25519"},
			"vpn":     {Host: "10.8.0.50", Port: 2222, Key: "~/.ssh/vpn_key", Jump: "bastion.vpn"},
		},
	})

	t.Run("default network", func(t *testing.T) {
		params := mustResolve(t, cfg, "webserver", "default")
		if params.Host != "1.2.3.4" {
			t.Errorf("Host = %q, want %q", params.Host, "1.2.3.4")
		}
		if params.Port != 22 {
			t.Errorf("Port = %d, want 22", params.Port)
		}
		if params.User != "deploy" {
			t.Errorf("User = %q, want %q", params.User, "deploy")
		}
	})

	t.Run("vpn network overrides", func(t *testing.T) {
		params := mustResolve(t, cfg, "webserver", "vpn")
		if params.Host != "10.8.0.50" {
			t.Errorf("Host = %q, want %q", params.Host, "10.8.0.50")
		}
		if params.Port != 2222 {
			t.Errorf("Port = %d, want 2222", params.Port)
		}
		// User not set in vpn profile — should inherit from default
		if params.User != "deploy" {
			t.Errorf("User = %q, want inherited %q", params.User, "deploy")
		}
		if params.Jump != "bastion.vpn" {
			t.Errorf("Jump = %q, want %q", params.Jump, "bastion.vpn")
		}
	})

	t.Run("unknown network falls back to default", func(t *testing.T) {
		params := mustResolve(t, cfg, "webserver", "office")
		if params.Host != "1.2.3.4" {
			t.Errorf("Host = %q, want %q", params.Host, "1.2.3.4")
		}
	})

	t.Run("unknown host returns error", func(t *testing.T) {
		_, err := Resolve(cfg, "notahost", "default")
		if err == nil {
			t.Error("expected error for unknown host, got nil")
		}
	})

	t.Run("missing default profile returns error", func(t *testing.T) {
		badCfg := makeConfig(map[string]config.HostConfig{
			"broken": {"vpn": {Host: "10.0.0.1"}},
		})
		_, err := Resolve(badCfg, "broken", "default")
		if err == nil {
			t.Error("expected error for missing default profile, got nil")
		}
	})

	t.Run("vpn override with user", func(t *testing.T) {
		// Exercises the merged.User = override.User assignment branch.
		cfg2 := makeConfig(map[string]config.HostConfig{
			"srv": {
				"default": {Host: "1.2.3.4", User: "alice"},
				"vpn":     {Host: "10.0.0.1", User: "root"},
			},
		})
		params := mustResolve(t, cfg2, "srv", "vpn")
		if params.User != "root" {
			t.Errorf("User = %q, want %q", params.User, "root")
		}
	})

	t.Run("empty network string falls back to default", func(t *testing.T) {
		params := mustResolve(t, cfg, "webserver", "")
		if params.Host != "1.2.3.4" {
			t.Errorf("Host = %q, want %q", params.Host, "1.2.3.4")
		}
	})
}

func TestBuildArgv(t *testing.T) {
	tests := []struct {
		name   string
		params config.SSHParams
		parsed ParsedArgs
		want   []string
	}{
		{
			name:   "full params",
			params: config.SSHParams{Host: "10.0.0.1", Port: 2222, User: "alice", Key: "/home/alice/.ssh/key", Jump: "bastion"},
			parsed: ParsedArgs{Remaining: []string{}},
			want:   []string{RealSSH, "-p", "2222", "-i", "/home/alice/.ssh/key", "-l", "alice", "-J", "bastion", "10.0.0.1"},
		},
		{
			name:   "minimal params",
			params: config.SSHParams{Host: "myserver"},
			parsed: ParsedArgs{Remaining: []string{}},
			want:   []string{RealSSH, "myserver"},
		},
		{
			name:   "parsed user fallback",
			params: config.SSHParams{Host: "myserver"},
			parsed: ParsedArgs{User: "bob", Remaining: []string{}},
			want:   []string{RealSSH, "-l", "bob", "myserver"},
		},
		{
			name:   "params user wins over parsed user",
			params: config.SSHParams{Host: "myserver", User: "alice"},
			parsed: ParsedArgs{User: "bob", Remaining: []string{}},
			want:   []string{RealSSH, "-l", "alice", "myserver"},
		},
		{
			name:   "remaining args appended",
			params: config.SSHParams{Host: "myserver"},
			parsed: ParsedArgs{Remaining: []string{"-v", "uptime"}},
			want:   []string{RealSSH, "myserver", "-v", "uptime"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildArgv(tt.params, tt.parsed)
			if len(got) != len(tt.want) {
				t.Fatalf("argv len = %d, want %d\ngot:  %v\nwant: %v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("argv[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
