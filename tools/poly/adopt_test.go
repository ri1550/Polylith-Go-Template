package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// oldMod is a stand-in for the template's module path; a literal, so that
// adopt rewriting this file's imports can never make the test self-referential.
const oldMod = "example.com/template/workspace"

func TestAdoptPlanRewritesDeletesAndDates(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module "+oldMod+"\n\ngo 1.27\n")
	write("components/greeting/greeting.go", "package greeting\n")
	write("components/keep/keep.go", "package keep\n\nimport _ \""+oldMod+"/components/greeting\"\n")
	write("bases/api/api.go", "package api\n")
	write("projects/hello/main.go", "package main\n\nimport _ \""+oldMod+"/bases/api\"\n\nfunc main() {}\n")
	write("README.md", "# W\n\nintro\n\n<!-- examples:start (x) -->\nexample stuff\n<!-- examples:end -->\n\n## Next\n")
	write(".golangci.yml", "local-prefixes:\n  - "+oldMod+"\n")
	write("docs/adr/0000-adr-template.md", templateADR)
	write("docs/adr/0001-x.md", strings.Replace(strings.Replace(goodADR, "2026-01-01", "YYYY-MM-DD", 1), "[owner]", "[you]", 1))

	actions, err := adoptPlan(root, "github.com/acme/inventory", false, "platform team", time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range actions {
		if err := a.run(); err != nil {
			t.Fatalf("%s: %v", a.desc, err)
		}
	}
	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		return string(b)
	}
	if !strings.Contains(read("go.mod"), "module github.com/acme/inventory") {
		t.Error("go.mod not rewritten")
	}
	if strings.Contains(read("components/keep/keep.go"), oldMod) || strings.Contains(read(".golangci.yml"), oldMod) {
		t.Error("imports or lint config still mention the template module")
	}
	for _, gone := range exampleUnits {
		if _, err := os.Stat(filepath.Join(root, gone)); err == nil {
			t.Errorf("%s still exists", gone)
		}
	}
	if readme := read("README.md"); strings.Contains(readme, "example stuff") || !strings.Contains(readme, "intro\n\n## Next") {
		t.Errorf("README block not removed cleanly:\n%s", readme)
	}
	if adr := read("docs/adr/0001-x.md"); !strings.Contains(adr, "date: 2026-10-07") || !strings.Contains(adr, "decision-makers: [platform team]") {
		t.Errorf("ADR not dated:\n%s", adr)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/adr/index/README.md")); err != nil {
		t.Error("index not regenerated")
	}
}

func TestAdoptPlanWithoutChangesIsMinimal(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/acme/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs/adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	actions, err := adoptPlan(root, "github.com/acme/x", true, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || !strings.Contains(actions[0].desc, "index") {
		t.Errorf("expected only the index refresh, got %d actions", len(actions))
	}
}
