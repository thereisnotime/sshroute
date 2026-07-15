package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
	"github.com/thereisnotime/sshroute/internal/ssh"
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

func withInvalidConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml{{"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path
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

func TestRunConnect_ReconnectDryRun(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"myserver": {"default": {Host: "1.2.3.4", Port: 22, User: "alice"}},
		},
	})

	reconnect = true
	dryRun = true
	defer func() { reconnect = false; dryRun = false }()

	// Dry-run must print the resolved command and return without entering the loop.
	if err := runConnect(connectCmd, []string{"myserver"}); err != nil {
		t.Fatalf("runConnect --reconnect --dry-run error: %v", err)
	}
}

func TestRunConnect_ReconnectDryRunFallback(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"myserver": {
				"default": {Host: "1.2.3.4", Port: 22, User: "alice"},
				"vpn":     {Host: "10.0.0.1", Port: 22, User: "alice"},
			},
		},
	})

	reconnect = true
	fallback = true
	dryRun = true
	defer func() { reconnect = false; fallback = false; dryRun = false }()

	if err := runConnect(connectCmd, []string{"myserver"}); err != nil {
		t.Fatalf("runConnect --reconnect --fallback --dry-run error: %v", err)
	}
}

func TestConnectCmd_ReconnectFlags(t *testing.T) {
	if connectCmd.Flags().Lookup("reconnect") == nil {
		t.Error("connect command is missing the --reconnect flag")
	}
	if connectCmd.Flags().Lookup("reconnect-delay") == nil {
		t.Error("connect command is missing the --reconnect-delay flag")
	}
}

// TestRunConnectReconnect_ResolveError drives the non-dry-run reconnect path (signal
// context, attempt closure, Supervise) to its error exit: a host with no profile for
// the detected network makes the first attempt fail to resolve, so the loop returns
// the error immediately instead of spinning.
func TestRunConnectReconnect_ResolveError(t *testing.T) {
	dryRun = false
	fallback = false
	cfg := &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"vpn": {Host: "1.2.3.4"}}, // no "default" profile for the detected network
		},
	}
	if err := runConnectReconnect(cfg, "srv", ssh.ParsedArgs{}); err == nil {
		t.Error("expected a resolve error in reconnect mode")
	}
}

// TestRunConnectReconnect_CleanExit drives the non-dry-run reconnect happy path: a
// single-route attempt whose (stubbed) ssh exits 0 must stop supervision and return
// nil without reconnecting.
func TestRunConnectReconnect_CleanExit(t *testing.T) {
	dryRun = false
	fallback = false
	old := ssh.RealSSH
	ssh.RealSSH = "/bin/true"
	defer func() { ssh.RealSSH = old }()
	cfg := &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Port: 22, User: "u"}},
		},
	}
	if err := runConnectReconnect(cfg, "srv", ssh.ParsedArgs{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAttemptOnce_SingleRouteSuccess(t *testing.T) {
	cfg := &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Port: 22, User: "u"}},
		},
	}
	old := ssh.RealSSH
	ssh.RealSSH = "/bin/true"
	fallback = false
	defer func() { ssh.RealSSH = old; fallback = false }()

	code, err := attemptOnce(context.Background(), cfg, "srv", "default", ssh.ParsedArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("code = %d, want 0", code)
	}
}

func TestAttemptOnce_SingleRouteResolveError(t *testing.T) {
	cfg := &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"vpn": {Host: "1.2.3.4"}}, // no "default" profile to resolve
		},
	}
	fallback = false
	defer func() { fallback = false }()

	if _, err := attemptOnce(context.Background(), cfg, "srv", "default", ssh.ParsedArgs{}); err == nil {
		t.Error("expected resolve error for missing default profile")
	}
}

func TestAttemptOnce_FallbackNonRetryable(t *testing.T) {
	cfg := &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Port: 22, User: "u"}},
		},
	}
	old := ssh.RealSSH
	ssh.RealSSH = "/bin/false" // exit 1 — a non-255 failure must stop, not retry
	fallback = true
	defer func() { ssh.RealSSH = old; fallback = false }()

	code, err := attemptOnce(context.Background(), cfg, "srv", "default", ssh.ParsedArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
}

