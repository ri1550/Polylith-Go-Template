package main

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

var (
	adrFileRE   = regexp.MustCompile(`^docs/adr/(\d{4})-.+\.md$`)
	adrRefRE    = regexp.MustCompile(`(?i)\badr-(\d{4})\b`)
	markerRE    = regexp.MustCompile(`(?i)\[interface-impact:\s*([a-z]+)\s*\]`)
	validMarker = map[string]bool{"none": true, "new": true}
)

// brickSources returns the non-test Go sources of every brick root package in
// a git tree, keyed by brick path (components/<name>, bases/<name>).
func brickSources(tree string) (map[string]map[string][]byte, error) {
	paths, err := gitLines("ls-tree", "-r", "--name-only", tree, "--", "components", "bases")
	if err != nil {
		return nil, err
	}
	out := map[string]map[string][]byte{}
	for _, p := range paths {
		parts := strings.Split(p, "/")
		if len(parts) != 3 || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			continue
		}
		src, err := git("show", tree+":"+p)
		if err != nil {
			return nil, err
		}
		brick := parts[0] + "/" + parts[1]
		if out[brick] == nil {
			out[brick] = map[string][]byte{}
		}
		out[brick][parts[2]] = []byte(src)
	}
	return out, nil
}

// surfaceChange is the difference in one brick's public interface between two trees.
type surfaceChange struct {
	Brick   string // components/<name> or bases/<name>
	From    string // the brick's previous path when Status is a rename, else ""
	Status  string // "new", "removed", "changed", "renamed" or "renamed, changed"
	Removed []string
	Added   []string
}

// Name returns the bare brick name.
func (c surfaceChange) Name() string { return path.Base(c.Brick) }

// FormerName returns the bare name the brick had before a rename, or "".
func (c surfaceChange) FormerName() string {
	if c.From == "" {
		return ""
	}
	return path.Base(c.From)
}

// Kind returns the affects key for the brick: "components" or "bases".
func (c surfaceChange) Kind() string { return path.Dir(c.Brick) }

// brickRenames asks git which brick root packages moved between two trees,
// keyed old brick path to new brick path.
func brickRenames(oldTree, newTree string) map[string]string {
	lines, err := gitLines("diff", "--name-status", "-M", oldTree, newTree, "--", "components", "bases")
	if err != nil {
		return nil
	}
	renames := map[string]string{}
	for _, line := range lines {
		if !strings.HasPrefix(line, "R") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			continue
		}
		oldBrick, ok1 := rootBrick(parts[1])
		newBrick, ok2 := rootBrick(parts[2])
		if ok1 && ok2 && oldBrick != newBrick {
			renames[oldBrick] = newBrick
		}
	}
	return renames
}

// rootBrick returns the brick path of a root-package Go file, if that is what p is.
func rootBrick(p string) (string, bool) {
	parts := strings.Split(p, "/")
	if len(parts) != 3 || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}

// surfaceChanges compares the public interface of every brick between two git
// trees. A brick that git saw move is reported once, as a rename, with any
// surface difference between its old and new location.
func surfaceChanges(oldTree, newTree string) ([]surfaceChange, error) {
	oldSrc, err := brickSources(oldTree)
	if err != nil {
		return nil, err
	}
	newSrc, err := brickSources(newTree)
	if err != nil {
		return nil, err
	}
	before := map[string][]string{}
	for b, src := range oldSrc {
		if before[b], err = surface(src); err != nil {
			return nil, fmt.Errorf("%s at %s: %w", b, oldTree, err)
		}
	}
	after := map[string][]string{}
	for b, src := range newSrc {
		if after[b], err = surface(src); err != nil {
			return nil, fmt.Errorf("%s at %s: %w", b, newTree, err)
		}
	}
	renames := brickRenames(oldTree, newTree)
	return pairChanges(before, after, renames), nil
}

