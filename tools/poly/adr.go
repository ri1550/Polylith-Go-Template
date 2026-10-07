package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

const (
	adrDir      = "docs/adr"
	adrIndexDir = "docs/adr/index"
	adrTemplate = "0000-adr-template.md"
)

var (
	adrFilenameRE  = regexp.MustCompile(`^(\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)
	adrTitleRE     = regexp.MustCompile(`(?m)^#\s+(.*)$`)
	supersededRE   = regexp.MustCompile(`(?i)^superseded by adr-(\d{4})$`)
	adrRequired    = []string{"status", "date", "decision-makers", "affects", "interface-impact"}
	validImpact    = map[string]bool{"none": true, "new": true, "breaking": true}
	statusPrefix   = []string{"proposed", "accepted", "deprecated", "superseded by adr-"}
	affectsKinds   = []string{"components", "bases", "projects"}
	affectsDirs    = map[string]string{"components": "components", "bases": "bases", "projects": "projects"}
	placeholderRE  = regexp.MustCompile(`(?i)YYYY-MM-DD|\[you\]|names or roles|ADR-NNNN:|Short title of the decision`)
	nonAlnumRE     = regexp.MustCompile(`[^a-z0-9]+`)
	placeholderMsg = "still contains a template placeholder (date, decision-makers or title); fill it in"
)

// adrRecord is one parsed decision record.
type adrRecord struct {
	File   string
	Number string
	Title  string
	Body   string
	Data   map[string]any
}

// frontMatter parses the YAML between the leading --- fences.
func frontMatter(text string) (map[string]any, error) {
	if !strings.HasPrefix(text, "---") {
		return nil, fmt.Errorf("no YAML front matter (the file must start with ---)")
	}
	parts := strings.SplitN(text, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("front matter is not closed with a second ---")
	}
	data := map[string]any{}
	if err := yaml.Unmarshal([]byte(parts[1]), &data); err != nil {
		return nil, fmt.Errorf("front matter is not valid YAML: %v", err)
	}
	return data, nil
}

// readADRs parses every NNNN-*.md record in dir, skipping the template and README.
// Files whose front matter cannot be parsed are returned with a nil Data and the error.
func readADRs(dir string) ([]adrRecord, map[string]error, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var records []adrRecord
	broken := map[string]error{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == adrTemplate || strings.EqualFold(name, "readme.md") || !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, nil, err
		}
		text := string(raw)
		rec := adrRecord{File: name, Title: strings.TrimSuffix(name, ".md"), Body: text}
		if m := adrFilenameRE.FindStringSubmatch(name); m != nil {
			rec.Number = m[1]
		}
		if m := adrTitleRE.FindStringSubmatch(text); m != nil {
			rec.Title = strings.TrimSpace(m[1])
		}
		data, err := frontMatter(text)
		if err != nil {
			broken[name] = err
		}
		rec.Data = data
		records = append(records, rec)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].File < records[j].File })
	return records, broken, nil
}

// knownUnits lists the directories under components/, bases/ and projects/.
// A nil set means the directory is absent.
func knownUnits(root string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for kind, dir := range affectsDirs {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			out[kind] = nil
			continue
		}
		set := map[string]bool{}
		for _, e := range entries {
			if e.IsDir() {
				set[e.Name()] = true
			}
		}
		out[kind] = set
	}
	return out
}

