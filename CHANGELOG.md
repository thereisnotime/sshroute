# Changelog

All notable changes to sshroute are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/).

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
