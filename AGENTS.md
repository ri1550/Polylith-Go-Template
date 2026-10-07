# AGENTS.md
Read this file before touching any code in this repository. It tells you what the codebase is, where every kind of context lives, how to read a brick before editing it, what to write afterwards, and where to stop and ask. The rules here are enforced by the checks named next to them; everything else is convention that you carry.

Quick routes:
- Setting up a fresh clone: run `make hooks` (CONTRIBUTING.md > One-time setup).
- Creating a brick or a project: README.md > Create new bricks. There is no scaffolding command.
- Adopting this template for a new repository: CONTRIBUTING.md > Setting up the repository.
- Recording a decision: `go run ./tools/poly adr new "<title>"`.

## What this is
A Polylith workspace written in Go. Polylith is a monorepo architecture where code lives as small, composable bricks that projects assemble into deployables.

Layout:

    components/<name>/     a brick: business logic, shared across projects (a Go package)
    bases/<name>/          a brick: a thin entry point for one kind of outside world (HTTP, CLI, queue)
    projects/<name>/       a deployable: one main.go that wires components into a base, plus deploy files
    development/           scratch: throwaway programs, outside ./... (go run ./development/<name>)
    docs/adr/              the decision log, plus a generated per-brick index
    tools/poly/            the in-repo Polylith tool; the checks that keep the process honest
    go.mod                 the one module; its module path is the workspace namespace

Tests live next to the code they test, as `_test.go` files inside the brick.

Vocabulary, used throughout:
- **Brick**: a component or a base. A directory holding a Go package.
- **Public interface** (also "surface"): every exported identifier in a brick's root package. There is no interface file; exported-ness is the contract. `go doc -all ./components/<name>` prints it.
- **Implementation**: everything unexported, plus the brick's `internal/` subpackages, which the compiler keeps private to the brick.
- **Composition root**: a project's `main()`, the one place that constructs components and hands them to a base.
- **ADR**: an Architecture Decision Record under `docs/adr/`: a decision, the alternatives, the reasons.

## Polylith rules
Each rule names what enforces it.
1. Components hold business logic and are shared. Components import only other components. Enforced by `poly check`.
2. Bases are thin entry points. A base imports components, never another base. Enforced by `poly check`. "Thin" is review's job: a base parses the protocol and delegates.
3. Projects are the composition root. A project's root package is `package main`, is exactly one Go file, and imports at least one base; it may import components in order to construct them and pass them to the base. Enforced by `poly check`. "No business logic" is review's job; the one-file limit keeps it visible.
4. Bricks are used through their root package only. Importing another brick's subpackage bypasses its interface. Enforced by `poly check`; for `internal/` the compiler enforces it too.
5. Nothing imports `projects/`, `development/` or `tools/`. Enforced by `poly check`.
6. Brick names are unique across components and bases. Enforced by `poly check`.
7. All Go code lives inside the layout above. Enforced by `poly check`.

Run the checks yourself at any time:

    go run ./tools/poly check       # rules 1 to 7
    go run ./tools/poly deps        # what each brick uses and is used by; what each project ships
    go run ./tools/poly diff        # bricks and projects changed since the last stable-* tag
    go run ./tools/poly interface   # exported API changes in the staged tree, and whether an ADR names them

## Guiding principle: Best Simple System for Now (BSSN)
Build the simplest thing that meets the need right now, written to an appropriate standard, with no speculative future-proofing (ADR-0003). Apply it to code and to context alike:
- Do not record a decision you have not made. Deferred choices (deployment shape, a tool, a policy) get no ADR.
- Do not add a layer where the code is self-evident. A trivial brick needs no README, no `internal/`, no Example.
- Do not future-proof the context: no speculative fields, no per-brick scaffolding "just in case".
- When something can be removed and the system still works for now, remove it. That includes ADRs and prose.

## The context layers
Each kind of context has exactly one home. If something does not fit one of these rows, it does not go in the codebase.

| Layer | Owns | Mutable? | Enforced by |
|-------|------|----------|-------------|
| Exported identifiers of the brick's root package | The public interface (the contract) | yes | compiler, `poly check`, `poly interface` |
| `Example*` functions in `_test.go` | How to call it (usage) | yes | `go test` runs them and checks `// Output:` |
| Tests in `_test.go` (external `_test` package) | How it behaves (guarantees, edge cases) | yes | `go test` in CI; `testpackage` lint keeps them at the interface |
| Package comment, promoted to a brick `README.md` when long | Why it is shaped this way: intent, invariants, non-obvious constraints | yes | `revive` requires the comment to exist; its truth is review's job |
| `docs/adr/` | Decisions with real alternatives, and the rationale | append-only | `poly adr lint`, `poly interface` |
| Commit message / PR body | What one change did | immutable | none |

