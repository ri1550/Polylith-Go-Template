package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// The spec (docs/spec/) and the build queue (docs/queue.yaml): one lint, one
// status. The conventions they enforce are written in docs/spec/README.md and
// the header of docs/queue.yaml; this file only checks shape and derives state.

const (
	specDir   = "docs/spec"
	queueFile = "docs/queue.yaml"
	glossary  = "00-glossary.md"
)

var (
	specHeadings = []string{"Purpose", "Vocabulary", "Data", "Behavior rules", "Open questions"}
	notDomains   = map[string]bool{"readme.md": true, "template.md": true, strings.ToLower(glossary): true}
	h2RE         = regexp.MustCompile(`(?m)^## (.*)$`)
	ruleRE       = regexp.MustCompile(`(?m)^(\d+)\. `)
	openRE       = regexp.MustCompile(`(?m)^- OPEN:`)
	citationRE   = regexp.MustCompile(`\b([a-z0-9][a-z0-9-]*) rule (\d+)\b`)
	termRE       = regexp.MustCompile(`(?m)^\*\*([^*]+)\*\*\s*—`)
	linkRE       = regexp.MustCompile(`!?\[[^\]]*\]\(([^)\s]+)\)`)
	stepIDRE     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	rangeRE      = regexp.MustCompile(`^(\d+)(?:-(\d+))?$`)
	stepKeys     = map[string]bool{"id": true, "rules": true, "brick": true, "after": true}
	testDirs     = []string{"components", "bases"} // where _test.go files that cite rules live
)

// ruleSet is the rule numbers each domain has (or that tests cite), by domain.
type ruleSet map[string]map[int]bool

func (r ruleSet) add(domain string, n int) {
	if r[domain] == nil {
		r[domain] = map[int]bool{}
	}
	r[domain][n] = true
}

func sortedInts(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func joinInts(ns []int) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}

// --- spec ------------------------------------------------------------------

func domainFiles(root string) []string {
	matches, _ := filepath.Glob(filepath.Join(root, specDir, "*.md"))
	var out []string
	for _, m := range matches {
		if !notDomains[strings.ToLower(filepath.Base(m))] {
			out = append(out, m)
		}
	}
	sort.Strings(out)
	return out
}

func rel(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(r)
	}
	return path
}

// sections splits a domain file into its "## " sections, in file order.
func sections(text string) (names []string, bodies map[string]string) {
	bodies = map[string]string{}
	locs := h2RE.FindAllStringSubmatchIndex(text, -1)
	for i, loc := range locs {
		name := strings.TrimSpace(text[loc[2]:loc[3]])
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		names = append(names, name)
		bodies[name] = text[loc[1]:end]
	}
	return names, bodies
}

// specInfo is what the spec says, plus the structural problems found reading it.
type specInfo struct {
	Rules    ruleSet
	Open     map[string]int
	Problems []string
}

func readSpec(root string) specInfo {
	info := specInfo{Rules: ruleSet{}, Open: map[string]int{}}
	shared := glossaryTerms(root)
	for _, path := range domainFiles(root) {
		raw, err := os.ReadFile(path)
		if err != nil {
			info.Problems = append(info.Problems, err.Error())
			continue
		}
		text := string(raw)
		domain := strings.TrimSuffix(filepath.Base(path), ".md")
		names, bodies := sections(text)
		if strings.Join(names, "|") != strings.Join(specHeadings, "|") {
			info.Problems = append(info.Problems, fmt.Sprintf("%s: headings must be exactly %q, in that order; found %q", rel(root, path), specHeadings, names))
		}
		info.Rules[domain] = map[int]bool{}
		var numbers []int
		for _, m := range ruleRE.FindAllStringSubmatch(bodies["Behavior rules"], -1) {
			n, _ := strconv.Atoi(m[1])
			numbers = append(numbers, n)
		}
		for i, n := range numbers {
			if n != i+1 {
				info.Problems = append(info.Problems, fmt.Sprintf("%s: behavior rules must be numbered 1..%d in order with no gaps or repeats; found %s", rel(root, path), len(numbers), joinInts(numbers)))
				break
			}
		}
		for _, n := range numbers {
			info.Rules[domain][n] = true
		}
		info.Open[domain] = len(openRE.FindAllString(bodies["Open questions"], -1))
		for _, m := range termRE.FindAllStringSubmatch(bodies["Vocabulary"], -1) {
			if term := strings.TrimSpace(m[1]); shared[strings.ToLower(term)] {
				info.Problems = append(info.Problems, fmt.Sprintf("%s: Vocabulary redefines '%s', which %s/%s already defines; link to it instead", rel(root, path), term, specDir, glossary))
			}
		}
	}
	return info
}

