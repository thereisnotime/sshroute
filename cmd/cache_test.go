package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thereisnotime/sshroute/internal/config"
	"github.com/thereisnotime/sshroute/internal/routecache"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

// cacheTestCfg has no networks, so the detected network is always "default" and
// the base order is lan, vpn (priority 0, by name), then default.
func cacheTestCfg() *config.Config {
	return &config.Config{
		Networks: make(map[string]config.NetworkDefinition),
		Hosts: map[string]config.HostConfig{
			"srv": {
				"default": {Host: "pub.example.com"},
				"vpn":     {Host: "10.0.0.1"},
				"lan":     {Host: "192.168.77.1"},
			},
		},
	}
}

// withCacheEnv points the route cache at a temp file and clears any env overrides.
func withCacheEnv(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state", "state.json")
	t.Setenv("SSHROUTE_CACHE_PATH", path)
	t.Setenv("SSHROUTE_CACHE", "")
	noCache = false
	t.Cleanup(func() { noCache = false })
	return path
}

// fakeSSH writes an ssh stand-in that appends its target host to a log and exits 0
// only for okHost, 255 otherwise. It is installed via SSHROUTE_SSH.
func fakeSSH(t *testing.T, okHost string) (logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "calls.log")
	script := filepath.Join(dir, "ssh")
	body := "#!/bin/sh\nfor a in \"$@\"; do last=\"$a\"; done\n" +
		"echo \"$last\" >> '" + logPath + "'\n" +
		"[ \"$last\" = '" + okHost + "' ] && exit 0\nexit 255\n"
	if err := os.WriteFile(script, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(script, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSHROUTE_SSH", script)
	old := ssh.RealSSH
	ssh.RealSSH = script
	t.Cleanup(func() { ssh.RealSSH = old })
	oldStatus := ssh.FallbackStatus
	ssh.FallbackStatus = &bytes.Buffer{}
	t.Cleanup(func() { ssh.FallbackStatus = oldStatus })
	return logPath
}

func readCalls(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(data))
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String(), runErr
}

func loadState(t *testing.T, path string) *routecache.State {
	t.Helper()
	st, err := (&routecache.Store{Path: path}).Load()
	if err != nil {
		t.Fatalf("loading state: %v", err)
	}
	return st
}

func runFallbackConnect(t *testing.T) error {
	t.Helper()
	fallback = true
	defer func() { fallback = false }()
	return runConnect(connectCmd, []string{"srv"})
}

func TestConnectFallback_RecordsThenUsesCache(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	logPath := fakeSSH(t, "10.0.0.1")

	if err := runFallbackConnect(t); err != nil {
		t.Fatalf("first connect: %v", err)
	}
	if got, want := readCalls(t, logPath), []string{"192.168.77.1", "10.0.0.1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first run tried %v, want %v", got, want)
	}
	st := loadState(t, statePath)
	h := st.Hosts["srv"]
	if h == nil || h.Networks["default"].Profile != "vpn" || h.LastSuccess.Profile != "vpn" {
		t.Fatalf("success not recorded: %+v", h)
	}
	if _, ok := h.Failures["lan"]; !ok {
		t.Errorf("failure of lan not recorded: %+v", h.Failures)
	}

	_ = os.Remove(logPath)
	if err := runFallbackConnect(t); err != nil {
		t.Fatalf("second connect: %v", err)
	}
	if got := readCalls(t, logPath); !reflect.DeepEqual(got, []string{"10.0.0.1"}) {
		t.Errorf("second run tried %v, want the cached route only", got)
	}
}

func TestConnectFallback_StaleCacheFallsThrough(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	logPath := fakeSSH(t, "pub.example.com")

	// Cache says vpn, but now only default works.
	store := &routecache.Store{Path: statePath, LockTimeout: time.Second}
	if err := store.Record("srv", "default", routecache.Outcome{Succeeded: "vpn"}); err != nil {
		t.Fatal(err)
	}
	if err := runFallbackConnect(t); err != nil {
		t.Fatalf("connect: %v", err)
	}
	want := []string{"10.0.0.1", "192.168.77.1", "pub.example.com"}
	if got := readCalls(t, logPath); !reflect.DeepEqual(got, want) {
		t.Errorf("tried %v, want %v (cached first, then the rest once)", got, want)
	}
	if p := loadState(t, statePath).Hosts["srv"].Networks["default"].Profile; p != "default" {
		t.Errorf("cache not updated to the working profile, got %q", p)
	}
}