// pairChanges turns before/after surfaces into one change per brick,
// folding git-detected renames into a single entry.
func pairChanges(before, after map[string][]string, renames map[string]string) []surfaceChange {
	bricks := map[string]bool{}
	for b := range before {
		bricks[b] = true
	}
	for b := range after {
		bricks[b] = true
	}
	renamedTo := map[string]bool{}
	for _, to := range renames {
		renamedTo[to] = true
	}
	var changes []surfaceChange
	for _, brick := range sortedKeys(bricks) {
		if renamedTo[brick] {
			continue // reported from its old name below
		}
		if to, moved := renames[brick]; moved {
			_, wasThere := before[brick]
			_, isThere := after[to]
			if wasThere && isThere {
				removed, added := diffLines(before[brick], after[to])
				status := "renamed"
				if len(removed) > 0 || len(added) > 0 {
					status = "renamed, changed"
				}
				changes = append(changes, surfaceChange{Brick: to, From: brick, Status: status, Removed: removed, Added: added})
				continue
			}
		}
		removed, added := diffLines(before[brick], after[brick])
		if len(removed) == 0 && len(added) == 0 {
			continue
		}
		status := "changed"
		_, wasThere := before[brick]
		_, isThere := after[brick]
		switch {
		case !wasThere:
			status = "new"
		case !isThere:
			status = "removed"
		}
		changes = append(changes, surfaceChange{Brick: brick, Status: status, Removed: removed, Added: added})
	}
	return changes
}

func diffLines(before, after []string) (removed, added []string) {
	b := map[string]bool{}
	for _, l := range before {
		b[l] = true
	}
	a := map[string]bool{}
	for _, l := range after {
		a[l] = true
	}
	for _, l := range before {
		if !a[l] {
			removed = append(removed, l)
		}
	}
	for _, l := range after {
		if !b[l] {
			added = append(added, l)
		}
	}
	sort.Strings(removed)
	sort.Strings(added)
	return removed, added
}

func printChanges(changes []surfaceChange) {
	for _, c := range changes {
		if c.From != "" {
			fmt.Printf("  %s (%s, from %s)\n", c.Brick, c.Status, c.From)
		} else {
			fmt.Printf("  %s (%s brick)\n", c.Brick, c.Status)
		}
		for _, l := range c.Removed {
			fmt.Printf("    - %s\n", l)
		}
		for _, l := range c.Added {
			fmt.Printf("    + %s\n", l)
		}
	}
}

// adrScope is what one ADR says it affects: kind ("components", "bases",
// "projects") to the names listed, where "*" means every brick of that kind.
type adrScope struct {
	Number  string
	Affects map[string][]string
}

func (a adrScope) covers(c surfaceChange) bool {
	for _, n := range a.Affects[c.Kind()] {
		if n == "*" || n == c.Name() || (c.FormerName() != "" && n == c.FormerName()) {
			return true
		}
	}
	return false
}

// adrScopeFromGit parses the front matter of an ADR as it exists in a git tree.
func adrScopeFromGit(tree, file string) (adrScope, error) {
	text, err := git("show", tree+":"+file)
	if err != nil {
		return adrScope{}, err
	}
	data, err := frontMatter(text)
	if err != nil {
		return adrScope{}, fmt.Errorf("%s: %v", file, err)
	}
	scope := adrScope{Affects: map[string][]string{}}
	if m := adrFileRE.FindStringSubmatch(file); m != nil {
		scope.Number = m[1]
	}
	if affects, ok := data["affects"].(map[string]any); ok {
		for _, kind := range affectsKinds {
			values, _ := affectsList(affects, kind)
			scope.Affects[kind] = values
		}
	}
	return scope, nil
}

// recording is the evidence a pull request offers for its interface changes.
type recording struct {
	ADRs    []adrScope // ADR files in the diff, plus ADRs referenced by number in commit messages
	Markers []string   // [interface-impact: x] values found in commit messages
}

