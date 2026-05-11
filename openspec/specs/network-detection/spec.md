## Purpose
Detect the active network environment before every SSH connection by evaluating configured check rules in priority order, so the correct per-network SSH parameters can be selected.

## Requirements

### Requirement: Priority-ordered network evaluation
Networks SHALL be evaluated in ascending `priority` order (lowest number first). Alphabetical order breaks ties. The first network whose checks all pass SHALL be returned as the active network.

#### Scenario: Single matching network
- **WHEN** one network's checks all pass
- **THEN** that network name is returned

#### Scenario: Multiple networks, priority wins
- **WHEN** two networks both have passing checks
- **THEN** the one with the lower priority number is returned

#### Scenario: No network matches
- **WHEN** no configured network's checks pass
- **THEN** `"default"` is returned

### Requirement: AND logic within a network
All checks within a single network definition SHALL pass for the network to be considered active.

#### Scenario: Partial check failure
- **WHEN** a network has two checks and one fails
- **THEN** that network is not selected and evaluation continues to the next

### Requirement: Route check
A `route` check SHALL pass when the specified subnet or IP string appears in the kernel routing table output.

#### Scenario: Route present
- **WHEN** `match` appears in `ip route show` output
- **THEN** the check passes

#### Scenario: Route absent
- **WHEN** `match` does not appear in `ip route show` output
- **THEN** the check fails

### Requirement: Interface check
An `interface` check SHALL pass when the named interface exists and its `operstate` reads `"up"`.

#### Scenario: Interface up
- **WHEN** `/sys/class/net/<iface>/operstate` contains `"up"`
- **THEN** the check passes

#### Scenario: Interface down or missing
- **WHEN** the operstate file is absent or contains any value other than `"up"`
- **THEN** the check fails

### Requirement: Ping check
A `ping` check SHALL pass when the specified host responds to ICMP echo within the configured timeout.

#### Scenario: Host responds in time
- **WHEN** the host sends an ICMP reply before the timeout elapses
- **THEN** the check passes

#### Scenario: Host unreachable or timeout
- **WHEN** no reply arrives within the timeout
- **THEN** the check fails

#### Scenario: Default timeout
- **WHEN** no `timeout` is specified
- **THEN** a 2-second timeout is applied

### Requirement: Exec check
An `exec` check SHALL pass when the configured shell command exits with code 0.

#### Scenario: Command exits 0
- **WHEN** the command exits successfully
- **THEN** the check passes

#### Scenario: Command fails or cannot run
- **WHEN** the command exits non-zero or cannot be executed
- **THEN** the check fails
