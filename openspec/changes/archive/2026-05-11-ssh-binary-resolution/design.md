# Design: SSH binary resolution

## Key decisions

### const → var for RealSSH

`RealSSH` changed from a package-level constant to a variable so it can be set at process startup after the config is loaded. All call sites remain unchanged — they reference `ssh.RealSSH`.

### Resolver function

`ResolveSSHBinary(cfg *config.Config) string` is called once at startup in `cmd/connect.go` and `cmd/root.go` (shadow mode) and its result assigned to `ssh.RealSSH`.

```
SSHROUTE_SSH env var
  → cfg.SSHBinary (yaml: ssh_binary)
    → exec.LookPath("ssh") — skip if same inode as os.Args[0]
      → "/usr/bin/ssh"
```

### Shadow-mode self-detection

`LookPath("ssh")` would return sshroute itself when running in shadow mode, causing infinite recursion. Guard with `os.SameFile` inode comparison:

```go
func sameFile(a, b string) bool {
    ai, _ := os.Stat(a)
    bi, _ := os.Stat(b)
    return os.SameFile(ai, bi)
}
```

If `LookPath` returns the same inode as `os.Args[0]`, skip it and fall through to the hardcoded path.

### Config field

```go
type Config struct {
    Networks  map[string]NetworkDefinition `yaml:"networks"`
    Hosts     map[string]HostConfig        `yaml:"hosts"`
    SSHBinary string                       `yaml:"ssh_binary,omitempty"`
}
```

### GoReleaser android/arm64

Added `android` to `goos` and an ignore matrix entry for `android/amd64`. Android uses `arm64` only. The official Go toolchain does not publish android/arm64 prebuilts, so the go.mod toolchain directive must not exceed what's available as a prebuilt for the linux/arm64 cross-compilation host.

## Files changed

- `internal/ssh/executor.go` — const→var, `ResolveSSHBinary`, `sameFile`
- `internal/config/config.go` — `SSHBinary` field
- `cmd/connect.go` — call `ResolveSSHBinary` after loadConfig
- `cmd/root.go` — call `ResolveSSHBinary` in shadow mode
- `.goreleaser.yaml` — android/arm64 build target
- `README.md` — Android/Termux installation section
