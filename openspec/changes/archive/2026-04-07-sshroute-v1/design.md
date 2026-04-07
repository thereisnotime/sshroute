# Design: sshroute v1

## Architecture

Three-layer separation (following atscale-cli pattern):

```
cmd/          — Cobra CLI layer. Flags, validation, output formatting only.
internal/     — All business logic. No Cobra imports. Pure functions.
main.go       — Entrypoint. Calls cmd.Execute() or shadow mode handler.
```

## Package Design

### internal/config
- `Config` struct: top-level, contains `Networks` and `Hosts` maps
- `NetworkConfig`: list of `NetworkCheck` (type + match/host/command/timeout)
- `HostConfig`: map of network name → `SSHParams`
- `SSHParams`: Host, Port, User, Key, Jump strings
- Loader: XDG-aware path (~/.config/sshroute/config.yaml), ~ expansion
- Validator: check required fields, valid check types, port ranges

### internal/network
- `Detector` orchestrates checks for each configured network in order
- Returns the first network name whose checks all pass, or "default"
- Four check types:
  - `route`: `ip route show` → grep for subnet string
  - `interface`: read `/sys/class/net/<iface>/operstate` → check "up"
  - `ping`: ICMP echo with configurable timeout (uses golang.org/x/net/icmp)
  - `exec`: run shell command, exit 0 = active

### internal/ssh
- `Router`: given Config + active network name + host alias → resolved SSHParams
  (merges default params with network-specific overrides)
- `Args`: parse raw SSH argv — extract [user@]host, collect remaining flags
  SSH flags with required values: -b -c -D -E -e -F -I -i -J -L -l -m -o -p -Q -R -S -W -w
- `Executor`: build final []string argv, syscall.Exec to /usr/bin/ssh

### internal/output
- `Formatter` interface: `Format(w io.Writer, data any) error`
- Factory: `New(format string) Formatter` → table | json | yaml
- Struct tags: `json:"x" yaml:"x" table:"X"` for reflection-based table rendering

### cmd (Cobra commands)
- `root.go`: detect shadow mode via `filepath.Base(os.Args[0]) == "ssh"`,
  global flags: `--config`, `--output`, `--verbose`, `--dry-run`
- `connect`: resolve + exec (or print command if --dry-run)
- `list`: show all hosts with their network profiles
- `add`: add/edit host via flags (`--host`, `--port`, `--user`, `--key`, `--jump`, `--network`)
- `remove`: remove host from config
- `network`: show detected network; subcommands: list, test <name>
- `config`: show config path; subcommand: edit (open $EDITOR)
- `version`: Version, Commit, Date, Go version

## Shadow Mode

```
$PATH: ~/.local/bin (sshroute symlinked as ssh) → /usr/bin
ssh myserver
  → ~/.local/bin/ssh (sshroute binary)
  → filepath.Base(os.Args[0]) == "ssh" → shadow mode
  → parse argv, extract host alias
  → host in config? → resolve + syscall.Exec /usr/bin/ssh <resolved args>
  → host unknown? → syscall.Exec /usr/bin/ssh <original argv>
```

## Key Decisions

- **syscall.Exec not os/exec**: replace process entirely, no wrapper process in ps tree,
  signals pass through naturally, exit codes are exact
- **First-network-wins**: detection runs in config order, returns on first match
- **Default profile always required**: every host must have a `default` key in config
- **No caching of network detection**: run fresh on every invocation (fast checks ~1ms)
- **YAML v3**: goccy/go-yaml for performance, or gopkg.in/yaml.v3 for compatibility
