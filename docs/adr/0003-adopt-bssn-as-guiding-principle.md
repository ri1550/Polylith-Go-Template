---
status: accepted
date: YYYY-MM-DD
decision-makers: [you]
consulted: []
informed: []
affects:
  components: []
  bases: []
  projects: []
interface-impact: none
---

# ADR-0003: Adopt Best Simple System for Now as the guiding principle
## Context and problem statement
A codebase's complexity accumulates through many small decisions. Without a shared, explicit criterion, those decisions tend toward over-engineering, premature abstraction, and speculative infrastructure that is individually reasonable but collectively expensive. We need a guiding principle that can be applied consistently without a policy meeting for every call.

## Decision drivers
- Decisions should converge on a clear answer without case-by-case deliberation.
- Speculative complexity is expensive: it is paid for by everyone who reads or changes the code later, not only by the person who added it.
- The right level of abstraction and structure depends on actual, current need, not anticipated future need.
- The principle must apply to the context system and documentation as much as to the code itself.

## Considered options
- No explicit guiding principle (decide ad hoc)
- YAGNI (You Aren't Gonna Need It)
- Best Simple System for Now (BSSN)
- DRY (Don't Repeat Yourself) as the primary lens
- KISS (Keep It Simple, Stupid)

## Decision outcome
Chosen option: Best Simple System for Now (BSSN). Build the simplest thing that meets the need right now, written to an appropriate standard, with no speculative future-proofing.

BSSN over YAGNI because BSSN is framed positively. Build the simplest right thing, rather than only avoiding extras and because "appropriate standard" explicitly includes code quality. BSSN over DRY because DRY is a code-level principle that can drive premature abstraction when applied before real duplication exists. BSSN over KISS because KISS focuses on simplicity in isolation; BSSN explicitly ties simplicity to the actual current need. No explicit principle leads to local-optimum decisions that accumulate into structural debt.

### Consequences
- Good: every decision has a clear filter. Does it meet an actual current need, at appropriate quality, with no speculation?
- Good: deferred decisions (on deployment shape, tooling, policy) are not failures; they are the correct response to not yet having a concrete need.
- Bad: requires judgment to apply; "appropriate standard" and "right now" need context that tooling cannot supply.
- Neutral: this principle governs code, the context system, and AGENTS.md equally.

### Confirmation
Applied throughout this decision log: each ADR's decision drivers include the BSSN filter. AGENTS.md carries BSSN as its guiding principle section. Deviations (adding structure before need, keeping complexity for future-proofing) should be challenged by citing this ADR.

## More information
- Best Simple System for Now (BSSN): https://dannorth.net/blog/best-simple-system-for-now/
