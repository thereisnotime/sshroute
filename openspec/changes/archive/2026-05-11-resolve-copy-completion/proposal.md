# Proposal: resolve, copy, and shell completion

## Problem

Three gaps in the v0.2.x CLI:

1. **No way to inspect resolved parameters without connecting.** Debugging network detection or auditing what key/host/jump would be used required reading the config and mentally resolving the inheritance chain.

2. **File transfer required manual parameter lookup.** `scp` doesn't know about sshroute's config, so users had to resolve the IP, port, key, and jump manually and pass them as scp flags.

3. **No tab completion for host aliases.** Every subcommand that takes an alias required typing it in full. With large configs this was error-prone.

## Proposed Solution

- `sshroute resolve <alias> [--network <n>]` — print the fully resolved SSH parameters for a host as table/json/yaml. Includes the exact `ssh` command that would be run.
- `sshroute copy <alias> <src> <dst>` — scp wrapper that resolves the same parameters as `connect` and rewrites `<alias>:<path>` to `user@host:<path>`. Supports `SSHROUTE_SCP` env override and `--dry-run`.
- Dynamic alias completion via Cobra's `ValidArgsFunction` — host aliases are read from the live config at completion time for `connect`, `remove`, `resolve`, and `copy`.
