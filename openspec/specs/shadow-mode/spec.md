## Purpose
Allow sshroute to transparently intercept all SSH calls system-wide when installed as `ssh` earlier in `$PATH`, routing configured aliases through the resolver and passing unknown hosts straight to the real SSH binary.

## Requirements

### Requirement: Shadow mode activation
Shadow mode SHALL activate when `filepath.Base(os.Args[0])` equals `"ssh"`.

#### Scenario: Called via ssh symlink
- **WHEN** the binary is invoked through a symlink named `ssh`
- **THEN** shadow mode activates and the first positional SSH argument is treated as the target

### Requirement: Known alias interception
When the target matches a configured alias, sshroute SHALL resolve SSH params and exec-replace itself with the real SSH binary.

#### Scenario: Target is a configured alias
- **WHEN** the extracted target matches a key in `cfg.Hosts`
- **THEN** the real SSH binary is exec'd with fully resolved parameters

### Requirement: Unknown host passthrough
When the target does not match any configured alias, the original argv SHALL be passed unchanged to the real SSH binary.

#### Scenario: Target not in config
- **WHEN** the target is absent from `cfg.Hosts`
- **THEN** the real SSH binary is exec'd with the original unmodified argv

### Requirement: Config failure passthrough
When the config file cannot be loaded, shadow mode SHALL fall through to passthrough rather than returning an error, so normal SSH usage is never broken.

#### Scenario: Config missing or malformed
- **WHEN** the config file does not exist or cannot be parsed
- **THEN** the original argv is passed through to the real SSH binary

### Requirement: Recursion prevention
The SSH binary resolution step SHALL skip any binary with the same inode as `os.Args[0]`, preventing shadow mode from exec'ing itself.

#### Scenario: sshroute is first ssh in PATH
- **WHEN** `~/.local/bin/ssh` is sshroute and is first in PATH
- **THEN** the real SSH binary from further in PATH (or the hardcoded fallback) is used
