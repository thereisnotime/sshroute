package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.2.3", "v1.2.3", 0},
		{"1.2.3", "v1.2.3", 0},
		{"v1.2.4", "v1.2.3", 1},
		{"v1.3.0", "v1.2.9", 1},
		{"v2.0.0", "v1.9.9", 1},
		{"v0.2.7", "v0.2.8", -1},
		{"dev", "v0.2.8", -1},          // a source build is always older than a release
		{"v0.2.8-1-gabc", "v0.2.8", 0}, // git-describe suffix ignored
		{"v1.2.3+meta", "v1.2.3", 0},   // build metadata ignored
	}
	for _, tt := range tests {
		if got := CompareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("CompareVersions(%q,%q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestChecksumFor(t *testing.T) {
	body := "abc123  sshroute_1.0.0_linux_amd64.tar.gz\ndef456  sshroute_1.0.0_darwin_arm64.tar.gz\n"
	got, err := ChecksumFor(body, "sshroute_1.0.0_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "def456" {
		t.Errorf("checksum = %q, want def456", got)
	}
	if _, err := ChecksumFor(body, "missing.tar.gz"); err == nil {
		t.Error("expected error for a filename not in checksums")
	}
}

func TestAssetName(t *testing.T) {
	name := AssetName("v1.2.3")
	if !strings.HasPrefix(name, "sshroute_1.2.3_") {
		t.Errorf("AssetName = %q, want sshroute_1.2.3_<os>_<arch>.*", name)
	}
	if !strings.HasSuffix(name, ".tar.gz") && !strings.HasSuffix(name, ".zip") {
		t.Errorf("AssetName = %q, want a .tar.gz or .zip suffix", name)
	}
}

func makeArchive(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestExtractBinary(t *testing.T) {
	want := []byte("\x7fELF-not-really-but-fine")
	got, err := extractBinary(bytes.NewReader(makeArchive(t, "sshroute", want)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("extracted = %q, want %q", got, want)
	}

	if _, err := extractBinary(bytes.NewReader(makeArchive(t, "README.md", []byte("x")))); err == nil {
		t.Error("expected error when the archive has no sshroute binary")
	}
}

func releaseServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, `{"tag_name":%q,"html_url":"https://example/releases/%s","assets":[{"name":"checksums.txt","browser_download_url":"https://example/checksums.txt"}]}`, tag, tag)
	}))
}

func TestLatestRelease(t *testing.T) {
	srv := releaseServer(t, "v9.9.9")
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()

	rel, err := LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.TagName != "v9.9.9" {
		t.Errorf("tag = %q, want v9.9.9", rel.TagName)
	}
	if rel.assetURL("checksums.txt") == "" {
		t.Error("expected checksums.txt asset URL")
	}
}

func TestRun_AlreadyLatest(t *testing.T) {
	srv := releaseServer(t, "v1.0.0")
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()

	var out bytes.Buffer
	applied, err := Run(context.Background(), "v1.0.0", Options{Out: &out})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied {
		t.Error("should not apply when already on the latest version")
	}
	if !strings.Contains(out.String(), "already on the latest") {
		t.Errorf("output = %q, want an up-to-date message", out.String())
	}
}

func TestRun_CheckOnly(t *testing.T) {
	srv := releaseServer(t, "v9.9.9")
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()

	var out bytes.Buffer
	applied, err := Run(context.Background(), "v0.1.0", Options{CheckOnly: true, Out: &out})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied {
		t.Error("check-only must not apply an update")
	}
	if !strings.Contains(out.String(), "update available") {
		t.Errorf("output = %q, want an 'update available' message", out.String())
	}
}

