## ADDED Requirements

### Requirement: cache — inspect and clear the route cache
`sshroute cache show [alias]` SHALL list cached routes (per alias: the alias-scope entry with key `*`, then one row per network) with profile, network, last success time, age and status (`fresh`, `expired`, `failed`, `unknown-profile`), and support `--output table|json|yaml`. `sshroute cache clear [alias]` SHALL remove the entries for one alias, or all entries when no alias is given. `sshroute cache path` SHALL print the effective state file path.

#### Scenario: Show as JSON
- **WHEN** `sshroute cache show srv -o json` is run
- **THEN** a JSON array of rows for `srv` is printed, each with `alias`, `key`, `profile`, `network`, `succeeded_at` and `status`

#### Scenario: Clear one alias
- **WHEN** `sshroute cache clear srv` is run
- **THEN** entries for `srv` are removed and entries for other aliases are kept

#### Scenario: Empty cache
- **WHEN** `sshroute cache show` is run and nothing is cached
- **THEN** `no cached routes` is printed

### Requirement: resolve --fallback shows the try-order
`sshroute resolve <alias> --fallback` SHALL print every profile in the order `connect --fallback` would try them, with the source of each (`cache`, `detected`, `priority`, `default`), the cached success time for a cached entry, and the resolved parameters. `--no-cache` SHALL show the order without the cache.

#### Scenario: Cached route shown first
- **WHEN** the cache holds a fresh entry for the alias
- **THEN** the first row is that profile with source `cache` and its `cached_at` time

## MODIFIED Requirements

### Requirement: connect — exec SSH with resolved params
`sshroute connect <alias> [ssh-args...]` SHALL detect the active network, resolve SSH params for the alias, and exec-replace itself with the real SSH binary. Extra arguments after the alias are appended to the SSH invocation unchanged. When `--fallback` or `--reconnect` is set, sshroute instead runs ssh as a supervised subprocess so it can inspect the exit code. With `--fallback`, the route cache decides which profile is tried first unless `--no-cache` is passed.

#### Scenario: Successful connection
- **WHEN** the alias exists and a network is detected
- **THEN** the process is replaced by the real SSH binary with resolved params

#### Scenario: --dry-run
- **WHEN** `--dry-run` is passed
- **THEN** the resolved SSH command is printed to stdout and no connection is made

#### Scenario: --fallback
- **WHEN** `--fallback` is passed
- **THEN** profiles are tried in order, cached route first; only SSH connection failures (exit 255) trigger a retry; auth or remote command failures stop immediately

#### Scenario: --fallback --dry-run
- **WHEN** `--fallback --dry-run` is passed
- **THEN** a `[dry-run] cache:` line with the cache decision and a `[dry-run] order:` line listing each profile and its source are printed before the commands

#### Scenario: --no-cache
- **WHEN** `--fallback --no-cache` is passed
- **THEN** the normal fallback order is used and the cache is not updated

### Requirement: Global flags
The following flags SHALL apply to every command:

#### Scenario: --config flag
- **WHEN** `--config /path/to/config.yaml` is passed
- **THEN** that file is used instead of the default `~/.config/sshroute/config.yaml`

#### Scenario: --output flag
- **WHEN** `-o json` or `-o yaml` is passed to a list or resolve command
- **THEN** output is rendered in the requested format

#### Scenario: --dry-run flag
- **WHEN** `--dry-run` is passed to connect or copy
- **THEN** the resolved command is printed and nothing is executed

#### Scenario: --verbose flag
- **WHEN** `-v` or `SSHROUTE_VERBOSE=1` is set on any command
- **THEN** debug-level log lines, including route cache hits, misses and writes, are written to stderr
