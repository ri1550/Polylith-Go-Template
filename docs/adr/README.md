# Architecture Decision Records
The single decision log for the workspace. Every architecturally significant decision is a numbered, immutable Markdown file here. Create one with `go run ./tools/poly adr new "<title>"`.

## Why one central log
- Significant decisions usually span bricks or the whole workspace; they have no single brick home.
- Bricks are split, merged, renamed and deleted; records must outlive the current shape of the code.
- One sequence is one greppable log and works with ADR tooling (MADR, adr-tools, log4brains).

Per-brick discoverability comes from the `affects` field and the generated index under `index/` (one view per brick or project, each including the workspace-wide ADRs). A one-line `See ADR-NNNN.` in a brick's package comment points the other way.

## Conventions
- Filenames `NNNN-short-kebab-title.md`; numbers sequential, never reused.
- Immutable. To change a decision, write a new ADR and set the old one's status to `superseded by ADR-NNNN`. The status line is the only edit allowed.
- `affects` is historical: it may name a brick that does not exist yet or no longer exists. The lint warns, never blocks, and you never edit an old ADR to follow a rename.
- `interface-impact`: `none` | `new` | `breaking`, meaning whether the decision changes a brick's exported API. A `breaking` decision is the one that must be recorded here, because it ripples to every project that consumes the brick.
- Status lifecycle: `proposed` → `accepted` → `deprecated` | `superseded by ADR-NNNN`. An ADR drafted by an agent stays `proposed` until a person accepts it; `date` and `decision-makers` are filled in at that point.
- ADRs are started by the developer. An agent drafts one only when asked, and never offers one unprompted. A choice that is still open is an `OPEN:` line in `docs/spec/`, not a proposed ADR (see `docs/spec/README.md`, "Decisions", and ADR-0005).
- `0000-adr-template.md` is the template; `poly adr new` copies it. Do not edit it in place.

## Enforcement
Where a decision can be checked by tooling, say how in its `Confirmation` section. The workspace offers `go run ./tools/poly check` for layout and dependency rules, `poly interface` for surface changes, and the compiler for `internal/`.
