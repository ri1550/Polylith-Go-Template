---
status: accepted
date: 2026-10-07
decision-makers: [template author]
consulted: []
informed: []
affects:
  components: []
  bases: []
  projects: ["*"]
interface-impact: none
---

# ADR-0004: Use a single Go module for the whole workspace
## Context and problem statement
Go code is organised in modules, and a monorepo can be one module, one module per deployable, or a `go.work` workspace of many modules. The choice shapes dependency management, how projects are built, and the developer and CI workflow. We need to pick one for the workspace.

## Decision drivers
- One build, one dependency graph and one `go test ./...` for the whole workspace, so developers and agents always see everything.
- Reproducible builds across projects and CI.
- Each project binary should carry only what it uses.
- BSSN: one module for the whole workspace, no per-project variation until something forces it.

## Considered options
- One module at the workspace root
- One module per project, bricks as separate modules, glued with `go.work`
- One module per project that vendors or `replace`s the bricks
- Bricks published as versioned modules from a separate repository

## Decision outcome
Chosen option: one module at the workspace root (`go.mod` names the module path, which is the workspace namespace). All bricks, projects, scratch code and tooling share it and the single `go.sum`.

One module over the alternatives because Go already gives us what the multi-module layouts are for: the linker includes in each binary only the packages it imports, so a project pays nothing for bricks it does not use, and `go build ./projects/<name>` produces a lean artifact without per-project packaging configuration. Multi-module layouts add version bookkeeping, `replace` directives and tidy-per-module friction while all the code is one deployable family. Published bricks reintroduce the cross-repository versioning problem Polylith avoids.

This decision is reversible: Polylith's layout does not depend on module boundaries, so splitting into modules later is contained to `go.mod` files, not the bricks.

### Consequences
- Good: one `go build ./...`, `go test ./...` and `go vet ./...`; one lockfile (`go.sum`); no per-project dependency lists to keep in sync.
- Bad: a third-party dependency added for one project appears in the shared `go.mod` and `go.sum`, even though it is only linked into the binaries that import it.
- Neutral: the in-repo tooling (`tools/poly`) and its YAML dependency live in the same module.

### Confirmation
`go build ./...` and `go test ./...` in CI cover every brick and project in one run. A green CI run is the confirmation.

## More information
- Go modules reference: https://go.dev/ref/mod
- Best Simple System for Now (BSSN): https://dannorth.net/blog/best-simple-system-for-now/
