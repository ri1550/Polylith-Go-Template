# AGENTS.md
## What this is
This is a Polylith workspace written in Go. This file governs how context about the codebase (why things are the way they are, what each part does, how to work on it) lives in the codebase itself, and how you, the agent, read it before editing and write it after.

The guiding idea: a brick's full context should always be a few cheap reads, regardless of how large the codebase grows. That holds only because each kind of context has exactly one home and is reached by query or by sitting next to the code, never by reading piles, and because the content in those homes is kept honest by discipline and review — the automated checks enforce structure, not substance. Your job on the write side is distillation, not capture.

Layout:

    bases/            thin entry points (one per app or service); each is a Go package
    components/       business logic, shared across projects; each is a Go package
    projects/         deployable artifacts: a package main that calls a base, plus deploy files
    development/      scratch space: throwaway package main programs (go run ./development/<x>)
    docs/adr/         the decision log (and a generated per-brick index)
    tools/poly/       the in-repo Polylith tool and the automated checks that keep the process honest
    go.mod            the one module for the whole workspace; its module path is the namespace

Tests live next to the code they test, as `_test.go` files inside the brick.

## Polylith rules
- Bricks are `components` and `bases`. Components hold business logic and are shared across projects. Bases are thin entry points to an app or service: they parse what the outside world hands them and delegate to components.
- A brick is a directory under `components/` or `bases/` holding a Go package. Its **root package is its public interface**: every exported identifier is the contract other bricks depend on. Unexported identifiers are implementation. When the implementation outgrows one package, move it to the brick's `internal/` subpackage; the compiler then refuses imports from outside the brick.
- Bricks are used through their root package only. Importing another brick's subpackage bypasses its interface and `poly check` rejects it.
- Components must not import bases. Dependency flows base -> components, never the reverse, so components stay reusable. Bases must not import other bases; code two bases share is a component.
- Projects (`projects/`) combine a base with components into a deployable artifact. A project is a `package main` whose `main()` hands control to a base, next to its deployment infrastructure (Dockerfile, deploy scripts). Put no business logic there, and import only bases.
- The whole workspace is one Go module. Go's linker includes in each project binary only the packages it imports, so there is no per-project dependency list to maintain.
- All of this is enforced by `go run ./tools/poly check` and by the compiler (`internal/`).

## Guiding principle: Best Simple System for Now (BSSN)
Build the simplest thing that meets the need right now, written to an appropriate standard, with no speculative future-proofing. See ADR-0003 for the rationale.

In practice:
- Do not record a decision you have not made. No ADR for a deployment shape, tool, or policy that is still open. Polylith lets you defer these on purpose.
- Do not add a layer where the code is self-evident. A trivial brick needs no README and no `internal/`; an obvious change needs no ADR.
- Do not future-proof the context itself: no speculative fields, no per-brick scaffolding "just in case." Add structure when a real need appears, not before.
- When something can be taken away and the system still works for now, take it away. That applies to ADRs and READMEs as much as to code.

## The context layers
Five homes, each owning one kind of context:

| Layer                                        | Owns                                                 | Mutable?    | Enforced? |
|----------------------------------------------|------------------------------------------------------|-------------|-----------|
| Exported identifiers of the brick's root package | The brick's public interface (its surface)       | yes         | compiler, `poly check`, `poly interface` |
| `_test.go` files (external `_test` package)  | How the brick behaves and how to call it             | yes         | CI, `testpackage` lint |
| Package comment / brick `README.md`          | Why the brick is shaped this way: intent, invariants | yes         | no |
| `docs/adr/`                                  | Decisions with real alternatives, and their rationale | append-only | `poly adr lint` |
| Commit message / PR body                     | What a specific change did                           | immutable   | no |

Anything that doesn't belong to one of these layers does not go in the codebase.

## Reading a brick before editing
Load the full context in this order before touching any brick:
1. **What the brick exposes (its public surface):** `go doc -all ./components/<brick>`. This prints every exported identifier with its doc comment, nothing else. It is the interface contract: the compiler enforces visibility and `poly interface` detects any change to it, so it cannot drift.
2. **How it behaves and how to call it:** the brick's `_test.go` files. They are working examples guaranteed current, because CI fails the moment code and test disagree, and they sit in an external `_test` package, so they only use what you can use. Read these to learn what a brick does, before anything else.
3. **Why it is shaped this way (intent, invariants, non-obvious constraints):** the package comment at the top of the brick's root package, or its `README.md`. The one prose layer; carries only what a test cannot assert.
4. **Why a decision was made (with alternatives):** `docs/adr/`, filtered to the brick rather than read whole (see below).

Four reads, all current, at any codebase size. For what a specific change did, also check the commit message or PR description.

### Finding the decisions that touch a brick
Do not read the whole ADR log. Query it by the `affects` field:

    grep -rl "<brick_name>" docs/adr/

This returns the handful of decisions touching that brick out of however many total. Or read the prebuilt `docs/adr/index/components-<brick_name>.md` (or `bases-<brick_name>.md`), which the pre-commit hook keeps current.

