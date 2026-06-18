## Why

There is no way to pass arbitrary SSH `-o Key=Value` flags through sshroute config, forcing users who need options like `ConnectTimeout` or `ServerAliveInterval` to work around sshroute with shell aliases or raw `ssh`, breaking the single-source-of-truth model.

## What Changes

- Add an `options` map (`map[string]string`) field to `SSHParams` in the config schema.
- Network profile resolution merges `options` from `default` first, then applies network-profile overrides key-by-key (network values win on conflict).
- `BuildArgv` emits one `-o Key=Value` argument pair per entry, sorted by key for deterministic argv output.

## Capabilities

### New Capabilities

None — this extends an existing capability.

### Modified Capabilities

- `host-resolution`: New `options` field on host profiles; inheritance and argv-emission rules for options.

## Impact

- `internal/config/config.go` — new `Options` field on `SSHParams`
- `internal/ssh/executor.go` — emit `-o Key=Value` in `BuildArgv`
- `internal/ssh/router.go` — merge `Options` during network profile resolution
- Fully additive — existing configs with no `options` field behave identically
