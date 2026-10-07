---
status: accepted
date: 2026-10-07
decision-makers: [workspace owner]
consulted: [template author]
informed: []
affects:
  components: ["*"]
  bases: ["*"]
  projects: []
interface-impact: none
---

# ADR-0011: Example functions are the usage layer
## Context and problem statement
ADR-0005 assigns "how to call a brick" to its tests. Tests assert behaviour but do not read as usage: a newcomer following the reading protocol gets signatures from `go doc` and then has to infer the calling convention from assertions. Go has a native layer between the two that the template did not use.

## Decision drivers
- Usage documentation must be enforced or it drifts.
- It should appear where the reader already looks: `go doc`.
- BSSN: no new tooling, no new file type.

## Considered options
- Tests only (status quo)
- Usage snippets in the package comment (unenforced prose)
- Example functions in the external `_test` package

## Decision outcome
Chosen option: Example functions. Each brick with a non-trivial calling convention carries an `ExampleXxx` function in its `_test.go` file with an `// Output:` comment. `go test` compiles and runs it and fails if the output changes; `go doc` prints it next to the symbol (`go doc -ex` lists them; `go doc <pkg> ExampleXxx` prints the source). It lives in the external `_test` package, so it uses only the exported API, exactly like a consumer.

The context layer table in AGENTS.md now reads: layer 2 is "how to call it" (Example functions) and "how it behaves" (tests), both enforced by CI.

### Consequences
- Good: usage is executable documentation; a stale example fails the build.
- Good: nothing new to install or learn beyond a Go convention.
- Bad: one more thing to write per brick. Not mandatory for trivial bricks (BSSN); required when the calling convention is not obvious from the signature.

### Confirmation
`go test ./...` runs every Example with an `// Output:` comment. The example bricks ship one each.

## More information
- ADR-0005 (the context system this refines)
- Go testing package, Examples: https://pkg.go.dev/testing#hdr-Examples
