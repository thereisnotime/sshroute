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

func TestRunAdd_DefaultNetwork(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})

	addHost = "1.2.3.4"
	addPort = 22
	addUser = "alice"
	addKey = ""
	addJump = ""
	addNetwork = "default"
	defer func() { addHost = ""; addPort = 22; addUser = ""; addNetwork = "default" }()

	if err := runAdd(addCmd, []string{"newserver"}); err != nil {
		t.Fatalf("runAdd error: %v", err)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}
	if cfg.Hosts["newserver"]["default"].Host != "1.2.3.4" {
		t.Errorf("host not saved correctly")
	}
}

func TestRunAdd_NonDefaultNetwork_SeedsDefault(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})

	addHost = "10.0.0.1"
	addPort = 2222
	addUser = "root"
	addKey = ""
	addJump = ""
	addNetwork = "vpn"
	defer func() { addHost = ""; addPort = 22; addUser = ""; addNetwork = "default" }()

	if err := runAdd(addCmd, []string{"newserver"}); err != nil {
		t.Fatalf("runAdd error: %v", err)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}
	// Should have auto-seeded a default profile.
	if _, ok := cfg.Hosts["newserver"]["default"]; !ok {
		t.Error("expected default profile to be auto-seeded")
	}
	if cfg.Hosts["newserver"]["vpn"].Host != "10.0.0.1" {
		t.Errorf("vpn profile not saved correctly")
	}
}

func TestRunAdd_UpdateExisting(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "old.host"}},
		},
	})

	addHost = "new.host"
	addPort = 22
	addUser = ""
	addKey = ""
	addJump = ""
	addNetwork = "default"
	defer func() { addHost = ""; addNetwork = "default" }()

	if err := runAdd(addCmd, []string{"srv"}); err != nil {
		t.Fatalf("runAdd error: %v", err)
	}

	cfg, _ := config.Load(cfgFile)
	if cfg.Hosts["srv"]["default"].Host != "new.host" {
		t.Errorf("expected host to be updated")
	}
}

func TestRunConnect_UnknownHost(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})

	err := runConnect(connectCmd, []string{"doesnotexist"})
	if err == nil {
		t.Error("expected error for unknown host")
	}
}

func TestRunConnect_DryRun(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"myserver": {"default": {Host: "1.2.3.4", Port: 22, User: "alice"}},
		},
	})

	dryRun = true
	defer func() { dryRun = false }()

	if err := runConnect(connectCmd, []string{"myserver"}); err != nil {
		t.Fatalf("runConnect dry-run error: %v", err)
	}
}

func TestRunNetwork_EmptyConfig(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})

	if err := runNetwork(networkCmd, nil); err != nil {
		t.Fatalf("runNetwork error: %v", err)
	}
}

func TestRunNetworkTest_NotFound(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})

	err := runNetworkTest(networkTestCmd, []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown network")
	}
}

func TestRunNetworkTest_NoChecks(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"empty": {Priority: 10},
		},
		Hosts: make(map[string]config.HostConfig),
	})

	if err := runNetworkTest(networkTestCmd, []string{"empty"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunConfigEdit_EditorNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfgFile = path
	t.Setenv("EDITOR", "nonexistent_editor_xyz_abc")

	err := runConfigEdit(configEditCmd, nil)
	if err == nil {
		t.Fatal("expected error when editor binary does not exist")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("error = %q, want 'not found in PATH'", err.Error())
	}
}

func TestRunConfigEdit_EditorNotFound_FileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Pre-create the file so the os.Stat branch sees it as existing.
	if err := os.WriteFile(path, []byte("# existing config\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	cfgFile = path
	t.Setenv("EDITOR", "nonexistent_editor_xyz_abc")

	err := runConfigEdit(configEditCmd, nil)
	if err == nil {
		t.Fatal("expected error when editor binary does not exist")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("error = %q, want 'not found in PATH'", err.Error())
	}
}

func TestRunConfig_DefaultConfigPath(t *testing.T) {
	// With cfgFile unset, runConfig must fall back to DefaultConfigPath which
	// reads SSHROUTE_CONFIG (or the OS default). Point it at a temp path so
	// the test is hermetic and the printed path is predictable.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = "" // force the DefaultConfigPath() branch

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := runConfig(configCmd, nil)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("runConfig error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), path) {
		t.Errorf("output %q does not contain path %q", buf.String(), path)
	}
}

func TestRunNetworkTest_WithChecks(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"local": {
				Priority: 10,
				Checks: []config.NetworkCheck{
					{Type: config.CheckTypeExec, Command: "true"},
				},
			},
		},
		Hosts: make(map[string]config.HostConfig),
	})

	if err := runNetworkTest(networkTestCmd, []string{"local"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
