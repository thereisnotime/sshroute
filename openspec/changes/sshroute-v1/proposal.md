# Proposal: sshroute v1

## Problem

SSH connections to the same logical host often require different parameters depending on which
network you're on. On VPN1 you reach it at 10.8.0.50:2222 via a jump host; directly connected
you use a public IP on port 22; in the office you use a LAN address. Today this is handled by
maintaining multiple Host entries in ~/.ssh/config or manually switching parameters — both are
error-prone and tedious.

## Proposed Solution

`sshroute` is a Go CLI that sits between you and ssh. It:

1. Reads a YAML config that maps logical host aliases to per-network SSH parameters
   (host, port, user, key, jump host).
2. Detects the active network by running lightweight checks (routing table, interface state,
   ping, or custom exec commands).
3. Resolves the correct parameter set and execs the real `/usr/bin/ssh` with those params.

It works in two modes:
- **Direct**: `sshroute connect myserver` — explicit invocation with full CLI
- **Shadow**: installed as `~/.local/bin/ssh`, intercepts all SSH calls transparently;
  unknown hosts pass through to `/usr/bin/ssh` unmodified.

## Why Build This?

- No existing tool does per-VPN SSH routing transparently
- `Match exec` in ~/.ssh/config can't override `Hostname` and is unwieldy for many hosts
- A purpose-built tool can be versioned, tested, and distributed properly