func glossaryTerms(root string) map[string]bool {
	out := map[string]bool{}
	raw, err := os.ReadFile(filepath.Join(root, specDir, glossary))
	if err != nil {
		return out
	}
	for _, m := range termRE.FindAllStringSubmatch(string(raw), -1) {
		out[strings.ToLower(strings.TrimSpace(m[1]))] = true
	}
	return out
}

// testFiles lists every _test.go file under the brick directories.
func testFiles(root string) []string {
	var out []string
	for _, dir := range testDirs {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, "_test.go") {
				out = append(out, p)
			}
			return nil
		})
	}
	sort.Strings(out)
	return out
}

// citedRules collects every `<domain> rule N` citation in the test files.
func citedRules(root string) ruleSet {
	cited := ruleSet{}
	for _, path := range testFiles(root) {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, m := range citationRE.FindAllStringSubmatch(string(raw), -1) {
			n, _ := strconv.Atoi(m[2])
			cited.add(m[1], n)
		}
	}
	return cited
}

// checkCitations verifies every citation in tests and spec files names a rule that exists.
func checkCitations(root string, rules ruleSet) []string {
	var problems []string
	files := testFiles(root)
	files = append(files, domainFiles(root)...)
	if g := filepath.Join(root, specDir, glossary); fileExists(g) {
		files = append(files, g)
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := string(raw)
		for _, loc := range citationRE.FindAllStringSubmatchIndex(text, -1) {
			domain := text[loc[2]:loc[3]]
			n, _ := strconv.Atoi(text[loc[4]:loc[5]])
			line := strings.Count(text[:loc[0]], "\n") + 1
			if _, ok := rules[domain]; !ok {
				problems = append(problems, fmt.Sprintf("%s:%d: cites '%s rule %d' but %s/%s.md does not exist", rel(root, path), line, domain, n, specDir, domain))
			} else if !rules[domain][n] {
				problems = append(problems, fmt.Sprintf("%s:%d: cites '%s rule %d' but that rule does not exist", rel(root, path), line, domain, n))
			}
		}
	}
	return problems
}

// checkLinks verifies every relative Markdown link under docs/ resolves. The
// generated ADR index is skipped.
func checkLinks(root string) []string {
	var problems []string
	skip := filepath.Join(root, adrIndexDir)
	_ = filepath.WalkDir(filepath.Join(root, "docs"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p == skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".md") {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		text := string(raw)
		for _, loc := range linkRE.FindAllStringSubmatchIndex(text, -1) {
			if text[loc[0]] == '!' {
				continue // an image
			}
			target := text[loc[2]:loc[3]]
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "#") || strings.Contains(target, "<") {
				continue
			}
			target, _, _ = strings.Cut(target, "#")
			if target != "" && !fileExists(filepath.Join(filepath.Dir(p), target)) {
				line := strings.Count(text[:loc[0]], "\n") + 1
				problems = append(problems, fmt.Sprintf("%s:%d: link to '%s' does not resolve", rel(root, p), line, target))
			}
		}
		return nil
	})
	return problems
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// --- queue -----------------------------------------------------------------

// queueStep is one step as written; validation happens in checkQueue.
type queueStep map[string]any

func loadQueue(root string) ([]queueStep, []string) {
	raw, err := os.ReadFile(filepath.Join(root, queueFile))
	if err != nil {
		return nil, nil // no queue is an empty queue
	}
	var data any
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return nil, []string{fmt.Sprintf("%s: not valid YAML: %v", queueFile, err)}
	}
	if data == nil {
		return nil, nil
	}
	top, ok := data.(map[string]any)
	if !ok {
		return nil, []string{fmt.Sprintf("%s: top level must be a mapping with the single key `steps`", queueFile)}
	}
	for k := range top {
		if k != "steps" {
			return nil, []string{fmt.Sprintf("%s: top level must be a mapping with the single key `steps` (found `%s`)", queueFile, k)}
		}
	}
	if top["steps"] == nil {
		return nil, nil
	}
	list, ok := top["steps"].([]any)
	if !ok {
		return nil, []string{fmt.Sprintf("%s: `steps` must be a list", queueFile)}
	}
	var steps []queueStep
	var problems []string
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: steps[%d] must be a mapping with id and rules", queueFile, i+1))
			continue
		}
		steps = append(steps, queueStep(m))
	}
	return steps, problems
}