// unrecorded returns the changed bricks that no ADR in the recording covers
// and no valid marker excuses, plus notes explaining what was ignored.
func unrecorded(changes []surfaceChange, rec recording) (missing []surfaceChange, notes []string) {
	excused := false
	for _, m := range rec.Markers {
		if validMarker[m] {
			excused = true
		} else {
			notes = append(notes, fmt.Sprintf("[interface-impact: %s] is not a marker; a %s change needs an ADR file (markers are none or new)", m, m))
		}
	}
	if excused {
		return nil, notes
	}
	for _, c := range changes {
		covered := false
		for _, a := range rec.ADRs {
			if a.covers(c) {
				covered = true
				break
			}
		}
		if !covered {
			missing = append(missing, c)
		}
	}
	return missing, notes
}

// gatherRecording collects the ADR files in a list of changed paths (read from
// tree), the ADRs that commit messages reference by number, and the markers.
func gatherRecording(tree string, changedPaths []string, messages string) (recording, []string) {
	var rec recording
	var notes []string
	seen := map[string]bool{}
	for _, p := range changedPaths {
		if !adrFileRE.MatchString(p) {
			continue
		}
		scope, err := adrScopeFromGit(tree, p)
		if err != nil {
			notes = append(notes, err.Error()) // deleted or unparsable: cannot count it
			continue
		}
		rec.ADRs = append(rec.ADRs, scope)
		seen[scope.Number] = true
	}
	for _, m := range adrRefRE.FindAllStringSubmatch(messages, -1) {
		num := m[1]
		if seen[num] {
			continue
		}
		seen[num] = true
		files, _ := gitLines("ls-tree", "-r", "--name-only", tree, "--", "docs/adr")
		found := false
		for _, f := range files {
			if strings.HasPrefix(path.Base(f), num+"-") {
				scope, err := adrScopeFromGit(tree, f)
				if err == nil {
					rec.ADRs = append(rec.ADRs, scope)
					found = true
				}
				break
			}
		}
		if !found {
			notes = append(notes, fmt.Sprintf("a commit message references ADR-%s, but no such ADR exists", num))
		}
	}
	for _, m := range markerRE.FindAllStringSubmatch(messages, -1) {
		rec.Markers = append(rec.Markers, strings.ToLower(m[1]))
	}
	return rec, notes
}

func printMissing(missing []surfaceChange, notes []string) {
	for _, n := range notes {
		fmt.Printf("  note: %s\n", n)
	}
	fmt.Println("Not recorded:")
	for _, c := range missing {
		names := "`" + c.Name() + "`"
		if c.FormerName() != "" {
			names += " or `" + c.FormerName() + "`"
		}
		fmt.Printf("  - %s (%s): no ADR in this change names %s in affects.%s\n", c.Brick, c.Status, names, c.Kind())
	}
	fmt.Println()
	fmt.Println("Do one of these:")
	fmt.Println("  1. Add an ADR under docs/adr/ whose `affects` names the brick (go run ./tools/poly adr new \"<title>\"; the agent can draft it).")
	fmt.Println("  2. Reference an existing ADR that names it in a commit message, e.g. 'ADR-0012: ...'.")
	fmt.Println("  3. If no consumer can observe this change (including a brick nothing uses yet), add '[interface-impact: none]' to a commit message.")
	fmt.Println("  4. If the change is additive and an ADR was deliberately declined, add '[interface-impact: new]' to a commit message.")
	fmt.Println("A breaking change always needs an ADR. See CONTRIBUTING.md.")
}

// emptyTree returns the hash of the empty tree for this repository.
func emptyTree() (string, error) {
	out, err := gitWithStdin("", "mktree")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// runInterface reports public interface changes.
//
//	--mode pre-commit  staged tree vs HEAD; prints the diff; warns about unrecorded bricks; never blocks
//	--mode ci          HEAD vs the merge base with origin/<BASE_REF>; prints the diff; blocks on unrecorded bricks
//	--between A B      prints the diff between any two refs
func runInterface(args []string) (int, error) {
	mode := "pre-commit"
	var between []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--between" && i+2 < len(args):
			between = args[i+1 : i+3]
			i += 2
		case args[i] == "--mode" && i+1 < len(args):
			mode = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--mode="):
			mode = strings.TrimPrefix(args[i], "--mode=")
		default:
			return 2, fmt.Errorf("usage: poly interface [--mode pre-commit|ci] | --between <old> <new>")
		}
	}
	if between != nil {
		changes, err := surfaceChanges(between[0], between[1])
		if err != nil {
			return 1, err
		}
		if len(changes) == 0 {
			fmt.Printf("No public interface changes between %s and %s.\n", between[0], between[1])
			return 0, nil
		}
		fmt.Printf("Public interface changes between %s and %s:\n", between[0], between[1])
		printChanges(changes)
		return 0, nil
	}
	switch mode {
	case "pre-commit":
		return interfacePreCommit()
	case "ci":
		return interfaceCI()
	default:
		return 2, fmt.Errorf("unknown mode %q (want pre-commit or ci)", mode)
	}
}

