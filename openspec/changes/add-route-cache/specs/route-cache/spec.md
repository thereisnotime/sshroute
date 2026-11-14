## ADDED Requirements

### Requirement: Record fallback outcomes
When the route cache is enabled and a fallback walk finishes (connect `--fallback`, connect `--reconnect --fallback`, or shadow mode with `SSHROUTE_FALLBACK=1`), sshroute SHALL record, for the alias: the profile that connected (ssh exited with a non-negative code other than 255) with the detected network and the time, and the time and detected network of each profile that failed to connect (exit 255). Dry runs SHALL NOT write.

#### Scenario: Late route connects
- **WHEN** the first two profiles exit 255 and the third exits 0
- **THEN** the third profile is recorded as the last success for the alias and detected network, and the first two are recorded as failures

#### Scenario: Remote command fails after connecting
- **WHEN** a profile's ssh exits with code 1
- **THEN** that profile is recorded as a success, because the connection itself worked

#### Scenario: Interrupted attempt
- **WHEN** ssh is killed by a signal or the reconnect loop is interrupted by SIGINT/SIGTERM
- **THEN** that attempt is not recorded as a success

### Requirement: Cached route is tried first
When a usable cache entry exists, the fallback order SHALL start with the cached profile followed by the normal fallback order without that profile. Each profile SHALL be tried at most once per walk.

#### Scenario: Cache hit
- **WHEN** the cache says profile `lan` connected last time and the normal order is `vpn, office, lan, default`
- **THEN** the walk tries `lan, vpn, office, default`

#### Scenario: Stale cache entry
- **WHEN** the cached profile exits 255
- **THEN** sshroute records the failure and continues with the rest of the normal order, and the next walk does not try that profile first unless it has connected since

#### Scenario: Cached profile removed from config
- **WHEN** the cached profile no longer exists for the host
- **THEN** the entry is ignored and the normal order is used

### Requirement: Entry freshness
A cache entry SHALL be used only if it is younger than the configured `ttl` and its profile has no recorded connection failure newer than the success. A `ttl` of `0` SHALL disable expiry. In network scope only failures recorded on the same detected network SHALL invalidate an entry.

#### Scenario: Expired entry
- **WHEN** the entry is older than `ttl`
- **THEN** it is ignored and the normal order is used

#### Scenario: Failure on another network
- **WHEN** scope is `network` and the cached profile failed on a different network after the cached success
- **THEN** the entry for the current network is still used

### Requirement: Cache scope
With `scope: network` (default) entries SHALL be keyed by alias and detected network. With `scope: alias` lookups SHALL use the most recent success for the alias on any network.

#### Scenario: Network scope on a new network
- **WHEN** scope is `network` and the alias has only been cached on network `office`, and the detected network is `home`
- **THEN** there is no cache hit

#### Scenario: Alias scope on a new network
- **WHEN** scope is `alias` and the alias last connected via `lan` on network `office`, and the detected network is `home`
- **THEN** `lan` is tried first

### Requirement: Cache configuration and overrides
The cache SHALL be configurable under `cache:` with `enabled` (default true), `path`, `ttl` (default `7d`), `scope` (default `network`) and `lock_timeout` (default `1s`). `--no-cache` SHALL disable it for one invocation; `SSHROUTE_CACHE` set to `0/false/no/off` or `1/true/yes/on` SHALL override `enabled`; `SSHROUTE_CACHE_PATH` SHALL override `path`. Precedence SHALL be flag, then env, then config. Invalid `ttl`, `scope` or `lock_timeout` values SHALL fail config validation.

#### Scenario: Disabled by env
- **WHEN** `SSHROUTE_CACHE=0` is set
- **THEN** the cache is neither read nor written

#### Scenario: Flag beats env
- **WHEN** `SSHROUTE_CACHE=1` is set and `--no-cache` is passed
- **THEN** the cache is neither read nor written

#### Scenario: Invalid scope
- **WHEN** `cache.scope` is `host`
- **THEN** loading the config fails with an error naming `cache.scope`

### Requirement: Default state file location
Without `path` or `SSHROUTE_CACHE_PATH`, the state file SHALL be `$XDG_STATE_HOME/sshroute/state.json` when `XDG_STATE_HOME` is set, `%LOCALAPPDATA%\sshroute\state.json` on Windows, and `~/.local/state/sshroute/state.json` otherwise (Linux, macOS, Termux).

#### Scenario: No XDG_STATE_HOME on Linux or macOS
- **WHEN** `XDG_STATE_HOME` is unset
- **THEN** the state file is `~/.local/state/sshroute/state.json`

### Requirement: Versioned state file format
The state file SHALL be JSON with a top-level `"version": 1` and a `hosts` object keyed by alias, each with optional `last_success` (`profile`, `network`, `succeeded_at`), `networks` (detected network to the same route object) and `failures` (profile to `network`, `failed_at`). Times SHALL be RFC 3339 in UTC. A file with a higher version SHALL be neither used nor overwritten.

#### Scenario: External reader
- **WHEN** another tool reads `hosts.<alias>.last_success`
- **THEN** it gets the profile, detected network and time of the last successful fallback connection

#### Scenario: File from a newer sshroute
- **WHEN** the state file has `"version": 2`
- **THEN** sshroute ignores it for lookups and leaves it unchanged

### Requirement: Safe concurrent writes
Writes SHALL take an exclusive lock on a sibling `.lock` file, re-read the current state, apply the change and atomically replace the state file via a temp file and rename. If the lock is not acquired within `lock_timeout` the write SHALL be skipped. Reads SHALL NOT take the lock.

#### Scenario: Parallel connects to different aliases
- **WHEN** many sshroute processes record outcomes for different aliases at the same time
- **THEN** every alias's entry is present afterwards and the file is valid JSON

#### Scenario: Lock held too long
- **WHEN** the lock is held by another process for longer than `lock_timeout`
- **THEN** the write is skipped and sshroute exits normally

### Requirement: Corrupt state is never fatal
A missing, unreadable or unparseable state file SHALL be treated as an empty cache and replaced on the next write. Cache read or write errors SHALL NOT change the outcome of a connection.

#### Scenario: Garbage in the state file
- **WHEN** the state file contains invalid JSON
- **THEN** the fallback walk uses the normal order, succeeds as usual, and the file is rewritten with valid content

### Requirement: Plain connect ignores the cache
`connect` without `--fallback` or `--reconnect --fallback` SHALL neither read nor write the route cache.

#### Scenario: Plain connect
- **WHEN** `sshroute connect srv` runs without `--fallback`
- **THEN** the profile for the detected network is exec'd regardless of any cache entry