func TestAttemptOnce_FallbackAllConnectFail(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "ssh255.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 255\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.Chmod(script, 0o700); err != nil {
		t.Fatalf("setup chmod: %v", err)
	}
	cfg := &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {
				"default": {Host: "1.2.3.4", Port: 22, User: "u"},
				"vpn":     {Host: "10.0.0.1", Port: 22, User: "u"},
			},
		},
	}
	old := ssh.RealSSH
	ssh.RealSSH = script
	fallback = true
	defer func() { ssh.RealSSH = old; fallback = false }()

	code, err := attemptOnce(context.Background(), cfg, "srv", "default", ssh.ParsedArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != ssh.SSHConnectFailure {
		t.Errorf("code = %d, want %d", code, ssh.SSHConnectFailure)
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

func TestRunNetworkTest_FailingCheck(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"nope": {
				Priority: 10,
				Checks: []config.NetworkCheck{
					{Type: config.CheckTypeExec, Command: "false"},
				},
			},
		},
		Hosts: make(map[string]config.HostConfig),
	})

	// A failing check must still complete without error — it just prints FAIL/NOT ACTIVE.
	if err := runNetworkTest(networkTestCmd, []string{"nope"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunNetworkTest_ErrorCheck(t *testing.T) {
	// A ping check with an invalid timeout passes config validation (host is
	// non-empty) but Detect returns an error at runtime — covering the
	// "ERROR:" stderr branch in runNetworkTest.
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"bad": {
				Priority: 10,
				Checks: []config.NetworkCheck{
					{Type: config.CheckTypePing, Host: "127.0.0.1", Timeout: "not_a_duration"},
				},
			},
		},
		Hosts: make(map[string]config.HostConfig),
	})

	if err := runNetworkTest(networkTestCmd, []string{"bad"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunNetworkList_NoChecksNetwork(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"bare": {Priority: 5},
		},
		Hosts: make(map[string]config.HostConfig),
	})

	if err := runNetworkList(networkListCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunNetworkList_JSONOutput(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"vpn": {Priority: 10, Checks: []config.NetworkCheck{{Type: config.CheckTypeExec, Command: "true"}}},
		},
		Hosts: make(map[string]config.HostConfig),
	})
	old := output
	output = "json"
	defer func() { output = old }()

	if err := runNetworkList(networkListCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunList_JSONOutput(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Port: 22, User: "alice"}},
		},
	})
	old := output
	output = "json"
	defer func() { output = old }()

	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunList_YAMLOutput(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Port: 22, User: "alice"}},
		},
	})
	old := output
	output = "yaml"
	defer func() { output = old }()

	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunList_DetectError(t *testing.T) {
	// A ping check with an invalid timeout passes validation but causes Detect
	// to error at runtime; runList must still succeed, falling back to "unknown".
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"broken": {Priority: 1, Checks: []config.NetworkCheck{
				{Type: config.CheckTypePing, Host: "127.0.0.1", Timeout: "not_a_duration"},
			}},
		},
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Port: 22, User: "alice"}},
		},
	})

	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunList_FilterByTag(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"web":     {"default": {Host: "1.2.3.4", Tags: []string{"production", "web"}}},
			"db":      {"default": {Host: "5.6.7.8", Tags: []string{"production", "database"}}},
			"staging": {"default": {Host: "9.10.11.12", Tags: []string{"staging"}}},
		},
	})
	listFilterTags = []string{"production"}
	defer func() { listFilterTags = nil }()

	old := output
	output = "json"
	defer func() { output = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "web") || !strings.Contains(out, "db") {
		t.Errorf("expected web and db in output, got: %s", out)
	}
	if strings.Contains(out, "staging") {
		t.Errorf("staging should be filtered out, got: %s", out)
	}
}

func TestRunList_FilterByText(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"web": {"default": {Host: "1.2.3.4", User: "alice", Comment: "frontend server"}},
			"db":  {"default": {Host: "5.6.7.8", User: "postgres"}},
		},
	})
	listFilterText = "frontend"
	defer func() { listFilterText = "" }()

	old := output
	output = "json"
	defer func() { output = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "web") {
		t.Errorf("expected web in output, got: %s", out)
	}
	if strings.Contains(out, "\"db\"") {
		t.Errorf("db should be filtered out, got: %s", out)
	}
}

