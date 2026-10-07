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
	return &Pkg{ImportPath: mod + "/" + rel, Rel: rel, Name: name, Imports: imports, Kind: kind, Brick: brick}
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
		pkg("projects/hello", "main", "bases/api"),
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
		{"project imports component", []*Pkg{pkg("components/a", "a"), pkg("projects/p", "main", "components/a")}, "projects import only bases"},
		{"anything imports a project", []*Pkg{pkg("projects/p", "main"), pkg("bases/a", "a", "projects/p")}, "projects are leaves"},
		{"anything imports development", []*Pkg{pkg("development/s", "s"), pkg("components/a", "a", "development/s")}, "development/"},
		{"anything imports tools", []*Pkg{pkg("tools/poly", "main"), pkg("components/a", "a", "tools/poly")}, "tools/"},
		{"bypassing the interface", []*Pkg{pkg("components/a", "a"), pkg("components/a/util", "util"), pkg("components/b", "b", "components/a/util")}, "root package"},
		{"brick is package main", []*Pkg{pkg("components/a", "main")}, "must not be package main"},
		{"project is not main", []*Pkg{pkg("projects/p", "p")}, "must be package main"},
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
		pkg("projects/hello", "main", "bases/api"),
	}}
	direct, transitive := brickDeps(ws)
	if got := sortedKeys(direct["project/hello"]); strings.Join(got, ",") != "api" {
		t.Errorf("direct deps of hello = %v, want [api]", got)
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
