# Design: resolve, copy, and shell completion

## resolve command

`cmd/resolve.go` — `sshroute resolve <alias> [--network <n>]`

Calls `network.Detect` (or uses `--network` override), then `ssh.Resolve`, and formats a `ResolveRow` struct. `ResolveRow` includes the full `ssh` command string built from `ssh.BuildArgv`.

```go
type ResolveRow struct {
    Alias, Network, Host string
    Port                 int
    User, Key, Jump      string
    Command              string   // full argv joined with spaces
}
```

Output via the existing `outfmt.New(output).Format(...)` path — table/json/yaml all work for free.

`--network` flag bypasses `network.Detect` entirely. When omitted, a detection error falls back to `"default"` with a debug log (non-fatal — useful when no network checks are configured).

## copy command

`cmd/copy.go` — `sshroute copy <alias> <src> <dst>`

Resolution path is identical to `connect`: `loadConfig → network.Detect → ssh.Resolve`. Builds an scp argv rather than an ssh argv.

### Remote path rewriting

`<alias>:<path>` in src or dst is rewritten to `user@host:<path>`. This mirrors scp's own convention while keeping the user-facing API alias-based.

```go
func rewriteRemote(arg, alias, remote string) string {
    prefix := alias + ":"
    if strings.HasPrefix(arg, prefix) {
        return remote + ":" + arg[len(prefix):]
    }
    return arg
}
```

### SCP binary resolution

`resolveSCPBinary()` checks `SSHROUTE_SCP` env var, then `exec.LookPath("scp")`, then falls back to `/usr/bin/scp`. Simpler than the SSH resolver — no shadow-mode recursion risk for scp.

### Execution

Uses `exec.Command` + pipe stdin/stdout/stderr (not `syscall.Exec`) because scp doesn't need signal transparency — it's a file transfer, not an interactive session. Error is wrapped and returned.

## Shell completion

`cmd/completion.go` — `completeAliases` function satisfies `cobra.CompletionFunc`.

```go
func completeAliases(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
    if len(args) > 0 { return nil, cobra.ShellCompDirectiveNoFileComp }
    cfg, err := config.Load(cfgFile)
    if err != nil { return nil, cobra.ShellCompDirectiveError }
    aliases := make([]string, 0, len(cfg.Hosts))
    for alias := range cfg.Hosts { aliases = append(aliases, alias) }
    sort.Strings(aliases)
    return aliases, cobra.ShellCompDirectiveNoFileComp
}
```

Registered on `connect`, `remove`, `resolve`, and `copy` via `ValidArgsFunction: completeAliases`. Returns after the first arg to avoid completing further positional arguments (src/dst for copy are paths, not aliases).

## Documentation

`docs/` folder added with five guides: homelab, corporate, shadow-mode, shell-completion, scripting. README updated with a Documentation section and resolve/copy entries in the Commands reference.

## Files changed

- `cmd/completion.go` — `completeAliases` function
- `cmd/resolve.go` — resolve command
- `cmd/copy.go` — copy command
- `cmd/connect.go` — wire `ValidArgsFunction`
- `cmd/remove.go` — wire `ValidArgsFunction`
- `cmd/cmd_test.go` — tests for all three new commands
- `docs/` — five new guide pages + index
- `README.md` — Documentation section, resolve/copy command reference
