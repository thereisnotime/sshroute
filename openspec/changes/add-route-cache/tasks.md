## 1. Config

- [x] 1.1 Add `CacheConfig` (`enabled`, `path`, `ttl`, `scope`, `lock_timeout`) to `internal/config/config.go`
- [x] 1.2 Add `ParseDuration` with a `d` suffix in `internal/config/duration.go`
- [x] 1.3 Validate `cache.ttl`, `cache.lock_timeout` and `cache.scope` in `internal/config/validator.go`

## 2. Route cache package

- [x] 2.1 `internal/routecache/state.go`: versioned `State`, `HostState`, `Route`, `Failure`, `Apply`, `Clear`, decode/encode with corrupt and newer-version handling
- [x] 2.2 `internal/routecache/lookup.go`: `Lookup` and `RouteStatus` (TTL, failure-after-success, missing profile, scope)
- [x] 2.3 `internal/routecache/store.go`: `ResolveOptions` (flag > env > config), `DefaultPath`, lock-free `Load`, locked `Update`/`Record` with timeout and atomic rename
- [x] 2.4 File locks: `lock_flock.go` (Linux, macOS, Android, BSD), `lock_windows.go` (LockFileEx), `lock_other.go` (no-op)

## 3. Fallback order

- [x] 3.1 `internal/ssh/fallback.go`: `ResolveOrderPreferred` with per-entry source, shared `RunFallback` returning attempts

## 4. Command wiring

- [x] 4.1 `cmd/cache.go`: `cacheOptions`, `planFallback`, `recordFallback`, dry-run plan output
- [x] 4.2 `cmd/connect.go`: use the plan and record outcomes in `--fallback`, `--reconnect --fallback` and their dry runs; add `--no-cache`
- [x] 4.3 `cmd/root.go`: shadow mode `SSHROUTE_FALLBACK=1` uses the plan and records; `-v` enables debug logging for all commands
- [x] 4.4 `cmd/resolve.go`: `--fallback` order table/JSON with sources, `--no-cache`
- [x] 4.5 `cmd/cache.go`: `cache show [alias]`, `cache clear [alias]`, `cache path`

## 5. Tests

- [x] 5.1 `internal/routecache`: apply/clear, decode (corrupt, missing version, newer version), lookup matrix (fresh, expired, ttl 0, failures per scope, removed profile), options precedence, default path, file format, corrupt rewrite, newer version untouched, lock timeout, 50 concurrent goroutines, 8 concurrent processes
- [x] 5.2 `internal/ssh`: `ResolveOrderPreferred` (cached first, dedupe, missing, cached default/detected), `RunFallback` (hit, stale, remote failure, exhausted, unresolvable, runner error)
- [x] 5.3 `internal/config`: `ParseDuration`, cache validation, YAML round trip
- [x] 5.4 `cmd`: record then reuse, stale fall-through, exhausted, `--no-cache`, `SSHROUTE_CACHE=0`, `enabled: false`, corrupt state, dry-run output, reconnect attempt records, interrupted attempt not recorded, `cache show/clear/path`, `resolve --fallback`

## 6. Docs

- [x] 6.1 README: `cache:` config, `--no-cache`, env vars, `cache` command, `resolve --fallback`
- [x] 6.2 `docs/route-cache.md`: behaviour, paths per OS, state file format for external readers
- [x] 6.3 Example config with a `cache:` section

## 7. Validation

- [x] 7.1 `just test` (race, shuffle) passes
- [x] 7.2 `go vet ./...` and `gofmt -s -l .` clean
- [x] 7.3 gosec clean; cross-builds for linux, darwin, windows, android
