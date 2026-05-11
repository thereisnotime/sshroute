# Shadow mode

Shadow mode makes sshroute intercept every `ssh` invocation system-wide — from your terminal, from `git`, from `rsync`, from `scp` — without changing anything at the call site.

## How it works

sshroute is installed as `ssh` somewhere earlier in `$PATH` than the real `/usr/bin/ssh`. When called as `ssh`, it checks whether the target matches any configured alias. If it does, it resolves the right parameters and exec-replaces itself with the real SSH binary. If the target is not in your config, it passes the call through to `/usr/bin/ssh` unchanged.

```
git push origin main
  → git calls: ssh git@github.com ...
  → sshroute: "github.com" not in config → passthrough
  → /usr/bin/ssh git@github.com ... (unchanged)

rsync -avz eagle:/data ./local/
  → rsync calls: ssh -l admin eagle ...
  → sshroute: "eagle" is in config → resolve
  → /usr/bin/ssh -p 22 -i ~/.ssh/id_homelab.pem -l admin 192.168.77.91 ...
```

sshroute detects when it is being called as its own binary to prevent infinite recursion.

## Installation

```sh
mkdir -p ~/.local/bin
ln -s $(which sshroute) ~/.local/bin/ssh
```

Add `~/.local/bin` to the front of `PATH` in `~/.bashrc` or `~/.zshrc`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Reload your shell or open a new terminal. Verify:

```sh
which ssh          # should print ~/.local/bin/ssh
ssh --version      # prints the real OpenSSH version (passthrough)
```

## git over SSH

Once shadow mode is active, `git push`, `git pull`, and `git clone` for any remote that uses SSH are transparently routed. No changes to `.gitconfig` or remote URLs.

If you have an internal GitLab/Gitea instance:

```yaml
hosts:
  gitlab-internal:
    default:
      host: gitlab.example.com
      port: 22
      user: git
      key: ~/.ssh/id_gitlab
    corp-vpn:
      host: 10.10.0.50
      port: 22
      user: git
      key: ~/.ssh/id_gitlab
```

`git clone git@gitlab-internal:team/repo.git` uses the VPN path when connected, public path otherwise.

For github.com and other external hosts, git calls pass straight through — sshroute never touches them.

## rsync

```sh
# These work without any flags — sshroute intercepts the ssh call rsync makes
rsync -avz eagle:/var/log/app/ ./logs/
rsync -avz ./release.tar.gz nas01:/mnt/tank/releases/
```

If you need rsync to use a specific SSH option not covered by the config, pass it via `-e`:

```sh
rsync -avz -e "ssh -o StrictHostKeyChecking=no" eagle:/data/ ./
```

This still goes through shadow-mode sshroute; the extra flag is appended.

## scp

`scp` calls the SSH binary independently. Shadow mode covers it automatically:

```sh
scp eagle:/etc/hosts ./hosts-eagle
scp ./deploy.sh ferret:/opt/scripts/
```

Alternatively, use `sshroute copy` directly — it handles the same resolution without requiring shadow mode:

```sh
sshroute copy eagle eagle:/etc/hosts ./hosts-eagle
sshroute copy ferret ./deploy.sh ferret:/opt/scripts/
```

## Ansible

Ansible uses SSH under the hood. With shadow mode, host aliases in your sshroute config are resolvable by Ansible without custom inventory plugins or SSH args:

```ini
# inventory.ini
[k3s_nodes]
ferret
fossa
genet

[routers]
opnsense
```

```sh
ansible -i inventory.ini k3s_nodes -m ping
```

Each node name is resolved by sshroute before the real SSH connection is made.

If Ansible is configured with `ansible_ssh_executable` or `ansible_ssh_extra_args`, those take precedence — shadow mode may not apply. In that case, point Ansible at sshroute explicitly:

```ini
[defaults]
ssh_executable = ~/.local/bin/ssh
```

## Custom SSH binary path

sshroute looks for the real SSH binary by checking:

1. `SSHROUTE_SSH` environment variable
2. `ssh_binary` field in the config file
3. `ssh` from `$PATH`, skipping itself
4. `/usr/bin/ssh` as a last-resort fallback

On systems where `ssh` is not at `/usr/bin/ssh` (Termux, Nix, Homebrew), the auto-detection via `$PATH` usually works. To pin it explicitly:

```yaml
# ~/.config/sshroute/config.yaml
ssh_binary: /opt/homebrew/bin/ssh
```

Or at runtime:

```sh
SSHROUTE_SSH=/opt/homebrew/bin/ssh sshroute connect myserver
```

## Disabling shadow mode for one command

Prefix with the full path to bypass sshroute:

```sh
/usr/bin/ssh user@somehost
```

Or unset PATH manipulation temporarily:

```sh
env PATH=/usr/bin:/bin ssh user@somehost
```
