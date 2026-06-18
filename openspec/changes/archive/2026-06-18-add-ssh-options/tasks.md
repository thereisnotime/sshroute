## 1. Config Schema

- [x] 1.1 Add `Options map[string]string \`yaml:"options,omitempty"\`` to `SSHParams` in `internal/config/config.go`

## 2. Resolution

- [x] 2.1 Merge `Options` in `resolveRecursive` (`internal/ssh/router.go`): copy default keys first, then apply network-profile overrides

## 3. Argv Construction

- [x] 3.1 Emit `-o Key=Value` pairs in `BuildArgv` (`internal/ssh/executor.go`), sorted by key for determinism

## 4. Tests

- [x] 4.1 Executor tests: options present in argv, empty options produce no extra `-o` flags, deterministic ordering
- [x] 4.2 Router tests: options inherited from default, network profile overrides individual keys, nil when absent

## 5. Validation

- [x] 5.1 `just test` passes with race detector
- [x] 5.2 `go vet ./...` clean
- [x] 5.3 `gofmt -s -l .` clean
