## Why

`connect --fallback` walks profiles in a fixed order: the detected network's profile, then the rest by network priority, then `default`. Each dead route costs a full `ConnectTimeout` (typically 5s). For a host whose working route is third or fourth in that order, every connect burns 10-15s on attempts that are known to fail, and a tool that runs ~40 `sshroute connect --fallback` processes in parallel pays that on every run. sshroute already learns which route works the moment ssh exits; it just forgets it.

## What Changes

- Add a route cache, on by default. After a fallback walk, sshroute records which profile connected (ssh exit other than 255) for the alias, the detected network at the time and a timestamp, plus the last connection failure time per profile.
- The next fallback walk for that alias tries the cached profile first, then the normal `ResolveOrder` minus that profile. A stale entry costs exactly one failed attempt; the failure is recorded so the entry is not used again until that profile connects.
- Entries are keyed by alias + detected network by default (`scope: network`); `scope: alias` keys by alias only.
- New `cache:` config section: `enabled` (true), `path` (XDG state dir), `ttl` (7d), `scope` (network), `lock_timeout` (1s).
- `--no-cache` on `connect` and `resolve`; `SSHROUTE_CACHE=0|1` and `SSHROUTE_CACHE_PATH` env overrides for wrappers and shadow mode.
- New `sshroute cache show [alias]`, `cache clear [alias]`, `cache path`.
- `resolve --fallback` prints the full try-order with the source of each entry (cache, detected, priority, default); `connect --fallback --dry-run` prints the cache decision and order; `-v` logs cache hits, misses and writes.
- `-v` / `SSHROUTE_VERBOSE=1` now enables debug logging for every command (it was only wired up in shadow mode).
- The state file is a small versioned JSON document (`"version": 1`) with a documented format so other tools can read "last connected via <profile> at <time>".

## Capabilities

### New Capabilities

- `route-cache`: recording fallback outcomes, cache-aware try-order, state file format, locking and corruption handling, configuration.

### Modified Capabilities

- `commands`: `connect --no-cache`, `resolve --fallback` / `--no-cache`, new `cache` command, `-v` for all commands.
- `shadow-mode`: `SSHROUTE_FALLBACK=1` uses and updates the route cache.

## Impact

- `internal/routecache/` (new): state model, lookup, store with file locking (flock on Linux/macOS/Android/BSD, LockFileEx on Windows) and atomic writes.
- `internal/ssh/fallback.go` (new): `ResolveOrderPreferred` and a shared `RunFallback` used by connect, reconnect and shadow mode.
- `internal/config`: `CacheConfig`, `ParseDuration` (accepts `d`), validation.
- `cmd/cache.go` (new), `cmd/connect.go`, `cmd/resolve.go`, `cmd/root.go`.
- `golang.org/x/sys` moves from an indirect to a direct dependency (Windows locking). No new modules.
- Plain `connect` without `--fallback` is unchanged: it still `syscall.Exec`s and neither reads nor writes the cache.