## Routing what you learned
How to route what you learned this session into the codebase, and what to discard. Most session material is not durable; if a thing does not clearly match one of these, discard it rather than inventing a home for it.

- **Made a choice with real alternatives someone might later reverse?** ADR. If the choice changes behavior, the test that pins the new behavior is part of recording it.
- **Changed a brick's public surface?** The exported-identifier change itself, a test asserting the new contract behaves as intended, plus an ADR if it was a genuine decision and a package-comment or README update if the prose needs it.
- **Established or changed how a brick is supposed to behave** (an edge case, an input/output guarantee, a regression you just fixed)? A test. This is the executable half of "how it works," and it is preferred over prose whenever the behavior can be asserted.
- **Learned a durable gotcha or invariant that no assertion can capture**, something about intent or rationale rather than behavior? Package comment or brick README.
- **Just describes what this diff does?** Commit or PR body.
- **Exploration that concluded nothing durable?** Discard. Do not write it anywhere.

When something could live as either a test or a README line, the test wins. It is enforced; the prose is not. Prose only catches what an assertion structurally cannot, which is why intent and rationale are the README's job.

## Working with the developer
This file is guidance you follow while working; it catches most process gaps in conversation. It cannot by itself force anyone to do anything. The hard guarantee comes from the automated gates under "What is enforced vs trusted" (CI and the pre-commit hook). Use both: you guide while the work happens, the gates block what slips through.

Assume the developer may be junior or moving fast and may not know Polylith vocabulary. The system works only if you carry the process, not them.

How to behave:
- **Do the bookkeeping yourself.** When a change needs an ADR, a test, or a README update, draft it and ask the developer to confirm. Do not tell them to go write it. Make the correct path the easy one.
- **Explain the term as you use it.** When you say base, brick interface, exported identifier, or `interface-impact`, add a one-line plain explanation. Do not assume the vocabulary is known.
- **One question at a time.** Do not interrogate. Ask the single most important confirmation, act on the answer, move on.
- **Prefer the simplest action (BSSN).** If a brick is trivial, say a README is not needed rather than creating one "to be safe." Ask before adding structure.

