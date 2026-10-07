package main

import (
	"flag"
	"fmt"
	"strings"
)

const stableTagPattern = "stable-*"

// latestStableTag returns the most recently created tag matching stable-*, or "".
func latestStableTag() (string, error) {
	lines, err := gitLines("tag", "-l", stableTagPattern, "--sort=-creatordate")
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", nil
	}
	return lines[0], nil
}

// changedUnits maps changed repository paths to the bricks and projects they belong to.
func changedUnits(paths []string) map[string]bool {
	units := map[string]bool{}
	for _, path := range paths {
		kind, brick := brickOfPath(path)
		if brick == "" || (kind != KindComponent && kind != KindBase && kind != KindProject) {
			continue
		}
		units[string(kind)+"/"+brick] = true
	}
	return units
}

// runDiff shows which bricks and projects changed since a reference point
// (the latest stable-* tag by default), and which projects are affected
// through their dependencies. Uncommitted changes count.
func runDiff(args []string) (int, error) {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	since := fs.String("since", "", "git ref to compare against (default: latest stable-* tag)")
	if err := fs.Parse(args); err != nil {
		return 2, err
	}
	ref := *since
	if ref == "" {
		tag, err := latestStableTag()
		if err != nil {
			return 1, err
		}
		if tag == "" {
			fmt.Println("No stable-* tag found. Mark a known-good point with `git tag stable-1`, or pass --since <ref>.")
			return 0, nil
		}
		ref = tag
	}

	paths, err := gitLines("diff", "--name-only", ref)
	if err != nil {
		return 1, err
	}
	untracked, err := gitLines("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return 1, err
	}
	changed := changedUnits(append(paths, untracked...))
	if len(changed) == 0 {
		fmt.Printf("No bricks or projects changed since %s.\n", ref)
		return 0, nil
	}

	fmt.Printf("Changed since %s:\n", ref)
	for _, u := range sortedKeys(changed) {
		kind, name, _ := strings.Cut(u, "/")
		fmt.Printf("  %-9s %s\n", kind, name)
	}

	ws, err := loadWorkspace()
	if err != nil {
		return 1, err
	}
	_, transitive := brickDeps(ws)
	var affected []string
	for k, deps := range transitive {
		if !strings.HasPrefix(k, "project/") {
			continue
		}
		if changed[k] {
			affected = append(affected, strings.TrimPrefix(k, "project/"))
			continue
		}
		for dep := range deps {
			if changed["component/"+dep] || changed["base/"+dep] {
				affected = append(affected, strings.TrimPrefix(k, "project/"))
				break
			}
		}
	}
	fmt.Println()
	fmt.Printf("Projects affected: %s\n", orNone(sortedStrings(affected)))
	return 0, nil
}

func sortedStrings(s []string) []string {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return sortedKeys(m)
}
