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
Do these in order on a fresh clone, then commit.
1. **Module path.** Replace `github.com/myorg/workspace` everywhere (portable: `-i.bak` then delete the backups):

       grep -rl --include='*.go' --include='go.mod' --include='*.yml' --include='*.md' 'github.com/myorg/workspace' . \
         | xargs sed -i.bak 's#github.com/myorg/workspace#github.com/<org>/<repo>#g' && find . -name '*.bak' -delete

2. **Example bricks.** Delete `components/greeting`, `bases/api`, `projects/hello`, and the marked block in `README.md`. The layout directories keep their `.keep` files. This removes two public surfaces nothing uses, so commit it with `[interface-impact: none]` in the message.
3. **Inherited tag.** If you cloned instead of using GitHub's **Use this template** button: `git tag -d stable-template-baseline`. Tag your own first known-good state later with `git tag -a stable-1 -m "..."`; `poly diff` uses the newest `stable-*` tag.
4. **LICENSE.** Replace the copyright holder, or the license.
5. **ADRs 0001 to 0010** are the decisions this template is built on. Adopting the template adopts them: set each one's `date:` to the day you adopt it and replace `[you]` in `decision-makers:` with the names or roles that did. Supersede any you later reverse with a new ADR.
6. **Branch protection** (GitHub): Settings > Branches > Add rule for `main`; require a pull request; require the **checks** workflow to pass. This is what makes CI a gate rather than advice.

`make all` must be green after step 2 with zero bricks; it is.

## Why all of this
Context that lives next to the code and is kept true by checks is worth more than documents that drift. `AGENTS.md` has the map of where every kind of context lives; `docs/adr/` has the decisions.
