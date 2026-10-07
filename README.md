# Workspace
A Polylith monorepo in Go. Code lives as small bricks (components and bases) that projects assemble into deployables, and the repository keeps its own context, so a human or a coding agent can learn what any part is for and why it is the way it is in five cheap reads.

If you are a coding agent: read `AGENTS.md` first. It is the operating manual.

## Getting started
You need [Go](https://go.dev/dl/) 1.27 or newer, git, and [golangci-lint](https://golangci-lint.run/docs/welcome/install/). Then, from this directory:

    make hooks      # turn the pre-commit checks on for this clone (once)
    make all        # check, test, lint: should be green

## Run the example
<!-- examples:start (delete this block when you delete the example bricks) -->
The example project wires the `greeting` component into the `api` base:

    go run ./projects/hello          # -> Hello, world!
    go run ./projects/hello Ada      # -> Hello, Ada!
    go doc -all ./components/greeting   # the component's public interface

`components/greeting`, `bases/api` and `projects/hello` exist to show the shapes: a component with an Example, a base that receives its dependencies, a project that constructs and wires. `docs/spec/greeting.md` is the matching example spec domain whose two rules the greeting tests cite. `make adopt` deletes all four.
<!-- examples:end -->

## Common commands

    make check                         # layout and boundaries, vet (incl. the Polylith analyzer), decision log, tidy go.mod
    make test                          # go test -race -shuffle=on ./...
    make lint                          # golangci-lint
    make build                         # every project into bin/
    go run ./tools/poly check          # brick boundaries and layout
    go run ./tools/poly deps           # what each brick uses; what each project ships
    go run ./tools/poly diff           # bricks and projects changed since the last stable-* tag
    go run ./tools/poly interface      # exported API changes in the staged tree
    go run ./tools/poly spec status    # what works (spec rules with a citing test) and what is queued
    go run ./tools/poly adr new "..."  # start a decision record (when you ask for one)
    make adopt MODULE=...              # once, on a fresh clone: make the template yours

## Create new bricks
A brick is a directory with a Go package in it; there is no scaffolding command.

    mkdir -p components/<name>     # business logic: package <name>, exported API, _test.go beside it
    mkdir -p bases/<name>          # a thin entry point: parses one protocol, delegates to components
    mkdir -p projects/<name>       # a deployable: one main.go that wires components into a base

Rules in one line each (full version in `AGENTS.md`): components import only components; bases import components, never bases; a project is one `main.go` that imports at least one base and may import components to construct them; every brick is used through its root package; `internal/` is private to its brick.

## Build and deploy
`go build -o bin/<name> ./projects/<name>` (or `make build`) produces a static binary containing only the bricks that project imports. Put the Dockerfile or deploy scripts next to the project's `main.go`.

## Where to read more
- Contributor setup and the day-to-day loop: `CONTRIBUTING.md`.
- The full process, the context layers, and the rules with their enforcement: `AGENTS.md`.
- The decisions behind the setup: `docs/adr/`.
- The in-repo tool: `tools/poly/README.md`.

## Layout

    components/    business logic, shared across projects
    bases/         thin entry points (one per kind of outside world)
    projects/      deployables: one main.go each, plus deploy files
    development/   scratch programs, outside ./... (go.mod ignore)
    docs/spec/     the behavior contract: one file per product domain, numbered rules, OPEN: questions
    docs/queue.yaml the build queue: which rules to implement next, in order (agent-maintained)
    docs/adr/      the decision log and its generated per-brick index
    tools/poly/    the in-repo Polylith tool and checks

## Namespace
The module path in `go.mod` (`github.com/myorg/workspace`) is the namespace. `make adopt MODULE=<your path>` renames it everywhere (`CONTRIBUTING.md` > Setting up the repository).