func parseRange(v any) (lo, hi int, ok bool) {
	m := rangeRE.FindStringSubmatch(strings.TrimSpace(stringOf(v)))
	if m == nil {
		return 0, 0, false
	}
	lo, _ = strconv.Atoi(m[1])
	hi = lo
	if m[2] != "" {
		hi, _ = strconv.Atoi(m[2])
	}
	if lo < 1 || hi < lo {
		return 0, 0, false
	}
	return lo, hi, true
}

func checkQueue(steps []queueStep, rules ruleSet) []string {
	var problems []string
	var seen []string
	for i, step := range steps {
		where := fmt.Sprintf("%s: steps[%d]", queueFile, i+1)
		for k := range step {
			if !stepKeys[k] {
				problems = append(problems, fmt.Sprintf("%s: unknown key `%s`; allowed: id, rules, brick, after", where, k))
			}
		}
		id, _ := step["id"].(string)
		switch {
		case !stepIDRE.MatchString(id):
			problems = append(problems, fmt.Sprintf("%s: id must be kebab-case (got %q)", where, stringOf(step["id"])))
		case contains(seen, id):
			problems = append(problems, fmt.Sprintf("%s: duplicate id %q", where, id))
		}
		stepRules, ok := step["rules"].(map[string]any)
		if !ok || len(stepRules) == 0 {
			problems = append(problems, fmt.Sprintf("%s: rules must be a non-empty mapping of <domain>: \"n\" or \"lo-hi\"", where))
		} else {
			for domain, rng := range stepRules {
				if _, _, ok := parseRange(rng); !ok {
					problems = append(problems, fmt.Sprintf("%s: rules[%s] must look like \"3\" or \"1-4\" (got %q)", where, domain, stringOf(rng)))
				}
				if _, ok := rules[domain]; !ok {
					problems = append(problems, fmt.Sprintf("%s: rules names domain %q, but %s/%s.md does not exist", where, domain, specDir, domain))
				}
			}
		}
		if after, present := step["after"]; present {
			if a, _ := after.(string); !contains(seen, a) {
				problems = append(problems, fmt.Sprintf("%s: after names %q, which is not an earlier step", where, stringOf(after)))
			}
		}
		if brick, present := step["brick"]; present {
			if _, ok := brick.(string); !ok {
				problems = append(problems, fmt.Sprintf("%s: brick must be a string", where))
			}
		}
		if id != "" {
			seen = append(seen, id)
		}
	}
	return problems
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// stepState derives a step's state from the spec and the tests: nothing is written down.
func stepState(step queueStep, rules, cited ruleSet) string {
	stepRules, _ := step["rules"].(map[string]any)
	if len(stepRules) == 0 {
		return "MALFORMED, run `go run ./tools/poly spec lint`"
	}
	var missing []string
	total, have := 0, 0
	domains := make([]string, 0, len(stepRules))
	for d := range stepRules {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	for _, domain := range domains {
		lo, hi, ok := parseRange(stepRules[domain])
		if !ok {
			return "MALFORMED, run `go run ./tools/poly spec lint`"
		}
		var absent []int
		for n := lo; n <= hi; n++ {
			total++
			if !rules[domain][n] {
				absent = append(absent, n)
			} else if cited[domain][n] {
				have++
			}
		}
		if len(absent) > 0 {
			missing = append(missing, fmt.Sprintf("%s has no rule %s", domain, joinInts(absent)))
		}
	}
	switch {
	case len(missing) > 0:
		return fmt.Sprintf("WAITING ON SPEC (%s; see its OPEN: lines)", strings.Join(missing, "; "))
	case have == 0:
		return "READY"
	case have < total:
		return fmt.Sprintf("IN PROGRESS (%d/%d rules have a citing test)", have, total)
	}
	return "DONE, delete this step"
}

func fmtStepRules(step queueStep) string {
	stepRules, _ := step["rules"].(map[string]any)
	domains := make([]string, 0, len(stepRules))
	for d := range stepRules {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	parts := make([]string, len(domains))
	for i, d := range domains {
		parts[i] = d + " " + stringOf(stepRules[d])
	}
	return strings.Join(parts, ", ")
}

// --- verbs -----------------------------------------------------------------

// specLint returns every structural problem in the spec, its citations, the
// links under docs/, and the queue.
func specLint(root string) (problems []string, domains, rules, steps int) {
	info := readSpec(root)
	problems = append(problems, info.Problems...)
	problems = append(problems, checkCitations(root, info.Rules)...)
	problems = append(problems, checkLinks(root)...)
	q, qProblems := loadQueue(root)
	problems = append(problems, qProblems...)
	problems = append(problems, checkQueue(q, info.Rules)...)
	for _, r := range info.Rules {
		rules += len(r)
	}
	return problems, len(info.Rules), rules, len(q)
}

func runSpecLint() (int, error) {
	problems, domains, rules, steps := specLint(".")
	if len(problems) > 0 {
		fmt.Println("Spec lint found problems:")
		for _, p := range problems {
			fmt.Printf("  - %s\n", p)
		}
		return 1, nil
	}
	fmt.Printf("Spec lint passed: %d domain(s), %d rule(s), %d queue step(s), links resolve.\n", domains, rules, steps)
	return 0, nil
}

// specStatus renders what works and what is next, all derived. It never fails:
// a SessionStart hook prints it, and a broken spec must not break a session.
func specStatus(root string) string {
	info := readSpec(root)
	cited := citedRules(root)
	steps, qProblems := loadQueue(root)
	problems := append(info.Problems, qProblems...)
	var lines []string

	if len(info.Rules) == 0 {
		lines = append(lines, fmt.Sprintf("[spec] no domains in %s/ yet. Nothing can be built until the developer asks for one.", specDir))
	} else {
		lines = append(lines, fmt.Sprintf("[spec] %d domain(s). Rules with a citing test are what works today.", len(info.Rules)))
		domains := make([]string, 0, len(info.Rules))
		for d := range info.Rules {
			domains = append(domains, d)
		}
		sort.Strings(domains)
		for _, domain := range domains {
			var have, uncited []int
			for _, n := range sortedInts(info.Rules[domain]) {
				if cited[domain][n] {
					have = append(have, n)
				} else {
					uncited = append(uncited, n)
				}
			}
			line := fmt.Sprintf("  %s: %d rule(s), %d cited, %d open question(s)", domain, len(info.Rules[domain]), len(have), info.Open[domain])
			if len(uncited) > 0 {
				line += "; uncited: " + joinInts(uncited)
			}
			var stray []int
			for _, n := range sortedInts(cited[domain]) {
				if !info.Rules[domain][n] {
					stray = append(stray, n)
				}
			}
			if len(stray) > 0 {
				line += "; cited but not in spec: " + joinInts(stray)
			}
			lines = append(lines, line)
		}
		var unknown []string
		for d := range cited {
			if _, ok := info.Rules[d]; !ok {
				unknown = append(unknown, d)
			}
		}
		sort.Strings(unknown)
		for _, d := range unknown {
			lines = append(lines, fmt.Sprintf("  %s: cited by tests but %s/%s.md does not exist", d, specDir, d))
		}
	}

	if len(steps) > 0 {
		lines = append(lines, fmt.Sprintf("[queue] %s: %d step(s), in order.", queueFile, len(steps)))
		for i, step := range steps {
			head := fmt.Sprintf("  %d. %s", i+1, stringOf(step["id"]))
			if b := stringOf(step["brick"]); b != "" {
				head += " (" + b + ")"
			}
			head += " rules: " + fmtStepRules(step)
			if a := stringOf(step["after"]); a != "" {
				head += " [after " + a + "]"
			}
			lines = append(lines, head+": "+stepState(step, info.Rules, cited))
		}
	} else {
		lines = append(lines, fmt.Sprintf("[queue] %s: empty. Add a step when a domain has rules to build and the developer wants them built.", queueFile))
	}
	if len(problems) > 0 {
		lines = append(lines, fmt.Sprintf("[spec] %d lint problem(s); run `go run ./tools/poly spec lint`.", len(problems)))
	}
	return strings.Join(lines, "\n")
}

func runSpecStatus() (code int, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[spec] status could not run: %v. Run `go run ./tools/poly spec lint`.\n", r)
			code, err = 0, nil
		}
	}()
	fmt.Println(specStatus("."))
	return 0, nil
}
