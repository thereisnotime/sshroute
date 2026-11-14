# Route cache

`sshroute connect --fallback` tries profiles in a fixed order: the profile for the detected network, then the other profiles by network priority, then `default`. Every dead route costs a full `ConnectTimeout`. The route cache remembers which profile connected last time and tries it first, so a host whose working route sits late in that order connects on the first attempt.

## When it applies

| Mode | Reads cache | Writes cache |
|---|---|---|
| `connect --fallback` | yes | yes |
| `connect --reconnect --fallback` | yes, every cycle | yes, every cycle |
| shadow mode with `SSHROUTE_FALLBACK=1` | yes | yes |
| `connect --fallback --dry-run`, `resolve --fallback` | yes | no |
| plain `connect` (no `--fallback`) | no | no |

Plain `connect` replaces itself with ssh (`syscall.Exec`), so it cannot see ssh's exit code. Using a cached profile there would turn a stale entry into a plain failure with nothing to fall back to, so plain `connect` ignores the cache.

## How it decides

After a fallback walk:

- every profile whose ssh exited 255 is recorded as a connection failure on the detected network;
- the profile whose ssh exited with any other code is recorded as the last success. A remote command that fails with exit 1 still counts, because the route worked.

On the next walk the cached profile goes first, followed by the normal order without it. An entry is used only if:

- the profile still exists for the host in the config;
- it is younger than `cache.ttl` (default `7d`, `0` disables expiry);
- that profile has not failed to connect since the success. With `scope: network`, only a failure on the same detected network counts.

If the cached profile fails, sshroute records the failure and carries on through the rest of the order. A stale entry costs exactly one failed attempt and is not used again until that profile connects.

Success is only known when ssh exits. An interactive session records its route at logout. A long session that drops also exits 255 and is recorded as a failure, which puts that host back on the normal order next time.

### Scope

- `scope: network` (default): remembered per alias and detected network. Useful when the best route depends on where you are (LAN at home, WireGuard elsewhere).
- `scope: alias`: the last route that worked for the alias on any network.

Both scopes write the same data, so switching scope takes effect immediately.

## Configuration

```yaml
cache:
  enabled: true        # default true
  path: ~/.local/state/sshroute/state.json
  ttl: 7d              # Go durations ("12h", "90m") plus a "d" suffix; 0 = never expires
  scope: network       # network | alias
  lock_timeout: 1s     # how long a write waits for the lock before it is skipped
```

| Override | Effect |
|---|---|
| `--no-cache` on `connect` / `resolve` | Neither read nor write for this run. Wins over everything |
| `SSHROUTE_CACHE=0` (`false`, `off`, `no`) | Disable without touching argv |
| `SSHROUTE_CACHE=1` (`true`, `on`, `yes`) | Enable even when `cache.enabled: false` |
| `SSHROUTE_CACHE_PATH` | State file path. Wins over `cache.path` |

## State file location

| Platform | Default path |
|---|---|
| Any, with `XDG_STATE_HOME` set | `$XDG_STATE_HOME/sshroute/state.json` |
| Linux, Termux | `~/.local/state/sshroute/state.json` |
| macOS | `~/.local/state/sshroute/state.json` (same XDG layout as the config in `~/.config/sshroute/`) |
| Windows | `%LOCALAPPDATA%\sshroute\state.json` |

`sshroute cache path` prints the path in effect, after config and env overrides. The directory is created with mode 0700 and the file with 0600. A `state.json.lock` file next to it is used for locking; leave it in place.

## Debugging

```sh
sshroute resolve myserver --fallback            # order with SOURCE (cache, detected, priority, default) and CACHED AT
sshroute connect myserver --fallback --dry-run  # prints "[dry-run] cache: ..." and "[dry-run] order: ..."
sshroute -v connect myserver --fallback         # logs cache hit/miss and what was recorded
sshroute cache show                             # everything cached, with status
sshroute cache clear myserver                   # forget one host
```

Example dry run:

```
[dry-run] cache: hit [network scope, key home-wg]: profile "internet" connected on network "home-wg" at 2026-11-14T10:00:00Z
[dry-run] order: internet (cache), home-wg (detected), zone-a (priority), default (default)
[dry-run] /usr/bin/ssh -J gateway 10.0.0.5
...
```

`cache show` statuses: `fresh` (will be used), `expired` (older than `ttl`), `failed` (the profile failed to connect since), `unknown-profile` (the profile or host is no longer in the config).

## State file format

Other tools can read the state file directly, for example to show "last connected via <profile> at <time>" per host. The format is versioned; this is version 1.

```json
{
  "version": 1,
  "hosts": {
    "eagle": {
      "last_success": {
        "profile": "internet",
        "network": "home-wg",
        "succeeded_at": "2026-11-14T10:00:00Z"
      },
      "networks": {
        "home-wg": {
          "profile": "internet",
          "network": "home-wg",
          "succeeded_at": "2026-11-14T10:00:00Z"
        },
        "default": {
          "profile": "default",
          "network": "default",
          "succeeded_at": "2026-11-12T18:30:12Z"
        }
      },
      "failures": {
        "zone-a": {
          "network": "home-wg",
          "failed_at": "2026-11-14T09:59:55Z"
        }
      }
    }
  }
}
```

| Field | Meaning |
|---|---|
| `version` | Format version. Readers should refuse or ignore versions they do not know |
| `hosts.<alias>` | One object per sshroute alias. Any of its fields may be missing |
| `last_success` | Most recent successful fallback connection on any network |
| `networks.<network>` | Most recent success while `<network>` was the detected network (`default` when none matched) |
| `failures.<profile>` | Most recent connection failure (ssh exit 255) of that profile, and the detected network at the time |
| `profile` | The sshroute profile name (a network name or `default`) that connected |
| `network` | The detected network when it happened |
| `succeeded_at`, `failed_at` | RFC 3339, UTC, second precision |

Reading it from a script:

```sh
jq -r '.hosts | to_entries[] | select(.value.last_success)
       | "\(.key)\t\(.value.last_success.profile)\t\(.value.last_success.succeeded_at)"' \
  "$(sshroute cache path)"
```

Notes for readers:

- Read the file without locking. It is only ever replaced by an atomic rename, so you always see a complete file.
- Do not write to it. If you need to reset entries, use `sshroute cache clear`.
- Ignore fields you do not recognise; new optional fields may be added without a version bump. An incompatible change bumps `version`, and an older sshroute leaves a newer file untouched.
- The file may be missing (nothing cached yet, or the cache is disabled).

## Concurrency and failure handling

Many `sshroute connect --fallback` processes can run in parallel. Each writes once, after its ssh exits:

1. take an exclusive lock on `state.json.lock` (`flock` on Linux, macOS, Android and the BSDs, `LockFileEx` on Windows), polling until `lock_timeout`;
2. re-read the current file, apply this run's outcome, write a temp file in the same directory and rename it over `state.json`.

Re-reading under the lock means parallel runs never drop each other's entries. If the lock is not acquired in time the write is skipped (logged with `-v`); the connection is never held up waiting for it. On other platforms there is no lock: writes stay atomic, but two simultaneous writers can lose one update.

A missing, unreadable or corrupt state file is treated as empty and rewritten on the next write. A cache error never changes the result of a connection.
