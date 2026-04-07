# sshroute

`sshroute` is a network-aware SSH router. It detects which network you're on — VPN, office LAN, direct internet — and automatically selects the correct SSH parameters (host, port, user, identity file, jump host) for a given logical host alias. Instead of maintaining multiple `Host` blocks in `~/.ssh/config` or manually swapping keys and addresses, you define each host once with per-network overrides and let sshroute figure out the rest.

## Installation

**Go install:**
```sh
go install github.com/thereisnotime/sshroute@latest
```

**Download a prebuilt binary** from the [releases page](https://github.com/thereisnotime/sshroute/releases) and place it in your `$PATH`.

**Shadow mode** (transparent SSH replacement):
```sh
mkdir -p ~/.local/bin
ln -s $(which sshroute) ~/.local/bin/ssh
# Make sure ~/.local/bin appears before /usr/bin in your PATH
export PATH="$HOME/.local/bin:$PATH"
```

Once symlinked as `ssh`, sshroute intercepts all SSH calls automatically. Hosts not defined in your config are passed through to the real `/usr/bin/ssh` unchanged.

## Config file

Default location: `~/.config/sshroute/config.yaml`

Override with `--config /path/to/config.yaml` or the `SSHROUTE_CONFIG` environment variable.

```yaml
networks:
  vpn-work:
    - type: route
      match: "10.8.0.0/24"
    - type: interface
      match: tun0
  office:
    - type: ping
      host: 192.168.1.1
      timeout: 500ms
  corp-vpn:
    - type: exec
      command: "ip link show wg0 | grep -q UP"

hosts:
  myserver:
    default:
      host: myserver.example.com
      port: 22
      user: alice
      key: ~/.ssh/id_ed25519
    vpn-work:
      host: 10.8.0.50
      port: 2222
      jump: bastion.vpn
    office:
      host: 192.168.1.100
      user: alice
```

The `default` profile is required for every host and is used when no network check matches.

## Network detection types

| Type        | Description                                              | Key fields              |
|-------------|----------------------------------------------------------|-------------------------|
| `route`     | Checks whether a subnet is present in the routing table  | `match` (CIDR)          |
| `interface` | Checks whether a named network interface is up           | `match` (iface name)    |
| `ping`      | Sends an ICMP ping and checks for a reply within timeout | `host`, `timeout`       |
| `exec`      | Runs a shell command; passes if exit code is 0           | `command`               |

All checks within a network definition must pass (AND logic). Networks are evaluated in config order; the first full match wins.

## Usage

```sh
# Connect to a host — network is detected automatically
sshroute connect myserver

# List all configured hosts
sshroute list

# Add or update a host
sshroute add myserver \
  --host myserver.example.com \
  --user alice \
  --key ~/.ssh/id_ed25519

# Add a network-specific override
sshroute add myserver \
  --network vpn-work \
  --host 10.8.0.50 \
  --port 2222 \
  --jump bastion.vpn

# Remove a host
sshroute remove myserver

# Show active network
sshroute network

# List all network definitions
sshroute network list

# Test whether a specific network is active
sshroute network test vpn-work

# Print version info
sshroute version

# Change output format (table is default)
sshroute list -o json
sshroute list -o yaml

# Preview the SSH command without running it
sshroute connect myserver --dry-run
```

## Shadow mode

When the binary (or a symlink to it) is named `ssh`, sshroute operates transparently:

```sh
ssh myserver          # routed via sshroute config
ssh -p 2222 other     # "other" not in config → passed through as-is
```

This means existing scripts, `git`, `rsync`, `scp`, and anything else that shells out to `ssh` will benefit automatically without any changes.

## Building from source

Requires [Go 1.22+](https://go.dev/dl/) and [just](https://just.systems/).

```sh
git clone git@github.com:thereisnotime/sshroute.git
cd sshroute

just build          # outputs bin/sshroute
just build-all      # cross-compile for linux/darwin × amd64/arm64
just test           # run tests with race detector
just install        # go install with version ldflags
```
