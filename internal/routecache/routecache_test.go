package routecache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thereisnotime/sshroute/internal/config"
)

var t0 = time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)

func allExist(string) bool { return true }

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{
		Path:        filepath.Join(t.TempDir(), "sub", "state.json"),
		LockTimeout: 10 * time.Second,
		Now:         func() time.Time { return t0 },
	}
}

func TestApplyRecordsSuccessAndFailures(t *testing.T) {
	st := NewState()
	st.Apply("srv", "vpn", Outcome{Succeeded: "lan", Failed: []string{"vpn", "office"}}, t0.Add(500*time.Millisecond))

	h := st.Hosts["srv"]
	if h == nil || h.LastSuccess == nil {
		t.Fatalf("missing host state: %+v", h)
	}
	want := Route{Profile: "lan", Network: "vpn", SucceededAt: t0}
	if *h.LastSuccess != want {
		t.Errorf("LastSuccess = %+v, want %+v (truncated to seconds, UTC)", *h.LastSuccess, want)
	}
	if h.Networks["vpn"] != want {
		t.Errorf("Networks[vpn] = %+v, want %+v", h.Networks["vpn"], want)
	}
	for _, p := range []string{"vpn", "office"} {
		if f := h.Failures[p]; f.Network != "vpn" || !f.FailedAt.Equal(t0) {
			t.Errorf("Failures[%s] = %+v", p, f)
		}
	}
}

func TestApplyEmptyOutcomeIsNoop(t *testing.T) {
	st := NewState()
	st.Apply("srv", "vpn", Outcome{}, t0)
	if len(st.Hosts) != 0 {
		t.Errorf("empty outcome created state: %+v", st.Hosts)
	}
}

func TestApplyFailureOnlyKeepsPreviousSuccess(t *testing.T) {
	st := NewState()
	st.Apply("srv", "vpn", Outcome{Succeeded: "lan"}, t0)
	st.Apply("srv", "vpn", Outcome{Failed: []string{"lan"}}, t0.Add(time.Hour))
	h := st.Hosts["srv"]
	if h.LastSuccess.Profile != "lan" || !h.LastSuccess.SucceededAt.Equal(t0) {
		t.Errorf("LastSuccess changed by a failure-only outcome: %+v", h.LastSuccess)
	}
}

