# poly: the workspace tool
There is no Polylith tool for Go, so this program provides the commands the process in `AGENTS.md` depends on. Run it from the workspace root with `go run ./tools/poly <command>`. It is part of the module, so `go vet`, `go test` and the linter cover it like any other code. Decisions: ADR-0008 (build it in-repo), ADR-0009 (project rule), ADR-0010 (exact gates).

The rules themselves live in `tools/polylith`, one definition shared by two front ends: `poly check` (summary, cross-package rules, the CI step) and `tools/polyvet`, a `go vet` analyzer that reports the same per-package rules at the offending import with file and line. `make vet` runs `go vet ./...` and then the analyzer; `-vettool` replaces the standard analyzers for its invocation, which is why both run.

| Command | What it does | Blocks? |
|---------|--------------|---------|
| `check` | Layout and dependency rules: components import only components; bases import components, never bases; a project is `package main`, one Go file, imports at least one base; bricks are used through their root package; nothing imports `projects/`, `development/` or `tools/`; brick names unique; no Go code outside the layout; every package loads. | Yes, locally and in CI |
| `deps` | Per brick: what it uses and what uses it (bricks and projects). Per project: what it wires directly, every brick it ships, and every module from outside the workspace it links. | Never |
| `diff [--since <ref>]` | Bricks and projects changed since the newest `stable-*` tag (or `--since`), each labelled `new`, `changed` or `removed`, including uncommitted work, and the projects affected through dependencies. Says so when the baseline is the template's own tag. | Never |
| `interface --mode pre-commit` | Diffs the exported API of every brick root package between HEAD and the staged tree. Always prints the `-`/`+` lines, labels each brick `new`, `removed`, `changed`, `renamed` or `renamed, changed` (a brick git saw move is one entry, and an ADR naming either its old or new name covers it), and names the bricks that no staged ADR covers. | No |
| `interface --mode ci` | Same diff between the merge base with `origin/<BASE_REF>` and HEAD. Fails unless every changed brick is covered: an ADR in the diff whose `affects` names it (or `*`), a commit message referencing an existing `ADR-NNNN` that does, or a `[interface-impact: none]` or `[interface-impact: new]` marker in a commit message. `breaking` is not a marker. | CI: yes |
| `interface --between <old> <new>` | The same diff between any two refs. | Never |
| `adopt --module <path>` | One-time, on a fresh clone of the template: rewrites the module path in every Go, Markdown, YAML and go.mod file; deletes the example bricks and the README's marked example block (`--keep-examples` to skip); deletes the template's baseline tag; with `--decision-makers "..."` dates the shipped ADRs today and attributes them; regenerates the index. Prints the plan first; `--dry-run` stops there. | Never |
| `adr new <title>` | Creates the next ADR from the template: next number, slug, today's date, title. | Never |
| `adr lint` | Fails on: bad filename, duplicate number, unparsable or missing front matter fields, the template's title left in place, invalid status or `interface-impact`, `superseded by` pointing at a missing ADR, malformed `affects`. Warns on: names in `affects` that do not exist today (historical by design), a `YYYY-MM-DD` date or `[you]` decision-makers (filled in at adoption), gaps in numbering. | Yes, locally and in CI |
| `adr index` | Regenerates `docs/adr/index/`: one view per brick or project, each including the workspace-wide (`*`) ADRs, plus a README. The hook stages it. | Refreshes files locally; not enforced in CI (parallel branches would conflict on generated files) |
| `spec lint` | Validates `docs/spec/`: five headings per domain in order, contiguous rule numbers, every `<domain> rule N` citation in a `_test.go` file under `components/` or `bases/` and in `docs/spec/` resolves, no glossary term redefined, links under `docs/` resolve. Validates `docs/queue.yaml`: shape, unique kebab-case ids, rule ranges naming a real domain, `after` naming an earlier step. | Yes, locally and in CI |
| `spec status` | Derives what works and what is next: per domain, rules with a citing test and open questions; the queue with each step's state (waiting on spec, ready, in progress, done). Never fails. A SessionStart hook (`.claude/settings.json`) prints it into the agent's context. | Never |

## What "public interface" means here
A brick's surface is the set of exported identifiers in its root package, not a file. `interface` reads each brick's root-package sources straight from git (`git show <tree>:<path>`), parses them with `go/parser`, and renders one line per declaration:
- exported functions and methods on exported types, without bodies or comments;
- exported types, with unexported struct fields removed;
- exported constants with their values (a changed constant is a changed contract);
- exported variables without their initializers (a sentinel error's identity is the contract, its text is not);
- unexported types that appear in any of those signatures, and their exported methods, since callers can hold and use them.

Two trees give two sorted lists; the difference is the change. Renaming a local, reformatting, or editing a comment does not register.

## What `check` and `polyvet` divide between them
`poly check` loads the whole module with `go list` and so can check cross-package rules (unique brick names, every package loads) and print the summary; it is the CI step. `polyvet` sees one package at a time and reports the import and layout rules where they occur, so a violation shows up like any vet finding. Both call the same functions in `tools/polylith`.

## What `check` assumes
`check` reads the package graph from `go list -e`, which parses only import headers. It reports packages that fail to load (import cycles, missing packages) but is not a compiler; the hook runs it first so the Polylith rule speaks before `go vet` does on a cycle.

## Dependencies
`go.yaml.in/yaml/v3` for ADR front matter and the queue; `golang.org/x/tools` for the `polyvet` analyzer. Everything else is the standard library, `go list` and `git`.

## Editing a check
Every rule has a test in this directory; add one for any rule you add. If you change what a check enforces, update `AGENTS.md` and `CONTRIBUTING.md` in the same commit, and tell the developer so they can decide whether it deserves an ADR.