func TestRunList_CommentAndTagsInOutput(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Comment: "test server", Tags: []string{"zone01", "dev"}}},
		},
	})

	old := output
	output = "json"
	defer func() { output = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "test server") {
		t.Errorf("expected comment in output, got: %s", out)
	}
	if !strings.Contains(out, "zone01,dev") {
		t.Errorf("expected tags in output, got: %s", out)
	}
}

func TestRunConnect_DryRunWithExtraArgs(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"myserver": {"default": {Host: "1.2.3.4", Port: 22, User: "alice"}},
		},
	})

	dryRun = true
	defer func() { dryRun = false }()

	if err := runConnect(connectCmd, []string{"myserver", "whoami"}); err != nil {
		t.Fatalf("runConnect dry-run with extra args error: %v", err)
	}
}

func TestRunConnect_DryRunWithJumpAlias(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"gateway": {"default": {Host: "gw.example.com", Port: 22, User: "admin", Key: "~/.ssh/gw_key"}},
			"backend": {"default": {Host: "192.168.1.10", Port: 2222, User: "root", Jump: "gateway"}},
		},
	})

	dryRun = true
	defer func() { dryRun = false }()

	if err := runConnect(connectCmd, []string{"backend"}); err != nil {
		t.Fatalf("runConnect dry-run with jump alias error: %v", err)
	}
}

func TestRunConnect_DetectError(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"broken": {Priority: 1, Checks: []config.NetworkCheck{
				{Type: config.CheckTypePing, Host: "127.0.0.1", Timeout: "not_a_duration"},
			}},
		},
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Port: 22, User: "alice"}},
		},
	})

	err := runConnect(connectCmd, []string{"srv"})
	if err == nil {
		t.Error("expected error from network detection failure")
	}
}

func TestRunConfigEdit_MkdirAllError(t *testing.T) {
	// /dev/null is a char device — MkdirAll creating a subdir inside it fails.
	cfgFile = "/dev/null/subdir/config.yaml"
	t.Setenv("SSHROUTE_CONFIG", "/dev/null/subdir/config.yaml")
	t.Setenv("EDITOR", "nonexistent_editor_xyz_abc")
	t.Cleanup(func() { cfgFile = "" })

	err := runConfigEdit(configEditCmd, nil)
	if err == nil {
		t.Fatal("expected error creating config directory")
	}
}

func TestRunConfigEdit_OpenFileError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write anywhere; cannot test OpenFile failure")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	path := filepath.Join(dir, "config.yaml")
	cfgFile = path
	t.Setenv("SSHROUTE_CONFIG", path)
	t.Setenv("EDITOR", "nonexistent_editor_xyz_abc")

	err := runConfigEdit(configEditCmd, nil)
	if err == nil {
		t.Fatal("expected error creating config file in read-only directory")
	}
	if !strings.Contains(err.Error(), "creating config file") {
		t.Errorf("error = %q, want 'creating config file'", err.Error())
	}
}

func TestRunConfigEdit_EmptyCfgFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("SSHROUTE_CONFIG", path)
	t.Setenv("EDITOR", "nonexistent_editor_xyz_abc")
	cfgFile = "" // force DefaultConfigPath() branch

	err := runConfigEdit(configEditCmd, nil)
	if err == nil {
		t.Fatal("expected error when editor binary does not exist")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("error = %q, want 'not found in PATH'", err.Error())
	}
}

func TestRunInit_MkdirAllError(t *testing.T) {
	// /dev/null is a char device; MkdirAll trying to create a subdir inside it
	// fails with ENOTDIR, covering the "creating config directory" error branch.
	cfgFile = "/dev/null/subdir/config.yaml"
	t.Setenv("SSHROUTE_CONFIG", "/dev/null/subdir/config.yaml")

	err := runInit(initCmd, nil)
	if err == nil {
		t.Fatal("expected error creating config directory")
	}
	// Restore valid cfgFile for subsequent tests.
	t.Cleanup(func() { cfgFile = "" })
}