## Reading a brick before editing
Do these four reads, in order, before touching a brick. They stay cheap at any codebase size.
1. **Surface**: `go doc -all ./components/<name>`. Every exported identifier with its doc comment. This is the contract; the compiler and `poly interface` keep it from drifting.
2. **Usage and behaviour**: the brick's `_test.go` files. Read the `Example*` functions first (they are the calling convention, verified on every test run), then the tests (they are the guarantees). Both live in an external `_test` package, so they use only what you can use.
3. **Why**: the package comment at the top of the root package, or the brick's `README.md` if it has one. The only prose layer; it carries what no assertion can.
4. **Decisions**: the ADRs that name this brick. Do not read the whole log. Either:

       grep -rl "<name>" docs/adr/*.md

   or open the generated view `docs/adr/index/components-<name>.md` (or `bases-<name>.md`). Each view includes the workspace-wide ADRs. A missing view means no ADR names the brick.

For what a specific change did, read the commit message or PR.

## Routing what you learned
At the end of a piece of work, route each thing you learned to exactly one home, or discard it. Most session material is not durable.
- A choice with real alternatives that someone might later reverse: **ADR** (`go run ./tools/poly adr new "<title>"`). If the choice changed behaviour, the test that pins it is part of the record.
- A changed public surface: the code change, a test for the new contract, an ADR whose `affects` names the brick (or a `[interface-impact: ...]` marker, see below), a package-comment update if the prose is now wrong.
- A new or changed guarantee (an edge case, an input/output rule, a fixed regression): a **test**. If the calling convention is not obvious from the signature, an **Example**.
- A durable gotcha or invariant that no assertion can express: the **package comment** or the brick README.
- What this diff does: the **commit message**.
- Exploration that concluded nothing durable: **discard**. Do not write it anywhere.

When something could be a test or a README line, write the test. Prose is for what an assertion structurally cannot say: intent and rationale.

## Working with the developer
You carry the process; the developer may be junior, fast-moving, and unfamiliar with Polylith vocabulary. The hard guarantees are the gates (next section). Your job is to make the correct path the easy one while the work happens.
- Do the bookkeeping yourself. When a change needs an ADR, a test, an Example or a comment, draft it and ask the developer to confirm. Do not tell them to go write it.
- Explain a term the first time you use it in a conversation: base, brick, surface, composition root, `interface-impact`.
- One question at a time. Ask the single most important confirmation, act on the answer, continue.
- Prefer the simplest action (BSSN). Say "this brick does not need a README" rather than writing one to be safe.
- An ADR you draft starts as `status: proposed`. Only the developer moves it to `accepted`. If nobody can confirm, leave it `proposed` and say so in the PR.

### Stop and confirm before
Scan this list before you act, not after.
- **Changing an existing public surface** (an exported function, method, type, field or constant in a brick's root package). Say: "this changes what other code depends on; `poly deps` shows these users: ... Intended?" Then: draft the ADR, update or add the test and Example. Adding a brand-new brick is not this case; use `[interface-impact: none]` if no project consumes it yet.
- **A decision with real alternatives.** Say: "this is a choice with other options (A, B). Record a short ADR so it is not relitigated?" Draft it if yes. Skip it if the alternatives are not real (BSSN).
- **Anything destructive or hard to reverse**: deleting or renaming a brick or an exported name, merging or splitting components, moving a dependency boundary. Run `go run ./tools/poly deps` and `go run ./tools/poly diff`, name the blast radius, confirm.
- **Business logic in a base or a project.** Logic lives in components; a base parses and delegates; a project's `main()` constructs and runs.
- **Adding a dependency.** First ask whether the standard library has it (Go 1.27 ships `uuid`, `encoding/json/v2`, post-quantum crypto). Then ask whether a few lines do the job for now. A module added for one project lands in the shared `go.mod`.
- **Editing an existing ADR** beyond its status line. ADRs are immutable; write a new one that supersedes it.

### Remind, but do not block, when
- A behaviour changed and no test asserts it.
- A public surface changed and no ADR names the brick.
- A package comment or README now contradicts the code.
- Scratch is left in `development/`.

### Pre-commit checklist
The hook enforces structure. Walk this for what it cannot see:
1. **Behaviour added or changed → a test asserts it**, and it would fail without the change.
2. **A real decision was made → ADR drafted, or explicitly declined** with a sentence saying why it was not significant.
3. **Prose still true**: package comment, README, Example.
4. **Scratch removed** from `development/`; anything durable routed to its layer.

## Conventions
### ADRs
- Location: `docs/adr/`, one global numbered sequence. Create with `go run ./tools/poly adr new "<title>"`, which picks the next number, fills the date, and copies `0000-adr-template.md` (MADR 4.0 plus the Polylith fields).
- Numbers are sequential and never reused. The lint warns on a gap and fails on a duplicate.
- Front matter: `status`, `date`, `decision-makers`, `affects` (components, bases, projects the decision touches; `*` means all), `interface-impact` (`none` | `new` | `breaking`). `date` and `decision-makers` are filled in when the decision is adopted; until then the lint only warns about the placeholders.
- `affects` is historical: it may name a brick that does not exist yet or no longer exists. The lint warns, never blocks. Never edit an old ADR to track a rename.
- Lifecycle: `proposed` → `accepted` → `deprecated` or `superseded by ADR-NNNN`. The status line is the only field you may edit on an existing ADR; the lint checks the target exists.
- One line per paragraph (no hard wrapping), so `grep -l` matches once per hit.
- Write an ADR for a decision, never to describe current state, and never for a brick whose shape involved no contested choice.

### Recording an interface change
CI blocks a pull request in which a brick's exported surface changed and nothing records it. For every changed brick, one of:
1. An ADR in the pull request whose `affects` names the brick. Best; required for a breaking change.
2. A commit message referencing an existing ADR that names it: `ADR-0012: ...`.
3. `[interface-impact: none]` in a commit message: no consumer can observe the change. Covers adding or removing a brick that no project uses yet.
4. `[interface-impact: new]` in a commit message: additive change, ADR deliberately declined.

`breaking` is not a marker. The hook prints the exact `-`/`+` lines of every surface change on every commit; `go run ./tools/poly interface --between <old> <new>` prints them for any two refs.

### Tests and Examples
- Colocated `_test.go` files, external package (`package greeting_test`). The `testpackage` linter enforces it, so tests can only reach the exported API.
- Assert the contract, not internals. A test pinned to an incidental detail breaks on every refactor and teaches nothing.
- `ExampleXxx` with an `// Output:` comment for any brick whose calling convention is not obvious from the signature. `go test` runs it; `go doc -ex` lists it.
- Run everything with `go test ./...`; CI adds `-race -shuffle=on`.

### Package comment and README
- Every package has a `// Package <name> ...` comment and every exported identifier a doc comment (`revive` enforces presence). The package comment states purpose and the one or two invariants a reader must know.
- Promote to a brick `README.md` only when the explanation needs sections or examples prose cannot avoid. Never both saying the same thing. Never copy a signature into prose; `go doc` prints the real one.
- When a decision changed this brick's contract, add one line: `See ADR-NNNN.`

### Projects and development
- `projects/<name>/main.go`: `package main`, one file, constructs components, calls a base, exits. Deployment files beside it. `make build` builds every project into `bin/`.
- `development/<name>/main.go`: scratch. Outside `./...` (go.mod `ignore`), so it never blocks a commit; run with `go run ./development/<name>`. Delete it when the exploration ends.
- Tag a known-good point with `git tag -a stable-1 -m "..."` after the first release-worthy state; `poly diff` compares against the newest `stable-*` tag.

## Guardrails
- The interface contract lives only in the exported identifiers. Never copy it into prose.
- Behavioural guarantees live in tests and Examples, not in prose.
- Raw session narrative does not go in the codebase. Only the distilled, routed artifacts above.
- Prose is for slow-changing things: purpose, invariants, constraints. Volatile detail belongs where it is enforced.

## What is enforced vs trusted
Two layers. This file is the soft layer: it catches gaps in conversation. The gates are the hard layer: they block non-compliant changes regardless of who made them.

Hard gates, local (`.githooks/pre-commit`, enabled by `make hooks`), in order: gofmt on staged files; `poly check`; `go vet ./...`; `go test ./...`; `poly adr lint`; the ADR index regenerated and staged for you; the interface heads-up (prints, never blocks); golangci-lint (required install; the hook fails without it).

Hard gates, CI (`.github/workflows/checks.yml`), required by branch protection on `main`: `go mod tidy -diff`; `go build ./...`; `poly check`; `go test -race -shuffle=on ./...`; golangci-lint; `poly adr lint`; `poly interface --mode ci` (blocks an unrecorded surface change); `govulncheck`.

Trusted (convention, made easy but not blocked): commit messages, the discard discipline, the truth of prose, "thin base" and "wiring-only project", and the honesty of `[interface-impact: ...]` markers. Code review is the backstop; the always-printed surface diff makes misuse visible.

## References
- Polylith (concepts): https://polylith.gitbook.io/polylith/
- MADR: https://adr.github.io/madr/
- Best Simple System for Now: https://dannorth.net/blog/best-simple-system-for-now/
- Go `internal` packages: https://go.dev/doc/go1.4#internalpackages
- Go Examples: https://pkg.go.dev/testing#hdr-Examples
