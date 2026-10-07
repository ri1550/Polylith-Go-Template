# Spec
This directory is the behavior contract: what the product intends to do, in product terms, before and while the code exists. Each file is one domain of the product. The target, not the current state.

**Code implements numbered rules from this directory and nothing else.** An `OPEN:` line is not buildable. If a brick needs behavior that no rule covers, add an `OPEN:` line, say so, and stop. Never invent a rule to unblock yourself.

This directory is not a decision record (`docs/adr/` is) and not the build queue (`docs/queue.yaml` is). What works today is written nowhere: it is the set of rules with a passing test citing them, and `go run ./tools/poly spec status` prints it.

## Before touching a brick
Read the spec domain for that brick before its exported surface, its tests, or its package comment. The domain tells you what the brick is for. If you cannot tell which domain a brick belongs to, that is a question for the developer, not a guess.

## Domain lifecycle
A domain file exists only because the developer decided the product should have that domain. An agent never creates, renames, or deletes a domain file unasked.

| Event | Who decides | What the agent does |
|-------|-------------|---------------------|
| A domain is created | The developer, by asking for one. | Copies `TEMPLATE.md` to `<domain>.md`, fills it from what the developer said, asks the developer to confirm before it lands. |
| Brick work reveals a domain is missing | The developer. | Says so and stops. Does not draft the file to unblock the brick. |
| A domain is withdrawn | The developer. | Confirms first, then names the blast radius: `grep -rn "<domain> rule" --include='*_test.go' components bases` for tests that cite it, `grep -rn "<domain>" docs/` for glossary entries and queue steps that point at it. Deletes the file in the same change that fixes those. Writes nothing in its place; git history is the record. |

The filename stem is the domain's identity. Tests name it in rule citations and `docs/queue.yaml` names it in step rules. Renaming a domain is a withdrawal plus a creation and carries the same blast radius.

## Rule lifecycle
A behavior rule moves through these states. Adding an `OPEN:` line is open to anyone. Every other transition is the developer's call: the agent drafts and proposes, the developer confirms before it lands.

| State | Form | How it gets here |
|-------|------|------------------|
| Unsettled | An `OPEN:` line under Open questions | Anyone adds it. This is the one spec edit an agent makes without asking, when brick work hits a question the spec does not answer. |
| Settled | A numbered rule under Behavior rules | The developer settles the question. The rule is appended with the next number and the `OPEN:` line is deleted, in the same change. No tombstone. |
| Pinned | A numbered rule that a test cites | A test names the rule. From here the test is authoritative for detail and the rule sentence stays in the spec as the citation target. Elaboration moves to the test; the sentence does not. |
| Withdrawn | The rule's number, kept, with one line saying what replaced it: `3. Withdrawn. Replaced by rule 7.` or `3. Withdrawn. No replacement.` | The developer withdraws it. Tests that cite it change in the same change. |

Rules that hold across every state:
- Numbers are permanent. Never renumber, never reuse, never insert. A new rule takes the next number even if an earlier one was withdrawn.
- Wording may change. The number may not. If a test cites the rule, the test changes in the same change as the wording.
- Each `OPEN:` line is one question. Never a placeholder such as "to be drafted". A section with nothing to say says so in one line.
- Resolving an `OPEN:` line always deletes the line. What replaces it is either a new rule or nothing. Never a rule with the question left standing beside it.

## Decisions
An undecided choice with real alternatives is an `OPEN:` line that names the alternatives. That is where it stays until the developer decides.

ADRs in `docs/adr/` are for architectural decisions that shape the direction of the project, and only the developer starts one. An `OPEN:` line is not a pending ADR. Never draft or offer an ADR because a spec question looks like a decision; write the `OPEN:` line and leave it there. When the developer settles the question, the answer is a rule. If the developer also wants it recorded as an ADR, they will say so.

## The shape of a domain file
Every domain file has these five headings, in this order, and no others. An empty section says "Nothing yet." rather than being removed, because the lint checks the headings.

| Section | What it holds | What it never holds |
|---------|---------------|---------------------|
| Purpose | What this domain is for, in two or three single-line paragraphs. | Rules. Implementation. |
| Vocabulary | Terms used only inside this domain, one line each, `**Term** — definition.` | Any term that another domain also uses. Those go in [00-glossary.md](00-glossary.md) and are linked, never redefined. |
| Data | What the domain is about and what it carries, in product terms: the things, what they know about themselves, how they relate. | Tables, columns, types, storage. That is the brick's business. |
| Behavior rules | Numbered rules, `1.` upward, one rule per line. Each rule is a sentence a test could assert. | Rationale. Alternatives. Anything not yet settled. |
| Open questions | One question per line, each line starting `- OPEN:`. The prefix is at the start of the line so that `grep -rn "^- OPEN:" docs/spec/` finds questions and not prose that mentions them. | Anything settled. |

A rule reads as a sentence about product behavior, not about code. "An account has exactly one denomination" is a rule. "The accounts table has a denomination_id column" is not.