func TestClear(t *testing.T) {
	st := NewState()
	st.Apply("a", "n", Outcome{Succeeded: "p"}, t0)
	st.Apply("b", "n", Outcome{Succeeded: "p"}, t0)
	if !st.Clear("a") {
		t.Error("Clear(a) = false, want true")
	}
	if st.Clear("a") {
		t.Error("second Clear(a) = true, want false")
	}
	if _, ok := st.Hosts["b"]; !ok {
		t.Error("Clear(a) removed b")
	}
	if !st.Clear("") || len(st.Hosts) != 0 {
		t.Error("Clear(\"\") did not remove everything")
	}
	if st.Clear("") {
		t.Error("Clear(\"\") on empty state = true")
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		version bool
	}{
		{"valid", `{"version":1,"hosts":{"a":{"last_success":{"profile":"p","network":"n","succeeded_at":"2026-11-14T10:00:00Z"}}}}`, false, false},
		{"null hosts", `{"version":1,"hosts":null}`, false, false},
		{"null host entry", `{"version":1,"hosts":{"a":null}}`, false, false},
		{"garbage", `{garbage`, true, false},
		{"empty", ``, true, false},
		{"missing version", `{"hosts":{}}`, true, false},
		{"newer version", `{"version":2,"hosts":{}}`, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, err := decode([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.version != errors.Is(err, ErrUnsupportedVersion) {
				t.Errorf("ErrUnsupportedVersion = %v, want %v", errors.Is(err, ErrUnsupportedVersion), tt.version)
			}
			if err == nil {
				if st.Hosts == nil {
					t.Error("Hosts is nil")
				}
				for a, h := range st.Hosts {
					if h == nil {
						t.Errorf("nil host %q kept", a)
					}
				}
			}
		})
	}
}

func TestLookup(t *testing.T) {
	ttl := 7 * 24 * time.Hour
	base := func() *State {
		st := NewState()
		st.Apply("srv", "vpn", Outcome{Succeeded: "lan"}, t0)
		return st
	}
	tests := []struct {
		name    string
		state   func() *State
		alias   string
		network string
		scope   Scope
		ttl     time.Duration
		now     time.Time
		exists  func(string) bool
		hit     bool
		status  Status
		key     string
	}{
		{"fresh network hit", base, "srv", "vpn", ScopeNetwork, ttl, t0.Add(time.Hour), allExist, true, StatusFresh, "vpn"},
		{"other network misses", base, "srv", "office", ScopeNetwork, ttl, t0, allExist, false, "", "office"},
		{"alias scope hits on any network", base, "srv", "office", ScopeAlias, ttl, t0, allExist, true, StatusFresh, AliasKey},
		{"unknown alias", base, "nope", "vpn", ScopeNetwork, ttl, t0, allExist, false, "", "vpn"},
		{"unknown alias, alias scope", base, "nope", "vpn", ScopeAlias, ttl, t0, allExist, false, "", AliasKey},
		{"expired", base, "srv", "vpn", ScopeNetwork, ttl, t0.Add(ttl + time.Second), allExist, false, StatusExpired, "vpn"},
		{"ttl 0 never expires", base, "srv", "vpn", ScopeNetwork, 0, t0.Add(1000 * ttl), allExist, true, StatusFresh, "vpn"},
		{"profile removed from config", base, "srv", "vpn", ScopeNetwork, ttl, t0, func(p string) bool { return p != "lan" }, false, StatusUnknownProfile, "vpn"},
		{
			"failed later on same network", func() *State {
				st := base()
				st.Apply("srv", "vpn", Outcome{Failed: []string{"lan"}}, t0.Add(time.Minute))
				return st
			}, "srv", "vpn", ScopeNetwork, ttl, t0.Add(time.Hour), allExist, false, StatusFailed, "vpn",
		},
		{
			"failed later on other network keeps network entry", func() *State {
				st := base()
				st.Apply("srv", "office", Outcome{Failed: []string{"lan"}}, t0.Add(time.Minute))
				return st
			}, "srv", "vpn", ScopeNetwork, ttl, t0.Add(time.Hour), allExist, true, StatusFresh, "vpn",
		},
		{
			"failed later anywhere invalidates alias scope", func() *State {
				st := base()
				st.Apply("srv", "office", Outcome{Failed: []string{"lan"}}, t0.Add(time.Minute))
				return st
			}, "srv", "vpn", ScopeAlias, ttl, t0.Add(time.Hour), allExist, false, StatusFailed, AliasKey,
		},
		{
			"failure before success is ignored", func() *State {
				st := NewState()
				st.Apply("srv", "vpn", Outcome{Failed: []string{"lan"}}, t0.Add(-time.Minute))
				st.Apply("srv", "vpn", Outcome{Succeeded: "lan"}, t0)
				return st
			}, "srv", "vpn", ScopeNetwork, ttl, t0, allExist, true, StatusFresh, "vpn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lk := tt.state().Lookup(tt.alias, tt.network, tt.scope, tt.ttl, tt.now, tt.exists)
			if lk.Hit != tt.hit || lk.Status != tt.status || lk.Key != tt.key {
				t.Errorf("Lookup = {Hit:%v Status:%q Key:%q Reason:%q}, want {Hit:%v Status:%q Key:%q}",
					lk.Hit, lk.Status, lk.Key, lk.Reason, tt.hit, tt.status, tt.key)
			}
			if lk.Reason == "" {
				t.Error("Reason is empty")
			}
			if lk.Hit && lk.Route.Profile != "lan" {
				t.Errorf("Route.Profile = %q, want lan", lk.Route.Profile)
			}
		})
	}
}

func TestResolveOptionsDefaults(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg/state")
	o, err := ResolveOptions(config.CacheConfig{}, false, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if !o.Enabled || o.TTL != DefaultTTL || o.Scope != ScopeNetwork || o.LockTimeout != DefaultLockTimeout {
		t.Errorf("defaults = %+v", o)
	}
	if want := filepath.Join("/xdg/state", "sshroute", "state.json"); o.Path != want {
		t.Errorf("Path = %q, want %q", o.Path, want)
	}
}

func TestResolveOptionsFromConfig(t *testing.T) {
	off := false
	c := config.CacheConfig{Enabled: &off, Path: "/custom/state.json", TTL: "12h", Scope: "alias", LockTimeout: "250ms"}
	o, err := ResolveOptions(c, false, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if o.Enabled || o.Disabled == "" {
		t.Errorf("enabled: false not honoured: %+v", o)
	}
	if o.Path != "/custom/state.json" || o.TTL != 12*time.Hour || o.Scope != ScopeAlias || o.LockTimeout != 250*time.Millisecond {
		t.Errorf("options = %+v", o)
	}
}

func TestResolveOptionsTildePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	o, err := ResolveOptions(config.CacheConfig{Path: "~/s.json"}, false, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "s.json"); o.Path != want {
		t.Errorf("Path = %q, want %q", o.Path, want)
	}
}

func TestResolveOptionsPrecedence(t *testing.T) {
	on, off := true, false
	tests := []struct {
		name    string
		enabled *bool
		env     string
		noCache bool
		want    bool
	}{
		{"default on", nil, "", false, true},
		{"config off", &off, "", false, false},
		{"env 0 beats config on", &on, "0", false, false},
		{"env false", nil, "false", false, false},
		{"env off", nil, "OFF", false, false},
		{"env 1 beats config off", &off, "1", false, true},
		{"env yes", &off, "yes", false, true},
		{"unrecognised env ignored", &off, "maybe", false, false},
		{"flag beats env 1", nil, "1", true, false},
		{"flag alone", nil, "", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(k string) string {
				switch k {
				case "SSHROUTE_CACHE":
					return tt.env
				case "SSHROUTE_CACHE_PATH":
					return "/env/state.json"
				}
				return ""
			}
			o, err := ResolveOptions(config.CacheConfig{Enabled: tt.enabled, Path: "/cfg/state.json"}, tt.noCache, getenv)
			if err != nil {
				t.Fatal(err)
			}
			if o.Enabled != tt.want {
				t.Errorf("Enabled = %v, want %v", o.Enabled, tt.want)
			}
			if !o.Enabled && o.Disabled == "" {
				t.Error("Disabled reason is empty")
			}
			if o.Path != "/env/state.json" {
				t.Errorf("SSHROUTE_CACHE_PATH not preferred over cache.path: %q", o.Path)
			}
		})
	}
}