func stringOf(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func affectsList(affects map[string]any, kind string) ([]string, bool) {
	raw, ok := affects[kind]
	if !ok {
		return nil, false
	}
	if raw == nil {
		return nil, true
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		out = append(out, stringOf(v))
	}
	return out, true
}

// lintResult separates what blocks (problems) from what only informs (warnings).
type lintResult struct {
	Problems []string
	Warnings []string
	Count    int
}

// lintADRs validates the decision log. Structure, placeholders, status values
// and supersede targets are problems. Names in `affects` are historical: an
// ADR may be written before its brick exists or outlive it, so an unknown
// name is only a warning. Gaps in numbering are a warning.
func lintADRs(root string) (lintResult, error) {
	dir := filepath.Join(root, adrDir)
	records, broken, err := readADRs(dir)
	if err != nil {
		return lintResult{}, err
	}
	units := knownUnits(root)
	var res lintResult
	seen := map[string]string{}
	numbers := map[int]bool{}

	for _, rec := range records {
		name := rec.File
		if rec.Number == "" {
			res.Problems = append(res.Problems, fmt.Sprintf("%s: filename must look like NNNN-kebab-title.md", name))
			continue
		}
		if prev, dup := seen[rec.Number]; dup {
			res.Problems = append(res.Problems, fmt.Sprintf("%s: ADR number %s is already used by %s; numbers are never reused", name, rec.Number, prev))
		} else {
			seen[rec.Number] = name
			n, _ := strconv.Atoi(rec.Number)
			numbers[n] = true
		}
		if err, bad := broken[name]; bad {
			res.Problems = append(res.Problems, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		data := rec.Data
		for _, field := range adrRequired {
			if stringOf(data[field]) == "" {
				res.Problems = append(res.Problems, fmt.Sprintf("%s: missing required field `%s`", name, field))
			}
		}
		if placeholderRE.MatchString(stringOf(data["date"])) || placeholderRE.MatchString(stringOf(data["decision-makers"])) || placeholderRE.MatchString(rec.Title) {
			res.Problems = append(res.Problems, fmt.Sprintf("%s: %s", name, placeholderMsg))
		}
		status := strings.ToLower(stringOf(data["status"]))
		if status != "" && !hasAnyPrefix(status, statusPrefix) {
			res.Problems = append(res.Problems, fmt.Sprintf("%s: status '%s' is not one of proposed / accepted / deprecated / superseded by ADR-NNNN", name, stringOf(data["status"])))
		}
		if m := supersededRE.FindStringSubmatch(status); m != nil {
			if !adrExists(dir, m[1]) {
				res.Problems = append(res.Problems, fmt.Sprintf("%s: status says superseded by ADR-%s, but no such ADR exists", name, m[1]))
			}
		}
		if impact := stringOf(data["interface-impact"]); impact != "" && !validImpact[strings.ToLower(impact)] {
			res.Problems = append(res.Problems, fmt.Sprintf("%s: interface-impact '%s' must be one of none / new / breaking", name, impact))
		}
		if raw, ok := data["affects"]; ok {
			problems, warnings := checkAffects(name, raw, units)
			res.Problems = append(res.Problems, problems...)
			res.Warnings = append(res.Warnings, warnings...)
		}
	}
	res.Count = len(seen)
	for n := 1; n <= len(numbers); n++ {
		if !numbers[n] {
			res.Warnings = append(res.Warnings, fmt.Sprintf("numbering has a gap at %04d; numbers are sequential and never reused", n))
			break
		}
	}
	return res, nil
}

func adrExists(dir, number string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, number+"-*.md"))
	return len(matches) > 0
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func checkAffects(name string, raw any, units map[string]map[string]bool) (problems, warnings []string) {
	affects, ok := raw.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: `affects` must have components/bases/projects lists", name)}, nil
	}
	for _, kind := range affectsKinds {
		values, ok := affectsList(affects, kind)
		if _, present := affects[kind]; !present {
			problems = append(problems, fmt.Sprintf("%s: `affects` is missing `%s`", name, kind))
			continue
		}
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: `affects.%s` must be a list", name, kind))
			continue
		}
		for _, v := range values {
			if v == "*" || units[kind][v] {
				continue
			}
			warnings = append(warnings, fmt.Sprintf("%s: `affects.%s` names '%s', which does not exist today (fine if it is planned or was removed)", name, kind, v))
		}
	}
	return problems, warnings
}

