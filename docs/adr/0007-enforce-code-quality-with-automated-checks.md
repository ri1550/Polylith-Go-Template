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

# ADR-0007: Enforce code quality with automated checks
## Context and problem statement
The codebase needs to stay architecturally sound, compiling and lint-clean as it grows. Relying on code review alone to catch broken brick boundaries, failing tests, or sloppy code does not scale and is exactly what fails under time pressure. We need code quality to be enforced at the gate.

## Decision drivers
- Brick boundaries and the dependency direction must be structurally enforced, not just documented.
- Behavioral guarantees must be verified on every change, not assumed.
- Common mistakes (unchecked errors, unused code, vet findings, formatting) should be caught before code review.
- Tests must exercise bricks through their public interface, since that is the contract other bricks see.
- Best Simple System for Now: use established tools, wire them into the same two-layer gate as ADR-0006.

## Considered options
- Rely on code review alone
- `go vet` and `gofmt` only
- `go vet` plus `staticcheck` run separately
- `golangci-lint` as the single linter runner with its standard set
- Any of the above, locally by convention only, hooks only, CI only, or both layers

## Decision outcome
Chosen option: Both layers, following the same approach as ADR-0006. The checks are `go build ./...` and `go vet ./...` (the compiler is the type checker; vet catches the classic mistakes), `go test ./...` (behavioral guarantees, in CI), `go run ./tools/poly check` (brick boundaries and layout), and `golangci-lint` with its standard linter set (`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`) plus `testpackage`, with `gofmt` and `goimports` as formatters. `golangci-lint` is configured in `.golangci.yml`; the pre-commit hook runs it when it is installed and CI always runs it.

`golangci-lint` over separate tools because it bundles the ones we would otherwise run one by one, caches well, and has first-class editor and GitHub Actions integration, so inline feedback matches what CI blocks. `testpackage` is enabled because tests in an external `_test` package can only reach the exported API, which makes "test at the interface" a checked rule rather than advice. The in-repo tooling under `tools/` is exempt from `testpackage` because it is `package main` and tests its own helpers.

### Consequences
- Good: architectural violations, compile errors, test failures and lint findings are all caught before code reaches `main`.
- Good: fast local feedback reduces failed CI runs; `go build` and `go vet` need nothing beyond the Go toolchain.
- Good: editor and gate use the same linter, so inline feedback matches what CI blocks.
- Bad: `golangci-lint` is an extra install for local use. The hook degrades gracefully (skips with a notice) when it is missing; CI still blocks.
- Neutral: the standard linter set is a starting point; stricter linters can be enabled as the codebase grows.

### Confirmation
`go run ./tools/poly check` enforces brick boundaries on every commit and in CI. `go test ./...` runs the test suite in CI. `golangci-lint run` lints the whole module, configured via `.golangci.yml`. All of them run in CI as part of the `checks` workflow required by branch protection on `main`.

## More information
- ADR-0002 (the Polylith architecture `poly check` enforces)
- ADR-0006 (context system checks that use the same two-layer approach)
- ADR-0008 (the in-repo tooling)
- `CONTRIBUTING.md` (setup and day-to-day workflow)
- golangci-lint: https://golangci-lint.run/
- Best Simple System for Now (BSSN): https://dannorth.net/blog/best-simple-system-for-now/
