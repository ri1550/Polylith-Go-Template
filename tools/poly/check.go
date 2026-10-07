package main

import (
	"fmt"
	"sort"
	"strings"
)

// runCheck validates the workspace layout and the brick dependency rules.
func runCheck() (int, error) {
	ws, err := loadWorkspace()
	if err != nil {
		return 1, err
	}
	problems := checkWorkspace(ws)
	if len(problems) > 0 {
		fmt.Println("poly check found problems:")
		fmt.Println()
		for _, p := range problems {
			fmt.Printf("  - %s\n", p)
		}
		fmt.Println()
		fmt.Println("The rules are in AGENTS.md under \"Polylith rules\". Ask the agent to help move the code to the right place.")
		return 1, nil
	}
	c, b, p := count(ws)
	fmt.Printf("poly check passed: %d component(s), %d base(s), %d project(s).\n", c, b, p)
	return 0, nil
}

func count(ws *Workspace) (components, bases, projects int) {
	for _, p := range ws.Pkgs {
		if !p.IsRoot() {
			continue
		}
		switch p.Kind {
		case KindComponent:
			components++
		case KindBase:
			bases++
		case KindProject:
			projects++
		}
	}
	return
}

// checkWorkspace returns every rule violation found, sorted. An empty result means the workspace is valid.
func checkWorkspace(ws *Workspace) []string {
	var problems []string
	byImport := ws.ByImport()

	// Brick names are unique across components and bases.
	owner := map[string]Kind{}
	for _, p := range ws.Pkgs {
		if (p.Kind == KindComponent || p.Kind == KindBase) && p.IsRoot() {
			if k, ok := owner[p.Brick]; ok && k != p.Kind {
				problems = append(problems, fmt.Sprintf(
					"brick name %q is used by both a component and a base; brick names must be unique", p.Brick))
			}
			owner[p.Brick] = p.Kind
		}
	}

	for _, p := range ws.Pkgs {
		if p.Err != "" {
			// A package that does not load has an unknown import list; say so rather than pass it.
			problems = append(problems, fmt.Sprintf("%s: does not load: %s", p.Rel, p.Err))
		}
		switch p.Kind {
		case KindOther:
			problems = append(problems, fmt.Sprintf(
				"%s: Go code outside the Polylith layout; it belongs in a component, a base, a project, development/ or tools/", p.Rel))
			continue
		case KindComponent, KindBase, KindProject:
			if p.Brick == "" {
				problems = append(problems, fmt.Sprintf(
					"%s: Go files directly under this directory; each brick or project needs its own subdirectory", p.Rel))
				continue
			}
			if p.IsRoot() && p.Kind != KindProject && p.Name == "main" {
				problems = append(problems, fmt.Sprintf(
					"%s: a brick must not be package main; bricks are libraries, executables live in projects/", p.Rel))
			}
			if p.IsRoot() && p.Kind == KindProject {
				problems = append(problems, checkProject(p, byImport)...)
			}
		}
		for _, imp := range p.Imports {
			target, ok := byImport[imp]
			if !ok {
				continue // standard library or a third-party module
			}
			if msg := importRule(p, target); msg != "" {
				problems = append(problems, fmt.Sprintf("%s imports %s: %s", p.Rel, target.Rel, msg))
			}
		}
	}
	sort.Strings(problems)
	return problems
}

// checkProject applies the structural rules that keep a project to wiring:
// package main, one Go file, at least one base (ADR-0009).
func checkProject(p *Pkg, byImport map[string]*Pkg) []string {
	var problems []string
	if p.Name != "main" {
		problems = append(problems, fmt.Sprintf(
			"%s: a project's root package must be package main (it is the deployable executable)", p.Rel))
	}
	if p.GoFiles > 1 {
		problems = append(problems, fmt.Sprintf(
			"%s: a project is wiring only and must be a single Go file (found %d); move logic into a base or component", p.Rel, p.GoFiles))
	}
	importsBase := false
	for _, imp := range p.Imports {
		if t, ok := byImport[imp]; ok && t.Kind == KindBase {
			importsBase = true
			break
		}
	}
	if !importsBase && p.Err == "" {
		problems = append(problems, fmt.Sprintf(
			"%s: a project must import at least one base; main() wires components into a base and runs it", p.Rel))
	}
	return problems
}

// importRule returns why `from` may not import `to`, or "" if the import is allowed.
func importRule(from, to *Pkg) string {
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
