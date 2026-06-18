package ssh

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thereisnotime/sshroute/internal/config"
)

func TestDryRun(t *testing.T) {
	argv := []string{RealSSH, "-p", "2222", "-l", "alice", "10.0.0.1"}

	// Capture stdout.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	DryRun(argv)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[dry-run]") {
		t.Errorf("output missing [dry-run] prefix: %q", out)
	}
	for _, arg := range argv {
		if !strings.Contains(out, arg) {
			t.Errorf("output missing arg %q: %q", arg, out)
		}
	}
}

func TestExpandTildeExecutor(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skip("cannot get current user:", err)
	}
	home := u.HomeDir

	tests := []struct {
		input string
		want  string
	}{
		{"~", home},
		{"~/.ssh/key", filepath.Join(home, ".ssh/key")},
		{"~/", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		got := expandTilde(tt.input)
		// filepath.Join strips trailing slash, so normalise both.
		if got != tt.want {
			t.Errorf("expandTilde(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExpandTildeExecutor_TildeOnly(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skip("cannot get current user")
	}
	got := expandTilde("~")
	if got != u.HomeDir {
		t.Errorf("expandTilde(~) = %q, want %q", got, u.HomeDir)
	}
}

func TestExpandTildeExecutor_NoTilde(t *testing.T) {
	got := expandTilde("/etc/hosts")
	if got != "/etc/hosts" {
		t.Errorf("expandTilde(/etc/hosts) = %q, want unchanged", got)
	}
	got = expandTilde("")
	if got != "" {
		t.Errorf("expandTilde('') = %q, want empty", got)
	}
}

func TestResolveSSHBinary_EnvVar(t *testing.T) {
	t.Setenv("SSHROUTE_SSH", "/custom/ssh")
	got := ResolveSSHBinary(nil)
	if got != "/custom/ssh" {
		t.Errorf("got %q, want /custom/ssh", got)
	}
}

func TestResolveSSHBinary_Config(t *testing.T) {
	t.Setenv("SSHROUTE_SSH", "")
	cfg := &config.Config{SSHBinary: "/configured/ssh"}
	got := ResolveSSHBinary(cfg)
	if got != "/configured/ssh" {
		t.Errorf("got %q, want /configured/ssh", got)
	}
}

func TestResolveSSHBinary_NilConfig(t *testing.T) {
	t.Setenv("SSHROUTE_SSH", "")
	got := ResolveSSHBinary(nil)
	if got == "" {
		t.Error("expected non-empty SSH binary path")
	}
}

func TestResolveSSHBinary_EmptyConfig(t *testing.T) {
	t.Setenv("SSHROUTE_SSH", "")
	got := ResolveSSHBinary(&config.Config{})
	if got == "" {
		t.Error("expected non-empty SSH binary path")
	}
}

func TestSameFile_Equal(t *testing.T) {
	f, err := os.CreateTemp("", "sshroute-same-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()
	if !sameFile(f.Name(), f.Name()) {
		t.Error("expected same file to return true")
	}
}

func TestSameFile_Different(t *testing.T) {
	a, err := os.CreateTemp("", "sshroute-a-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(a.Name())
	a.Close()

	b, err := os.CreateTemp("", "sshroute-b-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(b.Name())
	b.Close()

	if sameFile(a.Name(), b.Name()) {
		t.Error("expected different files to return false")
	}
}

func TestSameFile_NonExistentA(t *testing.T) {
	if sameFile("/nonexistent/sshroute-test-a", "/etc/hosts") {
		t.Error("expected false when first path does not exist")
	}
}

func TestSameFile_NonExistentB(t *testing.T) {
	if sameFile("/etc/hosts", "/nonexistent/sshroute-test-b") {
		t.Error("expected false when second path does not exist")
	}
}

func TestBuildArgv_PortFormatting(t *testing.T) {
	// Ensure port is formatted as decimal string, not e.g. hex.
	from := config.SSHParams{Host: "h", Port: 65535}
	got := BuildArgv(from, ParsedArgs{})
	found := false
	for i, a := range got {
		if a == "-p" && i+1 < len(got) {
			if got[i+1] != fmt.Sprintf("%d", 65535) {
				t.Errorf("port formatted as %q, want %q", got[i+1], "65535")
			}
			found = true
		}
	}
	if !found {
		t.Error("-p flag not found in argv")
	}
}

func TestBuildArgv_Options(t *testing.T) {
	params := config.SSHParams{
		Host: "myhost",
		Options: map[string]string{
			"ConnectTimeout":      "10",
			"ServerAliveInterval": "30",
		},
	}
	got := BuildArgv(params, ParsedArgs{})

	// Collect all -o values from argv.
	var opts []string
	for i, a := range got {
		if a == "-o" && i+1 < len(got) {
			opts = append(opts, got[i+1])
		}
	}

	want := map[string]bool{
		"ConnectTimeout=10":      true,
		"ServerAliveInterval=30": true,
	}
	for _, o := range opts {
		delete(want, o)
	}
	if len(want) > 0 {
		t.Errorf("missing -o entries in argv: %v; got opts: %v", want, opts)
	}
}

func TestBuildArgv_OptionsEmpty(t *testing.T) {
	params := config.SSHParams{Host: "myhost"}
	got := BuildArgv(params, ParsedArgs{})
	for i, a := range got {
		if a == "-o" && i+1 < len(got) && strings.Contains(got[i+1], "=") {
			// ProxyCommand is fine; reject anything that looks like an Options entry.
			if got[i+1] != "ProxyCommand=" && !strings.HasPrefix(got[i+1], "ProxyCommand=") {
				t.Errorf("unexpected -o %q with no Options set", got[i+1])
			}
		}
	}
}

func TestBuildArgv_OptionsDeterministicOrder(t *testing.T) {
	params := config.SSHParams{
		Host: "myhost",
		Options: map[string]string{
			"ConnectTimeout":        "5",
			"BatchMode":             "yes",
			"StrictHostKeyChecking": "no",
		},
	}

	first := BuildArgv(params, ParsedArgs{})
	second := BuildArgv(params, ParsedArgs{})

	if strings.Join(first, " ") != strings.Join(second, " ") {
		t.Errorf("BuildArgv is not deterministic:\nfirst:  %v\nsecond: %v", first, second)
	}
}
