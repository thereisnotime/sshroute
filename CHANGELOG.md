# Changelog

All notable changes to sshroute are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/).

---

## [0.2.8](https://github.com/thereisnotime/sshroute/compare/v0.2.7...v0.2.8) (2026-07-15)


### Features

* add --reconnect for supervised auto-reconnect ([3fe54e2](https://github.com/thereisnotime/sshroute/commit/3fe54e21404009bd1b94c5ac08874fedd26dd903))
* **connect:** add --reconnect to supervise and reconnect on drop ([4062175](https://github.com/thereisnotime/sshroute/commit/4062175d5dfe2bb5c3fa0ab19d2416e4b9ca5548))


### Bug Fixes

* **deps:** bump golang.org/x/text to v0.39.0 ([baef4bf](https://github.com/thereisnotime/sshroute/commit/baef4bfdf80c419ea06bee289eee6d7f8fa6f8ad))


### Dependencies

* bump go toolchain to 1.26.5 ([c17ea56](https://github.com/thereisnotime/sshroute/commit/c17ea5694f59ec61614e4dae83b0c0de4450c833))
* **deps:** bump go modules and GitHub Actions to latest ([e656ea5](https://github.com/thereisnotime/sshroute/commit/e656ea5e69faf26487a8086076b426b942ab055d))

## [0.1.1] - 2026-04-07

### Changed

- Expanded CLI reference documentation — every command and flag is now fully documented in the README.
- Added Community section to README covering how to obtain, report issues, and contribute.
- Added CONTRIBUTING.md with coding standards, project layout, and test instructions.
- Added SECURITY.md with private vulnerability reporting via GitHub Security Advisories.
- Added GitHub issue templates for bug reports and feature requests.
- Added CHANGELOG.md following Keep a Changelog format.
- OpenSSF Best Practices badge added to README.

### Fixed

- Restored canonical Apache 2.0 LICENSE text so pkg.go.dev detects the license correctly.
- Fixed gosec suppression annotations (`// #nosec G204/G304`) — previous `//nolint:gosec` comments only worked with golangci-lint, not gosec directly.
- Handled unhandled `f.Close()` error in `cmd/config.go`.
- Added interface name validation in `internal/network/iface.go` to prevent path traversal.
- Pinned `sigstore/cosign-installer`, `aquasecurity/trivy-action`, `github/codeql-action`, and all other GitHub Actions to SHA digests.
- Pinned `go install` tool versions (`gosec@v2.25.0`, `govulncheck@v1.1.4`).
- Pinned Dockerfile `FROM` images to digest SHAs.
- Updated `trivy-action` to v0.35.0 — v0.30.0 internally referenced a deleted `setup-trivy@v0.2.2` action, breaking CI.

### Added

- Unit tests across all internal packages — coverage improved from 0% to ~60%.
- Fuzz test for SSH argument parser (`FuzzParseArgs`).
- Branch protection now requires one approving review before merge.

---

## [0.1.0] - 2026-04-07

Initial public release.

### Added

- Network-aware SSH routing: detects active network or VPN and selects the right host, port, identity file, and jump host automatically.
- Four network detection methods: `route` (kernel routing table), `interface` (sysfs operstate), `ping` (ICMP with fallback to system `ping`), and `exec` (arbitrary shell command, exit 0 = match).
- Priority-ordered network evaluation — lower `priority` value is checked first; alphabetical tie-break for determinism.
- AND logic within a network: all checks must pass for the network to match.
- Per-host profiles: a required `default` profile plus optional per-network overrides; unset fields inherit from `default`.
- Shadow / transparent mode: install as `ssh` earlier in `$PATH`; unknown hosts pass through to `/usr/bin/ssh` unchanged.
- CLI commands: `init`, `connect`, `list`, `add`, `remove`, `network`, `network list`, `network test`, `config`, `config edit`, `version`.
- `--dry-run` flag to preview the resolved SSH command without executing.
- Output formats: `table` (default), `json`, `yaml` for all list commands.
- XDG-aware config path resolution with `$SSHROUTE_CONFIG` override.
- Atomic config writes (temp file + rename) with 0600 permissions.
- `sshroute init` to create a starter config with commented examples.
- Example configs: `basic`, `multi-network`, `wireguard-backconnect`, `jump-hosts`.
- GoReleaser pipeline: binaries for Linux/macOS/Windows on amd64 and arm64, cosign keyless signing, SBOM generation.
- CI: build, test with race detector, Codecov coverage, gosec SAST, govulncheck + Trivy SCA, OpenSSF Scorecard.

[0.1.0]: https://github.com/thereisnotime/sshroute/releases/tag/v0.1.0
