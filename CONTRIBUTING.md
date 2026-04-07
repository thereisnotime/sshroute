# Contributing to sshroute

Thanks for taking the time to contribute. Here is everything you need to get started.

## Getting started

```sh
git clone git@github.com:thereisnotime/sshroute.git
cd sshroute
go mod download
just build   # outputs bin/sshroute
just test    # run tests with race detector
```

Requirements: [Go 1.22+](https://go.dev/dl/), [just](https://just.systems/).

## Making changes

1. Fork the repo and create a branch from `main`.
2. Make your change. If it adds behaviour, add or update tests.
3. Run `just test` and confirm everything passes.
4. Run `go vet ./...` — no new warnings.
5. Open a pull request against `main`. Keep the title short and descriptive.

PRs require at least one approving review and all CI checks to pass before merging.

## Running the full CI suite locally

```sh
just test                                                    # unit tests + race detector
go install github.com/securego/gosec/v2/cmd/gosec@v2.25.0
gosec ./...                                                  # SAST
go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
govulncheck ./...                                            # SCA
```

## Project layout

```
cmd/            Cobra commands — flags and wiring only, no business logic
internal/
  config/       Config struct, YAML loader/saver, validator
  network/      Network detection: route, interface, ping, exec checks
  ssh/          Arg parser, SSH parameter resolver, executor
  output/       Table/JSON/YAML formatters
  version/      Build-time version string
main.go         Entry point — calls cmd.Execute()
examples/       Ready-to-use config examples
```

## Tests

Tests live next to the code they cover (`*_test.go`). The `internal/ssh` package also has a fuzz target (`fuzz_test.go`) that can be run with:

```sh
go test -fuzz=FuzzParseArgs ./internal/ssh/
```

Aim to keep overall coverage above 60%. Check with:

```sh
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
```

## Reporting issues

Bug reports and feature requests are welcome via [GitHub Issues](https://github.com/thereisnotime/sshroute/issues). For security vulnerabilities, please follow the process in [SECURITY.md](SECURITY.md).

## Coding standards

All contributions must follow the official Go coding standards:

- **[Effective Go](https://go.dev/doc/effective_go)** — the primary style reference for Go code.
- **[Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)** — common mistakes and idioms reviewed in Go PRs.
- **[Go Test Comments](https://github.com/golang/go/wiki/TestComments)** — conventions for writing good Go tests.

In addition:

- Format all code with `gofmt` (or `goimports`) before committing.
- Keep `cmd/` thin — business logic belongs in `internal/`.
- No magic numbers or unexplained constants — add a comment.
- Error messages are lowercase and do not end with punctuation (e.g. `"config not found"`, not `"Config not found."`).
- No `Co-Authored-By` trailers in commits.

## License

By contributing you agree that your work will be released under the [Apache 2.0 License](LICENSE).