func TestConnectFallback_AllExhaustedRecordsFailures(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	fakeSSH(t, "nothing-works")

	if err := runFallbackConnect(t); err == nil || !strings.Contains(err.Error(), "exhausted") {
		t.Fatalf("err = %v, want exhausted", err)
	}
	h := loadState(t, statePath).Hosts["srv"]
	if h == nil || h.LastSuccess != nil || len(h.Failures) != 3 {
		t.Errorf("state = %+v, want three failures and no success", h)
	}
}

func TestConnectFallback_NoCacheFlag(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	logPath := fakeSSH(t, "10.0.0.1")
	store := &routecache.Store{Path: statePath, LockTimeout: time.Second}
	if err := store.Record("srv", "default", routecache.Outcome{Succeeded: "vpn"}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(statePath)

	noCache = true
	if err := runFallbackConnect(t); err != nil {
		t.Fatal(err)
	}
	if got := readCalls(t, logPath); got[0] != "192.168.77.1" {
		t.Errorf("--no-cache still used the cache: tried %v", got)
	}
	after, _ := os.ReadFile(statePath)
	if !bytes.Equal(before, after) {
		t.Error("--no-cache wrote to the state file")
	}
}

func TestConnectFallback_EnvDisablesCache(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	t.Setenv("SSHROUTE_CACHE", "0")
	fakeSSH(t, "10.0.0.1")

	if err := runFallbackConnect(t); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Error("SSHROUTE_CACHE=0 still wrote the state file")
	}
}

func TestConnectFallback_ConfigDisablesCache(t *testing.T) {
	cfg := cacheTestCfg()
	off := false
	cfg.Cache.Enabled = &off
	withTempConfig(t, cfg)
	statePath := withCacheEnv(t)
	fakeSSH(t, "10.0.0.1")

	if err := runFallbackConnect(t); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Error("cache.enabled: false still wrote the state file")
	}
}

func TestConnectFallback_CorruptStateIgnored(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	fakeSSH(t, "10.0.0.1")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte("\x00garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runFallbackConnect(t); err != nil {
		t.Fatalf("corrupt state must not break connect: %v", err)
	}
	if loadState(t, statePath).Hosts["srv"] == nil {
		t.Error("corrupt state file was not rewritten")
	}
}

func TestConnectFallback_DryRunShowsCache(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	store := &routecache.Store{Path: statePath, LockTimeout: time.Second}
	if err := store.Record("srv", "default", routecache.Outcome{Succeeded: "vpn"}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(statePath)

	dryRun = true
	defer func() { dryRun = false }()
	out, err := captureStdout(t, func() error { return runFallbackConnect(t) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[dry-run] cache: hit", "[dry-run] order: vpn (cache), lan (priority), default (default)"} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run output missing %q:\n%s", want, out)
		}
	}
	if lines := strings.Count(out, "[dry-run] "); lines != 5 {
		t.Errorf("expected 2 plan lines + 3 commands, got %d:\n%s", lines, out)
	}
	after, _ := os.ReadFile(statePath)
	if !bytes.Equal(before, after) {
		t.Error("dry-run wrote to the state file")
	}
}