func TestRunInit_WriteFileError(t *testing.T) {
	// A directory at the config path makes os.WriteFile fail with EISDIR,
	// covering the "writing config file" error branch.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	cfgFile = path
	t.Setenv("SSHROUTE_CONFIG", path)
	initForce = true
	defer func() { initForce = false }()

	err := runInit(initCmd, nil)
	if err == nil {
		t.Fatal("expected error writing config file")
	}
	if !strings.Contains(err.Error(), "writing config file") {
		t.Errorf("error = %q, want 'writing config file'", err.Error())
	}
}

func TestRunNetworkList_DetectError(t *testing.T) {
	// A ping check with invalid timeout causes Detect to error; runNetworkList
	// must fall back to "unknown" rather than propagating the error.
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"broken": {Priority: 1, Checks: []config.NetworkCheck{
				{Type: config.CheckTypePing, Host: "127.0.0.1", Timeout: "bad_timeout"},
			}},
		},
		Hosts: make(map[string]config.HostConfig),
	})

	if err := runNetworkList(networkListCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunNetwork_DetectError(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"broken": {Priority: 1, Checks: []config.NetworkCheck{
				{Type: config.CheckTypePing, Host: "127.0.0.1", Timeout: "bad_timeout"},
			}},
		},
		Hosts: make(map[string]config.HostConfig),
	})

	err := runNetwork(networkCmd, nil)
	if err == nil {
		t.Error("expected error from network detection failure")
	}
}

func TestRunNetworkList_YAMLOutput(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"vpn": {Priority: 10, Checks: []config.NetworkCheck{{Type: config.CheckTypeExec, Command: "true"}}},
		},
		Hosts: make(map[string]config.HostConfig),
	})
	old := output
	output = "yaml"
	defer func() { output = old }()

	if err := runNetworkList(networkListCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunInit_EmptyCfgFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = "" // force DefaultConfigPath() branch

	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestVersionCmd(t *testing.T) {
	// Run the version command; it just prints to stdout and must not error.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	versionCmd.Run(versionCmd, nil)
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if buf.Len() == 0 {
		t.Error("expected version output, got nothing")
	}
}

// LoadConfig error tests — cover the "loading config" error branch in each command.

func TestRunConnect_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	err := runConnect(connectCmd, []string{"anyhost"})
	if err == nil || !strings.Contains(err.Error(), "loading config") {
		t.Errorf("expected 'loading config' error, got %v", err)
	}
}

func TestRunList_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	err := runList(listCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "loading config") {
		t.Errorf("expected 'loading config' error, got %v", err)
	}
}

func TestRunNetworkList_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	err := runNetworkList(networkListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "loading config") {
		t.Errorf("expected 'loading config' error, got %v", err)
	}
}

func TestRunNetwork_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	err := runNetwork(networkCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "loading config") {
		t.Errorf("expected 'loading config' error, got %v", err)
	}
}

func TestRunNetworkTest_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	err := runNetworkTest(networkTestCmd, []string{"anynet"})
	if err == nil || !strings.Contains(err.Error(), "loading config") {
		t.Errorf("expected 'loading config' error, got %v", err)
	}
}

func TestRunAdd_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	addHost = "1.2.3.4"
	addPort = 22
	addUser = "alice"
	addKey = ""
	addJump = ""
	addNetwork = "default"
	defer func() { addHost = ""; addPort = 22; addUser = ""; addNetwork = "default" }()

	err := runAdd(addCmd, []string{"newserver"})
	if err == nil || !strings.Contains(err.Error(), "loading config") {
		t.Errorf("expected 'loading config' error, got %v", err)
	}
}

func TestRunRemove_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	err := runRemove(removeCmd, []string{"anyhost"})
	if err == nil || !strings.Contains(err.Error(), "loading config") {
		t.Errorf("expected 'loading config' error, got %v", err)
	}
}

func TestRunAdd_SaveError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write anywhere")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path

	addHost = "1.2.3.4"
	addPort = 22
	addUser = "alice"
	addNetwork = "default"
	defer func() { addHost = ""; addPort = 22; addUser = ""; addNetwork = "default" }()

	// Make dir read-only so Save fails.
	os.Chmod(dir, 0o555)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	err := runAdd(addCmd, []string{"newserver"})
	if err == nil || !strings.Contains(err.Error(), "saving config") {
		t.Errorf("expected 'saving config' error, got %v", err)
	}
}

