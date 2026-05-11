# Homelab setup

A complete example for a typical homelab: multiple machines spread across one or more physical networks, accessible from home (LAN), over WireGuard when away, and through a public jump host when the tunnel is down.

## Network topology

```
Internet
  └─ router / WireGuard gateway (amgul)   192.168.77.1  (LAN)
                                           10.69.70.1    (WireGuard peer IP)
       ├─ main workstation (eagle)         192.168.77.91
       ├─ NAS (nas01)                      192.168.77.40
       ├─ k3s nodes (ferret, fossa, genet) 192.168.77.85-87
       └─ OPNsense router (opnsense)       192.168.77.1:22022
```

You carry a laptop. When you're home, you're on `192.168.77.0/24` directly. When you're away, you connect WireGuard and get a peer address from `10.69.70.0/24`. The gateway `amgul` is the only host with a public IP — everything else is reached through it.

## Network definitions

```yaml
networks:
  # WireGuard tunnel to the home network
  home-wg:
    priority: 10
    checks:
      - type: route
        match: 10.69.70.0      # WG subnet must be routable
      - type: ping
        host: 10.69.70.1       # AND the gateway peer must respond
        timeout: 1s

  # Direct LAN access (sitting at home)
  home-lan:
    priority: 20
    checks:
      - type: ping
        host: 192.168.77.1
        timeout: 300ms
```

`home-wg` has higher priority. If you're home with WireGuard running, it wins — you connect through the tunnel, which is fine. If you want LAN paths to win when on the local network, swap the priorities.

## Host definitions

```yaml
hosts:
  # WireGuard gateway — always the public entry point in default
  amgul:
    default:
      host: amgul.example.com
      port: 22
      user: admin
      key: ~/.ssh/id_homelab.pem
    home-wg:
      host: 10.69.70.1
    home-lan:
      host: 192.168.77.1

  # Workstation — only reachable on LAN or through the WG tunnel
  eagle:
    default:
      host: amgul.example.com
      port: 22
      user: admin
      key: ~/.ssh/id_homelab.pem
      jump: amgul.example.com   # jump through gateway when away with no tunnel
    home-wg:
      host: 192.168.77.91
      port: 22
      user: admin
      key: ~/.ssh/id_homelab.pem
    home-lan:
      host: 192.168.77.91
      port: 22
      user: admin
      key: ~/.ssh/id_homelab.pem

  # NAS — same pattern
  nas01:
    default:
      host: amgul.example.com
      port: 22
      user: root
      key: ~/.ssh/id_homelab.pem
      jump: amgul.example.com
    home-wg:
      host: 192.168.77.40
      user: root
      key: ~/.ssh/id_homelab.pem
    home-lan:
      host: 192.168.77.40
      user: root
      key: ~/.ssh/id_homelab.pem

  # k3s node — jump through gateway always except on LAN/WG
  ferret:
    default:
      host: amgul.example.com
      user: root
      key: ~/.ssh/id_homelab.pem
      jump: amgul.example.com
    home-wg:
      host: 192.168.77.85
      user: root
      key: ~/.ssh/id_homelab.pem
    home-lan:
      host: 192.168.77.85
      user: root
      key: ~/.ssh/id_homelab.pem

  # Router on a non-standard port
  opnsense:
    default:
      host: amgul.example.com
      port: 22
      user: admin
      key: ~/.ssh/id_homelab_router.pem
      jump: amgul.example.com
    home-wg:
      host: 192.168.77.1
      port: 22022
      user: admin
      key: ~/.ssh/id_homelab_router.pem
    home-lan:
      host: 192.168.77.1
      port: 22022
      user: admin
      key: ~/.ssh/id_homelab_router.pem
```

## Fallback mode

If your WireGuard tunnel goes down mid-session and you want sshroute to automatically try the public jump path, use `--fallback`:

```sh
sshroute connect eagle --fallback
```

This tries profiles in priority order, retrying only on SSH connection failures (exit 255). Auth failures or remote command errors stop immediately.

## Resolving what would be used

Useful when debugging detection:

```sh
# What will be used right now?
sshroute resolve eagle

# What would be used if I were on home-lan?
sshroute resolve eagle --network home-lan

# Get it as JSON for scripting
sshroute resolve nas01 --output json
```

## Copying files

Using `sshroute copy` instead of `scp` gives you the same network-aware resolution:

```sh
# Upload a file to the NAS (resolves to the right IP/key automatically)
sshroute copy nas01 ./backup.tar.gz nas01:/mnt/tank/backups/

# Download logs from the workstation
sshroute copy eagle eagle:/var/log/syslog ./eagle-syslog.txt
```

## Shadow mode for rsync and git

Set up shadow mode so every tool that calls `ssh` (rsync, git, etc.) goes through sshroute automatically:

```sh
mkdir -p ~/.local/bin
ln -s $(which sshroute) ~/.local/bin/ssh
# Ensure ~/.local/bin is early in PATH
```

Then `rsync -avz eagle:/home/admin/data ./` uses the correct key and IP without any extra flags.
