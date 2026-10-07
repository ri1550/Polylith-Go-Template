package main

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// brickDeps returns, for each brick or project (keyed "kind/name"), the set of
// brick names it imports directly, and the transitive set.
func brickDeps(ws *Workspace) (direct, transitive map[string]map[string]bool) {
	byImport := ws.ByImport()
	direct = map[string]map[string]bool{}
	key := func(p *Pkg) string { return string(p.Kind) + "/" + p.Brick }

	for _, p := range ws.Pkgs {
		if p.Brick == "" || (p.Kind != KindComponent && p.Kind != KindBase && p.Kind != KindProject) {
			continue
		}
		k := key(p)
		if direct[k] == nil {
			direct[k] = map[string]bool{}
		}
		for _, imp := range p.Imports {
			t, ok := byImport[imp]
			if !ok || t.Brick == "" || (t.Kind != KindComponent && t.Kind != KindBase) || key(t) == k {
				continue
			}
			direct[k][t.Brick] = true
		}
	}

	// Brick names are unique across components and bases, so a bare name resolves.
	nameKey := map[string]string{}
	for k := range direct {
		if kind, name, _ := strings.Cut(k, "/"); kind != string(KindProject) {
			nameKey[name] = k
		}
	}
	transitive = map[string]map[string]bool{}
	for k := range direct {
		seen := map[string]bool{}
		var walk func(string)
		walk = func(from string) {
			for dep := range direct[from] {
				if !seen[dep] {
					seen[dep] = true
					if next, ok := nameKey[dep]; ok {
						walk(next)
					}
				}
			}
		}
		walk(k)
		transitive[k] = seen
	}
	return direct, transitive
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func orNone(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}

// thirdPartyModules lists the modules outside this workspace that a package
// pulls in transitively (project binaries link exactly these).
func thirdPartyModules(ws *Workspace, pkgPath string) ([]string, error) {
	out, err := exec.Command("go", "list", "-deps", "-f", "{{with .Module}}{{.Path}}{{end}}", pkgPath).Output()
	if err != nil {
		return nil, fmt.Errorf("go list -deps %s: %w", pkgPath, err)
	}
	mods := map[string]bool{}
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" && l != ws.Module {
			mods[l] = true
		}
	}
	return sortedKeys(mods), nil
}

// runDeps prints what each brick uses, what uses it (bricks and projects),
// and what each project pulls in: bricks, and modules from outside the workspace.
func runDeps() (int, error) {
	ws, err := loadWorkspace()
	if err != nil {
		return 1, err
	}
	direct, transitive := brickDeps(ws)

	usedBy := map[string]map[string]bool{}
	for k, deps := range direct {
		kind, name, _ := strings.Cut(k, "/")
		label := name
		if kind == string(KindProject) {
			label = name + " (project)"
		}
		for dep := range deps {
			if usedBy[dep] == nil {
				usedBy[dep] = map[string]bool{}
			}
			usedBy[dep][label] = true
		}
	}

	keys := sortedKeys(toBoolMap(direct))
	fmt.Println("Bricks (direct dependencies):")
	nBricks := 0
	for _, kind := range []Kind{KindComponent, KindBase} {
		for _, k := range keys {
			if !strings.HasPrefix(k, string(kind)+"/") {
				continue
			}
			nBricks++
			name := strings.TrimPrefix(k, string(kind)+"/")
			fmt.Printf("  %-9s %-20s uses: %-30s used by: %s\n",
				kind, name, orNone(sortedKeys(direct[k])), orNone(sortedKeys(usedBy[name])))
		}
	}
	if nBricks == 0 {
		fmt.Println("  (none yet)")
	}
	fmt.Println()
	fmt.Println("Projects:")
	nProjects := 0
	for _, k := range keys {
		if !strings.HasPrefix(k, string(KindProject)+"/") {
			continue
		}
		nProjects++
		name := strings.TrimPrefix(k, "project/")
		mods, err := thirdPartyModules(ws, "./projects/"+name)
		if err != nil {
			return 1, err
		}
		fmt.Printf("  %s\n", name)
		fmt.Printf("    wires directly:  %s\n", orNone(sortedKeys(direct[k])))
		fmt.Printf("    ships bricks:    %s\n", orNone(sortedKeys(transitive[k])))
		fmt.Printf("    ships modules:   %s\n", orNone(mods))
	}
	if nProjects == 0 {
		fmt.Println("  (none yet)")
	}
	return 0, nil
}

func toBoolMap(m map[string]map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}
