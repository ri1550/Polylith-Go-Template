package main

import (
	"fmt"
	"sort"

	"github.com/myorg/workspace/tools/polylith"
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
		importsBase := false
		for _, imp := range p.Imports {
			if t, ok := byImport[imp]; ok && t.Kind == KindBase {
				importsBase = true
			}
		}
		layout := polylith.LayoutProblems(p, importsBase)
		problems = append(problems, layout...)
		if p.Kind == KindOther || (p.Brick == "" && p.Kind != KindDevelopment && p.Kind != KindTools) {
			continue
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
