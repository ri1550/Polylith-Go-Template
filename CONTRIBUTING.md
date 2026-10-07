# Contributing
This repository keeps its own context: why things are the way they are, what each part does, how to work on it. Small automated checks keep that context honest. You do not need to memorise the process; the checks guide you and the coding agent does the paperwork (ADRs, tests, Examples, comments). This file is setup and the day-to-day loop. The reasoning is in `AGENTS.md`.

## One-time setup (after cloning)
1. Install Go 1.27 or newer: https://go.dev/dl/
2. Install golangci-lint (required; the hook refuses to run without it): https://golangci-lint.run/docs/welcome/install/
3. Turn the checks on for this clone:

       make hooks

That is it. `make all` runs everything by hand at any time.

## The everyday loop
1. Edit. Ask the agent; it follows `AGENTS.md`.
2. Commit. The hook runs: gofmt, `poly check`, `go vet`, `go test`, the ADR lint, the ADR index (regenerated and staged for you), the interface heads-up, golangci-lint. If something is wrong the commit stops and says what to fix.
3. Push your branch and open a pull request. CI runs the same checks, plus `-race`, `go mod tidy -diff`, the interface gate and `govulncheck`.
4. Merge when green. With branch protection on, the button stays locked until then.

Local is the fast warning. CI is the gate.

## When a check stops you
Read the message, do the fix, try again.
- **gofmt**: run `gofmt -w <file>`, `git add`, commit again.
- **Polylith check**: a rule in `AGENTS.md` > Polylith rules was broken. The message names the import and the rule.
- **go vet / build**: the code does not type-check. The message names file and line.
- **go test**: a test or an Example failed. The output names it.
- **ADR lint**: a record under `docs/adr/` has a bad field, no title, or a `superseded by` pointing at nothing. Warnings (names that "do not exist today", a `YYYY-MM-DD` date, `[you]`) are informational: `affects` is historical, and a record is dated when it is adopted.
- **ADR index**: nothing to do; the hook regenerated and staged it.
- **Interface change heads-up** (local, never blocks): the hook printed the exact exported-API lines that changed and which bricks no ADR names. CI will block a pull request that leaves them unrecorded. Record it now: `go run ./tools/poly adr new "<title>"` and set `affects`, or use a marker (next item).
- **Interface changes are recorded** (CI, blocks): for each named brick, add an ADR whose `affects` names it, reference an existing one in a commit message (`ADR-0012: ...`), or add `[interface-impact: none]` (no consumer can observe it, including a brick nothing uses yet) or `[interface-impact: new]` (additive, ADR declined) to a commit message. A breaking change always needs an ADR.
- **golangci-lint**: the message names file, line and linter. "not on PATH" means install it or add `$(go env GOPATH)/bin` to your PATH.
- **go mod tidy -diff** (CI): run `go mod tidy` and commit `go.mod` and `go.sum`.

## Setting up the repository (admin, once)
On a fresh clone, from the workspace root:

    make adopt MODULE=github.com/<org>/<repo> MAKERS="<names or roles>"

It prints its plan and then: rewrites the module path everywhere; deletes the example bricks (`components/greeting`, `bases/api`, `projects/hello`) and the marked example block in `README.md`; deletes the template's baseline tag; dates ADRs 0001 to 0010 today and sets their `decision-makers` to `MAKERS` (adopting the template adopts its decisions; omit `MAKERS` to do that later by hand); regenerates the ADR index. `go run ./tools/poly adopt --module ... --dry-run` shows the plan without changing anything; `--keep-examples` keeps the example bricks.

Then:
1. `make all` (green with zero bricks), review `git status`, and commit with the message `adopt` printed. It ends in `[interface-impact: none]`, which is honest: the removed example bricks had no consumers.
2. Replace the copyright holder or the license in `LICENSE`.
3. Tag your first known-good state later with `git tag -a stable-1 -m "..."`; `poly diff` uses the newest `stable-*` tag.
4. Branch protection (GitHub): Settings > Branches > Add rule for `main`; require a pull request; require the **checks** workflow to pass. This is what makes CI a gate rather than advice.

## Why all of this
Context that lives next to the code and is kept true by checks is worth more than documents that drift. `AGENTS.md` has the map of where every kind of context lives; `docs/adr/` has the decisions.
