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

The layers that describe code which exists (its interface, its usage, its behaviour, its intent, the decisions behind it, what each change did) leave two things without a home. Nothing holds what the product is supposed to do before the code exists, so an agent asked to build a brick has nowhere to look for the target and nothing to stop it from inventing one. And nothing holds the order of work across sessions, so each session re-derives what to do next from the conversation.

## Decision drivers
- Any brick's full context should be a few cheap reads at any codebase size.
- Each kind of context needs exactly one home, reached by query or by sitting next to the code, never by reading piles.
- BSSN: capture only what is durable, discard the rest. The value is in the distillation, not the capture.
- An agent must implement intended behavior and nothing else, and must be able to tell the two apart without asking.
- Undecided questions need one home and one grep, so they are never silently resolved by the agent.
- Current state must not be written down by hand, because hand-maintained state drifts.
- The order of work has to survive a session ending and an agent's context being compacted.
- Agents offer architecture decision records far more often than they are wanted, and an unwanted ADR is noise in an append-only log.

## Considered options
- One central log plus colocated, query-tagged layers (the system below)
- Per-brick decision records and notes only
- An external wiki or doc tool
- Dumping raw session transcripts into the repo
- No formal capture; rely on commit history and memory
- For intended behaviour and order: a behavior spec with numbered rules and `OPEN:` lines, plus a minimal queue of rule ranges whose state is derived from test citations (chosen)
- A spec plus a full plan file holding in-flight state, a queue with done-when sentences, and a blocked list, maintained through a CLI
- A spec only, with order kept in the developer's head or an issue tracker
- No spec; tests as the only behavior layer, with intent agreed in conversation each session
- An ideas directory alongside the spec for parked, not-yet-intended work

## Decision outcome
Chosen option: a small set of layers, each owning one kind of context, plus a routing rule applied at the end of a session and a reading protocol applied before editing. The full, living specification is `AGENTS.md` at the workspace root; this ADR records the decision and its rationale, `AGENTS.md` records the mechanics.

The layers: `docs/spec/<domain>.md` for what the product intends, the exported identifiers of a brick's root package for the public interface (readable in one command with `go doc -all`), `Example` functions and tests in an external `_test` package for how to call a brick and how it behaves, the package comment or a brick `README.md` for intent and invariants, `docs/queue.yaml` for the order of work, `docs/adr/` for architectural decisions with alternatives, and commit or PR messages for what a change did. Decisions are found per brick by querying the `affects` field (`grep -rl <brick> docs/adr/`, or the generated `docs/adr/index/`), not by reading the whole log.

The spec is `docs/spec/`, one file per product domain with five fixed headings. Behavior rules are numbered sentences a test could assert, numbers are permanent, and unsettled questions are `- OPEN:` lines. A test pins a rule by citing it as `<domain> rule N` in its doc comment. Rules come from the developer; anything the agent derives on its own is an `OPEN:` line. What works today is not written anywhere: it is the set of rules with a passing test citing them, derived by `go run ./tools/poly spec status`.

The queue is `docs/queue.yaml`, an ordered list of steps, each a range of spec rules for one brick with an optional dependency on an earlier step. A step's state (waiting on spec, ready, in progress, done) is derived from whether its rules exist and whether tests cite them. Finished steps are deleted. There is no in-flight state, no done-when prose, and no blocked list: in-flight state belongs to the session, done-when is the rule text, and a blocked step is one waiting on a domain's `OPEN:` lines.

ADRs are the developer's instrument. An `OPEN:` line is not a pending ADR, and the agent never drafts or offers one unless the developer asks.

Central-plus-tagged over the alternatives because per-brick-only records fragment the log and have no home for cross-cutting decisions; an external wiki breaks the "context lives with the code" goal; and dumping raw transcripts recreates the signal-to-noise problem we are trying to avoid. The discipline that makes this work is routing each thing learned to exactly one layer and discarding what has no durable home.

The minimal spec and queue over the full plan file because each of the plan file's extra parts duplicated a layer that already existed, and the duplicate drifted: in-flight state duplicated the session, done-when prose restated the spec, and the blocked list filled with decisions that no queued step depended on, which ADR-0003 says not to record. Over a spec-only approach because order does not survive a session boundary. Over no spec because the agent then has no way to distinguish intended behavior from its own guesses. The ideas directory is not adopted: it leaks `OPEN:` lines and dead links into files whose job is to prevent exactly that.

### Consequences
- Good: context is co-located with code, queryable, and stays cheap to load as the codebase grows.
- Good: tests carry behavioral context as an enforced layer; prose is reserved for what an assertion cannot express.
- Good: Go's export rule makes the interface layer machine-readable: `go doc` prints it, and the interface check (ADR-0008) diffs it exactly.
- Good: an agent can tell what is intended, what is undecided, and what works, each by one grep or one command, with nothing maintained by hand except the rule text and the order.
- Good: fewer unwanted ADRs. The decision log stays a record of architectural direction.
- Bad: depends on discipline (good commit messages, the discard default, README accuracy) that cannot be fully enforced by tooling.
- Bad: a product question the developer has not answered blocks the brick that needs it. This is intended; it is the cost of never inventing a rule.
- Bad: the spec's substance (whether a rule is true and worth having) is trusted, not enforced. The lint checks shape and citations; the developer's confirmation is the backstop.
- Neutral: chores that cite no rule have no home in the queue. They are done now, dropped, or tracked in an issue tracker.
- Neutral: tests are treated as a context layer and are written at the interface, from an external `_test` package, not against implementation internals. This is recorded here rather than in a separate ADR for now; it will be split out only if a contested testing decision arises.

### Confirmation
The context layers are kept accurate by the automated checks described in ADR-0006, ADR-0007 and ADR-0010. `go run ./tools/poly spec lint` runs in the hook and CI and fails on wrong headings, broken rule numbering, a citation of a rule that does not exist, a broken link under `docs/`, a glossary term redefined in a domain, or a malformed queue. `go test ./...` holds every pinned rule. `poly spec status` is printed into the agent's context by a SessionStart hook (`.claude/settings.json`) so the derived state is read, not reconstructed. The example `greeting` component ships with a two-rule domain and tests citing both. Everything else is convention that `AGENTS.md` makes the path of least resistance.

## More information
- `AGENTS.md` (the living specification this ADR points at)
- `docs/spec/README.md` (the spec's conventions and lifecycles) and `docs/queue.yaml` (the queue's shape, in its header)
- ADR-0006 (the decision to enforce this process with automated checks)
- Best Simple System for Now (BSSN): https://dannorth.net/blog/best-simple-system-for-now/
