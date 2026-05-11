# Corporate / multi-environment setup

A realistic corporate setup: three environments (dev, staging, prod), each behind its own bastion, with multiple connection paths depending on whether you're in the office, on the corp VPN, or working remotely.

## Scenario

```
Environments
  dev       bastion-dev.corp.internal     10.10.1.1
  staging   bastion-stg.corp.internal     10.10.2.1
  prod      bastion-prod.corp.internal    10.10.3.1  (stricter key required)

Networks
  office    on-premises LAN, 192.168.10.0/24
  corp-vpn  WireGuard split-tunnel, wg-corp interface
  home      no VPN — must use public bastions
```

## Network definitions

```yaml
networks:
  # Office LAN — ping the internal gateway
  office:
    priority: 10
    checks:
      - type: ping
        host: 192.168.10.1
        timeout: 300ms

  # Corp WireGuard VPN — interface must be up and internal DNS resolvable
  corp-vpn:
    priority: 20
    checks:
      - type: interface
        match: wg-corp
      - type: exec
        # DNS check: only resolves when corp DNS is active
        command: "dig +short +time=1 bastion-dev.corp.internal | grep -qE '^[0-9]'"
```

`home` is the implicit fallback (`default`) — no checks, always matches when nothing else does.

## Host definitions

```yaml
hosts:
  # Dev application server
  app-dev:
    default:                              # remote / home: jump via public bastion
      host: app-dev.corp.internal
      port: 22
      user: deploy
      key: ~/.ssh/id_corp_dev
      jump: bastion-dev.example.com
    office:
      host: 10.10.1.50
      port: 22
      user: deploy
      key: ~/.ssh/id_corp_dev
    corp-vpn:
      host: 10.10.1.50
      port: 22
      user: deploy
      key: ~/.ssh/id_corp_dev
      jump: bastion-dev.corp.internal    # internal bastion, no public hop

  # Staging application server
  app-stg:
    default:
      host: app-stg.corp.internal
      port: 22
      user: deploy
      key: ~/.ssh/id_corp_stg
      jump: bastion-stg.example.com
    office:
      host: 10.10.2.50
      port: 22
      user: deploy
      key: ~/.ssh/id_corp_stg
    corp-vpn:
      host: 10.10.2.50
      port: 22
      user: deploy
      key: ~/.ssh/id_corp_stg
      jump: bastion-stg.corp.internal

  # Prod — stricter: dedicated prod key, access only via approved paths
  app-prod:
    default:
      host: app-prod.corp.internal
      port: 22
      user: deploy
      key: ~/.ssh/id_corp_prod            # separate prod key
      jump: bastion-prod.example.com
    corp-vpn:
      host: 10.10.3.50
      port: 22
      user: deploy
      key: ~/.ssh/id_corp_prod
      jump: bastion-prod.corp.internal
    # No office override — prod access always requires the VPN key path

  # Prod database — reachable only through the application server as jump
  db-prod:
    default:
      host: db-prod.corp.internal
      port: 22
      user: postgres
      key: ~/.ssh/id_corp_prod
      jump: bastion-prod.example.com
    corp-vpn:
      host: 10.10.3.100
      port: 22
      user: postgres
      key: ~/.ssh/id_corp_prod
      jump: bastion-prod.corp.internal
```

## Using exec checks for DNS-based detection

The `exec` check is useful when interface presence alone isn't reliable (e.g. a VPN that stays up across networks). Testing an internal DNS record that only resolves on corp infrastructure is more accurate:

```yaml
networks:
  corp-vpn:
    priority: 20
    checks:
      - type: interface
        match: wg-corp
      - type: exec
        command: "dig +short +time=1 vault.corp.internal | grep -qE '^[0-9]'"
```

Both must pass — the interface check rules out any local wg-corp interface that isn't the right one.

## Sharing config across a team

Keep the config in a team repo. Each developer checks it out and symlinks or copies it:

```sh
git clone git@github.com:your-org/infra-config.git ~/infra
ln -sf ~/infra/sshroute/config.yaml ~/.config/sshroute/config.yaml
```

Keys stay local — the config references them by path (`~/.ssh/id_corp_dev`), which each developer provisions separately via your secrets manager or onboarding script. Network detection is automatic; nobody has to know IPs or think about which bastion to use.

## Verifying resolution before connecting

Before connecting to a sensitive host, check what sshroute would actually use:

```sh
sshroute resolve app-prod
sshroute resolve app-prod --network corp-vpn
sshroute resolve db-prod --output json | jq '.[] | {host, jump, key}'
```

Use `--dry-run` to see the exact `ssh` command that would run:

```sh
sshroute connect db-prod --dry-run
```

## Fallback for resilience

When corp VPN is flaky, `--fallback` lets sshroute try the next profile automatically on connection failure:

```sh
sshroute connect app-stg --fallback
```

Profiles are tried in priority order; non-connection failures (auth errors, remote commands) stop immediately and don't retry.
