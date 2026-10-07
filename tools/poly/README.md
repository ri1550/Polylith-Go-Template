# poly: the workspace tool
There is no Polylith tool for Go, so this small program provides the commands the process in `AGENTS.md` depends on. Run it from the workspace root with `go run ./tools/poly <command>`. It is part of the module, so `go build`, `go vet`, `go test` and the linter cover it like any other code. See ADR-0008 for the decision.

| Command | What it does | Blocks? |
|---------|--------------|---------|
| `check` | Validates the layout and the brick dependency rules: components do not import bases, bases do not import other bases, projects import only bases, bricks are used through their root package, nothing imports `development/` or `tools/`, projects are `package main`, bricks are not, brick names are unique, no Go code outside the layout. | Yes, locally and in CI |
| `deps` | Shows what each brick uses, what uses it, and every brick each project pulls in transitively. | Never |
| `diff [--since <ref>]` | Shows which bricks and projects changed since the latest `stable-*` tag (or `--since`), including uncommitted changes, and which projects are affected through their dependencies. Mark a known-good point with `git tag stable-1`. | Never |
| `interface --mode pre-commit` | Diffs the exported API of every brick root package between HEAD and the staged tree. Prints what changed and a heads-up if no ADR is staged. | No |
| `interface --mode ci` | Same diff between the base branch (`BASE_REF`, `GITHUB_BASE_REF`, or `main`) and HEAD. Fails unless the PR includes an ADR file, references `ADR-NNNN` in a commit message, or notes `[interface-impact: none]`. | CI: yes |
| `adr lint` | Validates every ADR's filename, front matter, status, `interface-impact`, and `affects`, checks for duplicate numbers, and verifies that names in `affects` exist under `components/`, `bases/` or `projects/`. | Yes, locally and in CI |
| `adr index` | Regenerates `docs/adr/index/`, a per-brick view of which ADRs affect which brick. | Refreshes files locally; not enforced in CI (avoids merge conflicts on parallel branches) |

## How the interface check works
A brick's public interface in Go is the set of exported identifiers in its root package, not a single file. The check reads each brick's root-package sources straight from git (`git show <tree>:<path>`), parses them with `go/parser`, and renders every exported declaration on one line with bodies, comments and unexported struct fields stripped. Two trees give two sorted lists; the difference is the interface change. Renaming a local variable, reformatting, or adding a doc comment does not register. Adding, removing or changing an exported function, method, type, field, constant or variable does.

## What `check` assumes
`check` reads the package graph from `go list`, which parses only the import headers. It reports packages that fail to load (import cycles, missing packages) but it is not a compiler: the hook and CI run `go build ./...` first, and `check` assumes that passed.

## Dependencies
`go.yaml.in/yaml/v3` for ADR front matter. Everything else is the standard library, `go list` and `git`.

## Editing a check
These are intentionally simple (Best Simple System for Now). Every rule has a test in this directory; add one for any rule you add. If you change what a check enforces, update `AGENTS.md` and `CONTRIBUTING.md` to match, and consider whether the change deserves an ADR.
