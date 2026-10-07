package main

import (
	"flag"
	"fmt"
	"sort"
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

// unitStatus is what happened to a brick or project: "new", "removed" or "changed".
type unitStatus map[string]string

// changedUnits maps "status<TAB>path" lines (git diff --name-status) and
// untracked paths to the bricks and projects they belong to, with a status.
func changedUnits(nameStatus []string, untracked []string) unitStatus {
	type tally struct{ added, deleted, modified int }
	tallies := map[string]*tally{}
	bump := func(path string, code byte) {
		kind, brick := brickOfPath(path)
		if brick == "" || (kind != KindComponent && kind != KindBase && kind != KindProject) {
			return
		}
		key := string(kind) + "/" + brick
		if tallies[key] == nil {
			tallies[key] = &tally{}
		}
		switch code {
		case 'A':
			tallies[key].added++
		case 'D':
			tallies[key].deleted++
		default:
			tallies[key].modified++
		}
	}
	for _, line := range nameStatus {
		code, path, ok := strings.Cut(line, "\t")
		if !ok || code == "" {
			continue
		}
		if code[0] == 'R' || code[0] == 'C' { // "R100\told\tnew": count both ends
			old, newPath, _ := strings.Cut(path, "\t")
			bump(old, 'D')
			bump(newPath, 'A')
			continue
		}
		bump(path, code[0])
	}
	for _, path := range untracked {
		bump(path, 'A')
	}
	units := unitStatus{}
	for key, t := range tallies {
		switch {
		case t.added > 0 && t.deleted == 0 && t.modified == 0:
			units[key] = "new"
		case t.deleted > 0 && t.added == 0 && t.modified == 0:
			units[key] = "removed"
		default:
			units[key] = "changed"
		}
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
			fmt.Println("No stable-* tag found. Mark a known-good point with `git tag -a stable-1 -m \"...\"`, or pass --since <ref>.")
			return 0, nil
		}
		ref = tag
		if strings.Contains(tag, "template") {
			fmt.Printf("Note: %s is the template's own baseline. Tag your own known-good point (git tag -a stable-1 -m \"...\") once you have one.\n", tag)
		}
	}

	nameStatus, err := gitLines("diff", "--name-status", ref)
	if err != nil {
		return 1, err
	}
	untracked, err := gitLines("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return 1, err
	}
	changed := changedUnits(nameStatus, untracked)
	if len(changed) == 0 {
		fmt.Printf("No bricks or projects changed since %s.\n", ref)
		return 0, nil
	}

	fmt.Printf("Changed since %s:\n", ref)
	keys := make([]string, 0, len(changed))
	for k := range changed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, u := range keys {
		kind, name, _ := strings.Cut(u, "/")
		fmt.Printf("  %-9s %-20s %s\n", kind, name, changed[u])
	}

	ws, err := loadWorkspace()
	if err != nil {
		return 1, err
	}
	_, transitive := brickDeps(ws)
	affected := map[string]bool{}
	for k, status := range changed { // a changed or removed project is affected by definition
		if strings.HasPrefix(k, "project/") {
			affected[strings.TrimPrefix(k, "project/")+suffix(status)] = true
		}
	}
	for k, deps := range transitive {
		if !strings.HasPrefix(k, "project/") {
			continue
		}
		for dep := range deps {
			if _, ok := changed["component/"+dep]; ok {
				affected[strings.TrimPrefix(k, "project/")] = true
			}
			if _, ok := changed["base/"+dep]; ok {
				affected[strings.TrimPrefix(k, "project/")] = true
			}
		}
	}
	fmt.Println()
	fmt.Printf("Projects affected: %s\n", orNone(sortedKeys(affected)))
	return 0, nil
}

func suffix(status string) string {
	if status == "removed" {
		return " (removed)"
	}
	return ""
}
