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

# ADR-0005: Capture context in the codebase
## Context and problem statement
We want the why, what, and how of the codebase to live in the codebase itself, so that a developer or an AI agent can reconstruct any part's context without relying on memory, chat logs, or tribal knowledge. We also want useful context from each development session, whether by a human or an agent, to end up in the right place. The risk is the opposite failure: dumping everything and drowning the signal.

## Decision drivers
- Any brick's full context should be a few cheap reads at any codebase size.
- Each kind of context needs exactly one home, reached by query or by sitting next to the code, never by reading piles.
- BSSN: capture only what is durable, discard the rest. The value is in the distillation, not the capture.

## Considered options
- One central log plus colocated, query-tagged layers (the system below)
- Per-brick decision records and notes only
- An external wiki or doc tool
- Dumping raw session transcripts into the repo
- No formal capture; rely on commit history and memory

## Decision outcome
Chosen option: a small set of layers, each owning one kind of context, plus a routing rule applied at the end of a session and a reading protocol applied before editing. The full, living specification is `AGENTS.md` at the workspace root; this ADR records the decision and its rationale, `AGENTS.md` records the mechanics.

The layers: the exported identifiers of a brick's root package for the public interface (readable in one command with `go doc -all`), `_test.go` files in an external `_test` package for behavior and how to call a brick, the package comment or a brick `README.md` for intent and invariants, `docs/adr/` for decisions with alternatives, and commit or PR messages for what a change did. Decisions are found per brick by querying the `affects` field (`grep -rl <brick> docs/adr/`, or the generated `docs/adr/index/`), not by reading the whole log.

Central-plus-tagged over the alternatives because per-brick-only records fragment the log and have no home for cross-cutting decisions; an external wiki breaks the "context lives with the code" goal; and dumping raw transcripts recreates the signal-to-noise problem we are trying to avoid. The discipline that makes this work is routing each thing learned to exactly one layer and discarding what has no durable home.

### Consequences
- Good: context is co-located with code, queryable, and stays cheap to load as the codebase grows.
- Good: tests carry behavioral context as an enforced layer; prose is reserved for what an assertion cannot express.
- Good: Go's export rule makes the interface layer machine-readable: `go doc` prints it, and the interface check (ADR-0008) diffs it exactly.
- Bad: depends on discipline (good commit messages, the discard default, README accuracy) that cannot be fully enforced by tooling.
- Neutral: tests are treated as a context layer and are written at the interface, from an external `_test` package, not against implementation internals. This is recorded here rather than in a separate ADR for now; it will be split out only if a contested testing decision arises.

### Confirmation
The context layers are kept accurate by the automated checks described in ADR-0006 and ADR-0007. Everything else is convention that `AGENTS.md` makes the path of least resistance.

## More information
- `AGENTS.md` (the living specification this ADR points at)
- ADR-0006 (the decision to enforce this process with automated checks)
- Best Simple System for Now (BSSN): https://dannorth.net/blog/best-simple-system-for-now/
