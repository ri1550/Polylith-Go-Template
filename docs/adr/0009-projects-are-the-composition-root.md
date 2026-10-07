---
status: accepted
date: 2026-10-07
decision-makers: [workspace owner]
consulted: [template author]
informed: []
affects:
  components: ["*"]
  bases: ["*"]
  projects: ["*"]
interface-impact: none
---

# ADR-0009: Projects are the composition root
## Context and problem statement
ADR-0002 mapped Polylith's "projects hold no code" onto Go as "a project's `main` package imports only bases", and `poly check` enforced it. An audit with three independent build exercises showed the cost: Go has no project-level configuration that selects which components a deployable ships with, so under that rule the choice of implementation had to live inside the base. Every base then needed two exported entry points, an injectable constructor for tests and a zero-argument one for production, and constructor injection from `main()`, the idiomatic Go shape, was rejected with a message about business logic when the code was pure wiring. The rule also contradicted AGENTS.md's own prose that projects "combine a base with components".

## Decision drivers
- A project must be able to choose which components it runs with; that is the Polylith promise, and in Go the only place to make that choice is `main()`.
- Bases should be testable libraries with one constructor and injected dependencies.
- "No business logic in projects" must stay checkable in some structural form.
- BSSN: the smallest rule that keeps projects thin without forbidding the one thing they exist to do.

## Considered options
- Keep "projects import only bases" and document that bases construct their own dependencies and expose a zero-argument entry point
- Allow projects to import components and bases, with no further rule
- Allow projects to import components and bases, and bound the project structurally: one Go file, at least one base

## Decision outcome
Chosen option: projects may import components and bases, and are bounded structurally. A project's root package is `package main`, consists of exactly one non-test Go file, and imports at least one base. Inside that file, `main()` reads the deployment's environment, constructs the components, hands them to a base, and runs it. `poly check` enforces the three structural rules; code review enforces "wiring only". This replaces the "projects import only bases" rule in ADR-0002; everything else in ADR-0002 stands.

The bounded option over the unbounded one because a one-file limit is cheap to check and stops the common failure (a project growing helpers, handlers and types) without pretending to detect business logic. Over keeping the old rule because that rule only moved the wiring into the base, doubled every base's surface, and left projects unable to differ by component selection.

### Consequences
- Good: a second project can ship the same base with a different storage or transport component by changing one file.
- Good: bases shrink to one constructor with injected dependencies; tests inject fakes through the same door production uses.
- Good: the rule and the prose agree: projects combine a base with components.
- Bad: "wiring only" is a judgement; a determined contributor can put logic in `main()`. The one-file rule makes that visible in review.
- Neutral: environment and flag parsing is a project concern (which port, which backend); protocol handling stays in the base.

### Confirmation
`go run ./tools/poly check` fails a project whose root package is not `package main`, has more than one non-test Go file, or imports no base. `poly deps` lists the components a project imports directly. The example project `projects/hello` demonstrates the shape: `main()` passes a component function into the base.

## More information
- ADR-0002 (the architecture; its project rule is replaced here)
- `AGENTS.md` > "Polylith rules" (the live wording)
- Audit of 2026-10-07, agent B, journal step 6 and ADR-0009 of that exercise (the evidence)
