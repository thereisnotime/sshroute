# Tasks: SSH binary resolution

- [x] Change `RealSSH` from const to var in `internal/ssh/executor.go`
- [x] Add `ResolveSSHBinary(cfg)` with env → config → LookPath → fallback chain
- [x] Add `sameFile` helper to guard against shadow-mode self-detection
- [x] Add `SSHBinary string` field to `internal/config/config.go`
- [x] Call `ResolveSSHBinary` in `cmd/connect.go` after loadConfig
- [x] Call `ResolveSSHBinary` in `cmd/root.go` shadow mode path
- [x] Add `android/arm64` build target to `.goreleaser.yaml`
- [x] Add Android/Termux installation section to `README.md`
- [x] Add tests for `ResolveSSHBinary` and `sameFile` in `internal/ssh/executor_test.go`
- [x] Release v0.2.2
