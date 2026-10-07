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

# ADR-0008: Build the Polylith tooling in the repository
## Context and problem statement
This workspace's process leans on tooling: validating brick boundaries, showing dependencies and blast radius, and detecting public interface changes. Polylith's official tooling exists only for Clojure; there is no Polylith tool for Go. We need to decide where those capabilities come from.

## Decision drivers
- The brick rules must be checked by a machine, not by convention (ADR-0007).
- An interface change must be detected precisely: Go has no single interface file, so a path heuristic is not available.
- Contributors should need nothing beyond Go and git.
- BSSN: the smallest tool that covers what the process needs today; no scaffolding for commands nobody runs yet.

## Considered options
- No tooling: document the rules and rely on review
- Express the import rules declaratively with `golangci-lint`'s `depguard` and detect interface changes by file path
- Use `golang.org/x/exp/cmd/apidiff` for interface changes plus `depguard` for boundaries
- A small Go program in the repository (`tools/poly`) that implements the checks over `go list` and `git`
- Port the Clojure `poly` tool's command set wholesale

## Decision outcome
Chosen option: a small Go program at `tools/poly`, run with `go run ./tools/poly <command>`, in the same module and checked by the same compiler, tests and linter as everything else. It provides `check` (layout and import-direction rules over `go list -json`), `deps` (what each brick uses, what each project pulls in), `diff` (bricks and projects changed since the last `stable-*` tag), `interface` (the exported API of every brick root package, diffed between two git trees), `adr lint` and `adr index`.

The interface check parses each brick's root package with `go/parser` at two git trees (via `git show`, no checkout needed), renders every exported declaration without bodies or comments, and reports the lines added or removed. It therefore reacts to exactly what consumers can see, including additive changes that `apidiff` would call compatible but that the process still wants recorded, and ignores renamed locals, reformatting and new comments. `apidiff` needs both versions type-checked and built, which is slower and heavier for the same question. `depguard` can express most of the import rules but not "a base may import its own subpackages and no other base", and file-path heuristics for interface changes would fire on every implementation edit in a root package. Porting the Clojure tool's full command set would carry over features (per-project dependency validation, packaging, profiles) that Go's single module makes unnecessary (ADR-0004).

Scaffolding (`create`) and an `info` overview were deliberately left out: a brick is a directory with one file, and `go doc`, `go list` and `poly deps` already answer the overview questions. Add them when a real need appears.

### Consequences
- Good: every rule the process depends on is executable, versioned with the code, and readable by anyone who can read Go.
- Good: interface detection is exact, so `[interface-impact: none]` is rarely needed and its misuse stands out.
- Bad: the tool is ours to maintain. At the time of writing it is about 1441 lines of source and 284 lines of tests, and it is the only Go-specific moving part in the template.
- Neutral: the tool depends on `go.yaml.in/yaml/v3` (the maintained fork of the archived `gopkg.in/yaml.v3`) for ADR front matter; that requirement sits in the shared `go.mod` but is linked only into the tool.

### Confirmation
`go test ./tools/poly/...` pins the rules: every import-direction rule has a rejecting case, the surface extraction has a fixture, and the ADR lint has failing fixtures. `tools/poly/README.md` lists each command. The hook and CI call the tool, so a regression in it fails the gate it implements.

## More information
- ADR-0002 (the architecture this tool enforces)
- ADR-0006 and ADR-0007 (the gates that call it)
- `tools/poly/README.md`
- The Clojure `poly` tool, the reference for the command vocabulary: https://polylith.gitbook.io/poly/
- apidiff: https://pkg.go.dev/golang.org/x/exp/cmd/apidiff