func interfacePreCommit() (int, error) {
	oldTree := "HEAD"
	if _, err := git("rev-parse", "--verify", "-q", "HEAD^{tree}"); err != nil {
		empty, err := emptyTree()
		if err != nil {
			return 1, err
		}
		oldTree = empty
	}
	newTree, err := git("write-tree")
	if err != nil {
		return 1, err
	}
	newTree = strings.TrimSpace(newTree)
	changes, err := surfaceChanges(oldTree, newTree)
	if err != nil {
		return 1, err
	}
	if len(changes) == 0 {
		fmt.Println("No public interface changes.")
		return 0, nil
	}
	fmt.Println("This commit changes a public interface (a brick's exported API):")
	printChanges(changes)
	staged, err := gitLines("diff", "--cached", "--name-only", "--diff-filter=ACM")
	if err != nil {
		return 1, err
	}
	rec, notes := gatherRecording(newTree, staged, "") // the commit message does not exist yet
	missing, moreNotes := unrecorded(changes, rec)
	notes = append(notes, moreNotes...)
	fmt.Println()
	if len(missing) == 0 {
		fmt.Println("Every changed brick is named by an ADR staged in this commit. OK.")
		return 0, nil
	}
	printMissing(missing, notes)
	fmt.Println("Not blocking you now; CI blocks a pull request that leaves this unrecorded.")
	return 0, nil // never block locally
}

func interfaceCI() (int, error) {
	baseRef := os.Getenv("BASE_REF")
	if baseRef == "" {
		baseRef = os.Getenv("GITHUB_BASE_REF")
	}
	if baseRef == "" {
		baseRef = "main"
	}
	base := ""
	for _, candidate := range []string{"origin/" + baseRef, baseRef} {
		if _, err := git("rev-parse", "--verify", "-q", candidate+"^{commit}"); err == nil {
			base = candidate
			break
		}
	}
	if base == "" {
		return 1, fmt.Errorf("cannot resolve base branch %q (tried origin/%s and %s)", baseRef, baseRef, baseRef)
	}
	mergeBase, err := git("merge-base", base, "HEAD")
	if err != nil {
		return 1, err
	}
	mergeBase = strings.TrimSpace(mergeBase)

	changes, err := surfaceChanges(mergeBase, "HEAD")
	if err != nil {
		return 1, err
	}
	if len(changes) == 0 {
		fmt.Println("No public interface changes. OK.")
		return 0, nil
	}
	fmt.Printf("Public interface changes since %s:\n", base)
	printChanges(changes)
	files, err := gitLines("diff", "--name-only", mergeBase, "HEAD")
	if err != nil {
		return 1, err
	}
	messages, err := git("log", "--format=%B", mergeBase+"..HEAD")
	if err != nil {
		return 1, err
	}
	rec, notes := gatherRecording("HEAD", files, messages)
	missing, moreNotes := unrecorded(changes, rec)
	notes = append(notes, moreNotes...)
	fmt.Println()
	if len(missing) == 0 {
		for _, n := range notes {
			fmt.Printf("  note: %s\n", n)
		}
		fmt.Println("Every changed brick is recorded. OK.")
		return 0, nil
	}
	fmt.Println("This pull request changes public interfaces that no decision records:")
	fmt.Println()
	printMissing(missing, notes)
	return 1, nil
}
