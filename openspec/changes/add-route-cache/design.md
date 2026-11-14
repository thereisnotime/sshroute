## Context

Fallback mode already runs ssh as a child process and inspects its exit code: 255 means the connection failed, anything else means ssh reached the host (the session or remote command may still have failed). That is exactly the signal needed to learn which route works. Plain `connect` uses `syscall.Exec`, so sshroute is gone by the time ssh exits and cannot learn anything there.

## Goals / Non-Goals

- **Goal:** a host whose working route is late in the fallback order connects on the first attempt on the next run.
- **Goal:** never slower than today except for one failed attempt when the cached route has gone stale.
- **Goal:** safe under ~40 concurrent `connect --fallback` processes; never corrupt the file or lose another alias's entry; never hang a connection on the lock.
- **Goal:** a simple, versioned, documented state file other tools can read.
- **Non-Goal:** health probing, latency measurement or reordering beyond "last known good first".
- **Non-Goal:** changing plain (non-fallback) `connect`.

## Decisions

### What counts as success and failure
After a fallback walk, every attempt that exited 255 is a failure of that profile; the attempt that ended the walk with any other non-negative code is a success, even if the remote command failed (exit 1, 127, ...), because the route itself worked. A negative code (ssh killed by a signal) is neither. In reconnect mode, attempts cut short by SIGINT/SIGTERM are not recorded.

Known limitation: success is only known when ssh exits, so an interactive session records its route at logout. A long session that later drops also exits 255 and is recorded as a failure of that profile; the next run then falls back to the normal order, which is today's behaviour.

### Try-order
`ResolveOrderPreferred(cfg, alias, network, preferred)` returns `ResolveOrder` with the cached profile moved to the front and tagged `cache`. Each profile appears once. If the cached profile no longer exists in the config it is ignored. Every entry carries a source (`cache`, `detected`, `priority`, `default`) so dry-run and `resolve --fallback` can explain the order.

### When an entry is used
An entry is used only if all of these hold:
- the profile still exists for the host;
- it is younger than `ttl` (`0` disables expiry);
- the profile has no recorded failure newer than the success. In network scope only a failure on the same detected network counts, so a route that failed on a hotel network does not invalidate the same profile's entry for home.

Invalidation works through the failure record instead of deleting the entry, so `last_success` stays an accurate "last connected via X at T" for dashboards, and the next success simply overwrites it.

### Scope
Writes are identical for both scopes: each success updates `networks[<detected network>]` and `last_success`. The scope only changes the lookup: `network` reads `networks[<detected network>]`, `alias` reads `last_success`. Switching scope therefore works immediately with existing data.

### Plain connect does not use the cache
Without `--fallback`, connect resolves one profile and replaces itself with ssh. Using a cached profile there would mean a stale entry simply fails with no way to recover or to record the failure, so plain connect neither reads nor writes the cache. Users who want the cache use `--fallback` (or `SSHROUTE_FALLBACK=1` in shadow mode).

### Configuration and precedence
`cache:` in config.yaml: `enabled` (default true), `path`, `ttl` (default `7d`, Go durations plus a `d` suffix), `scope` (`network` | `alias`), `lock_timeout` (default `1s`). Invalid values fail config validation like every other field.

On/off precedence: `--no-cache` > `SSHROUTE_CACHE` (`0/false/no/off` or `1/true/yes/on`; anything else ignored) > `cache.enabled`. Path precedence: `SSHROUTE_CACHE_PATH` > `cache.path` > default. The env vars exist so wrappers and shadow mode can change behaviour without touching argv.

### Default path
1. `$XDG_STATE_HOME/sshroute/state.json` when `XDG_STATE_HOME` is set (any OS).
2. `%LOCALAPPDATA%\sshroute\state.json` on Windows.
3. `~/.local/state/sshroute/state.json` on Linux, macOS and Termux.

On macOS the config already defaults to `~/.config/sshroute/config.yaml` rather than `~/Library/Application Support`, so the state follows the same XDG layout. One path on Linux and macOS keeps external readers simple.

### State file format (version 1)
```json
{
  "version": 1,
  "hosts": {
    "<alias>": {
      "last_success": {"profile": "lan", "network": "corp-vpn", "succeeded_at": "2026-11-14T10:00:00Z"},
      "networks": {
        "<detected network>": {"profile": "lan", "network": "corp-vpn", "succeeded_at": "2026-11-14T10:00:00Z"}
      },
      "failures": {
        "<profile>": {"network": "corp-vpn", "failed_at": "2026-11-14T09:59:55Z"}
      }
    }
  }
}
```
Times are RFC 3339 UTC, second precision. `network` is the detected network name, `default` when none matched. Readers should ignore unknown fields. A future incompatible change bumps `version`; this build neither uses nor overwrites a file with a higher version.

### Concurrency
- Reads take no lock: the file is only ever replaced by rename, so a reader sees a complete old or new file.
- Writes: open `<path>.lock`, take an exclusive non-blocking lock and poll every 10ms until `lock_timeout`; on timeout the write is skipped (logged with `-v`). Under the lock: re-read the current file, apply this run's outcome, write a temp file in the same directory, rename over the state file. Re-reading under the lock is what keeps other aliases' entries from being lost.
- The lock lives on a separate file because the state file itself is replaced on every write.
- Locking: `flock(2)` on Linux, macOS, Android and the BSDs; `LockFileEx` on Windows. On any other platform the lock is a no-op: writes stay atomic, but two simultaneous writers can drop one update.
- One write per fallback walk (success and failures together), after ssh exits. The connect path never waits on the lock; at worst process exit is delayed by `lock_timeout`.
- No fsync: a file torn by a crash is just a corrupt cache, which is ignored and rewritten.

### Corruption
A missing file is an empty cache. An unreadable or unparseable file is treated as empty (logged with `-v`) and replaced on the next write. Cache problems never fail a connection: write errors are logged and dropped.

### `-v`
The root command now installs the debug slog handler for `-v` / `SSHROUTE_VERBOSE=1` (previously only shadow mode did), so `connect -v` shows cache hits, misses and writes.

## Risks / Trade-offs

- **A stale entry costs one attempt.** Bounded and self-correcting: the failure is recorded and the entry is skipped until that profile connects again.
- **Success is recorded at session end.** Fine for scripted runs; interactive sessions record at logout.
- **Long session drop looks like a connection failure.** Same as today's fallback behaviour; documented.
- **State in a shared file.** 0600 permissions, 0700 directory, contains only alias/profile/network names and timestamps.
