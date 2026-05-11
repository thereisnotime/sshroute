## Purpose
Resolve the correct SSH connection parameters (host, port, user, key, jump) for a given alias and active network by merging the host's default profile with any network-specific overrides.

## Requirements

### Requirement: Default profile required
Every host entry SHALL have a `default` network profile used when no detected network matches any configured network name.

#### Scenario: No network match
- **WHEN** the active network name is `"default"` or absent from the host's profiles
- **THEN** the `default` profile's parameters are used

### Requirement: Network profile inheritance
A per-network profile SHALL only need to specify fields that differ from the `default` profile; unset fields SHALL inherit from `default`.

#### Scenario: Partial override
- **WHEN** a `corp-vpn` profile sets only `host` and `port`
- **THEN** the resolved params use `corp-vpn.host`, `corp-vpn.port`, and all remaining fields from `default`

#### Scenario: Full override
- **WHEN** a network profile specifies all fields
- **THEN** no fields are inherited from `default`

### Requirement: Unknown alias is an error
Resolving or connecting to an alias not present in the config SHALL return an error.

#### Scenario: Alias missing from config
- **WHEN** the requested alias is not a key in `cfg.Hosts`
- **THEN** an error is returned and no SSH connection is attempted

### Requirement: Tilde expansion in key paths
Identity file paths starting with `~` SHALL be expanded to the current user's home directory before being passed to SSH.

#### Scenario: Key path with tilde
- **WHEN** `key: ~/.ssh/id_ed25519` is configured
- **THEN** the resolved key path is the absolute path under the user's home directory

### Requirement: Default port
When no `port` is specified in any applicable profile, the resolved port SHALL be 22.

#### Scenario: Port not set anywhere
- **WHEN** neither the `default` nor the active network profile specifies a port
- **THEN** the resolved port is 22

### Requirement: Jump host in argv
When a resolved profile includes a non-empty `jump` value it SHALL appear as `-J <jump>` in the SSH argv.

#### Scenario: Jump configured
- **WHEN** the active profile has a non-empty `jump`
- **THEN** `-J <jump>` appears in the built SSH argv

#### Scenario: No jump
- **WHEN** the active profile has an empty `jump`
- **THEN** no `-J` flag appears in the argv
