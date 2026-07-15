## Context

Releases are built by GoReleaser and contain, per tag: `sshroute_<ver>_<os>_<arch>.tar.gz` (`.zip` on Windows), `checksums.txt`, and a cosign `checksums.txt.bundle` (keyless, signed by the release workflow's OIDC identity). The self-update consumes exactly these.

## Decisions

### Verification: sha256 always, cosign when available
The archive's sha256 is matched against its line in `checksums.txt` (fetched over TLS from GitHub). That is the mandatory integrity gate. When `cosign` is found on PATH, the command additionally runs `cosign verify-blob` on `checksums.txt` using its bundle, pinned to the release workflow identity (`--certificate-identity-regexp` for `github.com/thereisnotime/sshroute/.github/workflows/*@refs/tags/v*`, issuer `token.actions.githubusercontent.com`). If cosign is present and verification fails, the update aborts. If cosign is absent, it is skipped with a notice — sha256 still gates. This gives supply-chain assurance for users who have cosign, without making it a hard dependency.

### Atomic self-replace, no dependency
The new binary is written to a temp file in the *same directory* as the resolved executable (so `rename` is atomic on one filesystem), `chmod`ed to the existing binary's permissions, then `os.Rename`d over the target. On Linux/macOS a running binary can be replaced this way. On Windows the running image cannot be renamed over, so the current file is moved to `<exe>.old` first, with rollback on failure. `os.Executable` is resolved through symlinks so the real file is replaced, not a symlink.

### No new dependencies
`archive/tar`, `compress/gzip`, `crypto/sha256`, `net/http`, and `os` cover everything. A hand-rolled 3-field `X.Y.Z` version comparison avoids pulling a semver module; a source build (`dev`) compares as older than any release so it always offers the update.

### Scope: release-binary installs
The command replaces whatever `os.Executable` points at. For `go install` / package-manager installs it would still work, but a later reinstall reverts it, so the help text directs those users to their installer.

## Risks / Trade-offs

- **Unwritable install dir** (e.g. `/usr/bin`): the temp-file create fails with a clear "insufficient permissions" message rather than a cryptic rename error.
- **GitHub API rate limit**: anonymous requests are limited but a single `update` call is one request; acceptable.
- **Decompression safety**: reads are bounded by a 100 MiB limit to guard against a malicious archive.

## Testability

Pure functions (version compare, checksum parse, asset name, tar extraction) are unit-tested directly. Release lookup and the `Run` control flow (already-current, `--check`) are tested against an `httptest` server via the overridable `apiBase`. The actual self-replace is not unit-tested (it would overwrite the test binary); it is exercised manually.
