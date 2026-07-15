## Context

`connect` has two existing execution modes:

- **direct** (`ssh.Exec` → `syscall.Exec`): replaces the process image so sshroute is invisible in the process tree. This is a stated project rule.
- **fallback** (`ssh.Run` → `exec.Command` subprocess): keeps sshroute resident so it can inspect the exit code and try the next profile.

Reconnection requires observing ssh's exit code and looping, so it cannot use `syscall.Exec`. It therefore builds on the fallback mode's subprocess approach.

## Goals / Non-Goals

- **Goal:** survive connection drops (sleep, network change) without the user re-running the command, selecting the correct route for wherever they are *now*.
- **Non-Goal:** seamless, no-blip roaming or predictive echo — that is mosh's domain and requires a stateful UDP protocol. Reconnect is a reconnect: a new ssh session each time, with a brief interruption. Session persistence across the blip is the user's multiplexer's job (tmux/zellij), unchanged.

## Decisions

### Deviation from the `syscall.Exec` rule
Reconnect mode uses a subprocess, so sshroute stays in the process tree for the duration of the session. This is an intentional, opt-in deviation, identical in kind to the existing `--fallback` mode. The default path (no `--reconnect`) still uses `syscall.Exec`.

### Reconnect trigger = exit code 255
ssh returns 255 only on connection-level failure (timeout, refused, unreachable, dropped link). A clean logout returns 0; a remote command returns its own code; auth failure is also surfaced by ssh distinctly from a mid-session drop. Keying reconnection on `== SSHConnectFailure` (255) means:
- a real drop reconnects,
- `exit` / detach (0) stops the loop — a strict improvement over `autossh`, which reconnects even on deliberate exit,
- an auth or remote-command failure stops immediately, so a misconfigured route does not spin forever.

An initial connection that fails with 255 is also retried — desirable, since "not up yet" and "dropped" are the same from the user's side; the delay plus the network re-detection each cycle bounds the spin.

### Route re-detection every attempt
Each attempt calls `network.Detect` and re-resolves, so the route follows the active network. This is the core advantage over `autossh` (fixed argv). `--fallback` composes: with it, each cycle walks the priority order; without it, each cycle resolves the single best route for the current network.

### Signal handling via `signal.NotifyContext` + `RunContext`
The loop derives a context cancelled on `SIGINT`/`SIGTERM`. The ssh child is run with `exec.CommandContext`, so a signal tears the child down; the loop then observes cancellation and returns. During an active session ssh holds the terminal in raw mode, so Ctrl-C is delivered to the *remote* end, not as a local signal — local signals only reach sshroute between attempts (during the delay) or via an explicit `SIGTERM` (e.g. `pkill`), where stopping is the desired outcome.

### Backoff
A fixed `--reconnect-delay` (default 2s) between attempts. Simple and predictable; each attempt also incurs network-detection latency, so the effective retry rate when offline is naturally throttled. Exponential/capped backoff is a possible future enhancement and is out of scope here.

## Testability

`Supervise(attempt AttemptFunc, cfg ReconnectConfig, stop <-chan struct{})` takes an injected attempt function and a `MaxTries` bound, so the loop logic (reconnect-on-255, stop-on-other, honour `stop`, honour `MaxTries`) is unit-tested with a fake attempt returning scripted exit codes — no real ssh or sleeps. `RunContext` is thin exec plumbing exercised via the existing subprocess test pattern.

## Risks / Trade-offs

- **Resident process:** sshroute is visible in the tree during a reconnect session. Accepted and scoped to opt-in, mirroring `--fallback`.
- **Fresh shell on reconnect:** each reconnect is a new login shell; users relying on in-session state must use a multiplexer. Documented; not a regression.
