## Why

sshroute is distributed as signed GitHub release binaries, but updating means manually finding the release, downloading the right archive, checking the hash, and replacing the binary. A built-in `update` command makes that a single, verified step for users who installed the release binary.

## What Changes

- Add `sshroute update [--check] [--force]`.
- It queries the latest GitHub release, downloads the archive for the running `GOOS`/`GOARCH`, verifies its sha256 against `checksums.txt`, and (when `cosign` is on PATH) verifies the cosign bundle for `checksums.txt`.
- It then atomically replaces the running executable (temp file in the same directory + rename; move-aside on Windows), preserving the existing file permissions.
- `--check` reports whether a newer version exists without installing; `--force` reinstalls the latest even if already current.
- Verification is mandatory: sha256 must match, and if `cosign` is installed its verification must pass, or the update aborts before touching the binary.

## Capabilities

### New Capabilities

None — this adds a subcommand to the existing `commands` capability.

### Modified Capabilities

- `commands`: adds the `update` subcommand.

## Impact

- `internal/update/` (new) — release lookup, download, sha256 + cosign verification, atomic self-replace
- `cmd/update.go` (new) — thin Cobra command
- No new dependencies; uses the standard library and an optional external `cosign`.
- Aimed at release-binary installs; `go install` / package-manager installs update via those tools.
