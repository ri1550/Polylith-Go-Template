package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	templateTag = "stable-template-baseline"
	readmeStart = "<!-- examples:start"
	readmeEnd   = "<!-- examples:end -->"
)

var (
	exampleUnits = []string{"components/greeting", "bases/api", "projects/hello", "docs/spec/greeting.md"}
	rewriteExts  = map[string]bool{".go": true, ".md": true, ".yml": true, ".yaml": true, ".mod": true}
	skipDirs     = map[string]bool{".git": true, "bin": true, "dist": true}
)

// adoptAction is one step of the adoption, described before it runs.
type adoptAction struct {
	desc string
	run  func() error
}

// runAdopt turns a fresh clone of the template into the adopter's workspace:
// new module path everywhere, example bricks removed, the README's example
// block removed, the template's baseline tag removed, the ADR index refreshed,
// and optionally the ADRs dated and attributed. It prints the plan first.
func runAdopt(args []string) (int, error) {
	fs := flag.NewFlagSet("adopt", flag.ContinueOnError)
	module := fs.String("module", "", "the new module path, e.g. github.com/acme/inventory (required)")
	keep := fs.Bool("keep-examples", false, "keep the example bricks")
	makers := fs.String("decision-makers", "", "date the shipped ADRs today and set their decision-makers, e.g. \"platform team\"")
	dry := fs.Bool("dry-run", false, "print the plan and change nothing")
	if err := fs.Parse(args); err != nil {
		return 2, err
	}
	if *module == "" {
		return 2, fmt.Errorf("usage: poly adopt --module <path> [--decision-makers \"...\"] [--keep-examples] [--dry-run]")
	}
	actions, err := adoptPlan(".", *module, *keep, *makers, time.Now())
	if err != nil {
		return 1, err
	}
	fmt.Println("Adoption plan:")
	for _, a := range actions {
		fmt.Printf("  - %s\n", a.desc)
	}
	if *dry {
		fmt.Println("Dry run: nothing changed.")
		return 0, nil
	}
	for _, a := range actions {
		if err := a.run(); err != nil {
			return 1, fmt.Errorf("%s: %w", a.desc, err)
		}
	}
	fmt.Println()
	fmt.Println("Done. Review with `git status`, run `make all`, then commit:")
	fmt.Printf("  git add -A && git commit -m \"Adopt the Polylith template as %s [interface-impact: none]\"\n", *module)
	fmt.Println("The marker is honest: the removed example bricks had no consumers. Then set branch protection on main (CONTRIBUTING.md).")
	return 0, nil
}

func adoptPlan(root, module string, keepExamples bool, makers string, now time.Time) ([]adoptAction, error) {
	oldModule, err := currentModule(root)
	if err != nil {
		return nil, err
	}
	var actions []adoptAction

	if oldModule != module {
		files, err := filesMentioning(root, oldModule)
		if err != nil {
			return nil, err
		}
		actions = append(actions, adoptAction{
			desc: fmt.Sprintf("rewrite the module path %s -> %s in %d file(s)", oldModule, module, len(files)),
			run: func() error {
				for _, f := range files {
					b, err := os.ReadFile(f)
					if err != nil {
						return err
					}
					if err := os.WriteFile(f, []byte(strings.ReplaceAll(string(b), oldModule, module)), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
		})
	}

	if !keepExamples {
		for _, unit := range exampleUnits {
			dir := filepath.Join(root, unit)
			if _, err := os.Stat(dir); err != nil {
				continue
			}
			actions = append(actions, adoptAction{desc: "delete the example " + unit, run: func() error { return os.RemoveAll(dir) }})
		}
		readme := filepath.Join(root, "README.md")
		if b, err := os.ReadFile(readme); err == nil && strings.Contains(string(b), readmeStart) {
			actions = append(actions, adoptAction{desc: "delete the example block from README.md", run: func() error {
				b, err := os.ReadFile(readme)
				if err != nil {
					return err
				}
				return os.WriteFile(readme, []byte(stripBlock(string(b))), 0o644)
			}})
		}
	}

	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		if tags, err := gitLines("-C", root, "tag", "-l", templateTag); err == nil && len(tags) > 0 {
			actions = append(actions, adoptAction{desc: "delete the template's baseline tag " + templateTag, run: func() error {
				_, err := git("-C", root, "tag", "-d", templateTag)
				return err
			}})
		}
	}

	if makers != "" {
		date := now.Format("2006-01-02")
		actions = append(actions, adoptAction{desc: fmt.Sprintf("date the shipped ADRs %s and set decision-makers to [%s]", date, makers), run: func() error {
			files, _ := filepath.Glob(filepath.Join(root, adrDir, "[0-9][0-9][0-9][0-9]-*.md"))
			for _, f := range files {
				if filepath.Base(f) == adrTemplate {
					continue
				}
				b, err := os.ReadFile(f)
				if err != nil {
					return err
				}
				text := strings.Replace(string(b), "date: YYYY-MM-DD", "date: "+date, 1)
				text = strings.Replace(text, "decision-makers: [you]", "decision-makers: ["+makers+"]", 1)
				if err := os.WriteFile(f, []byte(text), 0o644); err != nil {
					return err
				}
			}
			return nil
		}})
	}

	actions = append(actions, adoptAction{desc: "regenerate docs/adr/index/", run: func() error {
		_, _, err := writeADRIndex(root)
		return err
	}})
	return actions, nil
}

var moduleLineRE = regexp.MustCompile(`(?m)^module\s+(\S+)`)

func currentModule(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("run adopt from the workspace root: %w", err)
	}
	m := moduleLineRE.FindSubmatch(b)
	if m == nil {
		return "", fmt.Errorf("go.mod has no module line")
	}
	return string(m[1]), nil
}

// filesMentioning lists source and doc files under root that contain needle.
func filesMentioning(root, needle string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] && p != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !rewriteExts[filepath.Ext(p)] {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), needle) {
			out = append(out, p)
		}
		return nil
	})
	return out, err
}

// stripBlock removes the README's marked example block, markers included.
func stripBlock(text string) string {
	start := strings.Index(text, readmeStart)
	end := strings.Index(text, readmeEnd)
	if start < 0 || end < 0 || end < start {
		return text
	}
	end += len(readmeEnd)
	for end < len(text) && text[end] == '\n' {
		end++
	}
	return text[:start] + text[end:]
}
