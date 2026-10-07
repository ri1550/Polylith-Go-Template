---
status: accepted
date: YYYY-MM-DD
decision-makers: [you]
consulted: []
informed: []
affects:
  components: ["*"]
  bases: ["*"]
  projects: ["*"]
interface-impact: none
---

# ADR-0002: Adopt the Polylith architecture
## Context and problem statement
We are starting a new Go codebase that will likely grow to several deployable services or tools sharing a lot of code. We need a structure that makes sharing code easy, keeps clear boundaries between features, and lets us defer the choice of deployment shape (one binary, several services, functions) until we need it.

## Decision drivers
- Share code across multiple binaries without copy-paste or a sprawl of library modules.
- Clear separation between a feature's public interface and its implementation.
- Defer deployment decisions; focus on code and features first.
- Best Simple System for Now: avoid speculative infrastructure; one module, one build, grow structure only as needed.

## Considered options
- Polylith layout inside one Go module (components, bases, projects)
- The common Go layout (`cmd/`, `internal/`, `pkg/`) in one module
- Multiple repositories, one per service, with shared code as published modules
- A `go.work` multi-module monorepo without Polylith's brick model

## Decision outcome
Chosen option: Polylith, expressed in Go terms. Code lives as bricks: components (business logic, under `components/`) and bases (thin entry points, under `bases/`). Each brick is a Go package whose **root package is its public interface**: the exported identifiers are the contract, unexported identifiers are implementation, and private subpackages go under the brick's `internal/` so the compiler itself refuses outside imports. Projects under `projects/` are the deployable artifacts: a `package main` whose `main()` hands control to one or more bases, plus deployment infrastructure (Dockerfiles, deploy scripts). Projects hold no business logic. The whole workspace is one Go module; Go's linker includes in each project binary only the packages it imports, so one module costs nothing at deploy time.

Polylith over the alternatives because it gives code sharing and clear boundaries in one module and one build, separates interface from implementation by construction, and lets us defer deployment shape. The common `cmd/`+`internal/` layout has no interface/implementation convention at the feature level and no vocabulary for the entry-point versus logic split. Multiple repositories reintroduce the duplication and versioning problems we want to avoid. A `go.work` monorepo adds per-module versioning and tidy work for no gain while the code is one deployable family.

### Consequences
- Good: shared code with clear boundaries; interface/implementation separation is structural (exported names, `internal/`); deployment shape stays deferrable.
- Good: one module for the whole workspace suits test-driven work and gives agents full context with `go doc`, `go test ./...` and `go build ./...`.
- Bad: contributors must learn Polylith's vocabulary and the in-repo `poly` tool (ADR-0008); there is no official Polylith tooling for Go.
- Neutral: business logic must not live in `projects/`, only project wiring and infrastructure.

### Confirmation
`go run ./tools/poly check` runs in the pre-commit hook and in CI and enforces the brick boundaries: components must not import bases, bases must not import other bases, projects import only bases (replaced by ADR-0009: a project imports at least one base and may import components to construct them), bricks are consumed through their root package, and nothing imports `development/` or `tools/`. The compiler enforces `internal/`. A violation fails the build.

## More information
- ADR-0007 (the decision to enforce code quality with automated checks)
- ADR-0008 (the in-repo Polylith tooling)
- Polylith documentation (Clojure; concepts apply, tooling and examples do not): https://polylith.gitbook.io/polylith/
- Best Simple System for Now (BSSN): https://dannorth.net/blog/best-simple-system-for-now/