func TestRunRemove_SaveError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write anywhere")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4"}},
		},
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path

	// Make dir read-only so Save fails.
	os.Chmod(dir, 0o555)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	err := runRemove(removeCmd, []string{"srv"})
	if err == nil || !strings.Contains(err.Error(), "saving config") {
		t.Errorf("expected 'saving config' error, got %v", err)
	}
}

func TestRunAdd_WithCommentAndTags(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})

	addHost = "1.2.3.4"
	addPort = 22
	addUser = "alice"
	addComment = "my server"
	addTags = []string{"prod", "web"}
	addNetwork = "default"
	defer func() {
		addHost = ""
		addPort = 22
		addUser = ""
		addComment = ""
		addTags = nil
		addNetwork = "default"
	}()

	if err := runAdd(addCmd, []string{"tagged-server"}); err != nil {
		t.Fatalf("runAdd error: %v", err)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}
	host := cfg.Hosts["tagged-server"]["default"]
	if host.Comment != "my server" {
		t.Errorf("Comment = %q, want %q", host.Comment, "my server")
	}
	if len(host.Tags) != 2 || host.Tags[0] != "prod" {
		t.Errorf("Tags = %v, want [prod web]", host.Tags)
	}
}

func TestRunConnect_ResolveError(t *testing.T) {
	// Use a config where the host exists but has a jump alias that creates a
	// cycle, causing Resolve to fail.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"a": {"default": {Host: "a.example.com", Jump: "b"}},
			"b": {"default": {Host: "b.example.com", Jump: "a"}},
		},
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path

	err := runConnect(connectCmd, []string{"a"})
	if err == nil || !strings.Contains(err.Error(), "resolving params") {
		t.Errorf("expected 'resolving params' error, got %v", err)
	}
}

// resolve command tests

func TestRunResolve_UnknownHost(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})
	err := runResolve(resolveCmd, []string{"doesnotexist"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestRunResolve_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	err := runResolve(resolveCmd, []string{"anyhost"})
	if err == nil || !strings.Contains(err.Error(), "loading config") {
		t.Errorf("expected 'loading config' error, got %v", err)
	}
}

func TestRunResolve_DefaultNetwork(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Port: 22, User: "alice"}},
		},
	})
	old := output
	output = "json"
	defer func() { output = old }()

	if err := runResolve(resolveCmd, []string{"srv"}); err != nil {
		t.Fatalf("runResolve error: %v", err)
	}
}

func TestRunResolve_ExplicitNetwork(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {
				"default": {Host: "1.2.3.4", Port: 22},
				"vpn":     {Host: "10.0.0.1", Port: 2222},
			},
		},
	})
	resolveNetwork = "vpn"
	defer func() { resolveNetwork = "" }()

	old := output
	output = "json"
	defer func() { output = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	if err := runResolve(resolveCmd, []string{"srv"}); err != nil {
		t.Fatalf("runResolve error: %v", err)
	}
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "10.0.0.1") {
		t.Errorf("expected vpn host in output, got: %s", buf.String())
	}
}

// copy command tests

func TestBuildSCPArgv_Basic(t *testing.T) {
	params := config.SSHParams{Host: "1.2.3.4", Port: 22, User: "alice", Key: "~/.ssh/id_rsa"}
	argv := buildSCPArgv("/usr/bin/scp", params, "srv:remote.txt", "./local.txt", "srv")
	joined := strings.Join(argv, " ")
	if !strings.Contains(joined, "-P 22") {
		t.Errorf("expected -P 22, got: %s", joined)
	}
	if !strings.Contains(joined, "-i") {
		t.Errorf("expected -i key, got: %s", joined)
	}
	if !strings.Contains(joined, "alice@1.2.3.4:remote.txt") {
		t.Errorf("expected rewritten remote, got: %s", joined)
	}
}