## How tests cite rules
A test cites a rule as `<domain> rule <N>` in the doc comment above the test function, using the filename stem as the domain: `// TestTransfer asserts accounts rule 3.` An Example function (`ExampleXxx` with an `// Output:` comment) is a test too and may cite a rule the same way. That exact form is what the lint greps for, so use nothing else. Citing a rule pins it (see the rule lifecycle above).

A rule that depends on another domain's rule cites it the same way, inline: "A transfer moves value between two accounts (accounts rule 2)." One grammar for tests and prose means one grep finds every dependent of a rule.

## Developing a domain
Rules come from the developer. Material the agent derives on its own is always an `OPEN:` line, never a rule. That one distinction is what keeps the spec a contract rather than a draft.

A new domain file lands with its Purpose and Data filled, its Open questions listing what the developer left unsettled, and only the rules the developer actually stated. A fresh domain with "Nothing yet." under Behavior rules is correct. A request from the developer may state several rules, and those land in the same change.

When asked to develop a domain, work this loop:
1. Read [00-glossary.md](00-glossary.md), then the domain file.
2. List the `OPEN:` lines. Add any the reading surfaced.
3. Take the `OPEN:` line the next brick will hit first and ask the developer about it. One question at a time (AGENTS.md).
4. Turn the answer into a rule, appended with the next number, and delete the `OPEN:` line, in the same change. Show the developer the rule sentence before it lands.
5. Repeat until the developer stops or the list is empty.

A domain is buildable as soon as the rules a brick needs exist. `OPEN:` lines elsewhere in the file do not block building what is settled. Building what an `OPEN:` line covers is what is blocked.

A rule sentence passes when it is one behavior, in the present tense, in product language, using glossary terms, and a test could assert it. "An account has exactly one denomination" passes. "Accounts should probably support multiple currencies eventually" is an `OPEN:` line wearing a rule's number.

## Shared terms and the glossary
[00-glossary.md](00-glossary.md) defines every term that more than one domain uses, once. A domain's Vocabulary section defines terms that only it uses. There is no third place.

A term becomes shared the moment a second domain needs it. That is the only trigger. Do not add a term to the glossary because it might be shared later.

Before writing any term, in either place:
1. Search for it and its likely synonyms across the directory: `grep -rni "<term>" docs/spec/`.
2. If the glossary has it, link to it. Do not restate the definition, even in shorter form.
3. If another domain's Vocabulary has it, it is now shared. Move the definition to the glossary and replace the local one with a link, in the same change as the new use.
4. If nowhere has it and only this domain needs it, it is local Vocabulary.
5. If nowhere has it and two domains need it, it goes in the glossary.

Glossary edits are the developer's call, because a glossary change alters the language every domain is written in. Draft the entry, show it, land it on confirmation. One name per concept: when a domain arrives with a new name for a term the glossary already has, the glossary's name wins and the new one is not written down.

A glossary entry is one line: `**Term** — one-sentence definition. See [<domain>.md](<domain>.md).` The link names the domain that owns the term's behavior. An entry carries no `OPEN:` line and no rule. If what a term means is unsettled, the term is not in the glossary yet, and the question is an `OPEN:` line in the domain that will own it.

Withdrawing a term follows the domain rule: `grep -rn "<term>" docs/` first, fix every use in the same change, write nothing in its place.

## Conventions
- Filenames: `<domain>.md`, short kebab-case, no numbers. The two non-domain files are `00-glossary.md`, numbered so it sorts first, and `TEMPLATE.md`.
- `TEMPLATE.md` is the template. Copy it, do not edit it in place.
- Paragraphs are single lines (no hard-wrapping), same as ADRs, so grep hits stay on one line.
- `OPEN:` lines and numbered rules are the spec's exclusive forms. An ADR uses neither. `grep -rn "^- OPEN:" docs/spec/` is the list of everything unsettled in the product.
- Shared terms are defined once, in [00-glossary.md](00-glossary.md), by the process under "Shared terms and the glossary". A domain file links to the glossary and never restates a definition.
- No front matter, no generated index. The directory is small enough to `ls`. Add either when a real need appears, not before.

## Enforcement
| Rule | How it is held |
|------|----------------|
| Five headings, in order, no extras | `go run ./tools/poly spec lint`, blocks |
| Rules numbered from 1, no gaps, no repeats | `poly spec lint`, blocks |
| Every `<domain> rule N` citation in a `_test.go` file or in `docs/spec/` names a rule that exists | `poly spec lint`, blocks |
| No glossary term redefined in a Vocabulary section | `poly spec lint`, blocks |
| Links under `docs/` resolve | `poly spec lint`, blocks |
| Every step in `docs/queue.yaml` names a domain that exists | `poly spec lint`, blocks |
| Rules pinned by tests still hold | `go test`, blocks |
| A rule is true and worth having | Nobody. That is the developer's judgement, and an `OPEN:` line is the default until it is. |
