package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const goodADR = `---
status: accepted
date: 2026-01-01
decision-makers: [owner]
affects:
  components: [greeting]
  bases: []
  projects: ["*"]
interface-impact: none
---

# ADR-0001: Do the thing
Body.
`

const templateADR = `---
status: proposed              # proposed | accepted | deprecated | superseded by ADR-NNNN
date: YYYY-MM-DD
decision-makers: [names or roles]
affects:
  components: []
  bases: []
  projects: []
interface-impact: none
---

<!--
Copy this file. See AGENTS.md.
-->

# ADR-NNNN: Short title of the decision
## Context and problem statement
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
	res, err := lintADRs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 || len(res.Warnings) != 0 || res.Count != 1 {
		t.Errorf("res=%+v", res)
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
		{"no front matter", map[string]string{"0001-x.md": "# ADR-0001: no fences\n"}, "no YAML front matter"},
		{"bad status", map[string]string{"0001-x.md": strings.Replace(goodADR, "accepted", "done", 1)}, "status 'done'"},
		{"bad impact", map[string]string{"0001-x.md": strings.Replace(goodADR, "interface-impact: none", "interface-impact: huge", 1)}, "interface-impact 'huge'"},
		{"missing field", map[string]string{"0001-x.md": strings.Replace(goodADR, "date: 2026-01-01\n", "", 1)}, "missing required field `date`"},
		{"missing affects kind", map[string]string{"0001-x.md": strings.Replace(goodADR, "  bases: []\n", "", 1)}, "missing `bases`"},
		{"placeholder title", map[string]string{"0001-x.md": strings.Replace(goodADR, "# ADR-0001: Do the thing", "# ADR-NNNN: Short title of the decision", 1)}, "title is still the template"},
		{"superseded by a missing ADR", map[string]string{"0001-x.md": strings.Replace(goodADR, "status: accepted", "status: superseded by ADR-9999", 1)}, "no such ADR exists"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeWorkspace(t, tc.files)
			res, err := lintADRs(root)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(res.Problems, "\n"), tc.want) {
				t.Errorf("want a problem containing %q, got %v", tc.want, res.Problems)
			}
		})
	}
}

func TestLintOnlyWarnsOnHistoricalAffectsGapsAndPlaceholders(t *testing.T) {
	root := writeWorkspace(t, map[string]string{
		"0001-x.md": strings.Replace(goodADR, "[greeting]", "[planned]", 1),
		"0003-y.md": strings.Replace(strings.Replace(goodADR, "ADR-0001", "ADR-0003", 1), "date: 2026-01-01", "date: YYYY-MM-DD", 1),
	})
	res, err := lintADRs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 {
		t.Errorf("unknown names and gaps must not block: %v", res.Problems)
	}
	joined := strings.Join(res.Warnings, "\n")
	if !strings.Contains(joined, "'planned'") || !strings.Contains(joined, "gap at 0002") || !strings.Contains(joined, "placeholder") {
		t.Errorf("expected warnings for the unknown name, the gap and the placeholder, got %v", res.Warnings)
	}
}

func TestSupersededByExistingADRPasses(t *testing.T) {
	root := writeWorkspace(t, map[string]string{
		"0001-x.md": strings.Replace(goodADR, "status: accepted", "status: superseded by ADR-0002", 1),
		"0002-y.md": strings.Replace(goodADR, "ADR-0001", "ADR-0002", 1),
	})
	res, err := lintADRs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 {
		t.Errorf("got %v", res.Problems)
	}
}

func TestNewADRPicksTheNextNumberAndFillsTheTemplate(t *testing.T) {
	root := writeWorkspace(t, map[string]string{
		"0000-adr-template.md": templateADR,
		"0001-x.md":            goodADR,
		"0007-y.md":            goodADR,
	})
	path, err := newADR(root, "Use Postgres for users", time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "0008-use-postgres-for-users.md" {
		t.Errorf("path = %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"date: 2026-10-07", "# ADR-0008: Use Postgres for users", "status: proposed"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	if strings.Contains(text, "<!--") || strings.Contains(text, "YYYY") {
		t.Errorf("template boilerplate left in:\n%s", text)
	}
}

func TestIndexExpandsWildcardIntoEveryView(t *testing.T) {
	root := writeWorkspace(t, map[string]string{
		"0001-do-the-thing.md": goodADR,
		"0002-everything.md":   strings.Replace(strings.Replace(goodADR, "[greeting]", "[\"*\"]", 1), "ADR-0001: Do the thing", "ADR-0002: Everything", 1),
	})
	n, views, err := writeADRIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("n=%d, want 2", n)
	}
	got, err := os.ReadFile(filepath.Join(root, "docs/adr/index/components-greeting.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[ADR-0001: Do the thing](../0001-do-the-thing.md) (accepted)", "[ADR-0002: Everything](../0002-everything.md)"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("components-greeting.md lacks %q:\n%s", want, got)
		}
	}
	// The wildcard also creates views for units that exist on disk but no ADR names.
	if _, err := os.Stat(filepath.Join(root, "docs/adr/index/projects-hello.md")); err != nil {
		t.Errorf("expected projects-hello.md from the * expansion (views=%d)", views)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/adr/index/projects-_all_.md")); err != nil {
		t.Error("expected projects-_all_.md for the * wildcard")
	}
}