### Stop and confirm before
- **Changing a public interface** (an exported identifier in a brick's root package: adding, removing or changing a signature, an exported field, a constant value). Say plainly: "this changes what other code depends on; every project using `<brick>` is affected. Intended?" If yes, set `interface-impact`, draft the ADR, add or adjust the test. `go run ./tools/poly interface` shows exactly what changed.
- **Making a decision with real alternatives.** Say: "this is a choice with other options (A, B). Want me to record a short ADR so it is not relitigated later?" Draft it if yes. Drop it if it is not actually significant (BSSN).
- **Anything destructive or hard to reverse:** deleting or renaming a brick or an exported name, merging or splitting components, moving a dependency boundary. Confirm intent and name the blast radius (`go run ./tools/poly diff` shows the bricks and projects affected since the last stable tag; `go run ./tools/poly deps` shows who uses what).
- **Putting business logic in a base or a project.** Remind: logic lives in components, bases stay thin, projects hold only `main()` and deploy files.
- **Adding a dependency.** Ask whether it is needed now or whether a few lines do the job for now. A dependency added for one project lands in the shared `go.mod`.

### Remind, but do not block, when
- A behavior changed and no test was added for it.
- A public interface changed and no ADR is linked.
- A brick's package comment or README now contradicts the code.
- Scratch or exploration is left in `development/` that should be cleaned up.

### Pre-commit checklist
The automated checks enforce structure. This covers what they cannot:
1. **Behavior added or changed → a test asserts it.** If you changed what a brick does or fixed a bug, there should be a test that would fail without that change. A test that only passes incidentally teaches nothing and breaks on refactor.

2. **A real decision was made → ADR drafted, or explicitly declined.** A real decision is one with genuine alternatives that someone might later reverse. If yes, draft the ADR and confirm with the developer. If it was not actually significant, note that explicitly rather than leaving it implicit (BSSN: do not write ADRs for non-decisions either).

3. **Package comment / README still matches the code.** If the public surface or the brick's intent changed, check that the prose still accurately describes it. Stale prose is worse than no prose — it actively misleads.

4. **Scratch work discarded from `development/`.** Exploration, temporary programs, and spike code that concluded nothing durable should be removed. If something durable was found, route it to the right layer instead of leaving it in `development/`.

## Conventions
### ADRs
- Location: `docs/adr/`, a single global numbered sequence. Not per brick.
- Template: copy `docs/adr/0000-adr-template.md`. Do not edit it in place. It is MADR 4.0 (see References) extended with the `affects` and `interface-impact` fields.
- Filenames: `NNNN-short-kebab-title.md`, zero padded.
- Numbers are sequential and never reused.
- Body prose: single line per paragraph (no hard-wrapping). Markdown renders it the same; keeps grep -rl matches to a single line per hit.
- ADRs are immutable. To reverse a decision, write a new ADR and set the old one's status to `superseded by ADR-NNNN`.
- Status lifecycle: `proposed` -> `accepted` -> (`deprecated` | `superseded`).
- Front matter carries `affects` (the components, bases, and projects the decision touches) and `interface-impact` (`none` | `new` | `breaking`).
- A change that breaks a brick's exported API requires an ADR. An interface change is exactly the kind of decision worth recording.
- ADRs are for decisions. Do not write one to describe current state, and do not write one for a brick whose shape involved no contested choice.

### Tests
- Live next to the code, as `_test.go` files in the brick, in an external package (`package greeting_test`). The `testpackage` linter enforces this so tests can only reach the exported API. Run them with `go test ./...`. `go run ./tools/poly diff` shows which bricks a change affects; that signal also tells you the blast radius of what you touched.
- Assert the contract and documented behavior, not implementation internals. A test pinning an incidental detail breaks on every refactor and teaches the next reader nothing. Test at the interface; leave the implementation free to change.

### Package comment / README
- Use the package comment (the `// Package <name> ...` block above the `package` clause) for the short "what this exposes and how to call it" note; `go doc` and IDE hovers show it. Promote to a `README.md` in the brick when the description runs longer or needs examples.
- Do not write both saying the same thing; they will drift.
- Not mandatory per brick. A trivial one-function component is self-explanatory; a README there is noise. Write one when the brick has enough surface or non-obvious behavior to warrant it.

### development/
- Scratch lives in `development/<name>/main.go` and runs with `go run ./development/<name>`. It is in the module, so it must compile and pass `go vet` like everything else, but golangci-lint skips it and nothing may import it (`poly check`). Delete it when the exploration ends; if it found something durable, route that to the right layer instead.

### Projects
- `projects/<name>/main.go` is `package main` and does one thing: call a base. Anything else that belongs to the artifact (Dockerfile, deploy scripts, config samples) sits next to it.
- Build with `go build -o bin/<name> ./projects/<name>` or `make build`.

## Guardrails
- The interface contract lives only in the exported identifiers of the brick's root package. Never copy it into prose; the copy drifts and the prose is not enforced. `go doc` prints the real thing.
- Behavioral guarantees live in tests, not in README prose. If it can be asserted, assert it.
- Raw session narrative does not go in the codebase. Only the distilled, routed artifacts above.
- Keep the prose layer to slow-changing things: purpose, invariants, non-obvious constraints. Volatile specifics belong in the contract and the tests, where they are enforced. The more you push into prose, the more drift you buy.

## What is enforced vs trusted
Two layers hold the process together. The agent behavior above is the soft layer: it catches gaps in conversation, while the work happens. The gates below are the hard layer: they block non-compliant changes regardless of who or what made them. For junior or fast-moving developers, the gates are what actually force the process; the agent guidance only makes following it the easy path. Wire the gates up early.

Hard gates (block the change):
- Compiler and `go vet`: the code type-checks, `internal/` packages are private to their brick, classic mistakes are caught.
- `go run ./tools/poly check`: the Polylith rules above (dependency direction, root-package-only imports, no logic outside the layout, projects are `package main`).
- Tests (`go test ./...`) in CI: behavioral guarantees in tests.
- `golangci-lint`: the standard linter set plus `testpackage`, with `gofmt` and `goimports`.
- Decision log lint (`go run ./tools/poly adr lint`): front matter parses, statuses are valid, numbers are unique, `affects` names real bricks and projects.
- Interface-change check (`go run ./tools/poly interface --mode ci`): a PR that changes a brick's exported API must include an ADR file, reference one in a commit message, or justify `[interface-impact: none]`. This keeps the why-layer honest. Detection is exact: the exported declarations are diffed between the base branch and the PR.
- Pre-commit hook (`.githooks/pre-commit`, enabled by `make hooks`): runs gofmt, build, vet, `poly check`, the ADR lint, the ADR index refresh, the interface heads-up and (when installed) golangci-lint before a commit lands, so feedback comes in seconds rather than from a failed CI run. Highest-leverage gate for juniors.
- CI (`.github/workflows/checks.yml`) plus branch protection on `main`: the unskippable server-side gate. See CONTRIBUTING.md for the one-time setup.

The per-brick ADR index is generated by `go run ./tools/poly adr index` into `docs/adr/index/`. Setup and the day-to-day loop for developers live in CONTRIBUTING.md; the decisions behind this enforcement are in `docs/adr/`.

Trusted (convention, made easy but not blocked):
- Good commit and PR messages, the discard discipline, package comment and README accuracy. Distillation quality cannot be fully enforced; this file and the agent behavior make the correct path the easy one.
- The `[interface-impact: none]` escape in the interface-change check is self-reported. The check enforces that something was said; code review is the backstop against an agent or developer using it to avoid recording a real interface change.

## References
- Polylith (concepts): https://polylith.gitbook.io/polylith/
- MADR: https://adr.github.io/madr/
- Best Simple System for Now: https://dannorth.net/blog/best-simple-system-for-now/
- Go `internal` packages: https://go.dev/doc/go1.4#internalpackages
