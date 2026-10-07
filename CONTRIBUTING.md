# Contributing
This repository keeps its own context: why things are the way they are, what each part does, and how to work on it. A few small automated checks keep that context honest so the codebase stays understandable as it grows.

You do not need to memorize how it all works. The checks guide you, and the AI agent does most of the paperwork (writing ADRs, tests, and notes) for you. Your job is mostly to follow the setup below once, then answer the agent's questions as you work.

If you want the full reasoning, read `AGENTS.md`. This file is just how to get set up and what to do day to day.

## One-time setup (do this once after cloning)
1. **Install Go** 1.27 or newer: https://go.dev/dl/ (or your version manager).

2. **Turn the checks on for this clone:**

       make hooks

   This points git at the hook script in `.githooks/`, so the checks run every time you commit. Nothing else to install.

3. **Optional: install golangci-lint** so the linter also runs locally: https://golangci-lint.run/docs/welcome/install/. If it is missing, the hook says so and skips it; CI runs it regardless.

That is it. To run everything by hand at any time:

    make all        # check + test + lint

## The everyday loop
1. **Edit your code.** Ask the agent for help; it knows the rules in `AGENTS.md`.
2. **Commit.** When you run `git commit`, the checks run first. If something is wrong, the commit stops and tells you what to fix. Fix it and commit again.
3. **Push** your branch to GitHub.
4. **Open a pull request.** GitHub runs the same checks on its servers, plus the tests.
5. **Merge** once the checks are green. With branch protection on (see below), the button stays locked until they pass.

The checks on your machine are the fast warning. The checks on GitHub are the real gate. Same checks, two moments.

## When a check stops you, here is what it means
You do not need to understand the internals. Read the message, do the fix, try again.

- **"gofmt" failed.** A Go file is not formatted. Run `gofmt -w <file>`, `git add` it, commit again.

- **"go build" or "go vet" failed.** The code does not compile, or vet found a classic mistake. The message names the file and line. Ask the agent to fix it, or fix it yourself and commit again.

- **"Polylith check" failed.** A brick boundary was crossed: a component importing a base, a project importing a component directly, an import of a brick's subpackage instead of its root package, or Go code outside the layout. The message names the import and the rule. Ask the agent to help move the code to the right place.

- **Tests failed (on GitHub).** One or more tests are failing. The output names the file and test. Fix the code or the test, then push again. Run `go test ./...` locally first.

- **"ADR front matter lint" failed.** A decision record under `docs/adr/` is missing a field or has an invalid value. The message says which file and which field. Copy `docs/adr/0000-adr-template.md` if you are starting one from scratch, or ask the agent to fix the fields.

- **"Interface change heads-up" (local).** You changed a brick's public interface: an exported function, type, field or constant in its root package. The hook prints exactly what changed. This is a warning, not a block. If it is a real change to what other code depends on, ask the agent to draft an ADR for it. You will need one before the pull request can merge.

- **"Interface change must be recorded" (on GitHub) failed.** Same situation, but now it blocks the merge. Do one of: add an ADR describing the change (best, the agent can draft it), mention an ADR in a commit message like `ADR-NNNN: ...`, or if the edit was not really a contract change, add `[interface-impact: none]` to a commit message. Note: `[interface-impact: none]` is self-reported — code review is the backstop against misuse.

- **"golangci-lint" failed.** The linter found something: an unchecked error, unused code, a test in the wrong package. The message names the file, line and linter. Ask the agent to fix it, or fix it yourself and commit again.

- **"Refresh the ADR index" stopped the commit.** The check updated the generated index under `docs/adr/index/`. Nothing is wrong. Just `git add docs/adr/index` and commit again. (This only runs locally; CI does not enforce index freshness to avoid merge conflicts on parallel branches.)

## If you are setting up the repository (admin, one time)
Before anything else, replace the placeholders left by the starter:
- **Module path**: change `github.com/myorg/workspace` in `go.mod` and in every import to your own path (see the Namespace section of `README.md` for a one-liner).
- **ADR dates**: fill in the `date:` field in each ADR under `docs/adr/` with the date you are formally adopting the decision.
- **ADR decision-makers**: replace `[you]` in each ADR's front matter with the actual names or roles.
- **Example bricks**: delete `components/greeting`, `bases/api` and `projects/hello` once you have your own bricks.
- **LICENSE**: replace the copyright holder, or the license, with your own.
- **Inherited tags**: if you cloned or forked this repository instead of using GitHub's **Use this template** button, delete the template's own baseline tags. Otherwise `poly diff` compares your bricks against the template's baseline instead of your own.

      git tag -d $(git tag -l 'stable-*')

Then, make the checks a hard gate by configuring GitHub branch protection:
1. Go to **Settings > Branches**.
2. Under **Branch protection rules**, click **Add rule**.
3. Branch name pattern: `main`.
4. Tick **Require a pull request before merging**.
5. Tick **Require status checks to pass before merging**, then select the **checks** workflow.
6. Click **Create** (or **Save changes**).

Now no change can reach `main` without passing the checks, no matter who or what made it. This is the piece that actually enforces the process; everything else is guidance that makes following it easy.

## Why all of this
The short version: context that lives next to the code and stays accurate is worth far more than docs that drift. The checks stop the context from drifting. The full reasoning, and the map of where every kind of context lives, is in `AGENTS.md`. The decisions behind the setup are in `docs/adr/`.
