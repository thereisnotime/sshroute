## 1. Supervision core

- [x] 1.1 Add `internal/ssh/reconnect.go` with `RunContext(ctx, argv) (int, error)` (context-cancellable subprocess, mirrors `Run`)
- [x] 1.2 Add `ReconnectConfig{Delay, MaxTries}`, `AttemptFunc`, and `Supervise(attempt, cfg, stop)` — loop reconnecting only on `SSHConnectFailure` (255), stop on any other code, honour `stop` during the delay and `MaxTries`

## 2. Command wiring

- [x] 2.1 Register `--reconnect` (bool) and `--reconnect-delay` (duration, default 2s) on `connect` in `cmd/connect.go:init()`
- [x] 2.2 In `runConnect`, when `--reconnect` is set, build a `signal.NotifyContext` (SIGINT/SIGTERM) and drive `ssh.Supervise` with an attempt that re-detects the network, re-resolves (honouring `--fallback`), and runs via `RunContext`
- [x] 2.3 Extract the single-vs-fallback attempt into a helper shared by the reconnect path; keep the existing non-reconnect direct (`syscall.Exec`) and fallback paths behaving identically

## 3. Tests

- [x] 3.1 `internal/ssh` unit tests for `Supervise`: reconnect on 255 then stop on 0, stop immediately on a non-255 code, honour `MaxTries`, honour `stop` channel
- [x] 3.2 `cmd` tests: `--reconnect` flag registered and parsed; `--reconnect --dry-run` prints the resolved command without looping

## 4. Docs

- [x] 4.1 Document `--reconnect` / `--reconnect-delay` in README (connect section)

## 5. Validation

- [x] 5.1 `just test` passes with race detector
- [x] 5.2 `go vet ./...` clean
- [x] 5.3 `gofmt -s -l .` clean