// runADRLint validates the decision log under docs/adr/.
func runADRLint() (int, error) {
	if _, err := os.Stat(adrDir); err != nil {
		fmt.Printf("No %s/ directory found, nothing to lint.\n", adrDir)
		return 0, nil
	}
	res, err := lintADRs(".")
	if err != nil {
		return 1, err
	}
	for _, w := range res.Warnings {
		fmt.Printf("  warning: %s\n", w)
	}
	if len(res.Problems) > 0 {
		fmt.Println("ADR lint found problems:")
		fmt.Println()
		for _, p := range res.Problems {
			fmt.Printf("  - %s\n", p)
		}
		fmt.Println()
		fmt.Printf("Fix the files above. Each ADR must follow %s/%s. If you are unsure what a field means, see CONTRIBUTING.md.\n", adrDir, adrTemplate)
		return 1, nil
	}
	fmt.Printf("ADR lint passed: %d record(s) look good.\n", res.Count)
	return 0, nil
}

// runADRNew creates the next ADR from the template: picks the next number,
// slugs the title, fills the date, and prints the path.
func runADRNew(args []string) (int, error) {
	title := strings.TrimSpace(strings.Join(args, " "))
	if title == "" {
		return 2, fmt.Errorf("usage: poly adr new <title words>")
	}
	path, err := newADR(".", title, time.Now())
	if err != nil {
		return 1, err
	}
	fmt.Printf("Created %s\n", path)
	fmt.Println("Next: set `affects` to the bricks and projects this touches, write the sections, then commit it with the change it records.")
	return 0, nil
}