// fullReleaseServer serves a latest release plus its archive, checksums, and bundle.
// checksumOverride, if non-empty, replaces the correct hash to simulate tampering.
func fullReleaseServer(t *testing.T, tag string, archive []byte, checksumOverride string) *httptest.Server {
	t.Helper()
	assetName := AssetName(tag)
	hash := fmt.Sprintf("%x", sha256.Sum256(archive))
	if checksumOverride != "" {
		hash = checksumOverride
	}
	checksums := fmt.Sprintf("%s  %s\n", hash, assetName)

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	mux.HandleFunc("/repos/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":%q,"html_url":"https://x","assets":[`+
			`{"name":%q,"browser_download_url":%q},`+
			`{"name":"checksums.txt","browser_download_url":%q},`+
			`{"name":"checksums.txt.bundle","browser_download_url":%q}]}`,
			tag, assetName, srv.URL+"/archive", srv.URL+"/checksums", srv.URL+"/bundle")
	})
	mux.HandleFunc("/archive", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) })
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, checksums) })
	mux.HandleFunc("/bundle", func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "{}") })
	return srv
}

func TestRun_FullFlow(t *testing.T) {
	fakeCosign(t, 0) // shadow any real cosign so the signature path is deterministic
	fakeBin := []byte("\x7fELF new sshroute binary")
	srv := fullReleaseServer(t, "v9.9.9", makeArchive(t, "sshroute", fakeBin), "")
	defer srv.Close()

	oldBase, oldApply := apiBase, applyBinary
	apiBase = srv.URL
	var captured []byte
	applyBinary = func(b []byte) error { captured = b; return nil }
	defer func() { apiBase = oldBase; applyBinary = oldApply }()

	var out bytes.Buffer
	applied, err := Run(context.Background(), "v0.0.1", Options{Out: &out})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !applied {
		t.Fatal("expected an update to be applied")
	}
	if !bytes.Equal(captured, fakeBin) {
		t.Errorf("installed bytes = %q, want %q", captured, fakeBin)
	}
	if !strings.Contains(out.String(), "sha256 verified") {
		t.Errorf("output = %q, want a sha256-verified message", out.String())
	}
	if !strings.Contains(out.String(), "cosign signature verified") {
		t.Errorf("output = %q, want a cosign-verified message", out.String())
	}
}

func TestRun_WarnsWithoutCosign(t *testing.T) {
	// Point PATH at an empty dir so cosign cannot be found.
	t.Setenv("PATH", t.TempDir())
	fakeBin := []byte("\x7fELF binary")
	srv := fullReleaseServer(t, "v9.9.9", makeArchive(t, "sshroute", fakeBin), "")
	defer srv.Close()

	oldBase, oldApply := apiBase, applyBinary
	apiBase = srv.URL
	applyBinary = func([]byte) error { return nil }
	defer func() { apiBase = oldBase; applyBinary = oldApply }()

	var out bytes.Buffer
	applied, err := Run(context.Background(), "v0.0.1", Options{Out: &out})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !applied {
		t.Error("update should still apply on sha256 alone when cosign is absent")
	}
	if !strings.Contains(out.String(), "WARNING") || !strings.Contains(out.String(), "cosign is not installed") {
		t.Errorf("output = %q, want a no-cosign warning", out.String())
	}
}

func TestRun_ChecksumMismatch(t *testing.T) {
	srv := fullReleaseServer(t, "v9.9.9", makeArchive(t, "sshroute", []byte("real")), strings.Repeat("0", 64))
	defer srv.Close()

	oldBase, oldApply := apiBase, applyBinary
	apiBase = srv.URL
	applied := false
	applyBinary = func(b []byte) error { applied = true; return nil }
	defer func() { apiBase = oldBase; applyBinary = oldApply }()

	ok, err := Run(context.Background(), "v0.0.1", Options{Out: &bytes.Buffer{}})
	if err == nil {
		t.Fatal("expected a checksum-mismatch error")
	}
	if ok || applied {
		t.Error("must not install the binary on a checksum mismatch")
	}
}

func TestRun_NoAssetForPlatform(t *testing.T) {
	// releaseServer advertises only checksums.txt, no archive for this platform.
	srv := releaseServer(t, "v9.9.9")
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()

	if _, err := Run(context.Background(), "v0.0.1", Options{Out: &bytes.Buffer{}}); err == nil {
		t.Error("expected an error when the release has no asset for this platform")
	}
}

func fakeCosign(t *testing.T, exitCode int) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "cosign")
	if err := os.WriteFile(script, []byte(fmt.Sprintf("#!/bin/sh\nexit %d\n", exitCode)), 0o600); err != nil {
		t.Fatalf("write fake cosign: %v", err)
	}
	if err := os.Chmod(script, 0o700); err != nil {
		t.Fatalf("chmod fake cosign: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestVerifyCosign(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "{}")
	}))
	defer srv.Close()

	t.Run("passes when cosign succeeds", func(t *testing.T) {
		fakeCosign(t, 0)
		verified, err := verifyCosign(context.Background(), "checksums", srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !verified {
			t.Error("expected verified=true when cosign exits 0")
		}
	})

	t.Run("errors when cosign fails", func(t *testing.T) {
		fakeCosign(t, 1)
		if _, err := verifyCosign(context.Background(), "checksums", srv.URL); err == nil {
			t.Error("expected an error when cosign exits non-zero")
		}
	})
}

func TestDownload_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := download(context.Background(), srv.URL, io.Discard); err == nil {
		t.Error("expected an error on a 404 download")
	}
}

func TestVerifyCosign_BundleDownloadFails(t *testing.T) {
	fakeCosign(t, 0) // cosign is present, but the bundle can't be fetched
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := verifyCosign(context.Background(), "checksums", srv.URL); err == nil {
		t.Error("expected an error when the cosign bundle download fails")
	}
}

func TestLatestRelease_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()

	if _, err := LatestRelease(context.Background()); err == nil {
		t.Error("expected an error on a 500 response")
	}
}

func TestApplyTo(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "sshroute")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := applyTo(exe, []byte("new binary")); err != nil {
		t.Fatalf("applyTo: %v", err)
	}

	got, err := os.ReadFile(exe) // #nosec G304 -- test-controlled temp path
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "new binary" {
		t.Errorf("content = %q, want %q", got, "new binary")
	}
	fi, err := os.Stat(exe)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("perms = %o, want 0755 (should be preserved)", fi.Mode().Perm())
	}
	// no leftover temp files in the directory
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected only the replaced binary, found %d entries", len(entries))
	}
}

func TestApply(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "sshroute")
	if err := os.WriteFile(fake, []byte("old"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	old := executablePath
	executablePath = func() (string, error) { return fake, nil } // point Apply at a temp file
	defer func() { executablePath = old }()

	if err := Apply([]byte("new binary")); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := os.ReadFile(fake) // #nosec G304 -- test-controlled temp path
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "new binary" {
		t.Errorf("content = %q, want %q", got, "new binary")
	}
}

func TestReplaceMoveAside(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "bin")
	tmp := filepath.Join(dir, "tmpnew")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatalf("setup exe: %v", err)
	}
	if err := os.WriteFile(tmp, []byte("new"), 0o755); err != nil {
		t.Fatalf("setup tmp: %v", err)
	}

	if err := replaceMoveAside(exe, tmp); err != nil {
		t.Fatalf("replaceMoveAside: %v", err)
	}
	got, err := os.ReadFile(exe) // #nosec G304 -- test-controlled temp path
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
	if _, err := os.Stat(exe + ".old"); !os.IsNotExist(err) {
		t.Error(".old sidecar should have been cleaned up")
	}
}

func TestReplaceRename_Error(t *testing.T) {
	dir := t.TempDir()
	tmp := filepath.Join(dir, "tmp")
	if err := os.WriteFile(tmp, []byte("x"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Target directory does not exist, so the rename must fail.
	if err := replaceRename(filepath.Join(dir, "missing", "bin"), tmp); err == nil {
		t.Error("expected an error renaming into a missing directory")
	}
}

func TestReplaceMoveAside_Error(t *testing.T) {
	dir := t.TempDir()
	tmp := filepath.Join(dir, "tmp")
	if err := os.WriteFile(tmp, []byte("x"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// The target does not exist, so moving it aside must fail.
	if err := replaceMoveAside(filepath.Join(dir, "does-not-exist"), tmp); err == nil {
		t.Error("expected an error when the target is missing")
	}
}
