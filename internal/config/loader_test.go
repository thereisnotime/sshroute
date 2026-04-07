package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	t.Run("uses SSHROUTE_CONFIG env var", func(t *testing.T) {
		t.Setenv("SSHROUTE_CONFIG", "/tmp/my-sshroute.yaml")
		path, err := DefaultConfigPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if path != "/tmp/my-sshroute.yaml" {
			t.Errorf("path = %q, want %q", path, "/tmp/my-sshroute.yaml")
		}
	})

	t.Run("uses XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("SSHROUTE_CONFIG", "")
		t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
		path, err := DefaultConfigPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "/tmp/xdg/sshroute/config.yaml"
		if path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
	})

	t.Run("falls back to home directory", func(t *testing.T) {
		t.Setenv("SSHROUTE_CONFIG", "")
		t.Setenv("XDG_CONFIG_HOME", "")
		path, err := DefaultConfigPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".config", "sshroute", "config.yaml")
		if path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("missing file returns empty config", func(t *testing.T) {
		cfg, err := Load("/nonexistent/path/config.yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil config")
		}
		if len(cfg.Hosts) != 0 {
			t.Errorf("expected empty hosts, got %d", len(cfg.Hosts))
		}
		if len(cfg.Networks) != 0 {
			t.Errorf("expected empty networks, got %d", len(cfg.Networks))
		}
	})

	t.Run("valid YAML is parsed correctly", func(t *testing.T) {
		content := `
networks:
  vpn:
    priority: 10
    checks:
      - type: interface
        match: tun0
hosts:
  myserver:
    default:
      host: 1.2.3.4
      port: 22
      user: alice
`
		f := writeTempConfig(t, content)
		cfg, err := Load(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Hosts) != 1 {
			t.Errorf("expected 1 host, got %d", len(cfg.Hosts))
		}
		host := cfg.Hosts["myserver"]["default"]
		if host.Host != "1.2.3.4" {
			t.Errorf("host = %q, want %q", host.Host, "1.2.3.4")
		}
		if host.User != "alice" {
			t.Errorf("user = %q, want %q", host.User, "alice")
		}
		vpn := cfg.Networks["vpn"]
		if vpn.Priority != 10 {
			t.Errorf("priority = %d, want 10", vpn.Priority)
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		f := writeTempConfig(t, "{{invalid yaml{{")
		_, err := Load(f)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})

	t.Run("invalid config fails validation", func(t *testing.T) {
		content := `
hosts:
  myserver:
    vpn:
      host: 1.2.3.4
`
		f := writeTempConfig(t, content)
		_, err := Load(f)
		if err == nil {
			t.Fatal("expected validation error for missing default profile")
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("saves and reloads correctly", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")

		cfg := &Config{
			Networks: map[string]NetworkDefinition{
				"vpn": {Priority: 5, Checks: []NetworkCheck{{Type: CheckTypeInterface, Match: "wg0"}}},
			},
			Hosts: map[string]HostConfig{
				"myserver": {"default": {Host: "1.2.3.4", Port: 22, User: "root"}},
			},
		}

		if err := Save(path, cfg); err != nil {
			t.Fatalf("Save error: %v", err)
		}

		loaded, err := Load(path)
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		if loaded.Hosts["myserver"]["default"].Host != "1.2.3.4" {
			t.Errorf("host mismatch after save/load")
		}
		if loaded.Networks["vpn"].Priority != 5 {
			t.Errorf("priority mismatch after save/load")
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "nested", "deep", "config.yaml")
		cfg := &Config{
			Networks: make(map[string]NetworkDefinition),
			Hosts:    map[string]HostConfig{"h": {"default": {Host: "x"}}},
		}
		if err := Save(path, cfg); err != nil {
			t.Fatalf("Save error: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("file not created: %v", err)
		}
	})
}

func TestExpandTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		input string
		want  string
	}{
		{"~/.ssh/key", filepath.Join(home, ".ssh/key")},
		{"~/", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}
	for _, tt := range tests {
		got, err := expandTilde(tt.input)
		if err != nil {
			t.Errorf("expandTilde(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("expandTilde(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLoad_ExpandsKeyTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	content := `
hosts:
  myserver:
    default:
      host: 1.2.3.4
      key: ~/.ssh/id_ed25519
`
	f := writeTempConfig(t, content)
	cfg, err := Load(f)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	want := filepath.Join(home, ".ssh/id_ed25519")
	got := cfg.Hosts["myserver"]["default"].Key
	if got != want {
		t.Errorf("key = %q, want %q (tilde not expanded)", got, want)
	}
}

func TestSave_EmptyPath(t *testing.T) {
	// Save with empty path resolves to DefaultConfigPath.
	// We override via env so it goes somewhere temp.
	dir := t.TempDir()
	t.Setenv("SSHROUTE_CONFIG", filepath.Join(dir, "config.yaml"))

	cfg := &Config{
		Networks: make(map[string]NetworkDefinition),
		Hosts:    map[string]HostConfig{"h": {"default": {Host: "x"}}},
	}
	if err := Save("", cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `
hosts:
  s:
    default:
      host: 1.2.3.4
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("SSHROUTE_CONFIG", cfgPath)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Hosts["s"]["default"].Host != "1.2.3.4" {
		t.Errorf("host mismatch")
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "sshroute-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}