func TestConnectReconnectDryRunFallback_ShowsCache(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	withCacheEnv(t)
	noCache = true
	reconnect, fallback, dryRun = true, true, true
	defer func() { reconnect, fallback, dryRun = false, false, false }()
	out, err := captureStdout(t, func() error { return runConnect(connectCmd, []string{"srv"}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[dry-run] cache: disabled (--no-cache)") {
		t.Errorf("output = %s", out)
	}
}

func TestAttemptOnce_FallbackRecordsCache(t *testing.T) {
	cfg := cacheTestCfg()
	statePath := withCacheEnv(t)
	logPath := fakeSSH(t, "pub.example.com")
	fallback = true
	defer func() { fallback = false }()

	code, err := attemptOnce(context.Background(), cfg, "srv", "default", ssh.ParsedArgs{})
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if p := loadState(t, statePath).Hosts["srv"].Networks["default"].Profile; p != "default" {
		t.Errorf("reconnect attempt did not record, got %q", p)
	}
	_ = os.Remove(logPath)
	if _, err := attemptOnce(context.Background(), cfg, "srv", "default", ssh.ParsedArgs{}); err != nil {
		t.Fatal(err)
	}
	if got := readCalls(t, logPath); !reflect.DeepEqual(got, []string{"pub.example.com"}) {
		t.Errorf("second attempt tried %v, want cached route only", got)
	}
}

func TestAttemptOnce_CancelledContextDoesNotRecord(t *testing.T) {
	cfg := cacheTestCfg()
	statePath := withCacheEnv(t)
	fakeSSH(t, "pub.example.com")
	fallback = true
	defer func() { fallback = false }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = attemptOnce(ctx, cfg, "srv", "default", ssh.ParsedArgs{})
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Error("attempts cut short by a signal must not be recorded")
	}
}

func TestOutcomeFromAttempts(t *testing.T) {
	tests := []struct {
		name string
		in   []ssh.Attempt
		want routecache.Outcome
	}{
		{"none", nil, routecache.Outcome{}},
		{"success after failures", []ssh.Attempt{{Profile: "a", Code: 255}, {Profile: "b", Code: 0}}, routecache.Outcome{Succeeded: "b", Failed: []string{"a"}}},
		{"remote failure counts as connected", []ssh.Attempt{{Profile: "a", Code: 1}}, routecache.Outcome{Succeeded: "a"}},
		{"signal-killed proves nothing", []ssh.Attempt{{Profile: "a", Code: -1}}, routecache.Outcome{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := outcomeFromAttempts(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRecordFallback_WriteErrorIsSwallowed(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	// The state directory cannot be created under a regular file.
	opts := routecache.Options{Enabled: true, Path: filepath.Join(blocker, "state.json"), LockTimeout: time.Millisecond}
	recordFallback(opts, "srv", "default", []ssh.Attempt{{Profile: "vpn", Code: 0}})
}

func TestCacheOptions_InvalidSettingsDisable(t *testing.T) {
	withCacheEnv(t)
	cfg := cacheTestCfg()
	cfg.Cache.TTL = "nope"
	if o := cacheOptions(cfg); o.Enabled || !strings.Contains(o.Disabled, "invalid") {
		t.Errorf("options = %+v, want disabled for invalid settings", o)
	}
}

func TestCacheFlagsRegistered(t *testing.T) {
	if connectCmd.Flags().Lookup("no-cache") == nil {
		t.Error("connect is missing --no-cache")
	}
	if resolveCmd.Flags().Lookup("no-cache") == nil {
		t.Error("resolve is missing --no-cache")
	}
	if resolveCmd.Flags().Lookup("fallback") == nil {
		t.Error("resolve is missing --fallback")
	}
	for _, name := range []string{"show", "clear", "path"} {
		if c, _, err := rootCmd.Find([]string{"cache", name}); err != nil || c.Name() != name {
			t.Errorf("cache %s not registered", name)
		}
	}
}

func seedState(t *testing.T, path string) {
	t.Helper()
	store := &routecache.Store{Path: path, LockTimeout: time.Second}
	if err := store.Record("srv", "default", routecache.Outcome{Succeeded: "vpn"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Record("srv", "office", routecache.Outcome{Succeeded: "gone"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Record("old", "default", routecache.Outcome{Succeeded: "default"}); err != nil {
		t.Fatal(err)
	}
}

func TestCacheShow_JSON(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	seedState(t, statePath)
	old := output
	output = "json"
	defer func() { output = old }()

	out, err := captureStdout(t, func() error { return runCacheShow(cacheShowCmd, []string{"srv"}) })
	if err != nil {
		t.Fatal(err)
	}
	var rows []CacheRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	got := map[string]string{}
	for _, r := range rows {
		if r.Alias != "srv" {
			t.Errorf("filter leaked alias %q", r.Alias)
		}
		got[r.Key+"="+r.Profile] = r.Status
	}
	want := map[string]string{
		"*=gone":      "unknown-profile",
		"default=vpn": "fresh",
		"office=gone": "unknown-profile",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestCacheShow_TableAndEmpty(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	old := output
	output = "table"
	defer func() { output = old }()

	out, err := captureStdout(t, func() error { return runCacheShow(cacheShowCmd, nil) })
	if err != nil || !strings.Contains(out, "no cached routes") {
		t.Fatalf("empty show: out=%q err=%v", out, err)
	}
	seedState(t, statePath)
	out, err = captureStdout(t, func() error { return runCacheShow(cacheShowCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"LAST SUCCESS", "srv", "old", "fresh"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q:\n%s", want, out)
		}
	}
}

func TestCacheShow_ExpiredStatus(t *testing.T) {
	cfg := cacheTestCfg()
	cfg.Cache.TTL = "1h"
	withTempConfig(t, cfg)
	statePath := withCacheEnv(t)
	seedState(t, statePath)
	oldNow := cacheNow
	cacheNow = func() time.Time { return time.Now().Add(2 * time.Hour) }
	defer func() { cacheNow = oldNow }()
	old := output
	output = "json"
	defer func() { output = old }()

	out, err := captureStdout(t, func() error { return runCacheShow(cacheShowCmd, []string{"old"}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"status": "unknown-profile"`) || strings.Contains(out, "fresh") {
		t.Errorf("alias not in config should be unknown-profile, never fresh:\n%s", out)
	}
	out, _ = captureStdout(t, func() error { return runCacheShow(cacheShowCmd, []string{"srv"}) })
	if !strings.Contains(out, `"status": "expired"`) {
		t.Errorf("expected an expired row:\n%s", out)
	}
}

func TestCacheClear(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	seedState(t, statePath)

	out, err := captureStdout(t, func() error { return runCacheClear(cacheClearCmd, []string{"srv"}) })
	if err != nil || !strings.Contains(out, `cleared cached routes for "srv"`) {
		t.Fatalf("out=%q err=%v", out, err)
	}
	st := loadState(t, statePath)
	if st.Hosts["srv"] != nil || st.Hosts["old"] == nil {
		t.Errorf("clear srv removed the wrong entries: %+v", st.Hosts)
	}
	out, _ = captureStdout(t, func() error { return runCacheClear(cacheClearCmd, []string{"srv"}) })
	if !strings.Contains(out, "no cached routes") {
		t.Errorf("second clear: %q", out)
	}
	if _, err := captureStdout(t, func() error { return runCacheClear(cacheClearCmd, nil) }); err != nil {
		t.Fatal(err)
	}
	if n := len(loadState(t, statePath).Hosts); n != 0 {
		t.Errorf("clear all left %d hosts", n)
	}
}

func TestCachePath(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	out, err := captureStdout(t, func() error { return runCachePath(cachePathCmd, nil) })
	if err != nil || strings.TrimSpace(out) != statePath {
		t.Errorf("out=%q err=%v, want %q", out, err, statePath)
	}
}

func TestCacheCommands_LoadConfigError(t *testing.T) {
	withInvalidConfig(t)
	for name, fn := range map[string]func() error{
		"show":  func() error { return runCacheShow(cacheShowCmd, nil) },
		"clear": func() error { return runCacheClear(cacheClearCmd, nil) },
		"path":  func() error { return runCachePath(cachePathCmd, nil) },
	} {
		if err := fn(); err == nil {
			t.Errorf("cache %s: expected config error", name)
		}
	}
}

func TestResolveFallback_ShowsCachedRouteFirst(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	statePath := withCacheEnv(t)
	store := &routecache.Store{Path: statePath, LockTimeout: time.Second}
	if err := store.Record("srv", "default", routecache.Outcome{Succeeded: "vpn"}); err != nil {
		t.Fatal(err)
	}
	resolveFallback = true
	old := output
	output = "json"
	defer func() { resolveFallback = false; output = old }()

	out, err := captureStdout(t, func() error { return runResolve(resolveCmd, []string{"srv"}) })
	if err != nil {
		t.Fatal(err)
	}
	var rows []ResolveOrderRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	if len(rows) != 3 || rows[0].Profile != "vpn" || rows[0].Source != "cache" || rows[0].CachedAt == "" || rows[0].Order != 1 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[2].Profile != "default" || rows[2].CachedAt != "" {
		t.Errorf("last row = %+v", rows[2])
	}

	noCache = true
	out, _ = captureStdout(t, func() error { return runResolve(resolveCmd, []string{"srv"}) })
	if strings.Contains(out, `"source": "cache"`) {
		t.Errorf("--no-cache still shows a cache source:\n%s", out)
	}
}

func TestResolveFallback_Table(t *testing.T) {
	withTempConfig(t, cacheTestCfg())
	withCacheEnv(t)
	resolveFallback = true
	old := output
	output = "table"
	defer func() { resolveFallback = false; output = old }()
	out, err := captureStdout(t, func() error { return runResolve(resolveCmd, []string{"srv"}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SOURCE") || !strings.Contains(out, "priority") {
		t.Errorf("table = %s", out)
	}
}
