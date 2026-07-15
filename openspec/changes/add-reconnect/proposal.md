## Why

When a connection drops — laptop sleep, WiFi handoff, moving between networks — `sshroute connect` exits and the user has to re-run it by hand. Users work around this with `autossh`, but `autossh` re-execs a *fixed* ssh command: it is pinned to the IP it first resolved, so it cannot follow a network change (e.g. sleep on the LAN, wake on a hotspot leaves it retrying an unreachable LAN address). sshroute already knows how to pick the right route for the active network, so it is the natural place to supervise reconnection.

## What Changes

- Add a `--reconnect` flag to `sshroute connect`. When set, sshroute supervises the ssh subprocess and, when ssh exits because the connection dropped (exit code 255), it **re-detects the active network, re-resolves the route, and reconnects**. A clean exit (code 0) or any non-connection failure (auth error, remote-command exit) stops the loop and is propagated.
- Add a `--reconnect-delay` flag (default `2s`) controlling the wait between attempts.
- Reconnect composes with `--fallback`: each cycle re-runs the fallback try-order. Even without `--fallback`, each cycle re-detects the network, so route selection follows network changes across reconnects — the capability `autossh` lacks.
- Reconnect mode runs ssh as a subprocess (like `--fallback` already does) rather than `syscall.Exec`, so sshroute stays resident to supervise. `SIGINT`/`SIGTERM` tear down the child and stop the loop.

## Capabilities

### New Capabilities

None — this extends the existing `connect` command.

### Modified Capabilities

- `commands`: `connect` gains `--reconnect` / `--reconnect-delay` and supervised-reconnection behaviour.

## Impact

- `cmd/connect.go` — register `--reconnect` / `--reconnect-delay`; supervised loop wiring (signal context, network re-detection per attempt)
- `internal/ssh/reconnect.go` (new) — `Supervise` loop and `RunContext` (context-cancellable subprocess)
- `internal/ssh/executor.go` — unchanged public behaviour; `RunContext` lives alongside `Run`
- Fully additive — without `--reconnect`, `connect` behaves exactly as before (`syscall.Exec` for the direct case, subprocess retry for `--fallback`).
