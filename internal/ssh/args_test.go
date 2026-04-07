package ssh

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want ParsedArgs
	}{
		{
			name: "simple alias",
			argv: []string{"myserver"},
			want: ParsedArgs{Alias: "myserver", User: "", Remaining: []string{}},
		},
		{
			name: "user@host",
			argv: []string{"alice@myserver"},
			want: ParsedArgs{Alias: "myserver", User: "alice", Remaining: []string{}},
		},
		{
			name: "port flag before host",
			argv: []string{"-p", "2222", "myserver"},
			want: ParsedArgs{Alias: "myserver", User: "", Remaining: []string{"-p", "2222"}},
		},
		{
			name: "identity file flag",
			argv: []string{"-i", "~/.ssh/key", "myserver"},
			want: ParsedArgs{Alias: "myserver", User: "", Remaining: []string{"-i", "~/.ssh/key"}},
		},
		{
			name: "jump host flag",
			argv: []string{"-J", "bastion", "myserver"},
			want: ParsedArgs{Alias: "myserver", User: "", Remaining: []string{"-J", "bastion"}},
		},
		{
			name: "boolean flag before host",
			argv: []string{"-v", "myserver"},
			want: ParsedArgs{Alias: "myserver", User: "", Remaining: []string{"-v"}},
		},
		{
			name: "remote command after host",
			argv: []string{"myserver", "ls", "-la"},
			want: ParsedArgs{Alias: "myserver", User: "", Remaining: []string{"ls", "-la"}},
		},
		{
			name: "multiple flags and remote command",
			argv: []string{"-p", "22", "-v", "alice@myserver", "uptime"},
			want: ParsedArgs{Alias: "myserver", User: "alice", Remaining: []string{"-p", "22", "-v", "uptime"}},
		},
		{
			name: "empty argv",
			argv: []string{},
			want: ParsedArgs{Alias: "", User: "", Remaining: []string{}},
		},
		{
			name: "only flags no host",
			argv: []string{"-v", "-p", "22"},
			want: ParsedArgs{Alias: "", User: "", Remaining: []string{"-v", "-p", "22"}},
		},
		{
			name: "login flag",
			argv: []string{"-l", "bob", "myserver"},
			want: ParsedArgs{Alias: "myserver", User: "", Remaining: []string{"-l", "bob"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseArgs(tt.argv)
			if got.Alias != tt.want.Alias {
				t.Errorf("Alias = %q, want %q", got.Alias, tt.want.Alias)
			}
			if got.User != tt.want.User {
				t.Errorf("User = %q, want %q", got.User, tt.want.User)
			}
			if !reflect.DeepEqual(got.Remaining, tt.want.Remaining) {
				t.Errorf("Remaining = %v, want %v", got.Remaining, tt.want.Remaining)
			}
		})
	}
}
