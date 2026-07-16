// Package update implements the self-update command. It finds the latest GitHub
// release, downloads the release archive for the running platform, verifies its
// sha256 against checksums.txt (and the cosign bundle when cosign is available),
// then atomically replaces the running executable.
package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	owner = "thereisnotime"
	repo  = "sshroute"

	// maxArchiveBytes bounds decompression to guard against a malicious archive.
	maxArchiveBytes = 100 << 20 // 100 MiB
)

// Overridable for tests.
var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	apiBase    = "https://api.github.com"
	// applyBinary installs the verified binary; indirected so tests can exercise
	// the full download/verify/extract flow without replacing the test executable.
	applyBinary = Apply
	// executablePath resolves the running binary; indirected so Apply is testable
	// against a temp file rather than the test binary.
	executablePath = os.Executable
)

// Release is the subset of the GitHub release API that we use.
type Release struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

// Asset is a single downloadable release asset.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func (r *Release) assetURL(name string) string {
	for _, a := range r.Assets {
		if a.Name == name {
			return a.URL
		}
	}
	return ""
}

// Options controls Run.
type Options struct {
	CheckOnly bool
	Force     bool
	Out       io.Writer
}

// Run checks for a newer release and, unless CheckOnly is set, downloads,
// verifies, and installs it. It returns true if a new binary was applied.
func Run(ctx context.Context, current string, opts Options) (bool, error) {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	rel, err := LatestRelease(ctx)
	if err != nil {
		return false, err
	}
	latest := rel.TagName

	if CompareVersions(latest, current) <= 0 && !opts.Force {
		fmt.Fprintf(opts.Out, "already on the latest version (%s)\n", current)
		return false, nil
	}
	if opts.CheckOnly {
		fmt.Fprintf(opts.Out, "update available: %s -> %s\n%s\n", current, latest, rel.HTMLURL)
		return false, nil
	}

	assetName := AssetName(latest)
	assetURL := rel.assetURL(assetName)
	checksumsURL := rel.assetURL("checksums.txt")
	if assetURL == "" || checksumsURL == "" {
		return false, fmt.Errorf("release %s has no asset for %s/%s", latest, runtime.GOOS, runtime.GOARCH)
	}

	fmt.Fprintf(opts.Out, "downloading %s (%s)...\n", latest, assetName)
	archive := &bytes.Buffer{}
	sum, err := download(ctx, assetURL, archive)
	if err != nil {
		return false, err
	}

	checksums, err := downloadString(ctx, checksumsURL)
	if err != nil {
		return false, err
	}
	want, err := ChecksumFor(checksums, assetName)
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(sum, want) {
		return false, fmt.Errorf("checksum mismatch for %s: got %s, want %s", assetName, sum, want)
	}
	fmt.Fprintln(opts.Out, "sha256 verified")

	if bundleURL := rel.assetURL("checksums.txt.bundle"); bundleURL != "" {
		verified, err := verifyCosign(ctx, checksums, bundleURL)
		if err != nil {
			return false, err
		}
		if verified {
			fmt.Fprintln(opts.Out, "cosign signature verified")
		} else {
			fmt.Fprintln(opts.Out, "WARNING: cosign is not installed, so the release signature was NOT verified.")
			fmt.Fprintln(opts.Out, "         The download is checked for integrity via sha256 against checksums.txt")
			fmt.Fprintln(opts.Out, "         (fetched over HTTPS), but its provenance is not cryptographically proven.")
			fmt.Fprintln(opts.Out, "         Install cosign (https://github.com/sigstore/cosign) for full verification.")
		}
	}

	bin, err := extractBinary(archive)
	if err != nil {
		return false, err
	}
	if err := applyBinary(bin); err != nil {
		return false, err
	}
	fmt.Fprintf(opts.Out, "updated to %s\n", latest)
	return true, nil
}

// LatestRelease fetches the latest published release from GitHub.
func LatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req) // #nosec G107 -- url is built from constants and the fixed GitHub API host
	if err != nil {
		return nil, fmt.Errorf("querying latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned %s", resp.Status)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decoding release: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("latest release has no tag")
	}
	return &rel, nil
}

// AssetName returns the goreleaser archive name for the running platform.
func AssetName(version string) string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("sshroute_%s_%s_%s.%s", strings.TrimPrefix(version, "v"), runtime.GOOS, runtime.GOARCH, ext)
}

// ChecksumFor returns the hex sha256 recorded for filename in a checksums.txt body.
func ChecksumFor(checksums, filename string) (string, error) {
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum listed for %q", filename)
}

