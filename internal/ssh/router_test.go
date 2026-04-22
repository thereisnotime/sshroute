package ssh

import (
	"testing"

	"github.com/thereisnotime/sshroute/internal/config"
)

func expandTildeForTest(path string) string {
	return expandTilde(path)
}

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

	t.Run("vpn override with comment", func(t *testing.T) {
		cfg2 := makeConfig(map[string]config.HostConfig{
			"srv": {
				"default": {Host: "1.2.3.4", Comment: "default comment"},
				"vpn":     {Host: "10.0.0.1", Comment: "vpn comment"},
			},
		})
		params := mustResolve(t, cfg2, "srv", "vpn")
		if params.Comment != "vpn comment" {
			t.Errorf("Comment = %q, want %q", params.Comment, "vpn comment")
		}
	})

	t.Run("vpn override with tags", func(t *testing.T) {
		cfg2 := makeConfig(map[string]config.HostConfig{
			"srv": {
				"default": {Host: "1.2.3.4", Tags: []string{"prod"}},
				"vpn":     {Host: "10.0.0.1", Tags: []string{"internal", "secure"}},
			},
		})
		params := mustResolve(t, cfg2, "srv", "vpn")
		if len(params.Tags) != 2 || params.Tags[0] != "internal" {
			t.Errorf("Tags = %v, want [internal secure]", params.Tags)
		}
	})

	t.Run("vpn override with jump", func(t *testing.T) {
		cfg2 := makeConfig(map[string]config.HostConfig{
			"srv": {
				"default": {Host: "1.2.3.4", Jump: "bastion1"},
				"vpn":     {Host: "10.0.0.1", Jump: "bastion2"},
			},
		})
		params := mustResolve(t, cfg2, "srv", "vpn")
		if params.Jump != "bastion2" {
			t.Errorf("Jump = %q, want %q", params.Jump, "bastion2")
		}
	})
}

func TestResolveJumpAlias(t *testing.T) {
	cfg := makeConfig(map[string]config.HostConfig{
		"gateway": {
			"default": {Host: "gw.example.com", Port: 22, User: "admin", Key: "~/.ssh/gw_key"},
			"vpn":     {Host: "10.0.0.1"},
		},
		"backend": {
			"default": {Host: "192.168.1.10", Port: 2222, User: "root", Key: "~/.ssh/backend_key", Jump: "gateway"},
		},
	})

	t.Run("jump alias resolved on default network", func(t *testing.T) {
		params := mustResolve(t, cfg, "backend", "default")
		if params.Jump != "gateway" {
			t.Errorf("Jump = %q, want %q", params.Jump, "gateway")
		}
		if params.ResolvedJump == nil {
			t.Fatal("ResolvedJump is nil, want resolved gateway params")
		}
		if params.ResolvedJump.Host != "gw.example.com" {
			t.Errorf("ResolvedJump.Host = %q, want %q", params.ResolvedJump.Host, "gw.example.com")
		}
		if params.ResolvedJump.User != "admin" {
			t.Errorf("ResolvedJump.User = %q, want %q", params.ResolvedJump.User, "admin")
		}
		if params.ResolvedJump.Key != "~/.ssh/gw_key" {
			t.Errorf("ResolvedJump.Key = %q, want %q", params.ResolvedJump.Key, "~/.ssh/gw_key")
		}
	})

	t.Run("jump alias resolved with network override", func(t *testing.T) {
		params := mustResolve(t, cfg, "backend", "vpn")
		if params.ResolvedJump == nil {
			t.Fatal("ResolvedJump is nil, want resolved gateway params")
		}
		if params.ResolvedJump.Host != "10.0.0.1" {
			t.Errorf("ResolvedJump.Host = %q, want %q", params.ResolvedJump.Host, "10.0.0.1")
		}
	})

	t.Run("non-alias jump is not resolved", func(t *testing.T) {
		cfg2 := makeConfig(map[string]config.HostConfig{
			"srv": {
				"default": {Host: "srv.example.com", Jump: "bastion.example.com"},
			},
		})
		params := mustResolve(t, cfg2, "srv", "default")
		if params.Jump != "bastion.example.com" {
			t.Errorf("Jump = %q, want %q", params.Jump, "bastion.example.com")
		}
		if params.ResolvedJump != nil {
			t.Error("ResolvedJump should be nil for non-alias jump")
		}
	})

	t.Run("circular jump chain returns error", func(t *testing.T) {
		cfg3 := makeConfig(map[string]config.HostConfig{
			"a": {"default": {Host: "a.example.com", Jump: "b"}},
			"b": {"default": {Host: "b.example.com", Jump: "a"}},
		})
		_, err := Resolve(cfg3, "a", "default")
		if err == nil {
			t.Error("expected circular jump chain error, got nil")
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
		{
			name: "resolved jump uses ProxyCommand",
			params: config.SSHParams{
				Host: "192.168.1.10", Port: 2222, User: "root", Key: "/home/alice/.ssh/bk",
				Jump:         "gateway",
				ResolvedJump: &config.SSHParams{Host: "gw.example.com", Port: 22, User: "admin", Key: "~/.ssh/gw_key"},
			},
			parsed: ParsedArgs{Remaining: []string{}},
			want:   []string{RealSSH, "-p", "2222", "-i", "/home/alice/.ssh/bk", "-l", "root", "-o", "ProxyCommand=" + RealSSH + " -i " + expandTildeForTest("~/.ssh/gw_key") + " -p 22 -W %h:%p admin@gw.example.com", "192.168.1.10"},
		},
		{
			name: "resolved jump minimal (host only)",
			params: config.SSHParams{
				Host:         "target.internal",
				ResolvedJump: &config.SSHParams{Host: "jump.example.com"},
			},
			parsed: ParsedArgs{Remaining: []string{}},
			want:   []string{RealSSH, "-o", "ProxyCommand=" + RealSSH + " -W %h:%p jump.example.com", "target.internal"},
		},
		{
			name: "nested resolved jump (multi-hop)",
			params: config.SSHParams{
				Host: "deep.internal", User: "root",
				ResolvedJump: &config.SSHParams{
					Host: "mid.example.com", Port: 22, User: "hop2", Key: "/k2",
					ResolvedJump: &config.SSHParams{Host: "edge.example.com", Port: 443, User: "hop1", Key: "/k1"},
				},
			},
			parsed: ParsedArgs{Remaining: []string{}},
			want:   []string{RealSSH, "-l", "root", "-o", "ProxyCommand=" + RealSSH + " -i /k2 -p 22 -o ProxyCommand=" + RealSSH + " -i /k1 -p 443 -W %h:%p hop1@edge.example.com -W %h:%p hop2@mid.example.com", "deep.internal"},
		},
		{
			name: "resolved jump with raw jump on inner hop",
			params: config.SSHParams{
				Host: "target.internal",
				ResolvedJump: &config.SSHParams{
					Host: "mid.example.com", User: "admin",
					Jump: "external-bastion.example.com",
				},
			},
			parsed: ParsedArgs{Remaining: []string{}},
			want:   []string{RealSSH, "-o", "ProxyCommand=" + RealSSH + " -J external-bastion.example.com -W %h:%p admin@mid.example.com", "target.internal"},
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
