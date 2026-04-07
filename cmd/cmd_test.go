package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thereisnotime/sshroute/internal/config"
)


func withTempConfig(t *testing.T, cfg *config.Config) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("saving temp config: %v", err)
	}
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path
	return path
}

func TestCheckRuleString(t *testing.T) {
	tests := []struct {
		check config.NetworkCheck
		want  string
	}{
		{config.NetworkCheck{Match: "wg0"}, "wg0"},
		{config.NetworkCheck{Host: "192.168.1.1"}, "192.168.1.1"},
		{config.NetworkCheck{Command: "true"}, "true"},
		{config.NetworkCheck{Command: strings.Repeat("x", 50)}, strings.Repeat("x", 37) + "..."},
		{config.NetworkCheck{}, ""},
	}
	for _, tt := range tests {
		got := checkRuleString(tt.check)
		if got != tt.want {
			t.Errorf("checkRuleString(%+v) = %q, want %q", tt.check, got, tt.want)
		}
	}
}

func TestRunInit_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path

	// Remove file first to ensure it doesn't exist.
	os.Remove(path)

	err := runInit(initCmd, nil)
	if err != nil {
		t.Fatalf("runInit error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestRunInit_FailsIfExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path
	initForce = false

	err := runInit(initCmd, nil)
	if err == nil {
		t.Error("expected error when config already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want 'already exists'", err.Error())
	}
}

func TestRunInit_Force(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("old content"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path
	initForce = true
	defer func() { initForce = false }()

	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit --force error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "old content") {
		t.Error("expected old content to be overwritten")
	}
}

func TestRunConfig_PrintsPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	_ = runConfig(configCmd, nil)
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), path) {
		t.Errorf("output %q does not contain path %q", buf.String(), path)
	}
}

func TestRunNetworkList_EmptyConfig(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})

	buf := new(bytes.Buffer)
	networkListCmd.SetOut(buf)
	if err := runNetworkList(networkListCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunNetworkList_WithNetworks(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"vpn": {
				Priority: 10,
				Checks:   []config.NetworkCheck{{Type: config.CheckTypeInterface, Match: "wg0"}},
			},
		},
		Hosts: make(map[string]config.HostConfig),
	})

	if err := runNetworkList(networkListCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunList_EmptyConfig(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})

	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunList_WithHosts(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"myserver": {
				"default": {Host: "1.2.3.4", Port: 22, User: "alice"},
			},
		},
	})

	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRemove_ExistingHost(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"myserver": {"default": {Host: "1.2.3.4"}},
		},
	})

	if err := runRemove(removeCmd, []string{"myserver"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRemove_MissingHost(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})

	err := runRemove(removeCmd, []string{"doesnotexist"})
	if err == nil {
		t.Error("expected error when removing non-existent host")
	}
}
