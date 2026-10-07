package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodADR = `---
status: accepted
date: 2026-01-01
decision-makers: [you]
affects:
  components: [greeting]
  bases: []
  projects: ["*"]
interface-impact: none
---

# ADR-0001: Do the thing
Body.
`

func writeWorkspace(t *testing.T, adrs map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"docs/adr", "components/greeting", "bases/api", "projects/hello"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range adrs {
		if err := os.WriteFile(filepath.Join(root, "docs/adr", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLintAcceptsAValidRecord(t *testing.T) {
	root := writeWorkspace(t, map[string]string{"0001-do-the-thing.md": goodADR})
	problems, n, err := lintADRs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 || n != 1 {
		t.Errorf("problems=%v n=%d", problems, n)
	}
}

func TestLintReportsProblems(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"bad filename", map[string]string{"1-Bad_Name.md": goodADR}, "NNNN-kebab-title"},
		{"duplicate number", map[string]string{"0001-a.md": goodADR, "0001-b.md": goodADR}, "already used"},
		{"no front matter", map[string]string{"0002-x.md": "# ADR-0002: no fences\n"}, "no YAML front matter"},
		{"bad status", map[string]string{"0003-x.md": strings.Replace(goodADR, "accepted", "done", 1)}, "status 'done'"},
		{"bad impact", map[string]string{"0004-x.md": strings.Replace(goodADR, "interface-impact: none", "interface-impact: huge", 1)}, "interface-impact 'huge'"},
		{"missing field", map[string]string{"0005-x.md": strings.Replace(goodADR, "date: 2026-01-01\n", "", 1)}, "missing required field `date`"},
		{"unknown brick", map[string]string{"0006-x.md": strings.Replace(goodADR, "[greeting]", "[nope]", 1)}, "no such component exists"},
		{"missing affects kind", map[string]string{"0007-x.md": strings.Replace(goodADR, "  bases: []\n", "", 1)}, "missing `bases`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeWorkspace(t, tc.files)
			problems, _, err := lintADRs(root)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(problems, "\n"), tc.want) {
				t.Errorf("want a problem containing %q, got %v", tc.want, problems)
			}
		})
	}
}

func TestIndexGroupsByAffectedUnit(t *testing.T) {
	root := writeWorkspace(t, map[string]string{"0001-do-the-thing.md": goodADR})
	n, views, err := writeADRIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || views != 2 {
		t.Errorf("n=%d views=%d, want 1 and 2", n, views)
	}
	got, err := os.ReadFile(filepath.Join(root, "docs/adr/index/components-greeting.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "[ADR-0001: Do the thing](../0001-do-the-thing.md) (accepted)") {
		t.Errorf("unexpected index content:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/adr/index/projects-_all_.md")); err != nil {
		t.Error("expected projects-_all_.md for the * wildcard")
	}
}
