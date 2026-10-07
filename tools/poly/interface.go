package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

var adrFileRE = regexp.MustCompile(`^docs/adr/\d{4}-.+\.md$`)

// brickSources returns the non-test Go sources of every brick root package in
// a git tree, keyed by brick path (components/<name>, bases/<name>).
func brickSources(tree string) (map[string]map[string][]byte, error) {
	paths, err := gitLines("ls-tree", "-r", "--name-only", tree, "--", "components", "bases")
	if err != nil {
		return nil, err
	}
	out := map[string]map[string][]byte{}
	for _, path := range paths {
		parts := strings.Split(path, "/")
		if len(parts) != 3 || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := git("show", tree+":"+path)
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
	Brick   string
	Removed []string
	Added   []string
}

// surfaceChanges compares the public interface of every brick between two git trees.
func surfaceChanges(oldTree, newTree string) ([]surfaceChange, error) {
	oldSrc, err := brickSources(oldTree)
	if err != nil {
		return nil, err
	}
	newSrc, err := brickSources(newTree)
	if err != nil {
		return nil, err
	}
	bricks := map[string]bool{}
	for b := range oldSrc {
		bricks[b] = true
	}
	for b := range newSrc {
		bricks[b] = true
	}

	var changes []surfaceChange
	for _, brick := range sortedKeys(bricks) {
		before, err := surface(oldSrc[brick])
		if err != nil {
			return nil, fmt.Errorf("%s at %s: %w", brick, oldTree, err)
		}
		after, err := surface(newSrc[brick])
		if err != nil {
			return nil, fmt.Errorf("%s at %s: %w", brick, newTree, err)
		}
		removed, added := diffLines(before, after)
		if len(removed) > 0 || len(added) > 0 {
			changes = append(changes, surfaceChange{Brick: brick, Removed: removed, Added: added})
		}
	}
	return changes, nil
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
		fmt.Printf("  %s\n", c.Brick)
		for _, l := range c.Removed {
			fmt.Printf("    - %s\n", l)
		}
		for _, l := range c.Added {
			fmt.Printf("    + %s\n", l)
		}
	}
}

func hasADR(paths []string) bool {
	for _, p := range paths {
		if adrFileRE.MatchString(p) {
			return true
		}
	}
	return false
}

// emptyTree is the hash of the empty tree, used when the repository has no commits yet.
func emptyTree() (string, error) {
	cmd := exec.Command("git", "mktree")
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git mktree: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// runInterface reports public interface changes.
//
// --mode pre-commit compares the staged tree with HEAD and only warns.
// --mode ci compares HEAD with the merge base of origin/<BASE_REF> and blocks
// unless the change is recorded: an ADR file in the diff, an "ADR-NNNN"
// reference in a commit message, or "[interface-impact: none]" in a commit message.
func runInterface(args []string) (int, error) {
	fs := flag.NewFlagSet("interface", flag.ContinueOnError)
	mode := fs.String("mode", "pre-commit", "pre-commit (warn only) or ci (block)")
	if err := fs.Parse(args); err != nil {
		return 2, err
	}

	switch *mode {
	case "pre-commit":
		return interfacePreCommit()
	case "ci":
		return interfaceCI()
	default:
		return 2, fmt.Errorf("unknown mode %q (want pre-commit or ci)", *mode)
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
	changes, err := surfaceChanges(oldTree, strings.TrimSpace(newTree))
	if err != nil {
		return 1, err
	}
	if len(changes) == 0 {
		return 0, nil
	}
	staged, err := gitLines("diff", "--cached", "--name-only", "--diff-filter=ACM")
	if err != nil {
		return 1, err
	}
	if hasADR(staged) {
		fmt.Println("Public interface changed and an ADR is staged with it. OK.")
		return 0, nil
	}
	fmt.Println("Heads-up: this commit changes a public interface (a brick's exported API):")
	printChanges(changes)
	fmt.Println()
	fmt.Println("This changes what other code depends on. If it is a real interface change, add an ADR")
	fmt.Println("under docs/adr/ in this commit (the agent can draft it). CI will ask for this later.")
	fmt.Println("Not blocking you now.")
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
	files, err := gitLines("diff", "--name-only", mergeBase, "HEAD")
	if err != nil {
		return 1, err
	}
	if hasADR(files) {
		fmt.Println("Public interface changed and an ADR is included. OK.")
		return 0, nil
	}
	messages, err := git("log", "--format=%B", mergeBase+"..HEAD")
	if err != nil {
		return 1, err
	}
	lower := strings.ToLower(messages)
	if strings.Contains(lower, "adr-") || strings.Contains(lower, "[interface-impact:") {
		fmt.Println("Public interface changed and a commit message records it. OK.")
		return 0, nil
	}

	fmt.Println("This pull request changes a public interface but records no decision:")
	fmt.Println()
	printChanges(changes)
	fmt.Println()
	fmt.Println("Do one of these, then push again:")
	fmt.Println("  1. Add or update an ADR under docs/adr/ describing the change (best option; the agent can draft it).")
	fmt.Println("  2. Reference an ADR in a commit message, e.g. 'ADR-NNNN: ...'.")
	fmt.Println("  3. If this is NOT a contract change, add '[interface-impact: none]' to a commit message.")
	fmt.Println("See CONTRIBUTING.md for details.")
	return 1, nil
}
