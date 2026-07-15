# sshroute

<table>
  <tr>
    <th>CI</th>
    <th>Code</th>
    <th>OpenSpec</th>
    <th>Security</th>
  </tr>
  <tr>
    <td>
      <a href="https://github.com/thereisnotime/sshroute/actions/workflows/ci.yaml"><img src="https://github.com/thereisnotime/sshroute/actions/workflows/ci.yaml/badge.svg" alt="CI"></a><br>
      <a href="https://github.com/thereisnotime/sshroute/actions/workflows/release.yaml"><img src="https://github.com/thereisnotime/sshroute/actions/workflows/release.yaml/badge.svg" alt="Release"></a><br>
      <a href="https://github.com/thereisnotime/sshroute/actions/workflows/openspec-badge.yaml"><img src="https://github.com/thereisnotime/sshroute/actions/workflows/openspec-badge.yaml/badge.svg" alt="OpenSpec Badge"></a><br>
      <a href="https://github.com/thereisnotime/sshroute/actions/workflows/scorecard.yaml"><img src="https://github.com/thereisnotime/sshroute/actions/workflows/scorecard.yaml/badge.svg" alt="Scorecard"></a>
    </td>
    <td>
      <a href="https://github.com/thereisnotime/sshroute/releases/latest"><img src="https://img.shields.io/github/v/release/thereisnotime/sshroute" alt="Latest Release"></a><br>
      <a href="https://codecov.io/gh/thereisnotime/sshroute"><img src="https://codecov.io/gh/thereisnotime/sshroute/branch/main/graph/badge.svg" alt="codecov"></a><br>
      <a href="https://goreportcard.com/report/github.com/thereisnotime/sshroute"><img src="https://goreportcard.com/badge/github.com/thereisnotime/sshroute" alt="Go Report Card"></a><br>
      <a href="https://pkg.go.dev/github.com/thereisnotime/sshroute"><img src="https://pkg.go.dev/badge/github.com/thereisnotime/sshroute.svg" alt="Go Reference"></a>
    </td>
    <td>
      <a href="openspec/specs/"><img src="https://raw.githubusercontent.com/thereisnotime/sshroute/gh-pages/badges/number_of_specs.svg" alt="Specs"></a><br>
      <a href="openspec/specs/"><img src="https://raw.githubusercontent.com/thereisnotime/sshroute/gh-pages/badges/number_of_requirements.svg" alt="Requirements"></a><br>
      <a href="openspec/changes/"><img src="https://raw.githubusercontent.com/thereisnotime/sshroute/gh-pages/badges/tasks_status.svg" alt="Tasks"></a><br>
      <a href="openspec/changes/"><img src="https://raw.githubusercontent.com/thereisnotime/sshroute/gh-pages/badges/open_changes.svg" alt="Open Changes"></a>
    </td>
    <td>
      <a href="https://scorecard.dev/viewer/?uri=github.com/thereisnotime/sshroute"><img src="https://api.scorecard.dev/projects/github.com/thereisnotime/sshroute/badge" alt="OpenSSF Scorecard"></a><br>
      <a href="https://www.bestpractices.dev/projects/12389"><img src="https://www.bestpractices.dev/projects/12389/badge" alt="CII Best Practices"></a><br>
      <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License: Apache 2.0"></a>
    </td>
  </tr>
</table>

Network-aware SSH router. Detects your active network or VPN and automatically selects the right host, port, identity file, and jump host for each SSH connection — without touching `~/.ssh/config`.

## How it works

Define each logical host once with a `default` profile and optional per-network overrides. On every connection, sshroute detects which network you're on (VPN, office LAN, WireGuard peer, etc.) and resolves the correct SSH parameters before handing off to the real `/usr/bin/ssh`.

```
ssh myserver
  → sshroute detects: corp-vpn is active
  → resolves: 10.100.0.50:2222 via bastion.corp.internal
  → exec /usr/bin/ssh -p 2222 -i ~/.ssh/corp_key -J bastion.corp.internal 10.100.0.50
```

