---
status: proposed              # proposed | accepted | deprecated | superseded by ADR-NNNN
date: YYYY-MM-DD
decision-makers: [names or roles]
consulted: [names or roles]   # optional: subject-matter experts asked for input (two-way)
informed: [names or roles]    # optional: kept up to date (one-way)
affects:                      # Polylith bricks and projects this decision touches
  components: []              # e.g. [users, payments]
  bases: []                   # e.g. [httpapi]
  projects: []                # e.g. [billing]
interface-impact: none        # none | new | breaking  (does this change a brick's exported API?)
---

<!--
Based on MADR 4.0.0 (https://adr.github.io/madr/), extended with the Polylith-specific `affects` and `interface-impact` fields. Copy this file to NNNN-short-kebab-title.md. Do not edit it in place. See AGENTS.md for the full convention.
-->

# ADR-NNNN: Short title of the decision
## Context and problem statement
What is the situation, and what problem or requirement forces a decision now? Two to four sentences. State the architecturally significant requirement plainly.

## Decision drivers
- Driver 1 (e.g. must stay deployable as independent projects)
- Driver 2 (e.g. minimize coupling between components)
- Driver 3

## Considered options
- Option A
- Option B
- Option C

## Decision outcome
Chosen option: "Option X", because ...

State the decision in one or two sentences first, then the justification.

### Consequences
- Good: ...
- Bad: ...
- Neutral: ...

### Confirmation
How do we verify the decision is upheld over time? Prefer automated checks. Examples: `go run ./tools/poly check` passes, a test asserts the rule, the compiler rejects the import, CI fails on violation.

## Interface notes
Fill this in only when `interface-impact` is `new` or `breaking`. Describe the public contract being added or changed: which brick, the exported names or signatures, and what consumers must do to adapt. For a breaking change, note the migration path and which projects need updating.

## More information
Links to related ADRs, issues, the discussion that led here, or relevant sections of the Polylith / MADR docs.
