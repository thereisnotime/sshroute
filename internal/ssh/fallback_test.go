package ssh

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/thereisnotime/sshroute/internal/config"
)

func fallbackCfg() *config.Config {
	return &config.Config{
		Networks: map[string]config.NetworkDefinition{
			"vpn":    {Priority: 10},
			"office": {Priority: 20},
			"lan":    {Priority: 30},
		},
		Hosts: map[string]config.HostConfig{
			"srv": {
				"default": {Host: "pub.example.com"},
				"vpn":     {Host: "10.0.0.1"},
				"office":  {Host: "192.168.1.1"},
				"lan":     {Host: "192.168.77.1"},
			},
		},
	}
}

func TestResolveOrderPreferred(t *testing.T) {
	tests := []struct {
		name      string
		detected  string
		preferred string
		want      []OrderEntry
	}{
		{"no cache", "vpn", "", []OrderEntry{
			{"vpn", SourceDetected}, {"office", SourcePriority}, {"lan", SourcePriority}, {"default", SourceDefault},
		}},
		{"cached profile first, not repeated", "vpn", "lan", []OrderEntry{
			{"lan", SourceCache}, {"vpn", SourceDetected}, {"office", SourcePriority}, {"default", SourceDefault},
		}},
		{"cached equals detected", "vpn", "vpn", []OrderEntry{
			{"vpn", SourceCache}, {"office", SourcePriority}, {"lan", SourcePriority}, {"default", SourceDefault},
		}},
		{"cached default", "vpn", "default", []OrderEntry{
			{"default", SourceCache}, {"vpn", SourceDetected}, {"office", SourcePriority}, {"lan", SourcePriority},
		}},
		{"cached profile no longer exists", "vpn", "gone", []OrderEntry{
			{"vpn", SourceDetected}, {"office", SourcePriority}, {"lan", SourcePriority}, {"default", SourceDefault},
		}},
		{"no network detected", "default", "office", []OrderEntry{
			{"office", SourceCache}, {"vpn", SourcePriority}, {"lan", SourcePriority}, {"default", SourceDefault},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOrderPreferred(fallbackCfg(), "srv", tt.detected, tt.preferred)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveOrderPreferredUnknownAlias(t *testing.T) {
	if got := ResolveOrderPreferred(fallbackCfg(), "nope", "vpn", "vpn"); len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

// scriptedRunner returns exit codes by target host (the last argv element).
func scriptedRunner(codes map[string]int, seen *[]string) Runner {
	return func(argv []string) (int, error) {
		host := argv[len(argv)-1]
		*seen = append(*seen, host)
		if c, ok := codes[host]; ok {
			return c, nil
		}
		return SSHConnectFailure, nil
	}
}

func silenceFallbackStatus(t *testing.T) *strings.Builder {
	t.Helper()
	var b strings.Builder
	old := FallbackStatus
	FallbackStatus = &b
	t.Cleanup(func() { FallbackStatus = old })
	return &b
}

func TestRunFallback(t *testing.T) {
	order := ResolveOrderPreferred(fallbackCfg(), "srv", "vpn", "lan")
	tests := []struct {
		name      string
		codes     map[string]int
		wantCode  int
		wantTried []string
		wantAtt   []Attempt
	}{
		{"cached route works first try", map[string]int{"192.168.77.1": 0}, 0,
			[]string{"192.168.77.1"}, []Attempt{{"lan", 0}}},
		{"stale cache costs one attempt", map[string]int{"10.0.0.1": 0}, 0,
			[]string{"192.168.77.1", "10.0.0.1"}, []Attempt{{"lan", 255}, {"vpn", 0}}},
		{"remote failure stops the walk", map[string]int{"192.168.77.1": 2}, 2,
			[]string{"192.168.77.1"}, []Attempt{{"lan", 2}}},
		{"all fail", map[string]int{}, SSHConnectFailure,
			[]string{"192.168.77.1", "10.0.0.1", "192.168.1.1", "pub.example.com"},
			[]Attempt{{"lan", 255}, {"vpn", 255}, {"office", 255}, {"default", 255}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := silenceFallbackStatus(t)
			var tried []string
			code, att, err := RunFallback(fallbackCfg(), "srv", order, ParsedArgs{}, scriptedRunner(tt.codes, &tried))
			if err != nil {
				t.Fatal(err)
			}
			if code != tt.wantCode {
				t.Errorf("code = %d, want %d", code, tt.wantCode)
			}
			if !reflect.DeepEqual(tried, tt.wantTried) {
				t.Errorf("tried %v, want %v", tried, tt.wantTried)
			}
			if !reflect.DeepEqual(att, tt.wantAtt) {
				t.Errorf("attempts %v, want %v", att, tt.wantAtt)
			}
			if fails := strings.Count(status.String(), "failed (connection error)"); fails != countCode(tt.wantAtt, 255) {
				t.Errorf("status lines = %d, want %d", fails, countCode(tt.wantAtt, 255))
			}
		})
	}
}

func countCode(att []Attempt, code int) int {
	n := 0
	for _, a := range att {
		if a.Code == code {
			n++
		}
	}
	return n
}

func TestRunFallbackSkipsUnresolvable(t *testing.T) {
	silenceFallbackStatus(t)
	cfg := fallbackCfg()
	order := []OrderEntry{{"ghost", SourceCache}, {"vpn", SourceDetected}}
	cfg.Hosts["srv"]["ghost"] = config.SSHParams{Jump: "srv"} // circular jump: fails to resolve
	var tried []string
	code, att, err := RunFallback(cfg, "srv", order, ParsedArgs{}, scriptedRunner(map[string]int{"10.0.0.1": 0}, &tried))
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !reflect.DeepEqual(att, []Attempt{{"vpn", 0}}) {
		t.Errorf("attempts = %v", att)
	}
}

func TestRunFallbackRunnerError(t *testing.T) {
	silenceFallbackStatus(t)
	order := ResolveOrderPreferred(fallbackCfg(), "srv", "vpn", "")
	calls := 0
	run := func([]string) (int, error) {
		calls++
		if calls == 1 {
			return SSHConnectFailure, nil
		}
		return -1, errors.New("exec failed")
	}
	_, att, err := RunFallback(fallbackCfg(), "srv", order, ParsedArgs{}, run)
	if err == nil {
		t.Fatal("expected runner error")
	}
	if !reflect.DeepEqual(att, []Attempt{{"vpn", 255}}) {
		t.Errorf("attempts before error = %v", att)
	}
}
