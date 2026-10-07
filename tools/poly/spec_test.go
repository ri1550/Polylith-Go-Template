package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodDomain = `# Accounts
## Purpose
Money lives in accounts.

## Vocabulary
**Ledger** — the list of movements of one account.

## Data
An account has a balance.

## Behavior rules
1. An account starts with a zero balance.
2. An account has exactly one denomination.
3. A withdrawal larger than the balance is refused.

## Open questions
- OPEN: can an account be closed with a non-zero balance?
`

func specWorkspace(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestSpecLintAcceptsAWellFormedSpecAndQueue(t *testing.T) {
	root := specWorkspace(t, map[string]string{
		"docs/spec/accounts.md":                goodDomain,
		"docs/spec/00-glossary.md":             "# Glossary\n**Balance** — what an account holds. See [accounts.md](accounts.md).\n",
		"docs/spec/README.md":                  "# Spec\nSee [00-glossary.md](00-glossary.md) and `accounts rule 99` as an example citation that is not checked.\n",
		"components/accounts/accounts_test.go": "package accounts_test\n\n// TestOpen asserts accounts rule 1.\nfunc TestOpen() {}\n\n// ExampleWithdraw asserts accounts rule 3.\nfunc ExampleWithdraw() {}\n",
		"docs/queue.yaml":                      "steps:\n- id: open-accounts\n  rules: {accounts: \"1-2\"}\n  brick: components/accounts\n- id: withdrawals\n  rules: {accounts: \"3\"}\n  after: open-accounts\n",
	})
	problems, domains, rules, steps := specLint(root)
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if domains != 1 || rules != 3 || steps != 2 {
		t.Errorf("domains=%d rules=%d steps=%d", domains, rules, steps)
	}
	status := specStatus(root)
	for _, want := range []string{
		"accounts: 3 rule(s), 2 cited, 1 open question(s); uncited: 2",
		"1. open-accounts (components/accounts) rules: accounts 1-2: IN PROGRESS (1/2 rules have a citing test)",
		"2. withdrawals rules: accounts 3 [after open-accounts]: DONE, delete this step",
	} {
		if !strings.Contains(status, want) {
			t.Errorf("status lacks %q:\n%s", want, status)
		}
	}
}

func TestSpecLintReportsProblems(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"wrong headings", map[string]string{"docs/spec/a.md": strings.Replace(goodDomain, "## Data", "## Model", 1)}, "headings must be exactly"},
		{"rule gap", map[string]string{"docs/spec/a.md": strings.Replace(goodDomain, "2. An account", "4. An account", 1)}, "no gaps or repeats"},
		{"dangling citation", map[string]string{"docs/spec/a.md": goodDomain, "components/x/x_test.go": "package x_test\n// asserts a rule 9\n"}, "rule 9' but that rule does not exist"},
		{"citation of a missing domain", map[string]string{"docs/spec/a.md": goodDomain, "bases/x/x_test.go": "package x_test\n// asserts nope rule 1\n"}, "docs/spec/nope.md does not exist"},
		{"glossary term redefined", map[string]string{"docs/spec/a.md": goodDomain, "docs/spec/00-glossary.md": "# G\n**Ledger** — shared.\n"}, "redefines 'Ledger'"},
		{"broken link", map[string]string{"docs/spec/a.md": goodDomain, "docs/spec/README.md": "See [x](missing.md).\n"}, "link to 'missing.md' does not resolve"},
		{"queue bad id", map[string]string{"docs/spec/a.md": goodDomain, "docs/queue.yaml": "steps:\n- id: Bad_Id\n  rules: {a: \"1\"}\n"}, "id must be kebab-case"},
		{"queue bad range", map[string]string{"docs/spec/a.md": goodDomain, "docs/queue.yaml": "steps:\n- id: s\n  rules: {a: \"3-1\"}\n"}, "must look like"},
		{"queue unknown domain", map[string]string{"docs/spec/a.md": goodDomain, "docs/queue.yaml": "steps:\n- id: s\n  rules: {b: \"1\"}\n"}, "docs/spec/b.md does not exist"},
		{"queue after later step", map[string]string{"docs/spec/a.md": goodDomain, "docs/queue.yaml": "steps:\n- id: s\n  rules: {a: \"1\"}\n  after: t\n- id: t\n  rules: {a: \"2\"}\n"}, "not an earlier step"},
		{"queue unknown key", map[string]string{"docs/spec/a.md": goodDomain, "docs/queue.yaml": "steps:\n- id: s\n  rules: {a: \"1\"}\n  done: true\n"}, "unknown key `done`"},
		{"queue wrong top level", map[string]string{"docs/spec/a.md": goodDomain, "docs/queue.yaml": "tasks: []\n"}, "single key `steps`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := specWorkspace(t, tc.files)
			problems, _, _, _ := specLint(root)
			if !strings.Contains(strings.Join(problems, "\n"), tc.want) {
				t.Errorf("want a problem containing %q, got %v", tc.want, problems)
			}
		})
	}
}

func TestSpecStatusDerivesEveryStepState(t *testing.T) {
	root := specWorkspace(t, map[string]string{
		"docs/spec/a.md":  goodDomain,
		"docs/queue.yaml": "steps:\n- id: waiting\n  rules: {a: \"3-5\"}\n- id: ready\n  rules: {a: \"2\"}\n",
	})
	status := specStatus(root)
	for _, want := range []string{"waiting rules: a 3-5: WAITING ON SPEC (a has no rule 4, 5; see its OPEN: lines)", "ready rules: a 2: READY"} {
		if !strings.Contains(status, want) {
			t.Errorf("status lacks %q:\n%s", want, status)
		}
	}
	empty := specStatus(t.TempDir())
	if !strings.Contains(empty, "no domains") || !strings.Contains(empty, "empty") {
		t.Errorf("empty workspace status:\n%s", empty)
	}
}
