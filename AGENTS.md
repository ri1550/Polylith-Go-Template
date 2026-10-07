# AGENTS.md
Read this file before touching any code in this repository. It tells you what the codebase is, where every kind of context lives, how to read a brick before editing it, what to write afterwards, and where to stop and ask. The rules here are enforced by the checks named next to them; everything else is convention that you carry.

Quick routes:
- Starting a session: a `[spec]` summary should already be in your context (a SessionStart hook prints `go run ./tools/poly spec status`). If it is not, run that command before anything else.
- Setting up a fresh clone: run `make hooks` (CONTRIBUTING.md > One-time setup).
- Creating a brick or a project: README.md > Create new bricks. There is no scaffolding command.
- Adopting this template for a new repository: `make adopt MODULE=<path>` (CONTRIBUTING.md > Setting up the repository).
- An undecided product question: an `- OPEN:` line in its `docs/spec/<domain>.md`. Never a rule you made up, never an ADR you offered.
- Recording a decision the developer asked for: `go run ./tools/poly adr new "<title>"`.

## What this is
A Polylith workspace written in Go. Polylith is a monorepo architecture where code lives as small, composable bricks that projects assemble into deployables.

Layout:

    components/<name>/     a brick: business logic, shared across projects (a Go package)
    bases/<name>/          a brick: a thin entry point for one kind of outside world (HTTP, CLI, queue)
    projects/<name>/       a deployable: one main.go that wires components into a base, plus deploy files
    development/           scratch: throwaway programs, outside ./... (go run ./development/<name>)
    docs/spec/             the behavior contract: one file per product domain, plus the glossary
    docs/queue.yaml        the build queue: which spec rules to implement next, in order (agent-maintained)
    docs/adr/              the decision log, plus a generated per-brick index
    tools/poly/            the in-repo Polylith tool; the checks that keep the process honest
    go.mod                 the one module; its module path is the workspace namespace

Tests live next to the code they test, as `_test.go` files inside the brick.

Vocabulary, used throughout:
- **Brick**: a component or a base. A directory holding a Go package.
- **Public interface** (also "surface"): every exported identifier in a brick's root package. There is no interface file; exported-ness is the contract. `go doc -all ./components/<name>` prints it.
- **Implementation**: everything unexported, plus the brick's `internal/` subpackages, which the compiler keeps private to the brick.
- **Composition root**: a project's `main()`, the one place that constructs components and hands them to a base.
- **ADR**: an Architecture Decision Record under `docs/adr/`: an architectural decision, the alternatives, the reasons. Started by the developer, never offered by you.
- **Spec domain**: one file in `docs/spec/` describing one area of the product in product language: numbered behavior rules a test can assert, and `OPEN:` questions nobody has settled.
- **Queue step**: an entry in `docs/queue.yaml` naming a range of spec rules to implement in one brick. Its state is derived from the tests, never written.

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
- Do not record a decision you have not made. Deferred choices (deployment shape, a tool, a policy) get no ADR. An undecided product question is an `OPEN:` line in its spec domain; an undecided architectural choice that no spec domain needs answered is written nowhere.
- Do not add a layer where the code is self-evident. A trivial brick needs no README, no `internal/`, no Example.
- Do not future-proof the context: no speculative fields, no per-brick scaffolding "just in case".
- When something can be removed and the system still works for now, remove it. That includes ADRs and prose.

## The context layers
Each kind of context has exactly one home. If something does not fit one of these rows, it does not go in the codebase. What works today is deliberately not a row: it is the set of spec rules with a passing test citing them, and `go run ./tools/poly spec status` derives it.