## Why sshroute?

### For homelabbers

Your lab probably has at least two realities: you're either sitting at home on the LAN, or you're away and coming in over WireGuard or another VPN. The problem is `~/.ssh/config` doesn't know which one you're in — so you end up with separate aliases (`server-lan`, `server-vpn`), or a jump host that only works half the time, or you just memorize IPs.

sshroute solves this by detecting your current network before every connection. When the WireGuard interface is up and the peer route exists, it connects directly to the tunnel IP. When you're on the LAN, it uses the local address. When neither is reachable, it falls back to the public hostname. One alias, three realities, zero manual switching.

It also intercepts SSH transparently — `git push`, `rsync`, `scp` all go through it automatically once you set up shadow mode. No wrappers, no shell functions, no thinking.

### For corporate environments

Enterprise networks are worse. You have the public internet, maybe a site-to-site VPN, maybe a personal VPN split-tunnel, and inside that you have different jump hosts depending on which environment you're targeting — dev, staging, prod, each with their own bastion and key. Keeping this straight in `~/.ssh/config` means either one enormous config that breaks whenever infra changes, or you write a script that everyone on the team maintains differently.

sshroute lets you define the routing logic declaratively, keep it in a versioned YAML file, and share it across the team. The same config works for everyone — the right network is detected automatically based on what interfaces or routes are active on each machine. Keys, ports, users, and jump hosts resolve without the user having to think about it.

## How it compares

| Feature | `~/.ssh/config` | WireGuard-only | Teleport / Boundary | sshroute |
|---|---|---|---|---|
| Detects your current network | ❌ | ❌ | ❌ | ✅ |
| Picks the best path automatically | ❌ | ❌ | ❌ | ✅ |
| Falls back on connection failure | ❌ | ❌ | ✅ | ✅ |
| Auto-reconnect + re-route on drop | ❌ | ⚠️ tunnel roams | ⚠️ via fixed proxy | ✅ |
| One command per host, any location | ❌ | ⚠️ VPN must be up | ✅ | ✅ |
| Config size for 10 hosts × 4 paths | 📄 ~600 lines | 📄 ~600 lines + VPN config | 📄 server-side config | 📄 ~60 lines |
| Roaming mobile devices | ⚠️ manual aliases | ⚠️ VPN required | ✅ | ✅ |
| Jump host auto-chaining | ⚠️ manual `-J` | ➖ n/a | ✅ | ✅ |
| Works with scp / rsync / git / Ansible | ✅ | ✅ | ⚠️ partial | ✅ |
| No server-side install on targets | ✅ | ❌ | ❌ | ✅ |
| No auth server or daemon to run | ✅ | ❌ | ❌ | ✅ |
| No client agent | ✅ | ❌ | ❌ | ✅ |
| Open source, fully self-hosted | ✅ | ✅ | ⚠️ open-core | ✅ |

Teleport and Boundary are a different category — they add access control, audit logs, and certificate-based auth on top of routing. If that's what you need, use them. sshroute is for when you want the routing intelligence without the operational overhead of running a central auth server.

## Installation

### Binary download

