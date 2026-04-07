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
