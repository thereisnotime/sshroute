## Purpose
Define the behaviour of every sshroute CLI subcommand, including global flags, output formats, and tab completion of host aliases.

## Requirements

### Requirement: connect — exec SSH with resolved params
`sshroute connect <alias> [ssh-args...]` SHALL detect the active network, resolve SSH params for the alias, and exec-replace itself with the real SSH binary. Extra arguments after the alias are appended to the SSH invocation unchanged.

#### Scenario: Successful connection
- **WHEN** the alias exists and a network is detected
- **THEN** the process is replaced by the real SSH binary with resolved params

#### Scenario: --dry-run
- **WHEN** `--dry-run` is passed
- **THEN** the resolved SSH command is printed to stdout and no connection is made

#### Scenario: --fallback
- **WHEN** `--fallback` is passed
- **THEN** profiles are tried in priority order; only SSH connection failures (exit 255) trigger a retry; auth or remote command failures stop immediately

### Requirement: list — show all configured hosts
`sshroute list` SHALL print all configured aliases with their resolved SSH params for the active network. Supports `--output table|json|yaml`.

#### Scenario: Default table output
- **WHEN** no `--output` flag is given
- **THEN** a formatted table is printed with columns: ALIAS, NETWORK, HOST, PORT, USER, KEY, JUMP, COMMENT, TAGS

### Requirement: add — create or update a host profile
`sshroute add <alias>` SHALL write SSH params for the alias under the specified `--network` (default: `default`). Omitted flags retain their current values if the alias already exists.

#### Scenario: New alias
- **WHEN** the alias does not exist in the config
- **THEN** a new entry is created with the provided flags

#### Scenario: Partial update of existing alias
- **WHEN** the alias exists and only `--port` is passed
- **THEN** only the port is updated; all other fields remain unchanged

### Requirement: remove — delete a host entry
`sshroute remove <alias>` SHALL remove all profiles for the alias from the config. Fails if the alias does not exist.

#### Scenario: Alias exists
- **WHEN** the alias is present in the config
- **THEN** the entry is deleted and the config is saved

#### Scenario: Alias not found
- **WHEN** the alias is absent from the config
- **THEN** an error is returned and the config is not modified

### Requirement: network — report active network
`sshroute network` SHALL print the name of the currently detected network, or `"default"` if none match.

#### Scenario: Network detected
- **WHEN** a configured network's checks all pass
- **THEN** that network name is printed

#### Scenario: No match
- **WHEN** no network's checks pass
- **THEN** `"default"` is printed

### Requirement: network list — enumerate configured networks
`sshroute network list` SHALL list all configured networks with their priority, check rules, and current active state. Supports `--output table|json|yaml`.

#### Scenario: List output
- **WHEN** the command is run
- **THEN** every configured network is shown with its priority and whether it is currently active

### Requirement: network test — debug a network's checks
`sshroute network test <name>` SHALL run every check for the named network and print individual pass/fail results.

#### Scenario: Check results shown
- **WHEN** a valid network name is given
- **THEN** each check is evaluated and its result printed

### Requirement: resolve — show resolved SSH parameters
`sshroute resolve <alias> [--network <name>]` SHALL print the fully resolved SSH parameters for the alias. Includes the exact `ssh` command that would be run. Supports `--output table|json|yaml`.

#### Scenario: Auto-detected network
- **WHEN** `--network` is omitted
- **THEN** the active network is detected and used for resolution

#### Scenario: Forced network override
- **WHEN** `--network corp-vpn` is passed
- **THEN** the corp-vpn profile is resolved regardless of actual network state

#### Scenario: Unknown alias
- **WHEN** the alias is not in the config
- **THEN** an error is returned

### Requirement: copy — scp wrapper with resolved params
`sshroute copy <alias> <src> <dst>` SHALL resolve SSH params identically to `connect` and invoke `scp` with the correct port, key, and jump host. `<alias>:<path>` in src or dst SHALL be rewritten to `user@host:<path>`.

#### Scenario: Remote source rewrite
- **WHEN** src is `myserver:/remote/path`
- **THEN** scp receives `user@host:/remote/path` as the source argument

#### Scenario: Remote destination rewrite
- **WHEN** dst is `myserver:/remote/path`
- **THEN** scp receives `user@host:/remote/path` as the destination argument

#### Scenario: --dry-run
- **WHEN** `--dry-run` is passed
- **THEN** the resolved scp command is printed and scp is not executed

#### Scenario: SSHROUTE_SCP override
- **WHEN** `SSHROUTE_SCP=/path/to/scp` is set
- **THEN** that binary is used instead of the system scp

### Requirement: Dynamic alias tab completion
`connect`, `remove`, `resolve`, and `copy` SHALL complete host aliases from the live config when Tab is pressed. Aliases SHALL be returned sorted alphabetically. No alias completion occurs after the first positional argument.

#### Scenario: First argument completion
- **WHEN** Tab is pressed as the first argument
- **THEN** all keys from `cfg.Hosts` are returned, sorted

#### Scenario: Subsequent argument
- **WHEN** Tab is pressed after the alias is already provided
- **THEN** no aliases are suggested

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
- **WHEN** `-v` or `SSHROUTE_VERBOSE=1` is set
- **THEN** debug-level log lines are written to stderr
