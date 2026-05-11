## Purpose
Resolve the path to the real SSH binary at process startup through a prioritized chain of sources, so sshroute works on non-standard systems (Termux, Nix, Homebrew) and never recurses into itself in shadow mode.

## Requirements

### Requirement: Environment variable override
If `SSHROUTE_SSH` is set to a non-empty string it SHALL be used as the SSH binary path with no further resolution.

#### Scenario: Env var set
- **WHEN** `SSHROUTE_SSH=/custom/ssh` is exported
- **THEN** `/custom/ssh` is used regardless of config or PATH

### Requirement: Config field override
If `ssh_binary` is set in the config file and `SSHROUTE_SSH` is not set, the config value SHALL be used.

#### Scenario: Config field set, env var absent
- **WHEN** `ssh_binary: /data/data/com.termux/files/usr/bin/ssh` is configured
- **THEN** that path is used as the SSH binary

### Requirement: PATH auto-detection
If neither env var nor config field is set, `exec.LookPath("ssh")` SHALL be consulted to find the SSH binary.

#### Scenario: ssh found in PATH and not sshroute itself
- **WHEN** LookPath finds an `ssh` that is a different inode from `os.Args[0]`
- **THEN** that path is used

### Requirement: Shadow-mode recursion guard
If `LookPath("ssh")` resolves to the same file as `os.Args[0]` (by `os.SameFile` inode comparison), that result SHALL be discarded.

#### Scenario: LookPath returns sshroute itself
- **WHEN** sshroute is installed as `~/.local/bin/ssh` and LookPath finds it
- **THEN** the result is discarded and resolution falls through to the hardcoded fallback

### Requirement: Hardcoded fallback
When all other resolution steps produce no usable binary, `/usr/bin/ssh` SHALL be used.

#### Scenario: No env var, no config, no usable PATH result
- **WHEN** all resolution steps fail or are skipped
- **THEN** `/usr/bin/ssh` is returned
