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

# ADR-0006: Enforce the context system with automated checks
## Context and problem statement
ADR-0005 records how context lives in the codebase, and `AGENTS.md` describes the process for keeping it accurate. But a written process that relies on people, and on AI agents, remembering to follow it will drift, especially with junior or fast-moving contributors. We need the context system to be enforced, not just described.

## Decision drivers
- The process must hold even when the contributor does not know or recall it.
- Feedback should arrive fast, ideally before a bad change is even committed.
- There must be a gate that cannot be skipped, to actually protect `main`.
- Contributors should need nothing beyond Go and git on their machine.
- Best Simple System for Now: the least machinery that makes the correct path the easy one and blocks the wrong one. No heavyweight policy engine.

## Considered options
- Rely on code review alone
- Rely on `AGENTS.md` and the agent's good behavior alone
- Local hooks only
- CI checks only (server)
- Both layers: local hooks plus CI gated by branch protection
- For the local layer: a plain git hook script, the `pre-commit` framework (an extra runtime to install), or `lefthook` (an extra binary)

## Decision outcome
Chosen option: Both layers. A plain git pre-commit hook (`.githooks/pre-commit`, enabled per clone with `make hooks`) gives fast, local feedback: a warning or a blocked commit at the keyboard. CI re-runs the same checks on GitHub, and branch protection makes passing them a requirement to merge. The checks are subcommands of the in-repo tool (ADR-0008): the decision log lint and brick-name validation (`poly adr lint`), ADR index generation (`poly adr index`), and interface-change recording (`poly interface`). The hook ends with a bookkeeping reminder that points to the pre-commit checklist in `AGENTS.md`, so the agent sees it in context before the commit lands.

Both layers over the alternatives because review and good intentions do not scale and are exactly what fails with juniors; a local hook alone can be bypassed (`--no-verify`) and lives only on each machine; CI alone gives slow feedback and lets a broken change get committed and pushed before catching it. Together, the local layer keeps most problems from ever being committed, and the server layer is the unskippable gate.

A plain hook script over `pre-commit` or `lefthook` because it needs no install beyond git, Go and the linter, which keeps the setup to one `make hooks`. The price is losing the generic fixers those tools bundle (trailing whitespace, YAML syntax); `gofmt` covers Go files, and the rest is not worth a dependency for now.

Code quality checks (`go build`, `go vet`, `go test`, `golangci-lint`) follow the same two-layer approach and are covered in ADR-0007.

### Consequences
- Good: the context system is enforced regardless of who or what makes a change; the hard gate (branch protection) cannot be skipped.
- Good: fast local feedback reduces failed CI runs.
- Bad: a small amount of setup per contributor (`make hooks`) and one repository setting (branch protection) that an admin must enable. If branch protection is not enabled, CI advises but does not block.
- Neutral: the interface-change check is heuristic about *recording*, not about detection. Detection is exact (it diffs the exported API). Recording was first satisfied by any numbered ADR file in the pull request; ADR-0010 tightened it to an ADR whose `affects` names the changed brick, an `ADR-NNNN` reference to one that does, or a `[interface-impact: none|new]` marker. The markers are self-reported; code review is the backstop against misuse.

### Confirmation
The `checks` workflow runs in CI on every pull request and is required by branch protection on `main`. The checks enforce themselves: a misconfiguration shows up as a failing run. `poly adr lint` validates every ADR's filename, front matter, field values and unique numbering; names in `affects` are historical and only warned about (ADR-0010). `poly adr index` keeps `docs/adr/index/` current locally; it runs in the hook but is not enforced in CI to avoid merge conflicts on parallel branches. `poly interface --mode ci` blocks a merge that changes a brick's exported API without a recorded decision. The hook prints the reminder last on every run. `CONTRIBUTING.md` documents the per-contributor and admin setup.

## More information
- ADR-0005 (the context system this enforces)
- ADR-0007 (code quality checks that use the same two-layer approach)
- ADR-0008 (the tool that implements the checks)
- `CONTRIBUTING.md` (setup and day-to-day workflow)
- `tools/poly/README.md` (what each check does)
- `AGENTS.md` (the process these checks enforce)
- Best Simple System for Now (BSSN): https://dannorth.net/blog/best-simple-system-for-now/
