## MODIFIED Requirements

### Requirement: SSH options passthrough
A host profile SHALL support an `options` map of arbitrary SSH `-o Key=Value` flags that are appended to the SSH invocation.

#### Scenario: Options present in config
- **WHEN** a host profile has `options: {ConnectTimeout: "10"}`
- **THEN** `-o ConnectTimeout=10` appears in the built SSH argv

#### Scenario: No options configured
- **WHEN** a host profile has no `options` field
- **THEN** no additional `-o` flags are added to the SSH argv

#### Scenario: Options emitted in sorted key order
- **WHEN** multiple options are configured
- **THEN** they are emitted as `-o Key=Value` pairs sorted alphabetically by key, producing deterministic argv output

### Requirement: SSH options inheritance from default profile
The `options` map SHALL be inherited from the `default` profile and merged with any network-specific overrides; network-profile values override matching keys from `default`, non-overlapping keys are preserved.

#### Scenario: Network profile inherits all default options
- **WHEN** `default` has `options: {ConnectTimeout: "10"}` and the active network profile has no `options`
- **THEN** the resolved params include `ConnectTimeout=10`

#### Scenario: Network profile overrides a single option
- **WHEN** `default` has `options: {ConnectTimeout: "10", BatchMode: "yes"}` and the active network profile has `options: {ConnectTimeout: "5"}`
- **THEN** the resolved params include `ConnectTimeout=5` and `BatchMode=yes`
