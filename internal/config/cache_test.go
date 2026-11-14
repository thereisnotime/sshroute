package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"7d", 7 * 24 * time.Hour, false},
		{"1.5d", 36 * time.Hour, false},
		{"12h", 12 * time.Hour, false},
		{"90m", 90 * time.Minute, false},
		{"0", 0, false},
		{" 2h ", 2 * time.Hour, false},
		{"", 0, true},
		{"d", 0, true},
		{"soon", 0, true},
		{"-1h", 0, true},
		{"-2d", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseDuration(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseDuration(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseDuration(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestValidateCache(t *testing.T) {
	tests := []struct {
		name    string
		cache   CacheConfig
		wantErr string
	}{
		{"empty", CacheConfig{}, ""},
		{"full valid", CacheConfig{TTL: "7d", Scope: "alias", LockTimeout: "500ms"}, ""},
		{"network scope", CacheConfig{Scope: "network"}, ""},
		{"bad ttl", CacheConfig{TTL: "forever"}, "cache.ttl"},
		{"bad lock timeout", CacheConfig{LockTimeout: "1x"}, "cache.lock_timeout"},
		{"bad scope", CacheConfig{Scope: "host"}, "cache.scope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&Config{Cache: tt.cache})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestCacheConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	off := false
	in := &Config{
		Hosts: map[string]HostConfig{"srv": {"default": {Host: "h"}}},
		Cache: CacheConfig{Enabled: &off, Path: "~/state.json", TTL: "2d", Scope: "alias", LockTimeout: "2s"},
	}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	c := out.Cache
	if c.Enabled == nil || *c.Enabled || c.Path != "~/state.json" || c.TTL != "2d" || c.Scope != "alias" || c.LockTimeout != "2s" {
		t.Errorf("cache section did not round-trip: %+v", c)
	}
}

func TestCacheConfigOmittedWhenEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := Save(path, &Config{Hosts: map[string]HostConfig{"srv": {"default": {Host: "h"}}}}); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Cache.Enabled != nil {
		t.Error("empty cache section should stay unset")
	}
}
