# Proposal: SSH binary resolution

## Problem

On Android/Termux the SSH binary is not at `/usr/bin/ssh` — it lives under the Termux prefix (`/data/data/com.termux/files/usr/bin/ssh`). The original `const RealSSH = "/usr/bin/ssh"` made this hardcoded and non-overridable, causing sshroute to fail immediately on Termux with "exec /usr/bin/ssh: no such file or directory".

The same problem affects any non-standard Linux installation (Nix, Homebrew on Linux, custom OpenSSH builds).

## Proposed Solution

Replace the hardcoded constant with a runtime resolver that checks, in order:

1. `SSHROUTE_SSH` environment variable — for one-off overrides
2. `ssh_binary` field in the config file — for persistent per-machine config
3. `exec.LookPath("ssh")` — automatic detection from `$PATH`, skipping itself to prevent shadow-mode recursion
4. `/usr/bin/ssh` — last-resort fallback

Also ship Android/arm64 binaries via GoReleaser so Termux users can install without needing to build from source.
