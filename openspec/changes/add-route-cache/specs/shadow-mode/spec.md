## ADDED Requirements

### Requirement: Shadow-mode fallback uses the route cache
When shadow mode runs with `SSHROUTE_FALLBACK=1`, it SHALL build the fallback order with the route cache and record the outcome, exactly like `connect --fallback`. `SSHROUTE_CACHE=0` SHALL disable this.

#### Scenario: git push through shadow mode
- **WHEN** `SSHROUTE_FALLBACK=1` is set and a previous run cached profile `lan` for the alias
- **THEN** `lan` is tried first

#### Scenario: Disabled by env
- **WHEN** `SSHROUTE_FALLBACK=1` and `SSHROUTE_CACHE=0` are set
- **THEN** the normal fallback order is used and the cache is not updated
