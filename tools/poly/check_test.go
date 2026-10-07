package main

import (
	"strings"
	"testing"
)

const mod = "example.com/ws"

func pkg(rel, name string, imports ...string) *Pkg {
	kind, brick := classify(rel)
	for i, imp := range imports {
		imports[i] = mod + "/" + imp
	}
	return &Pkg{ImportPath: mod + "/" + rel, Rel: rel, Name: name, Imports: imports, Kind: kind, Brick: brick, GoFiles: 1}
}

func withFiles(p *Pkg, n int) *Pkg {
	p.GoFiles = n
	return p
}

func broken(p *Pkg, err string) *Pkg {
	p.Err = err
	return p
}

func TestCheckAcceptsAValidWorkspace(t *testing.T) {
	ws := &Workspace{Module: mod, Pkgs: []*Pkg{
		pkg("components/greeting", "greeting"),
		pkg("components/greeting/internal/fmt", "fmt"),
		pkg("components/users", "users", "components/greeting"),
		pkg("bases/api", "api", "components/users", "bases/api/internal/routes"),
		pkg("bases/api/internal/routes", "routes", "components/greeting"),
		// A project is the composition root: it wires components into a base (ADR-0009).
		pkg("projects/hello", "main", "bases/api", "components/users"),
		pkg("development/scratch", "main", "components/users", "bases/api"),
		pkg("tools/poly", "main", "components/users"),
	}}
	if problems := checkWorkspace(ws); len(problems) != 0 {
		t.Errorf("expected no problems, got:\n%s", strings.Join(problems, "\n"))
	}
}

func TestCheckRejectsRuleViolations(t *testing.T) {
	cases := []struct {
		name string
		pkgs []*Pkg
		want string
	}{
		{"component imports base", []*Pkg{pkg("bases/api", "api"), pkg("components/a", "a", "bases/api")}, "components must not import bases"},
		{"base imports base", []*Pkg{pkg("bases/a", "a"), pkg("bases/b", "b", "bases/a")}, "must not import another base"},
		{"anything imports a project", []*Pkg{pkg("projects/p", "main", "bases/a"), pkg("bases/a", "a", "projects/p")}, "projects are leaves"},
		{"anything imports development", []*Pkg{pkg("development/s", "s"), pkg("components/a", "a", "development/s")}, "development/"},
		{"anything imports tools", []*Pkg{pkg("tools/poly", "main"), pkg("components/a", "a", "tools/poly")}, "tools/"},
		{"bypassing the interface", []*Pkg{pkg("components/a", "a"), pkg("components/a/util", "util"), pkg("components/b", "b", "components/a/util")}, "root package"},
		{"importing internal", []*Pkg{pkg("components/a", "a"), pkg("components/a/internal/x", "x"), pkg("components/b", "b", "components/a/internal/x")}, "private to its brick"},
		{"brick is package main", []*Pkg{pkg("components/a", "main")}, "must not be package main"},
		{"project is not main", []*Pkg{pkg("projects/p", "p", "bases/a"), pkg("bases/a", "a")}, "must be package main"},
		{"project imports no base", []*Pkg{pkg("projects/p", "main", "components/a"), pkg("components/a", "a")}, "at least one base"},
		{"project has two files", []*Pkg{withFiles(pkg("projects/p", "main", "bases/a"), 2), pkg("bases/a", "a")}, "single Go file"},
		{"code outside the layout", []*Pkg{pkg("internal/x", "x")}, "outside the Polylith layout"},
		{"file directly under components", []*Pkg{pkg("components", "components")}, "own subdirectory"},
		{"duplicate brick name", []*Pkg{pkg("components/a", "a"), pkg("bases/a", "a")}, "must be unique"},
		{"package fails to load", []*Pkg{broken(pkg("components/a", "a"), "syntax error")}, "does not load"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			problems := checkWorkspace(&Workspace{Module: mod, Pkgs: tc.pkgs})
			if len(problems) == 0 {
				t.Fatalf("expected a problem containing %q, got none", tc.want)
			}
			if !strings.Contains(strings.Join(problems, "\n"), tc.want) {
				t.Errorf("expected a problem containing %q, got:\n%s", tc.want, strings.Join(problems, "\n"))
			}
		})
	}
}

func TestBrickDepsAreTransitiveForProjects(t *testing.T) {
	ws := &Workspace{Module: mod, Pkgs: []*Pkg{
		pkg("components/greeting", "greeting"),
		pkg("components/users", "users", "components/greeting"),
		pkg("bases/api", "api", "components/users"),
		pkg("projects/hello", "main", "bases/api", "components/greeting"),
	}}
	direct, transitive := brickDeps(ws)
	if got := sortedKeys(direct["project/hello"]); strings.Join(got, ",") != "api,greeting" {
		t.Errorf("direct deps of hello = %v, want [api greeting]", got)
	}
	if got := sortedKeys(transitive["project/hello"]); strings.Join(got, ",") != "api,greeting,users" {
		t.Errorf("transitive deps of hello = %v, want [api greeting users]", got)
	}
}

func TestBrickOfPath(t *testing.T) {
	cases := map[string]string{
		"components/greeting/greeting.go": "component/greeting",
		"bases/api/internal/x/y.go":       "base/api",
		"projects/hello/main.go":          "project/hello",
		"components/.keep":                "other/",
		"docs/adr/0001-x.md":              "other/",
		"README.md":                       "other/",
	}
	for path, want := range cases {
		kind, brick := brickOfPath(path)
		if got := string(kind) + "/" + brick; got != want {
			t.Errorf("brickOfPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestChangedUnitsLabelsNewRemovedAndChanged(t *testing.T) {
	status := changedUnits([]string{
		"A\tcomponents/fresh/fresh.go",
		"D\tbases/old/old.go",
		"D\tbases/old/old_test.go",
		"M\tcomponents/greeting/greeting.go",
		"A\tcomponents/greeting/extra.go",
		"R086\tbases/api/api.go\tbases/greetapi/greetapi.go",
		"M\tREADME.md",
	}, []string{"projects/spike/main.go"})
	want := unitStatus{
		"component/fresh":    "new",
		"base/old":           "removed",
		"component/greeting": "changed",
		"base/api":           "removed",
		"base/greetapi":      "new",
		"project/spike":      "new",
	}
	for k, v := range want {
		if status[k] != v {
			t.Errorf("%s = %q, want %q", k, status[k], v)
		}
	}
	if len(status) != len(want) {
		t.Errorf("got %v", status)
	}
}
