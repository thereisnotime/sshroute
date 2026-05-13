# Multi-zone homelab with roaming mobile devices

A real-world scenario that shows where sshroute pays off most: multiple physical
networks, a WireGuard gateway, and mobile devices that roam between them. The
equivalent plain SSH config would be ~600 lines of static aliases — and would
still require you to manually pick the right one every time.

---

## Topology

```
Internet
  └─ gateway.example.com:2222 (public SSH port)
       └─ gateway (zone-b, WireGuard server 172.16.0.1)
            ├─ zone-a LAN (192.168.10.x)  — servers, desktops
            │    ├─ workstation   192.168.10.50
            │    ├─ server1       192.168.10.60
            │    ├─ phone1 *      192.168.10.101   port 8022
            │    └─ phone2 *      192.168.10.102   port 8022
            └─ zone-b LAN (10.20.30.x)   — behind gateway
                 ├─ nas           10.20.30.40
                 ├─ phone1 *      10.20.30.101     port 8022 (when on zone-b)
                 └─ phone2 *      10.20.30.102     port 8022 (when on zone-b)

* Mobile devices get a different IP depending on which network they're on.
```

The gateway is the only host with a public address. Everything else is private.
Mobile devices make this interesting: their IP depends on which network they
joined, and you won't always know which one that is when you try to connect.

---

## Network detection

Four networks, checked in priority order. The first one whose check passes
selects the profile used for that connection.

| Priority | Name | Check | Meaning |
|---|---|---|---|
| 5 | `zone-a` | `nc 192.168.10.40 22` | On zone-a LAN directly |
| 10 | `zone-b` | `nc 10.20.30.40 22` | On zone-b LAN directly |
| 15 | `wireguard` | `nc 172.16.0.1 22` | WireGuard tunnel is up |
| 25 | `internet` | `nc gateway.example.com 2222` | On internet, gateway reachable |

Lower priority number = checked first = preferred. Direct LAN connections are
always fastest, so they get the lowest numbers. WireGuard adds one hop, and
the public gateway adds latency and goes through NAT — so those come last.

The `internet` check (`nc gateway.example.com 2222`) has a 5-second timeout.
It only matters when all three local checks have already failed, so the extra
wait is acceptable.

---

## Profile selection and fallback

Without `--fallback`, sshroute uses the detected network's profile (or
`default` if no network matches).

With `--fallback` (recommended for all connection wrappers), sshroute tries
profiles in priority order on SSH connection failure (`exit 255`). This means:

1. Try the direct LAN path — fails if you're not on that LAN.
2. Try WireGuard — fails if the tunnel is down.
3. Try the internet path — works if gateway is reachable.
4. Try `default` — last resort.

You always get the best available path without thinking about it.

---

## The profile inheritance gotcha

sshroute merges a named profile **on top of** the `default` profile. Only
non-empty fields in the override replace the default. This matters for `jump`:

```yaml
# WRONG — zone-a profile silently inherits jump: gateway from default
workstation:
    default:
        host: 192.168.10.50
        jump: gateway          # ← set in default
    zone-a:
        host: 192.168.10.50   # ← jump: gateway is inherited! 3 hops from LAN.
```

```yaml
# CORRECT — default has no jump; named profiles set it explicitly
workstation:
    default:
        host: 192.168.10.50   # no jump — direct last-resort attempt
    zone-a:
        host: 192.168.10.50   # inherits nothing, direct as intended
    zone-b:
        host: 192.168.10.50
        jump: gateway          # explicit
    wireguard:
        host: 192.168.10.50
        jump: gateway          # explicit
    internet:
        host: 192.168.10.50
        jump: gateway          # explicit
```

**Rule:** if any profile for a host needs a direct connection (no jump), keep
`default` jump-free and set `jump` explicitly in every profile that requires it.

---

## Mobile devices

Phones and tablets are the hardest case because their IP changes with their
network. The strategy is to keep a profile for each network, using the
appropriate IP for that network:

```yaml
phone1:
    default:
        host: 192.168.10.101   # zone-a IP, last resort direct
        port: 8022
        ...
    zone-a:
        host: 192.168.10.101   # direct on zone-a
        port: 8022
        ...
    zone-b:
        host: 10.20.30.101     # direct on zone-b
        port: 8022
        ...
    wireguard:
        host: 10.20.30.101     # zone-b IP, via WG hop
        port: 8022
        jump: gateway
        ...
    internet:
        host: 10.20.30.101     # zone-b IP, via gateway public
        port: 8022
        jump: gateway
        ...
```

With `--fallback`, this covers every case:

| You are on | Phone is on | Profile used |
|---|---|---|
| zone-a | zone-a | `zone-a` — direct |
| zone-b | zone-b | `zone-b` — direct |
| WireGuard | zone-b | `wireguard` — one WG hop |
| internet | zone-b | `internet` — one SSH hop via gateway |
| zone-a | zone-b | `zone-a` fails, `zone-b` fails, `wireguard` fails, `internet` succeeds |
| internet | zone-a | `internet` fails (zone-b IP unreachable), `default` attempted |

The one gap: if you're on the internet and the phone is on zone-a, no profile
covers it cleanly (zone-b IP won't reach a zone-a device). This requires
sshroute to support explicitly clearing an inherited field, or a secondary
zone-a path via a stable zone-a jump host. In practice, phones connected to
zone-a are usually reachable via direct zone-a or WireGuard anyway.

---

## Recursive jump resolution

When `jump` references another sshroute host alias, the jump host is resolved
using the **same active network**. This means chains work automatically:

```yaml
# internet + phone on zone-a via a stable zone-a relay host
phone1:
    internet-z09:
        host: 192.168.10.101
        port: 8022
        jump: relay            # relay.internet resolves to gateway public

relay:
    default:
        host: 192.168.10.200   # direct last resort
    internet:
        host: 192.168.10.200
        jump: gateway          # relay → gateway → relay → phone: 2 hops
```

SSH resolves this to: `-J gateway.example.com:2222,192.168.10.200 192.168.10.101`

---

## Why not just use SSH config?

A plain `~/.ssh/config` has no network detection. You would need one `Host`
block per path per device, manually named:

```
Host workstation-lan
    HostName 192.168.10.50
    User admin
    IdentityFile ~/.ssh/id_homelab.pem

Host workstation-wg
    HostName 192.168.10.50
    User admin
    IdentityFile ~/.ssh/id_homelab.pem
    ProxyJump gateway-wg

Host workstation-internet
    HostName 192.168.10.50
    User admin
    IdentityFile ~/.ssh/id_homelab.pem
    ProxyJump gateway-internet

Host gateway-wg
    HostName 172.16.0.1
    Port 22
    User admin
    IdentityFile ~/.ssh/id_homelab.pem

Host gateway-internet
    HostName gateway.example.com
    Port 2222
    User admin
    IdentityFile ~/.ssh/id_homelab.pem

# ... repeat for every host and every path
```

For the topology above (9 hosts × 4–5 paths × ~6 lines each) that's roughly
**550–650 lines** — and you still have to type `ssh workstation-wg` vs
`ssh workstation-internet` based on your current situation. sshroute detects
it for you and the command is always just `sshroute connect workstation`.

---

## Full example

See [`examples/multi-zone-roaming.yaml`](../examples/multi-zone-roaming.yaml)
for a complete, ready-to-adapt config for this topology.
