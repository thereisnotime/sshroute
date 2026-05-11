# Shell completion

sshroute provides dynamic shell completion for all subcommands and flags. Host aliases are completed live from your config — no static list to maintain.

## What gets completed

- Subcommand names (`connect`, `resolve`, `copy`, `remove`, `add`, etc.)
- Flags and their values
- Host aliases for `connect`, `remove`, `resolve`, and `copy` — pulled directly from the active config at completion time

## Bash

Add to `~/.bashrc`:

```sh
eval "$(sshroute completion bash)"
```

Or generate once and source from a file (faster shell startup):

```sh
sshroute completion bash > ~/.local/share/bash-completion/completions/sshroute
```

If you use the system bash-completion package, the file location may differ:

```sh
sshroute completion bash | sudo tee /etc/bash_completion.d/sshroute > /dev/null
```

Reload your shell or run `source ~/.bashrc`.

## Zsh

Add to `~/.zshrc`:

```sh
eval "$(sshroute completion zsh)"
```

If you see `command not found: compdef`, enable completions first:

```sh
autoload -Uz compinit
compinit
eval "$(sshroute completion zsh)"
```

Generate to a file for faster startup:

```sh
sshroute completion zsh > "${fpath[1]}/_sshroute"
```

## Fish

```sh
sshroute completion fish | source
```

To persist across sessions:

```sh
sshroute completion fish > ~/.config/fish/completions/sshroute.fish
```

## PowerShell

```powershell
sshroute completion powershell | Out-String | Invoke-Expression
```

## Verifying alias completion

After setting up completion, pressing Tab after a subcommand that accepts an alias should list your configured hosts:

```
$ sshroute connect <Tab>
amgul     eagle     edora     ferret    fossa     genet     nas01
```

```
$ sshroute resolve <Tab>
amgul     eagle     edora     ferret    fossa     genet     nas01
```

Completion is dynamic — adding or removing a host with `sshroute add` / `sshroute remove` is reflected immediately without regenerating anything.

## Using a non-default config

If you use `--config` to point at a different config file, completion respects it:

```sh
SSHROUTE_CONFIG=~/work/sshroute.yaml sshroute connect <Tab>
```

Aliases from `~/work/sshroute.yaml` are listed instead of the default config.
