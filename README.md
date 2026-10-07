# Workspace
A Polylith monorepo in Go. Code lives as small bricks (components and bases) that combine into deployable projects, and the repo keeps its own context so anyone, human or AI agent, can understand why things are the way they are.

## Getting started
You need [Go](https://go.dev/dl/) 1.27 or newer and git. Then, from this directory:

    git init        # if this is not already a git repo
    make hooks      # turn the pre-commit checks on for this clone

Optional but recommended: install [golangci-lint](https://golangci-lint.run/docs/welcome/install/) so the hook can lint locally. CI runs it either way.

## Run it
The example project builds the `api` base, which delegates to the `greeting` component:

    go run ./projects/hello          # -> Hello, world!
    go run ./projects/hello Ada      # -> Hello, Ada!

## Common commands

    go test ./...                      # run all tests
    make check                         # build, vet, brick boundaries, decision log
    make lint                          # golangci-lint
    make build                         # build every project into bin/
    go doc -all ./components/greeting  # a brick's public interface, with docs
    go run ./tools/poly check          # validate the workspace (boundaries, layout)
    go run ./tools/poly deps           # what each brick uses, what each project pulls in
    go run ./tools/poly diff           # bricks and projects changed since the last stable-* tag
    go run ./tools/poly interface      # what changed in a brick's public interface

## Create new bricks
There is no scaffolding command; a brick is a directory with a Go package in it:

    mkdir components/<name>            # business logic, shared
    mkdir bases/<name>                 # a thin entry point
    mkdir projects/<name>              # a deployable: package main that calls a base

A brick's public interface is the exported identifiers of its root package; its implementation is everything unexported, and `internal/` subpackages once it grows. Keep business logic in components, keep bases thin, and keep projects to `main()` plus deploy files.

## Build and deploy
A `project` under `projects/` is the deployable unit: `go build -o bin/<name> ./projects/<name>` produces a static binary containing only the bricks that project imports. Put the Dockerfile or deploy scripts next to the project's `main.go`.

## Where to read more
- New contributor: `CONTRIBUTING.md` (setup and the day-to-day loop in detail).
- AI agent, or the full process and where every kind of context lives: `AGENTS.md`.
- The decisions behind this setup: `docs/adr/`.
- The in-repo tooling: `tools/poly/README.md`.

## Layout

    bases/         thin entry points (one per app or service)
    components/    business logic, shared across projects
    projects/      deployable artifacts (package main plus deploy files, no business logic)
    development/   scratch space for throwaway programs
    docs/adr/      the decision log (and a generated per-brick index)
    tools/poly/    the in-repo Polylith tool and the automated checks

## Example bricks
`components/greeting`, `bases/api` and `projects/hello` are examples that demonstrate the interface/implementation split and the base-to-project wiring. Delete them once you have your own.

## Namespace
The module path in `go.mod` (`github.com/myorg/workspace`) is the namespace. To use your own, change it there and in every import (`grep -rl github.com/myorg/workspace . | xargs sed -i 's#github.com/myorg/workspace#<your path>#g'`).