Download the latest release from [GitHub Releases](https://github.com/thereisnotime/sshroute/releases). Binaries are available for Linux, macOS, and Android on AMD64 and ARM64.

### Go install

```sh
go install github.com/thereisnotime/sshroute@latest
```

### Android (Termux)

Download the `android_arm64` tarball from [GitHub Releases](https://github.com/thereisnotime/sshroute/releases), extract, and place the binary in `~/.local/bin`:

```sh
mkdir -p ~/.local/bin
curl -Lo "$TMPDIR/sshroute.tar.gz" \
  https://github.com/thereisnotime/sshroute/releases/latest/download/sshroute_android_arm64.tar.gz
tar -xzf "$TMPDIR/sshroute.tar.gz" -C ~/.local/bin sshroute
chmod +x ~/.local/bin/sshroute
```

Add `~/.local/bin` to your `PATH` in `~/.bashrc` or `~/.profile` if it isn't already:

```sh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Alternatively, compile from source with Termux's Go. Because the official Go toolchain doesn't publish android/arm64 binaries, set `GOTOOLCHAIN=local` to use what Termux ships:

```sh
GOTOOLCHAIN=local go install github.com/thereisnotime/sshroute@latest
```

After installing, set the SSH binary path since Termux doesn't have `/usr/bin/ssh`:

```yaml
# ~/.config/sshroute/config.yaml
ssh_binary: /data/data/com.termux/files/usr/bin/ssh
```

Or via environment variable: `export SSHROUTE_SSH=$(which ssh)`

### Docker

```sh
docker run --rm -v ~/.config/sshroute:/root/.config/sshroute \
  ghcr.io/thereisnotime/sshroute network
```

### Podman

```sh
podman run --rm -v ~/.config/sshroute:/root/.config/sshroute \
  ghcr.io/thereisnotime/sshroute network
```

On SELinux-enabled systems (Fedora, RHEL, etc.) add `:Z` to the volume flag:

```sh
podman run --rm -v ~/.config/sshroute:/root/.config/sshroute:Z \
  ghcr.io/thereisnotime/sshroute network
```

### Shadow mode (transparent SSH replacement)

Install sshroute as `ssh` earlier in your `$PATH`. All SSH calls — from your terminal, `git`, `rsync`, `scp` — are intercepted automatically. Hosts not in your config pass through to `/usr/bin/ssh` unchanged.

```sh
mkdir -p ~/.local/bin
ln -s $(which sshroute) ~/.local/bin/ssh

# Add to ~/.bashrc or ~/.zshrc if not already present:
export PATH="$HOME/.local/bin:$PATH"
```

## Quick start

```sh
# Add a host with a default profile
sshroute add myserver --host myserver.example.com --user alice --key ~/.ssh/id_ed25519

# Add a VPN-specific override
sshroute add myserver --network vpn --host 10.8.0.50 --port 2222 --jump bastion.vpn

# Connect — network is detected automatically
sshroute connect myserver

# Preview the resolved command without running it
sshroute connect myserver --dry-run

# See what network is currently active
sshroute network
```

## Commands

### Global flags

These flags apply to every command:

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--config` | `SSHROUTE_CONFIG` | `~/.config/sshroute/config.yaml` | Config file path |
| `-o, --output` | | `table` | Output format: `table`, `json`, `yaml` |
| `-v, --verbose` | `SSHROUTE_VERBOSE=1` | `false` | Debug logging to stderr |
| `--dry-run` | | `false` | Print resolved SSH command without executing |

### `init`

Create a starter config file with commented examples. Fails if the file already exists.

| Flag | Default | Description |
|---|---|---|
| `--force` | `false` | Overwrite an existing config file |

### `connect <alias>`

Detect the active network, resolve SSH parameters for `alias`, and exec the real SSH binary. Any extra arguments after the alias are passed through to SSH unchanged.

| Flag | Default | Description |
|---|---|---|
| `--fallback` | `false` | Try every profile in priority order, retrying the next one only on a connection failure (exit 255) |
| `--reconnect` | `false` | Supervise the connection and automatically reconnect when it drops, re-detecting the active network and re-resolving the route each time |
| `--reconnect-delay` | `2s` | Wait between reconnect attempts when `--reconnect` is set |

With `--reconnect`, sshroute keeps ssh alive across dropped connections (laptop sleep, WiFi handoff, roaming between networks). Because it re-detects the network on every reconnect, it follows you onto a different route: for example, sleeping on the LAN and waking on a hotspot reconnects over the public route instead of retrying the now-unreachable LAN address. A clean logout (exit 0) or an auth/remote-command failure stops the loop; only genuine connection drops reconnect. Reconnect runs ssh as a subprocess (like `--fallback`), so sshroute stays resident for the session; `SIGINT`/`SIGTERM` tears it down. Session state across the blip is your multiplexer's job (tmux/zellij); combine `--reconnect` with `-- tmux attach` or `-- zellij attach -c <name>` to land straight back in your session:

```sh
sshroute connect myserver --reconnect --fallback -- zellij attach -c work
```

### `list`

List all configured hosts and the SSH parameters that would be used on the current network. Supports `-o table|json|yaml`.

### `add <alias>`

Add a host or update an existing one. Omitted flags keep their current value. Run multiple times with different `--network` values to build per-network overrides.

| Flag | Default | Description |
|---|---|---|
| `--host` | | Hostname or IP address |
| `--port` | `22` | SSH port |
| `--user` | | SSH username |
| `--key` | | Path to identity file (supports `~`) |
| `--jump` | | Jump host — passed as `-J` to SSH |
| `--network` | `default` | Network profile to write the params into |

### `remove <alias>`

Remove all profiles for `alias` from the config.

### `network`

Print the name of the currently detected network (or `default` if none match).

### `network list`

List all configured networks with their priority, check rules, and current active state. Supports `-o table|json|yaml`.

### `network test <name>`

Run every check for network `name` and print pass/fail per rule. Useful for debugging detection logic.

### `config`

Print the resolved path to the config file.

### `config edit`

Open the config file in `$EDITOR` (falls back to `nano`). Creates the file and its parent directory if they do not exist.

### `resolve <alias>`

Print the SSH parameters that would be used for `alias` on the current network. Useful for debugging and scripting. Use `--network <name>` to override the detected network. Supports `-o table|json|yaml`.

| Flag | Default | Description |
|---|---|---|
| `--network` | auto-detect | Network profile to resolve against |

### `copy <alias> <src> <dst>`

Copy files to or from a configured host using `scp` with the same resolved parameters (key, port, jump) as `connect`. Use `<alias>:<path>` syntax for remote paths:

```sh
sshroute copy myserver ./local.txt myserver:/remote/path/
sshroute copy myserver myserver:/remote/file.txt ./local/
```

The `SSHROUTE_SCP` environment variable overrides the `scp` binary used.

### `version`

Print the version, git commit, build date, and Go runtime info.

### `update`

Update sshroute in place to the latest GitHub release. It downloads the archive for your platform, verifies its sha256 against `checksums.txt`, and — if [`cosign`](https://github.com/sigstore/cosign) is installed — verifies the release's cosign signature, before atomically replacing the running binary.

```sh
sshroute update            # download, verify, and install the latest release
sshroute update --check    # only report whether a newer version is available
sshroute update --force    # reinstall the latest even if already current
```

If sha256 (or cosign, when present) verification fails, the update aborts and the binary is left untouched. This targets installs of the release binary; if you installed via `go install` or a package manager, update with that instead.

## Config file

Default location: `~/.config/sshroute/config.yaml`

```yaml
networks:
  corp-vpn:
    priority: 10          # lower = checked first
    checks:
      - type: interface
        match: wg0
      - type: route
        match: 10.100.0.0

  office:
    priority: 20
    checks:
      - type: ping
        host: 192.168.1.1
        timeout: 500ms

hosts:
  myserver:
    default:              # required — used when no network matches
      host: myserver.example.com
      port: 22
      user: alice
      key: ~/.ssh/id_ed25519
      options:            # optional — passed as SSH -o Key=Value flags
        ConnectTimeout: "10"
        ServerAliveInterval: "30"
    corp-vpn:
      host: 10.100.0.50
      port: 2222
      key: ~/.ssh/corp_key
      jump: bastion.corp.internal
      options:
        ConnectTimeout: "5"   # overrides default for this network only
    office:
      host: 192.168.1.50
```

Every host must have a `default` profile. Network profiles only need to specify fields that differ from the default — unset fields inherit from `default`.

### Host profile fields

| Field | Type | Description |
|---|---|---|
| `host` | string | Hostname or IP address |
| `port` | int | SSH port (default: 22) |
| `user` | string | SSH user |
| `key` | string | Path to identity file (`~` is expanded) |
| `jump` | string | Jump host alias or `user@host` |
| `options` | map | Arbitrary SSH `-o Key=Value` flags (e.g. `ConnectTimeout`, `StrictHostKeyChecking`) |
| `comment` | string | Description shown in `sshroute list` |
| `tags` | list | Tags for filtering with `sshroute list --tag` |

`options` keys are merged from `default` into network profiles — network values override matching keys, non-overlapping keys are inherited.

## Network detection

Networks are evaluated in `priority` order (lowest value first). Alphabetical order breaks ties. The first network whose checks all pass is used; if none match, `default` applies.

| Check type | Passes when | Required fields |
|---|---|---|
| `route` | Subnet/IP appears in the kernel routing table | `match` |
| `interface` | Named interface exists and is operationally up | `match` |
| `ping` | Host responds to ICMP echo within timeout | `host`, `timeout` (optional, default 2s) |
| `exec` | Shell command exits with code 0 | `command` |

Multiple checks within one network definition use **AND** logic — all must pass.

## Examples

Ready-to-use config files are in [`examples/`](examples/):

| File | Use case |
|---|---|
| [`basic.yaml`](examples/basic.yaml) | Single host, VPN vs public fallback |
| [`multi-network.yaml`](examples/multi-network.yaml) | Office LAN, corp VPN, remote VPN, public |
| [`wireguard-backconnect.yaml`](examples/wireguard-backconnect.yaml) | WireGuard peer that backconnects to you |
| [`jump-hosts.yaml`](examples/jump-hosts.yaml) | Different bastions per network |
| [`multi-zone-roaming.yaml`](examples/multi-zone-roaming.yaml) | Multi-zone homelab with WireGuard gateway and roaming mobile devices |

## Documentation

In-depth guides are in [`docs/`](docs/):

| Guide | Description |
|---|---|
| [Homelab setup](docs/homelab.md) | Multi-zone homelab with WireGuard, jump hosts, NAS, k3s nodes |
| [Multi-zone roaming](docs/multi-zone-roaming.md) | Multiple LANs, WireGuard gateway, mobile devices that roam between networks |
| [Corporate / multi-environment](docs/corporate.md) | Dev/staging/prod with per-environment bastions and VPN detection |
| [Shadow mode](docs/shadow-mode.md) | Transparent SSH replacement — git, rsync, scp, Ansible |
| [Shell completion](docs/shell-completion.md) | Dynamic alias completion for bash, zsh, fish |
| [Scripting and automation](docs/scripting.md) | Using `resolve` and `copy` in scripts and CI pipelines |

## Output formats

All list commands support multiple output formats:

```sh
sshroute list                  # table (default)
sshroute list -o json          # JSON — for scripting
sshroute list -o yaml          # YAML
sshroute network list -o json
```

## Community

**Get the software** — download a pre-built binary from [Releases](https://github.com/thereisnotime/sshroute/releases), install with `go install github.com/thereisnotime/sshroute@latest`, or [build from source](#building-from-source).

**Feedback and bug reports** — open an issue on [GitHub Issues](https://github.com/thereisnotime/sshroute/issues). Use the bug report template for unexpected behaviour and the feature request template for ideas.

**Contributing** — see [CONTRIBUTING.md](CONTRIBUTING.md) for how to set up the project, run tests, and open a pull request. Security vulnerabilities should be reported privately via [GitHub Security Advisories](https://github.com/thereisnotime/sshroute/security/advisories/new).

## Building from source

Requires [Go 1.22+](https://go.dev/dl/) and [just](https://just.systems/).

```sh
git clone git@github.com:thereisnotime/sshroute.git
cd sshroute

just build        # outputs bin/sshroute
just build-all    # cross-compile linux/darwin × amd64/arm64
just test         # run tests with race detector
just install      # go install with version ldflags injected
```