| Layer | Owns | Mutable? | Enforced by |
|-------|------|----------|-------------|
| `docs/spec/<domain>.md` | What the product intends to do: numbered behavior rules and `OPEN:` questions, before and alongside the code | yes; rule numbers permanent | `poly spec lint`; `go test` through rule citations |
| Exported identifiers of the brick's root package | The public interface (the contract) | yes | compiler, `poly check`, `poly interface` |
| `Example*` functions in `_test.go` | How to call it (usage) | yes | `go test` runs them and checks `// Output:` |
| Tests in `_test.go` (external `_test` package) | How it behaves (guarantees, edge cases); which spec rules it pins | yes | `go test` in CI; `testpackage` lint keeps them at the interface; `poly spec lint` checks the citations |
| Package comment, promoted to a brick `README.md` when long | Why it is shaped this way: intent, invariants, non-obvious constraints | yes | `revive` requires the comment to exist; its truth is review's job |
| `docs/queue.yaml` | Which spec rules to build next, in what order. Agent-only; the developer hears it, they do not read it | freely rewritten; finished steps deleted | `poly spec lint` |
| `docs/adr/` | Architectural decisions that shape the project's direction, with alternatives and rationale | append-only | `poly adr lint`, `poly interface` |
| Commit message / PR body | What one change did | immutable | none |