func TestResolveOptionsInvalid(t *testing.T) {
	empty := func(string) string { return "" }
	if _, err := ResolveOptions(config.CacheConfig{TTL: "soon"}, false, empty); err == nil {
		t.Error("expected error for bad ttl")
	}
	if _, err := ResolveOptions(config.CacheConfig{LockTimeout: "x"}, false, empty); err == nil {
		t.Error("expected error for bad lock_timeout")
	}
}

func TestDefaultPathHome(t *testing.T) {
	if runtime.GOOS == windowsGOOS {
		t.Skip("windows uses LOCALAPPDATA")
	}
	t.Setenv("XDG_STATE_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	p, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "state", "sshroute", "state.json"); p != want {
		t.Errorf("DefaultPath = %q, want %q", p, want)
	}
}

func TestStoreLoadMissingFile(t *testing.T) {
	s := newTestStore(t)
	st, err := s.Load()
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if st.Version != Version || len(st.Hosts) != 0 {
		t.Errorf("state = %+v", st)
	}
}

func TestStoreRecordAndLoad(t *testing.T) {
	s := newTestStore(t)
	if err := s.Record("srv", "vpn", Outcome{Succeeded: "lan", Failed: []string{"vpn"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Record("other", "default", Outcome{Succeeded: "default"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Record("noop", "default", Outcome{}); err != nil {
		t.Fatal(err)
	}
	st, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Hosts) != 2 || st.Hosts["srv"].LastSuccess.Profile != "lan" || st.Hosts["other"].LastSuccess.Profile != "default" {
		t.Errorf("state = %+v", st.Hosts)
	}
	fi, err := os.Stat(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != windowsGOOS && fi.Mode().Perm() != fileMode {
		t.Errorf("state file mode = %v, want %v", fi.Mode().Perm(), fileMode)
	}
	// No temp files left behind.
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(s.Path), ".state-*"))
	if len(matches) != 0 {
		t.Errorf("temp files left behind: %v", matches)
	}
}

func TestStoreFileFormat(t *testing.T) {
	s := newTestStore(t)
	if err := s.Record("srv", "vpn", Outcome{Succeeded: "lan", Failed: []string{"vpn"}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	// Decode generically, the way an external tool would.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["version"] != float64(1) {
		t.Errorf("version = %v", raw["version"])
	}
	host := raw["hosts"].(map[string]any)["srv"].(map[string]any)
	last := host["last_success"].(map[string]any)
	if last["profile"] != "lan" || last["network"] != "vpn" || last["succeeded_at"] != "2026-11-14T10:00:00Z" {
		t.Errorf("last_success = %v", last)
	}
	fail := host["failures"].(map[string]any)["vpn"].(map[string]any)
	if fail["network"] != "vpn" || fail["failed_at"] != "2026-11-14T10:00:00Z" {
		t.Errorf("failures.vpn = %v", fail)
	}
	if _, ok := host["networks"].(map[string]any)["vpn"]; !ok {
		t.Errorf("networks.vpn missing: %v", host["networks"])
	}
}

func TestStoreCorruptFileIsIgnoredAndRewritten(t *testing.T) {
	s := newTestStore(t)
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.Path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := s.Load()
	if err == nil {
		t.Error("Load of corrupt file should report the parse error")
	}
	if st == nil || len(st.Hosts) != 0 {
		t.Fatalf("corrupt file must yield an empty usable state, got %+v", st)
	}
	if err := s.Record("srv", "vpn", Outcome{Succeeded: "lan"}); err != nil {
		t.Fatalf("Record over corrupt file: %v", err)
	}
	st, err = s.Load()
	if err != nil {
		t.Fatalf("file not rewritten: %v", err)
	}
	if st.Hosts["srv"] == nil {
		t.Error("entry missing after rewrite")
	}
}

func TestStoreNewerVersionIsNotOverwritten(t *testing.T) {
	s := newTestStore(t)
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		t.Fatal(err)
	}
	future := []byte(`{"version":2,"something":"else"}`)
	if err := os.WriteFile(s.Path, future, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Record("srv", "vpn", Outcome{Succeeded: "lan"}); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("Record err = %v, want ErrUnsupportedVersion", err)
	}
	data, _ := os.ReadFile(s.Path)
	if string(data) != string(future) {
		t.Errorf("newer-version file was modified: %s", data)
	}
}

func TestStoreLockTimeout(t *testing.T) {
	s := newTestStore(t)
	s.LockTimeout = 50 * time.Millisecond
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		t.Fatal(err)
	}
	holder, err := os.OpenFile(s.Path+lockSuffix, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Close() }()
	ok, err := tryLockFile(holder)
	if err != nil || !ok {
		t.Fatalf("could not take lock for test: ok=%v err=%v", ok, err)
	}

	start := time.Now()
	err = s.Record("srv", "vpn", Outcome{Succeeded: "lan"})
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != windowsGOOS && err == nil {
		t.Skip("no file locking on this platform")
	}
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("Record err = %v, want ErrLockTimeout", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("lock wait took %v, should give up near the 50ms timeout", elapsed)
	}
	if _, err := os.Stat(s.Path); !os.IsNotExist(err) {
		t.Error("state file written despite lock timeout")
	}

	// Once released, writes go through.
	if err := unlockFile(holder); err != nil {
		t.Fatal(err)
	}
	if err := s.Record("srv", "vpn", Outcome{Succeeded: "lan"}); err != nil {
		t.Fatalf("Record after unlock: %v", err)
	}
}

func TestStoreConcurrentGoroutines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	const writers = 50
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// A separate Store (and so a separate lock fd) per writer, like separate processes.
			s := &Store{Path: path, LockTimeout: 30 * time.Second}
			errs <- s.Record(fmt.Sprintf("host-%02d", i), "vpn", Outcome{Succeeded: "lan", Failed: []string{"vpn"}})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	assertHosts(t, path, writers)
}

// TestHelperWriter is not a real test: TestStoreConcurrentProcesses re-runs the test
// binary with ROUTECACHE_HELPER_* set so several OS processes write at once.
func TestHelperWriter(t *testing.T) {
	path := os.Getenv("ROUTECACHE_HELPER_PATH")
	if path == "" {
		t.Skip("helper process only")
	}
	id := os.Getenv("ROUTECACHE_HELPER_ID")
	n, _ := strconv.Atoi(os.Getenv("ROUTECACHE_HELPER_N"))
	s := &Store{Path: path, LockTimeout: 30 * time.Second}
	for j := 0; j < n; j++ {
		if err := s.Record(fmt.Sprintf("p%s-h%d", id, j), "vpn", Outcome{Succeeded: "lan"}); err != nil {
			t.Fatalf("helper %s write %d: %v", id, j, err)
		}
	}
}

func TestStoreConcurrentProcesses(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns processes")
	}
	path := filepath.Join(t.TempDir(), "state.json")
	const procs, perProc = 8, 10
	cmds := make([]*exec.Cmd, procs)
	for i := range cmds {
		cmd := exec.Command(os.Args[0], "-test.run=^TestHelperWriter$", "-test.count=1") // #nosec G204 -- re-running the test binary itself
		cmd.Env = append(os.Environ(),
			"ROUTECACHE_HELPER_PATH="+path,
			"ROUTECACHE_HELPER_ID="+strconv.Itoa(i),
			"ROUTECACHE_HELPER_N="+strconv.Itoa(perProc),
		)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		cmds[i] = cmd
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("helper %d: %v", i, err)
		}
	}
	assertHosts(t, path, procs*perProc)
}

func assertHosts(t *testing.T, path string, want int) {
	t.Helper()
	st, err := (&Store{Path: path}).Load()
	if err != nil {
		t.Fatalf("state file unreadable after concurrent writes: %v", err)
	}
	if len(st.Hosts) != want {
		t.Errorf("hosts = %d, want %d (entries lost)", len(st.Hosts), want)
	}
}