func TestBuildSCPArgv_NoPort(t *testing.T) {
	params := config.SSHParams{Host: "1.2.3.4"}
	argv := buildSCPArgv("/usr/bin/scp", params, "./local.txt", "srv:/dst", "srv")
	joined := strings.Join(argv, " ")
	if strings.Contains(joined, "-P") {
		t.Errorf("expected no -P flag when port is 0, got: %s", joined)
	}
}

func TestBuildSCPArgv_WithJump(t *testing.T) {
	params := config.SSHParams{Host: "1.2.3.4", Jump: "bastion.example.com"}
	argv := buildSCPArgv("/usr/bin/scp", params, "./local.txt", "srv:/dst", "srv")
	joined := strings.Join(argv, " ")
	if !strings.Contains(joined, "-J bastion.example.com") {
		t.Errorf("expected -J jump, got: %s", joined)
	}
}

func TestRewriteRemote_AliasPrefix(t *testing.T) {
	got := rewriteRemote("myserver:/path/to/file", "myserver", "alice@1.2.3.4")
	want := "alice@1.2.3.4:/path/to/file"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRewriteRemote_LocalPath(t *testing.T) {
	got := rewriteRemote("./local/file.txt", "myserver", "alice@1.2.3.4")
	if got != "./local/file.txt" {
		t.Errorf("local path should be unchanged, got %q", got)
	}
}

func TestResolveSCPBinary_EnvVar(t *testing.T) {
	t.Setenv("SSHROUTE_SCP", "/custom/scp")
	got := resolveSCPBinary()
	if got != "/custom/scp" {
		t.Errorf("got %q, want /custom/scp", got)
	}
}

func TestResolveSCPBinary_Default(t *testing.T) {
	t.Setenv("SSHROUTE_SCP", "")
	got := resolveSCPBinary()
	if got == "" {
		t.Error("expected non-empty scp binary path")
	}
}

func TestRunCopy_UnknownHost(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts:    make(map[string]config.HostConfig),
	})
	err := runCopy(copyCmd, []string{"doesnotexist", "./local.txt", "doesnotexist:/remote"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestRunCopy_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	err := runCopy(copyCmd, []string{"srv", "./local.txt", "srv:/remote"})
	if err == nil || !strings.Contains(err.Error(), "loading config") {
		t.Errorf("expected 'loading config' error, got %v", err)
	}
}

func TestRunCopy_DryRun(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {"default": {Host: "1.2.3.4", Port: 22, User: "alice"}},
		},
	})
	dryRun = true
	defer func() { dryRun = false }()

	if err := runCopy(copyCmd, []string{"srv", "./local.txt", "srv:/remote/path"}); err != nil {
		t.Fatalf("runCopy dry-run error: %v", err)
	}
}

// completion tests

func TestCompleteAliases_ReturnsAliases(t *testing.T) {
	withTempConfig(t, &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"alpha": {"default": {Host: "1.2.3.4"}},
			"beta":  {"default": {Host: "5.6.7.8"}},
		},
	})
	got, directive := completeAliases(connectCmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("unexpected directive: %v", directive)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 aliases, got %d: %v", len(got), got)
	}
}

func TestCompleteAliases_AlreadyHasArg(t *testing.T) {
	got, directive := completeAliases(connectCmd, []string{"srv"}, "")
	if got != nil {
		t.Errorf("expected nil completions after first arg, got %v", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("unexpected directive: %v", directive)
	}
}

func TestRunList_ResolveErrorSkipsHost(t *testing.T) {
	// Manually create a config file that has a host with a circular jump
	// chain. The validator won't catch this, but Resolve will fail.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfgContent := `
hosts:
  good:
    default:
      host: 1.2.3.4
      port: 22
  broken-a:
    default:
      host: a.example.com
      jump: broken-b
  broken-b:
    default:
      host: b.example.com
      jump: broken-a
`
	if err := os.WriteFile(path, []byte(cfgContent), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("SSHROUTE_CONFIG", path)
	cfgFile = path

	old := output
	output = "json"
	defer func() { output = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "good") {
		t.Errorf("expected 'good' host in output, got: %s", out)
	}
	if strings.Contains(out, "broken-a") || strings.Contains(out, "broken-b") {
		t.Errorf("broken hosts should be skipped, got: %s", out)
	}
}
