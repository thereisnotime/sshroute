# Scripting and automation

sshroute's `resolve` and `copy` commands are designed to integrate cleanly into shell scripts, CI pipelines, and other automation workflows.

## Extracting connection parameters

`sshroute resolve` prints the resolved SSH parameters for a host on the current (or specified) network. Use `--output json` to make it machine-readable:

```sh
sshroute resolve eagle --output json
```

```json
[
  {
    "alias": "eagle",
    "network": "home-wg",
    "host": "192.168.77.91",
    "port": 22,
    "user": "admin",
    "key": "/home/user/.ssh/id_homelab.pem",
    "jump": "",
    "command": "/usr/bin/ssh -p 22 -i /home/user/.ssh/id_homelab.pem -l admin 192.168.77.91"
  }
]
```

Extract individual fields with `jq`:

```sh
HOST=$(sshroute resolve eagle --output json | jq -r '.[0].host')
PORT=$(sshroute resolve eagle --output json | jq -r '.[0].port')
USER=$(sshroute resolve eagle --output json | jq -r '.[0].user')
KEY=$(sshroute resolve eagle --output json | jq -r '.[0].key')
```

Or capture everything at once:

```sh
eval "$(sshroute resolve eagle --output json | jq -r '.[0] |
  "HOST=\(.host)\nPORT=\(.port)\nUSER=\(.user)\nKEY=\(.key)"')"
```

## Running a command on a remote host

Use the resolved command directly:

```sh
CMD=$(sshroute resolve eagle --output json | jq -r '.[0].command')
$CMD "uptime && df -h"
```

Or let sshroute connect and pass extra arguments through:

```sh
sshroute connect eagle -- uptime
sshroute connect eagle -- "df -h /var"
```

## File sync script

A simple backup script that syncs a remote directory using `sshroute copy` (no need to know the current IP or key):

```sh
#!/bin/bash
set -euo pipefail

ALIAS=nas01
REMOTE_PATH="$ALIAS:/mnt/tank/backups"
LOCAL_PATH="$HOME/backups/$(date +%Y-%m-%d)"

mkdir -p "$LOCAL_PATH"
sshroute copy "$ALIAS" "$REMOTE_PATH/" "$LOCAL_PATH/"
echo "Synced to $LOCAL_PATH"
```

## Dry-run before executing

`--dry-run` prints the resolved command without running it — useful for auditing or logging:

```sh
sshroute connect eagle --dry-run
# → /usr/bin/ssh -p 22 -i ~/.ssh/id_homelab.pem -l admin 192.168.77.91

sshroute copy nas01 ./archive.tar.gz nas01:/mnt/tank/ --dry-run
# → [dry-run] /usr/bin/scp -P 22 -i ~/.ssh/id_homelab.pem ./archive.tar.gz root@192.168.77.40:/mnt/tank/
```

## CI / CD

In a pipeline, use `SSHROUTE_CONFIG` to point at a config baked into the repo:

```yaml
# GitHub Actions example
- name: Deploy to staging
  env:
    SSHROUTE_CONFIG: ./infra/sshroute-staging.yaml
    SSHROUTE_SSH: /usr/bin/ssh
  run: |
    sshroute connect app-stg -- ./scripts/deploy.sh
```

Use `--network` to force a specific profile rather than relying on auto-detection (CI runners rarely have the right interfaces):

```sh
sshroute resolve app-stg --network corp-vpn --output json | jq '.[0].command'
```

Or pass the command string directly to `eval`:

```sh
SSHCMD=$(sshroute resolve app-stg --network corp-vpn --output json | jq -r '.[0].command')
$SSHCMD "systemctl restart myapp"
```

## Using a different SCP binary

`sshroute copy` respects `SSHROUTE_SCP`:

```sh
SSHROUTE_SCP=/opt/homebrew/bin/scp sshroute copy nas01 ./file.txt nas01:/tmp/
```

## Checking the detected network in scripts

```sh
NETWORK=$(sshroute network)
if [ "$NETWORK" = "home-wg" ]; then
  echo "On WireGuard — using direct paths"
else
  echo "Falling back to public endpoints"
fi
```

## Looping over hosts

```sh
for alias in ferret fossa genet; do
  echo "=== $alias ==="
  sshroute connect "$alias" -- uptime
done
```

## Building SSH args from resolve output

Some tools (ansible, kubectl port-forward via SSH, etc.) need individual SSH flags rather than a full command string. Build them from the JSON output:

```sh
resolve() {
  sshroute resolve "$1" --output json | jq -r '.[0] |
    [
      if .port != 0 then "-p \(.port)" else empty end,
      if .key != "" then "-i \(.key)" else empty end,
      if .jump != "" then "-J \(.jump)" else empty end,
      "\(.user)@\(.host)"
    ] | join(" ")'
}

ansible -i "$(resolve eagle)," all -m ping
```
