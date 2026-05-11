# Tasks: resolve, copy, and shell completion

## resolve command
- [x] Create `cmd/resolve.go` with `ResolveRow` struct and `runResolve`
- [x] Add `--network` flag to override auto-detection
- [x] Wire `ValidArgsFunction: completeAliases`
- [x] Support table/json/yaml output via existing formatter

## copy command
- [x] Create `cmd/copy.go` with `buildSCPArgv`, `rewriteRemote`, `resolveSCPBinary`
- [x] `SSHROUTE_SCP` env var override for scp binary
- [x] `--dry-run` support
- [x] Wire `ValidArgsFunction: completeAliases`

## Shell completion
- [x] Create `cmd/completion.go` with `completeAliases` function
- [x] Register on connect, remove, resolve, copy commands
- [x] Sort aliases for stable completion output

## Tests
- [x] Tests for resolve command (table/json, unknown alias, network override)
- [x] Tests for copy command (dry-run output, remote path rewriting, unknown alias)
- [x] Tests for completeAliases (returns sorted aliases, no-op after first arg)
- [x] Coverage at 80.6%

## Documentation
- [x] `docs/homelab.md`
- [x] `docs/corporate.md`
- [x] `docs/shadow-mode.md`
- [x] `docs/shell-completion.md`
- [x] `docs/scripting.md`
- [x] `docs/README.md` index
- [x] README: Documentation section, resolve/copy in Commands reference

## Release
- [x] Release v0.2.4
