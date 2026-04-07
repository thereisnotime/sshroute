# Tasks: sshroute v1

## Phase 0 — Scaffold (main agent)
- [x] Create GitHub repo thereisnotime/sshroute
- [x] Clone + set git identity + SSH remote
- [x] openspec init + new change sshroute-v1
- [ ] go.mod + directory skeleton
- [ ] Stub files with exported interfaces
- [ ] .gitignore + initial commit + push

## Phase 1 — Parallel Agents

### A1: internal/config
- [ ] config.go — Config, NetworkConfig, NetworkCheck, HostConfig, SSHParams structs
- [ ] loader.go — Load/Save YAML, XDG path, ~ expansion
- [ ] validator.go — Validate structure and values

### A2: internal/network
- [ ] detector.go — Orchestrate checks, return active network name
- [ ] route.go — ip route check
- [ ] iface.go — /sys/class/net operstate check
- [ ] ping.go — ICMP ping with timeout
- [ ] exec.go — arbitrary command check

### A3: internal/ssh
- [ ] router.go — Merge default + network override → SSHParams
- [ ] args.go — Parse SSH argv, extract host alias
- [ ] executor.go — Build argv + syscall.Exec /usr/bin/ssh

### A4: cmd/
- [ ] root.go — Root command, shadow mode detection, global flags
- [ ] connect.go — connect subcommand
- [ ] list.go — list subcommand
- [ ] add.go — add subcommand
- [ ] remove.go — remove subcommand
- [ ] network.go — network subcommand + list/test
- [ ] config.go — config subcommand + edit
- [ ] version.go — version subcommand

### A5: Infra + output
- [ ] internal/version/version.go
- [ ] internal/output/formatter.go + table.go + json.go + yaml.go
- [ ] justfile
- [ ] .goreleaser.yaml
- [ ] Dockerfile
- [ ] .github/workflows/ci.yaml
- [ ] .github/workflows/release.yaml
- [ ] README.md

## Phase 2 — Integration (main agent)
- [ ] main.go — wire cmd.Execute()
- [ ] go mod tidy
- [ ] Fix cross-package interface mismatches
- [ ] just build
- [ ] Smoke test
- [ ] Commit + push
- [ ] openspec archive sshroute-v1 --skip-specs -y
