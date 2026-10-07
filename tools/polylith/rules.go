// Package polylith holds the workspace rules that both the poly command and
// the polyvet analyzer enforce, so there is exactly one definition of each.
// See AGENTS.md > Polylith rules for the prose and ADR-0009 for the project rule.
package polylith

import (
	"fmt"
	"strings"
)

// Kind says where a package sits in the Polylith layout.
type Kind string

// The kinds of package in the layout. Other is Go code outside it, which the rules reject.
const (
	KindComponent   Kind = "component"
	KindBase        Kind = "base"
	KindProject     Kind = "project"
	KindDevelopment Kind = "development"
	KindTools       Kind = "tools"
	KindOther       Kind = "other"
)

// Pkg is one Go package in the workspace.
type Pkg struct {
	ImportPath string
	Rel        string   // directory relative to the module root, forward slashes
	Name       string   // package clause name
	Imports    []string // regular and test imports, deduplicated
	Kind       Kind
	Brick      string // brick or project name (the directory under components/, bases/ or projects/)
	GoFiles    int    // number of non-test Go files in the package
	Err        string // why the package failed to load, if it did
}

// IsRoot reports whether the package is a brick's or project's root package
// (components/<name>, bases/<name>, projects/<name>) rather than a subpackage.
func (p *Pkg) IsRoot() bool {
	return p.Brick != "" && strings.Count(p.Rel, "/") == 1
}

// Classify maps a module-relative directory to its kind and brick name.
func Classify(rel string) (Kind, string) {
	parts := strings.Split(rel, "/")
	brick := ""
	if len(parts) > 1 {
		brick = parts[1]
	}
	switch parts[0] {
	case "components":
		return KindComponent, brick
	case "bases":
		return KindBase, brick
	case "projects":
		return KindProject, brick
	case "development":
		return KindDevelopment, brick
	case "tools":
		return KindTools, brick
	}
	return KindOther, ""
}

// ImportRule returns why `from` may not import `to`, or "" if the import is allowed.
func ImportRule(from, to *Pkg) string {
	if from.Kind == KindTools {
		return "" // tooling may import anything
	}
	sameUnit := from.Kind == to.Kind && from.Brick == to.Brick && from.Brick != ""
	if sameUnit {
		return "" // inside one brick or project, including its internal/ subpackages
	}
	switch to.Kind {
	case KindDevelopment:
		return "nothing may depend on development/ (scratch space)"
	case KindTools:
		return "nothing may depend on tools/"
	case KindProject:
		return "projects are leaves; nothing imports a project"
	case KindOther:
		return "code outside the Polylith layout"
	}
	// to is a component or a base in another brick.
	if !to.IsRoot() {
		if strings.Contains(to.Rel+"/", "/internal/") {
			return fmt.Sprintf("%s is private to its brick (the compiler rejects this too); use the brick's root package", to.Rel)
		}
		return "use the brick through its root package (its public interface), not a subpackage; if the subpackage is private, move it under internal/"
	}
	switch from.Kind {
	case KindComponent:
		if to.Kind == KindBase {
			return "components must not import bases (dependency flows base -> component, never the reverse)"
		}
	case KindBase:
		if to.Kind == KindBase {
			return "a base must not import another base; share the code as a component instead"
		}
	}
	return "" // projects may import bases and components: main() is the composition root (ADR-0009)
}

// LayoutProblems returns the rule violations that concern one package on its
// own: its place in the layout, its package name, and (for a project root)
// the structural rules that keep it to wiring. importsBase says whether any
// of its imports is a base.
func LayoutProblems(p *Pkg, importsBase bool) []string {
	var problems []string
	switch p.Kind {
	case KindOther:
		return []string{fmt.Sprintf("%s: Go code outside the Polylith layout; it belongs in a component, a base, a project, development/ or tools/", p.Rel)}
	case KindComponent, KindBase, KindProject:
		if p.Brick == "" {
			return []string{fmt.Sprintf("%s: Go files directly under this directory; each brick or project needs its own subdirectory", p.Rel)}
		}
	}
	if !p.IsRoot() {
		return nil
	}
	switch p.Kind {
	case KindComponent, KindBase:
		if p.Name == "main" {
			problems = append(problems, fmt.Sprintf("%s: a brick must not be package main; bricks are libraries, executables live in projects/", p.Rel))
		}
	case KindProject:
		if p.Name != "main" {
			problems = append(problems, fmt.Sprintf("%s: a project's root package must be package main (it is the deployable executable)", p.Rel))
		}
		if p.GoFiles > 1 {
			problems = append(problems, fmt.Sprintf("%s: a project is wiring only and must be a single Go file (found %d); move logic into a base or component", p.Rel, p.GoFiles))
		}
		if !importsBase && p.Err == "" {
			problems = append(problems, fmt.Sprintf("%s: a project must import at least one base; main() wires components into a base and runs it", p.Rel))
		}
	}
	return problems
}