## Reading a brick before editing
Do these five reads, in order, before touching a brick. They stay cheap at any codebase size.
0. **Intent**: the brick's spec domain, `docs/spec/<domain>.md`. Code implements numbered rules from there and nothing else; an `OPEN:` line is not buildable. If no domain covers the brick, that is a question for the developer, not a guess.
1. **Surface**: `go doc -all ./components/<name>`. Every exported identifier with its doc comment. This is the contract; the compiler and `poly interface` keep it from drifting.
2. **Usage and behaviour**: the brick's `_test.go` files. Read the `Example*` functions first (they are the calling convention, verified on every test run), then the tests (they are the guarantees). Each cites the spec rule it pins as `<domain> rule N` in its doc comment. Both live in an external `_test` package, so they use only what you can use.
3. **Why**: the package comment at the top of the root package, or the brick's `README.md` if it has one. The only prose layer; it carries what no assertion can.
4. **Decisions**: the ADRs that name this brick. Do not read the whole log. Either:

       grep -rl "<name>" docs/adr/*.md

   or open the generated view `docs/adr/index/components-<name>.md` (or `bases-<name>.md`). Each view includes the workspace-wide ADRs. A missing view means no ADR names the brick.

For what a specific change did, read the commit message or PR.

## The spec: docs/spec/
The spec is the behavior contract, one file per product domain, in product language. Rules are numbered sentences a test could assert, and the numbers are permanent. Unsettled questions are `- OPEN:` lines, so `grep -rn "^- OPEN:" docs/spec/` lists everything undecided in the product. A test pins a rule by citing it as `<domain> rule N` in the doc comment above the test or Example function.

The one distinction that keeps the spec a contract rather than a draft: rules come from the developer, and anything you derive on your own is an `OPEN:` line. Adding an `OPEN:` line is the one spec edit you make without asking. Every other edit (a new rule, a reworded rule, a glossary entry, a new domain) you draft, show, and land on confirmation. The lifecycles, the five-heading shape, the citation grammar, and the shared-term process live in `docs/spec/README.md`, not here.

## The queue: docs/queue.yaml
The queue holds the one thing no other layer holds: order. Each step is a range of spec rules to implement in one brick, with an optional `after` dependency. Everything else is derived, never written. A step is waiting on spec when it cites rules the domain does not have yet, ready when they exist, in progress when some have a citing test, and done when all do. A done step is deleted. `go run ./tools/poly spec status` prints all of this.

You do not read the queue by hand. A SessionStart hook (`.claude/settings.json`) prints `poly spec status` into your context at every session start. If no `[spec]` summary appeared, the hook did not run; run `go run ./tools/poly spec status` yourself before doing anything else. Open every session by saying, in one line, what the queue's head is and which domains have open questions it is waiting on. Then read the spec domain of the step you are about to work.

Edit the YAML directly; `poly spec lint` validates it at commit time. A step exists only because the developer asked for the work or a queued step depends on it. Nothing speculative. Chores that cite no rule do not go in the queue: do them now, drop them, or open an issue. Dead ends, sub-steps, and in-flight notes are session material: plan mode holds them while you work, the commit message holds what survives, and nothing else is written down.

## Routing what you learned
At the end of a piece of work, route each thing you learned to exactly one home, or discard it. Most session material is not durable.
- A product question the spec does not answer, or a choice with real alternatives that is not yet decided: an **`OPEN:` line** in the spec domain, naming the alternatives when it is a choice. Then stop building what it covers. Do not draft or offer an ADR; see "Decisions" below.
- The developer settled what the product should do: a **numbered rule** in the spec domain, appended with the next number, confirmed by the developer, with the `OPEN:` line it resolves deleted in the same change.
- The developer asked for an ADR: draft it with `go run ./tools/poly adr new "<title>"` and confirm it. If the choice changed behaviour, the test that pins it is part of the record. Only the developer starts an ADR.
- A changed public surface: the code change, a test for the new contract, a package-comment update if the prose is now wrong. The interface gate will want an ADR naming the brick or a `[interface-impact: ...]` marker (see below); tell the developer and let them choose.
- A new or changed guarantee (an edge case, an input/output rule, a fixed regression): a **test**, citing the spec rule it pins. If the calling convention is not obvious from the signature, an **Example**.
- Scoped build work the developer asked for, with rules to cite: a **step in `docs/queue.yaml`**, placed by dependency.
- A durable gotcha or invariant that no assertion can express: the **package comment** or the brick README.
- What this diff does: the **commit message**.
- Exploration that concluded nothing durable, in-flight notes, dead ends: **discard**. Do not write it anywhere.

When something could be a test or a README line, write the test. Prose is for what an assertion structurally cannot say: intent and rationale.

### Decisions
ADRs record architectural decisions that shape the direction of the project. Only the developer starts one. Do not draft one, and do not offer to, because a question looks like a decision, because a choice has alternatives, or because a change feels significant. Write the `OPEN:` line if a spec domain needs the answer, otherwise write nothing, and carry on. When the developer wants an ADR they will ask, and then you draft it from the template and confirm it with them. This is deliberate: agents offer ADRs far more often than they are wanted, and an unwanted ADR is noise in an append-only log (ADR-0005).

## Working with the developer
You carry the process; the developer may be junior, fast-moving, and unfamiliar with Polylith vocabulary. The hard guarantees are the gates (next section). Your job is to make the correct path the easy one while the work happens.
- Do the bookkeeping yourself. When a change needs a test, an Example, a comment, a spec `OPEN:` line, a queue update, or an ADR the developer has asked for, draft it and ask the developer to confirm. Do not tell them to go write it.
- Explain a term the first time you use it in a conversation: base, brick, surface, composition root, spec domain, `interface-impact`.
- One question at a time. Ask the single most important confirmation, act on the answer, continue.
- Prefer the simplest action (BSSN). Say "this brick does not need a README" rather than writing one to be safe.
- An ADR you were asked to draft starts as `status: proposed`. Only the developer moves it to `accepted`. If nobody can confirm, leave it `proposed` and say so in the PR.

### Stop and confirm before
Scan this list before you act, not after.
- **Changing an existing public surface** (an exported function, method, type, field or constant in a brick's root package). Say: "this changes what other code depends on; `poly deps` shows these users: ... Intended?" If yes, make the change, update or add the test and Example, and say that the interface gate will need either an ADR naming the brick or a `[interface-impact: none|new]` marker in a commit message. The developer chooses which; draft the ADR only if they ask. Adding a brand-new brick is not this case; `[interface-impact: none]` covers it while no project consumes it.
- **Hitting a choice with real alternatives.** Say: "this is a choice with other options (A, B). It is an `OPEN:` line in `<domain>.md` until you decide." Write the `OPEN:` line. Do not offer an ADR.
- **Adding, rewording, or withdrawing a numbered spec rule, creating a domain, or editing `docs/spec/00-glossary.md`.** Show the sentence, then land it on confirmation. Adding an `OPEN:` line needs no confirmation.
- **Anything destructive or hard to reverse**: deleting or renaming a brick, an exported name, or a spec domain, merging or splitting components, moving a dependency boundary. Run `go run ./tools/poly deps` and `go run ./tools/poly diff` (and `grep -rn "<domain> rule" --include='*_test.go' components bases` for a domain), name the blast radius, confirm.
- **Business logic in a base or a project.** Logic lives in components; a base parses and delegates; a project's `main()` constructs and runs.
- **Adding a dependency.** First ask whether the standard library has it (Go 1.27 ships `uuid`, `encoding/json/v2`, post-quantum crypto). Then ask whether a few lines do the job for now. A module added for one project lands in the shared `go.mod`.
- **Editing an existing ADR** beyond its status line. ADRs are immutable; write a new one that supersedes it.

### Remind, but do not block, when
- A behaviour changed and no test asserts it.
- A public surface changed and no ADR names the brick.
- A brick implements a spec rule and no test cites it as `<domain> rule N`.
- A queue step is done (every rule cited) and still in the file.
- A package comment or README now contradicts the code.
- Scratch is left in `development/`.

### Pre-commit checklist
The hook enforces structure. Walk this for what it cannot see:
1. **Behaviour added or changed → a test asserts it and cites its rule**, and it would fail without the change.
2. **A question or choice came up → it is an `OPEN:` line, or the developer settled it.** If still open, the `OPEN:` line naming the alternatives is in the spec domain. If settled, the rule is in the spec and the `OPEN:` line is gone. An ADR exists only if the developer asked for one.
3. **Prose still true**: package comment, README, Example.
4. **Scratch removed** from `development/`; anything durable routed to its layer.
5. **The queue matches the code.** A step whose rules all have a citing test is deleted in this commit. A step the developer asked for is added. Nothing in between is written down.

## Conventions
### ADRs
- Started by the developer. You draft one only when asked and never offer one unprompted.
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

### Spec
- Conventions, the rule lifecycle, the citation grammar, and the shared-term process live in `docs/spec/README.md`. Not repeated here.

### Queue
- Shape and the derived states are in the header of `docs/queue.yaml` and in `tools/poly/README.md`. Not repeated here.

### Tests and Examples
- Colocated `_test.go` files, external package (`package greeting_test`). The `testpackage` linter enforces it, so tests can only reach the exported API.
- Cite the spec rule a test pins as `<domain> rule N` in the doc comment above the function: `// TestTransfer asserts accounts rule 3.` That exact form is what `poly spec lint` and `poly spec status` grep for. An Example may cite a rule the same way.
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
- Rules come from the developer. Never invent a spec rule to unblock yourself; write the `OPEN:` line and stop.
- Never draft or offer an ADR unasked.
- The interface contract lives only in the exported identifiers. Never copy it into prose.
- Behavioural guarantees live in tests and Examples, not in prose.
- Raw session narrative does not go in the codebase. Only the distilled, routed artifacts above.
- Prose is for slow-changing things: purpose, invariants, constraints. Volatile detail belongs where it is enforced.

## What is enforced vs trusted
Two layers. This file is the soft layer: it catches gaps in conversation. The gates are the hard layer: they block non-compliant changes regardless of who made them.

Hard gates, local (`.githooks/pre-commit`, enabled by `make hooks`), in order: gofmt on staged files; `poly check`; `go vet ./...`; the same Polylith rules as a vet analyzer (`tools/polyvet`, file:line at the import); `go test ./...`; `poly adr lint`; `poly spec lint` (spec headings in order, rule numbering contiguous, `<domain> rule N` citations resolve, links under `docs/` resolve, no glossary term redefined, `docs/queue.yaml` well-formed and naming real domains); the ADR index regenerated and staged for you; the interface heads-up (prints, never blocks); golangci-lint (required install; the hook fails without it).

Hard gates, CI (`.github/workflows/checks.yml`), required by branch protection on `main`: `go mod tidy -diff`; `go build ./...`; `poly check`; `go test -race -shuffle=on ./...`; golangci-lint; `poly adr lint`; `poly spec lint`; `poly interface --mode ci` (blocks an unrecorded surface change); `govulncheck`.

Trusted (convention, made easy but not blocked): commit messages, the discard discipline, the truth of prose, "thin base" and "wiring-only project", the honesty of `[interface-impact: ...]` markers, and that a spec rule is true and worth having and a queue step was asked for. The lints check that they are well-formed; the developer's confirmation and code review are the backstop; the always-printed surface diff makes misuse visible.

## References
- Polylith (concepts): https://polylith.gitbook.io/polylith/
- MADR: https://adr.github.io/madr/
- Best Simple System for Now: https://dannorth.net/blog/best-simple-system-for-now/
- Go `internal` packages: https://go.dev/doc/go1.4#internalpackages
- Go Examples: https://pkg.go.dev/testing#hdr-Examples
