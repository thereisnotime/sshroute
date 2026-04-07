# sshroute

[![CI](https://github.com/thereisnotime/sshroute/actions/workflows/ci.yaml/badge.svg)](https://github.com/thereisnotime/sshroute/actions/workflows/ci.yaml)
[![Release](https://github.com/thereisnotime/sshroute/actions/workflows/release.yaml/badge.svg)](https://github.com/thereisnotime/sshroute/actions/workflows/release.yaml)
[![Latest Release](https://img.shields.io/github/v/release/thereisnotime/sshroute)](https://github.com/thereisnotime/sshroute/releases/latest)
[![codecov](https://codecov.io/gh/thereisnotime/sshroute/branch/main/graph/badge.svg)](https://codecov.io/gh/thereisnotime/sshroute)
[![Go Report Card](https://goreportcard.com/badge/github.com/thereisnotime/sshroute)](https://goreportcard.com/report/github.com/thereisnotime/sshroute)
[![Go Reference](https://pkg.go.dev/badge/github.com/thereisnotime/sshroute.svg)](https://pkg.go.dev/github.com/thereisnotime/sshroute)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/thereisnotime/sshroute/badge)](https://scorecard.dev/viewer/?uri=github.com/thereisnotime/sshroute)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Network-aware SSH router. Detects your active network or VPN and automatically selects the right host, port, identity file, and jump host for each SSH connection — without touching `~/.ssh/config`.

## How it works

Define each logical host once with a `default` profile and optional per-network overrides. On every connection, sshroute detects which network you're on (VPN, office LAN, WireGuard peer, etc.) and resolves the correct SSH parameters before handing off to the real `/usr/bin/ssh`.

```
ssh myserver
  → sshroute detects: corp-vpn is active
  → resolves: 10.100.0.50:2222 via bastion.corp.internal
  → exec /usr/bin/ssh -p 2222 -i ~/.ssh/corp_key -J bastion.corp.internal 10.100.0.50
```

## Installation

### Binary download

Download the latest release from [GitHub Releases](https://github.com/thereisnotime/sshroute/releases). Binaries are available for Linux and macOS on AMD64 and ARM64.

### Go install

```sh
go install github.com/thereisnotime/sshroute@latest
```

### Docker

```sh
docker run --rm -v ~/.config/sshroute:/root/.config/sshroute \
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

| Command | Description |
|---|---|
| `init` | Create a starter config file with commented examples |
| `connect <alias>` | Connect to a host — resolves params for the active network |
| `list` | List all configured hosts and their active profile |
| `add <alias>` | Add or update a host (use `--network` for per-network overrides) |
| `remove <alias>` | Remove a host from the config |
| `network` | Show the currently detected network name |
| `network list` | List all configured networks and their check rules |
| `network test <name>` | Run checks for a specific network and show pass/fail per rule |
| `config` | Print the config file path |
| `config edit` | Open the config in `$EDITOR` |
| `version` | Print version, commit, and build info |

## Global flags

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--config` | `SSHROUTE_CONFIG` | `~/.config/sshroute/config.yaml` | Config file path |
| `-o, --output` | | `table` | Output format: `table`, `json`, `yaml` |
| `-v, --verbose` | `SSHROUTE_VERBOSE=1` | `false` | Debug logging to stderr |
| `--dry-run` | | `false` | Print resolved SSH command without executing |

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
    corp-vpn:
      host: 10.100.0.50
      port: 2222
      key: ~/.ssh/corp_key
      jump: bastion.corp.internal
    office:
      host: 192.168.1.50
```

Every host must have a `default` profile. Network profiles only need to specify fields that differ from the default — unset fields inherit from `default`.

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

See the [`examples/`](examples/) directory for ready-to-use configs:

| File | Use case |
|---|---|
| [`basic.yaml`](examples/basic.yaml) | Single host, VPN vs public fallback |
| [`multi-network.yaml`](examples/multi-network.yaml) | Office LAN, corp VPN, remote VPN, public |
| [`wireguard-backconnect.yaml`](examples/wireguard-backconnect.yaml) | WireGuard peer that backconnects to you |
| [`jump-hosts.yaml`](examples/jump-hosts.yaml) | Different bastions per network |

### WireGuard backconnect

A common pattern: a remote machine tunnels back to you over WireGuard. Its peer IP falls outside your normal subnet CIDR. Use a route check combined with a ping to verify it's actually up before trying to connect directly:

```yaml
networks:
  wg-peer:
    priority: 10
    checks:
      - type: route
        match: "10.100.200.5"    # route must exist
      - type: ping
        host: "10.100.200.5"     # AND peer must respond
        timeout: 2s

hosts:
  remote-machine:
    default:
      host: remote-machine.example.com
      port: 22
      user: admin
      key: ~/.ssh/id_ed25519
    wg-peer:
      host: 10.100.200.5         # connect directly when peer is up
      user: admin
      key: ~/.ssh/wg_key
```

## Output formats

All list commands support multiple output formats:

```sh
sshroute list                  # table (default)
sshroute list -o json          # JSON — for scripting
sshroute list -o yaml          # YAML
sshroute network list -o json
```

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
