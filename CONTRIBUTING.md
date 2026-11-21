# Contributing to go-intl

Bug reports, translation corrections, documentation improvements, and patches are welcome. For a bug, include the locale, the smallest message pattern and arguments that reproduce it, and the output you expected. Use the [issue templates](https://github.com/Waterkyuu/go-intl/issues/new/choose) for bugs and feature requests. For a suspected vulnerability, follow the [security reporting instructions](SECURITY.md).

## Make a change

Fork the repository, create a branch from `main`, and keep the pull request focused on one change. The core package and the CLI share the root Go module. Runnable client examples have a separate module in `examples/clients/`, and the documentation site lives in `website/`.

Follow the existing package conventions. Place regression tests beside the affected Go implementation. When changing a public API or message syntax, update the examples and both READMEs where relevant. The repository's [Go development guidelines](AGENTS.md) cover package design, declaration comments, formatting, and tests.

## Check your work

For the core library and CLI, use Go 1.25 or later. Format changed Go files with `gofmt`, then run these commands from the repository root.

```sh
go vet ./...
go test -race ./...
go build ./...
```

To run the Go linter used by CI, install golangci-lint and run `task lint` or `golangci-lint run ./...`. For changes to a Bubble Tea, MyGo, or NSGo example, also run `go test -race ./...` from `examples/clients/` with Go 1.27.3 or later. For website changes, follow the [website development instructions](website/README.md).

## Report performance changes

For changes to lookup or message rendering, measure the affected workload before and after the patch on the same machine. For example, run the following command on each revision.

```sh
go test . -run '^$' -bench '^BenchmarkTranslate$' -benchmem -count=5
```

Include the Go version, CPU, and relevant results in the pull request. Identify whether the benchmark reuses a bound translator and argument map. The [benchmark report](PERFORMANCE.md) describes the workloads and reproduction commands.

## Open a pull request

Describe the observable change and list the checks you ran. Link a related issue if one exists. Use an English commit subject such as `fix(catalog): preserve empty fallback messages`. Project discussions and reviews follow the [code of conduct](CODE_OF_CONDUCT.md).