func newADR(root, title string, now time.Time) (string, error) {
	dir := filepath.Join(root, adrDir)
	records, _, err := readADRs(dir)
	if err != nil {
		return "", err
	}
	next := 1
	for _, r := range records {
		if n, err := strconv.Atoi(r.Number); err == nil && n >= next {
			next = n + 1
		}
	}
	number := fmt.Sprintf("%04d", next)
	slug := strings.Trim(nonAlnumRE.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if slug == "" {
		return "", fmt.Errorf("title %q has no letters or digits to make a filename from", title)
	}
	tmpl, err := os.ReadFile(filepath.Join(dir, adrTemplate))
	if err != nil {
		return "", fmt.Errorf("reading the template: %w", err)
	}
	text := string(tmpl)
	text = strings.Replace(text, "date: YYYY-MM-DD", "date: "+now.Format("2006-01-02"), 1)
	text = strings.Replace(text, "# ADR-NNNN: Short title of the decision", "# ADR-"+number+": "+title, 1)
	// Drop the "copy this file" comment; it is about the template, not this record.
	if start := strings.Index(text, "<!--"); start >= 0 {
		if end := strings.Index(text[start:], "-->"); end >= 0 {
			text = text[:start] + strings.TrimLeft(text[start+end+3:], "\n")
		}
	}
	path := filepath.Join(dir, number+"-"+slug+".md")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists", path)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// runADRIndex regenerates docs/adr/index/: one file per brick listing the ADRs
// that affect it, plus a README with the full list. Generated output; do not edit by hand.
func runADRIndex() (int, error) {
	if _, err := os.Stat(adrDir); err != nil {
		fmt.Printf("No %s/ directory, nothing to index.\n", adrDir)
		return 0, nil
	}
	n, views, err := writeADRIndex(".")
	if err != nil {
		return 1, err
	}
	fmt.Printf("Wrote index for %d ADR(s) across %d brick view(s).\n", n, views)
	return 0, nil
}

const generatedNote = "_Generated by `go run ./tools/poly adr index`. Do not edit by hand._"

// writeADRIndex builds one view per brick or project. An ADR whose `affects`
// says "*" for a kind appears in every view of that kind, so a per-brick
// view always includes the workspace-wide decisions.
func writeADRIndex(root string) (int, int, error) {
	records, _, err := readADRs(filepath.Join(root, adrDir))
	if err != nil {
		return 0, 0, err
	}
	type entry struct{ File, Title, Status string }
	var all []entry
	units := knownUnits(root)
	// Every unit that exists on disk or is named by any ADR gets a view.
	viewsFor := map[string]map[string]bool{}
	for _, kind := range affectsKinds {
		viewsFor[kind] = map[string]bool{}
		for u := range units[kind] {
			viewsFor[kind][u] = true
		}
	}
	for _, rec := range records {
		affects, _ := rec.Data["affects"].(map[string]any)
		for _, kind := range affectsKinds {
			values, _ := affectsList(affects, kind)
			for _, v := range values {
				if v != "*" {
					viewsFor[kind][v] = true
				}
			}
		}
	}

	byView := map[string][]entry{}
	for _, rec := range records {
		e := entry{rec.File, rec.Title, stringOf(rec.Data["status"])}
		all = append(all, e)
		affects, _ := rec.Data["affects"].(map[string]any)
		for _, kind := range affectsKinds {
			values, _ := affectsList(affects, kind)
			for _, v := range values {
				if v == "*" {
					byView[kind+"-_all_"] = append(byView[kind+"-_all_"], e)
					for u := range viewsFor[kind] {
						byView[kind+"-"+u] = append(byView[kind+"-"+u], e)
					}
				} else {
					byView[kind+"-"+v] = append(byView[kind+"-"+v], e)
				}
			}
		}
	}
	// An ADR can reach the same view twice ("*" and a name); keep one entry each.
	for k, entries := range byView {
		seen := map[string]bool{}
		var uniq []entry
		for _, e := range entries {
			if !seen[e.File] {
				seen[e.File] = true
				uniq = append(uniq, e)
			}
		}
		sort.Slice(uniq, func(i, j int) bool { return uniq[i].File < uniq[j].File })
		byView[k] = uniq
	}

	line := func(e entry) string {
		suffix := ""
		if e.Status != "" {
			suffix = " (" + e.Status + ")"
		}
		return fmt.Sprintf("- [%s](../%s)%s", e.Title, e.File, suffix)
	}

	indexDir := filepath.Join(root, adrIndexDir)
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return 0, 0, err
	}
	old, _ := filepath.Glob(filepath.Join(indexDir, "*.md"))
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			return 0, 0, err
		}
	}

	keys := make([]string, 0, len(byView))
	for k := range byView {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		kind, unit, _ := strings.Cut(k, "-")
		label := unit
		if unit == "_all_" {
			label = "* (every " + strings.TrimSuffix(kind, "s") + ")"
		}
		var body []string
		body = append(body, fmt.Sprintf("# ADRs affecting %s: %s", strings.TrimSuffix(kind, "s"), label))
		for _, e := range byView[k] {
			body = append(body, line(e))
		}
		body = append(body, "", generatedNote, "")
		if err := os.WriteFile(filepath.Join(indexDir, k+".md"), []byte(strings.Join(body, "\n")), 0o644); err != nil {
			return 0, 0, err
		}
	}

	readme := []string{
		"# ADR index",
		"Generated view of the decision log. Do not edit by hand; run `go run ./tools/poly adr index`.",
		"",
		"## All ADRs",
	}
	for _, e := range all {
		readme = append(readme, line(e))
	}
	readme = append(readme, "", "## By brick or project", "Each view includes the workspace-wide ADRs (`affects: [\"*\"]`).")
	for _, k := range keys {
		kind, unit, _ := strings.Cut(k, "-")
		readme = append(readme, fmt.Sprintf("- %s `%s`: [%s.md](%s.md) (%d)", strings.TrimSuffix(kind, "s"), unit, k, k, len(byView[k])))
	}
	readme = append(readme, "")
	if err := os.WriteFile(filepath.Join(indexDir, "README.md"), []byte(strings.Join(readme, "\n")), 0o644); err != nil {
		return 0, 0, err
	}
	return len(all), len(byView), nil
}
