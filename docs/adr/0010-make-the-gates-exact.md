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

# ADR-0010: Make the context system's gates exact
## Context and problem statement
ADR-0006 and ADR-0007 set up the two-layer enforcement. In use, the gates turned out to be present but loose: the interface-change check passed on the presence of any ADR file or any commit message containing `adr-`, so one ADR about anything unlocked every interface change in a pull request; the ADR lint validated `affects` against today's directories, so renaming or deleting a brick forced an edit to an immutable record, and that forced edit then satisfied the interface check; the hook skipped the linter silently when it was not on `PATH`; the hook ran no tests; `go build ./...` in the hook wrote a stray binary into the repository once the example project was deleted; and a scratch program in `development/` blocked every commit because it was part of `./...`. Nothing checked `go.mod` tidiness, data races, or known vulnerabilities. The context system itself had a gap of the same kind: "how to call a brick" was assigned to tests, which assert behaviour but do not read as usage, so the one layer a newcomer needs most was unenforced prose or inference.

## Decision drivers
- A gate that can be satisfied by accident is not a gate.
- ADRs may be written before the brick they name exists, so `affects` is historical by nature and cannot be validated against the filesystem.
- Contributors should get the same answer locally and in CI.
- Usage documentation must be enforced or it drifts, and it should appear where the reader already looks: `go doc`.
- BSSN: tighten what exists; add only checks that are one line and cost seconds; no new tooling or file types.

## Considered options
- Leave the gates as they are and rely on review
- Tighten the interface check to per-brick linkage; keep the rest
- Tighten the interface check, make `affects` historical, make lint mandatory, add the missing CI checks, and take `development/` out of `./...`
- For usage: keep tests only; usage snippets in the package comment (unenforced prose); or Example functions in the external `_test` package

## Decision outcome
Chosen option: the full set.

**Interface check is a linkage check.** For every brick whose exported surface changed, the pull request must contain an ADR whose `affects` names that brick (or `*`), or a commit message that references an existing `ADR-NNNN` that does, or a commit message carrying `[interface-impact: none]` or `[interface-impact: new]`. `none` means no consumer can observe the change, which includes adding or removing a brick that no project uses yet. `new` means the change is additive and an ADR was deliberately declined. `breaking` is not a marker: a breaking change needs an ADR file. The check always prints the rendered surface diff, labels each brick `new`, `removed` or `changed`, and names the bricks that are not recorded. `poly interface --between <old> <new>` prints the same diff for any two refs.

**`affects` is historical.** The ADR lint warns when a name in `affects` does not exist on disk and never fails on it. It warns on template placeholders left in a record (`YYYY-MM-DD`, `[you]`) and on gaps in the numbering, and it fails on a `superseded by ADR-NNNN` whose target does not exist.

**The hook runs the full local set.** `poly check` first (so the Polylith rule speaks before the compiler on an import cycle), then `go vet ./...` instead of `go build ./...` (vet type-checks every package and never writes a binary), `go test ./...`, the ADR lint, the ADR index refresh (the hook stages the generated files itself), the interface heads-up, and golangci-lint. The hook adds `$(go env GOPATH)/bin` to `PATH` and fails if golangci-lint is still missing; the linter is a required install.

**CI adds** `go mod tidy -diff`, `go test -race -shuffle=on`, and `govulncheck`.

**`development/` is ignored** through the `go.mod` `ignore` directive (Go 1.25), so scratch programs are outside `./...` for every tool and still run by explicit path.

**Example functions are the usage layer.** Each brick whose calling convention is not obvious from its signature carries an `ExampleXxx` function in its `_test.go` file with an `// Output:` comment. `go test` compiles and runs it and fails if the output changes; `go doc` prints it next to the symbol (`go doc -ex` lists them). It lives in the external `_test` package, so it uses only the exported API, exactly like a consumer. The context layer table in AGENTS.md reads: "how to call it" is Example functions, "how it behaves" is tests, both enforced by CI. Example functions over prose snippets because a stale example fails the build and a stale snippet does not; over tests alone because an assertion is not a calling convention.

### Consequences
- Good: an unrecorded interface change cannot merge; a recorded one is linked to the brick it touches.
- Good: renaming or deleting a brick never forces an edit to the decision log.
- Good: local and CI results match; the stray-binary and silent-skip paths are closed.
- Bad: golangci-lint becomes a required install.
- Bad: the hook is slower by one test run. On a small workspace that is under a second with the test cache.
- Neutral: `[interface-impact: none|new]` remains self-reported; code review is the backstop, and the always-printed diff makes misuse visible.
- Neutral: an Example is one more thing to write per brick. Not mandatory for trivial bricks (BSSN); required when the calling convention is not obvious from the signature.

### Confirmation
`go test ./tools/poly/...` pins every rule above, and `go test ./...` runs every Example with an `// Output:` comment; the example bricks ship one each. The hook and CI call the tool, so a regression fails the gate it implements. `tools/poly/README.md` lists what each command checks.

## More information
- ADR-0006 and ADR-0007 (the gates this tightens)
- ADR-0008 (the tool)
- Go 1.25 `ignore` directive: https://go.dev/doc/go1.25
- Go testing package, Examples: https://pkg.go.dev/testing#hdr-Examples