// CompareVersions compares two X.Y.Z versions (a leading "v" and any -pre/+meta
// suffix are ignored). It returns 1 if a > b, -1 if a < b, and 0 if equal.
func CompareVersions(a, b string) int {
	pa, pb := parseVer(a), parseVer(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			if pa[i] > pb[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func parseVer(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i != -1 {
		v = v[:i]
	}
	var out [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		if i > 2 {
			break
		}
		out[i], _ = strconv.Atoi(part)
	}
	return out
}

// download streams url into dst and returns the sha256 hex of the bytes written.
func download(ctx context.Context, url string, dst io.Writer) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req) // #nosec G107 -- url is a release asset URL from the GitHub API response
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading %s: %s", url, resp.Status)
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, h), io.LimitReader(resp.Body, maxArchiveBytes)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func downloadString(ctx context.Context, url string) (string, error) {
	var buf bytes.Buffer
	if _, err := download(ctx, url, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// extractBinary returns the sshroute binary bytes from a gzipped tar archive.
func extractBinary(r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("opening archive: %w", err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}
		base := filepath.Base(hdr.Name)
		if base == "sshroute" || base == "sshroute.exe" {
			return io.ReadAll(io.LimitReader(tr, maxArchiveBytes))
		}
	}
	return nil, fmt.Errorf("no sshroute binary found in archive")
}

// verifyCosign verifies the checksums.txt cosign bundle when cosign is on PATH.
// It returns (false, nil) when cosign is not installed, and an error when cosign
// is installed but verification fails.
func verifyCosign(ctx context.Context, checksums, bundleURL string) (bool, error) {
	cosign, err := exec.LookPath("cosign")
	if err != nil {
		return false, nil
	}
	dir, err := os.MkdirTemp("", "sshroute-cosign-")
	if err != nil {
		return false, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	checksumsPath := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksumsPath, []byte(checksums), 0o600); err != nil {
		return false, err
	}
	bundlePath := filepath.Join(dir, "checksums.txt.bundle")
	if bundle, err := downloadString(ctx, bundleURL); err != nil {
		return false, err
	} else if err := os.WriteFile(bundlePath, []byte(bundle), 0o600); err != nil {
		return false, err
	}

	// The keyless signing identity is a release workflow of this repo. Releases are
	// built either by the release-please workflow (running on refs/heads/main) or by
	// the tag-push Release workflow (refs/tags/v*), so accept both.
	cmd := exec.CommandContext(ctx, cosign, "verify-blob", // #nosec G204 -- cosign path from LookPath; args are constants and temp file paths
		"--bundle", bundlePath,
		"--certificate-identity-regexp", `^https://github\.com/thereisnotime/sshroute/\.github/workflows/.+@refs/(heads/main|tags/v.+)$`,
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
		checksumsPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("cosign verification failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return true, nil
}

// Apply atomically replaces the running executable with newBin.
func Apply(newBin []byte) error {
	exe, err := executablePath()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return applyTo(exe, newBin)
}

// applyTo atomically replaces the file at exe with newBin, preserving its
// permissions. Split from Apply so the replacement logic is testable without
// overwriting the running test binary.
func applyTo(exe string, newBin []byte) error {
	dir := filepath.Dir(exe)

	// Preserve the existing binary's permissions (falling back to 0755).
	mode := os.FileMode(0o755)
	if fi, err := os.Stat(exe); err == nil {
		mode = fi.Mode().Perm()
	}

	tmp, err := os.CreateTemp(dir, ".sshroute-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s (insufficient permissions? try running as the file owner): %w", dir, err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(newBin); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}

	replace := replaceRename
	if runtime.GOOS == "windows" {
		replace = replaceMoveAside
	}
	if err := replace(exe, tmpName); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// replaceRename swaps tmp in for exe with a single atomic rename. Works on
// Linux/macOS, where a running binary can be replaced this way.
func replaceRename(exe, tmp string) error {
	if err := os.Rename(tmp, exe); err != nil {
		return fmt.Errorf("replacing %s: %w", exe, err)
	}
	return nil
}

// replaceMoveAside swaps tmp in for exe by first moving the current file aside,
// which Windows requires because a running image cannot be renamed over. It rolls
// back on failure. The logic is rename-based, so it is exercised on any platform.
func replaceMoveAside(exe, tmp string) error {
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Rename(old, exe) // roll back
		return err
	}
	_ = os.Remove(old)
	return nil
}
