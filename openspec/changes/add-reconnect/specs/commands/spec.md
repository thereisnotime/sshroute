## ADDED Requirements

### Requirement: connect --reconnect supervises the connection
`sshroute connect <alias> --reconnect` SHALL run ssh as a supervised subprocess and, when ssh exits with a connection failure (exit code 255), re-detect the active network, re-resolve the route, and reconnect. Any other exit code SHALL stop supervision and be propagated. A `--reconnect-delay` duration (default `2s`) SHALL control the wait between attempts. `--reconnect` SHALL compose with `--fallback`.

#### Scenario: Reconnect after a dropped connection
- **WHEN** `--reconnect` is set and ssh exits with code 255
- **THEN** the active network is re-detected, the route is re-resolved, and ssh is launched again after the reconnect delay

#### Scenario: Clean exit stops supervision
- **WHEN** `--reconnect` is set and ssh exits with code 0
- **THEN** supervision stops and sshroute exits 0 without reconnecting

#### Scenario: Non-connection failure stops supervision
- **WHEN** `--reconnect` is set and ssh exits with a non-zero, non-255 code (auth failure or remote-command exit)
- **THEN** supervision stops immediately and that exit code is propagated

#### Scenario: Route follows the active network across reconnects
- **WHEN** `--reconnect` is set and the active network changes between attempts
- **THEN** the next attempt resolves the route for the newly detected network, not the previously used one

#### Scenario: Interrupt during the reconnect delay
- **WHEN** `--reconnect` is set and SIGINT or SIGTERM is received while waiting between attempts
- **THEN** the ssh child is torn down and supervision stops

## MODIFIED Requirements

### Requirement: connect — exec SSH with resolved params
`sshroute connect <alias> [ssh-args...]` SHALL detect the active network, resolve SSH params for the alias, and exec-replace itself with the real SSH binary. Extra arguments after the alias are appended to the SSH invocation unchanged. When `--fallback` or `--reconnect` is set, sshroute instead runs ssh as a supervised subprocess so it can inspect the exit code.

#### Scenario: Successful connection
- **WHEN** the alias exists and a network is detected
- **THEN** the process is replaced by the real SSH binary with resolved params

#### Scenario: --dry-run
- **WHEN** `--dry-run` is passed
- **THEN** the resolved SSH command is printed to stdout and no connection is made

#### Scenario: --fallback
- **WHEN** `--fallback` is passed
- **THEN** profiles are tried in priority order; only SSH connection failures (exit 255) trigger a retry; auth or remote command failures stop immediately
