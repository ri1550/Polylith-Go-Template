package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	adrDir      = "docs/adr"
	adrIndexDir = "docs/adr/index"
	adrTemplate = "0000-adr-template.md"
)

var (
	adrFilenameRE = regexp.MustCompile(`^(\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)
	adrTitleRE    = regexp.MustCompile(`(?m)^#\s+(.*)$`)
	adrRequired   = []string{"status", "date", "decision-makers", "affects", "interface-impact"}
	validImpact   = map[string]bool{"none": true, "new": true, "breaking": true}
	statusPrefix  = []string{"proposed", "accepted", "deprecated", "superseded by adr-"}
	affectsKinds  = []string{"components", "bases", "projects"}
	affectsDirs   = map[string]string{"components": "components", "bases": "bases", "projects": "projects"}
)

// adrRecord is one parsed decision record.
type adrRecord struct {
	File   string
	Number string
	Title  string
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
		rec := adrRecord{File: name, Title: strings.TrimSuffix(name, ".md")}
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
// A nil set means the directory is absent and existence is not checked.
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

// lintADRs validates the decision log and returns every problem found.
func lintADRs(root string) ([]string, int, error) {
	dir := filepath.Join(root, adrDir)
	records, broken, err := readADRs(dir)
	if err != nil {
		return nil, 0, err
	}
	units := knownUnits(root)
	var problems []string
	seen := map[string]string{}

	for _, rec := range records {
		name := rec.File
		if rec.Number == "" {
			problems = append(problems, fmt.Sprintf("%s: filename must look like NNNN-kebab-title.md", name))
			continue
		}
		if prev, dup := seen[rec.Number]; dup {
			problems = append(problems, fmt.Sprintf("%s: ADR number %s is already used by %s; numbers are never reused", name, rec.Number, prev))
		} else {
			seen[rec.Number] = name
		}
		if err, bad := broken[name]; bad {
			problems = append(problems, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		data := rec.Data
		for _, field := range adrRequired {
			if stringOf(data[field]) == "" {
				problems = append(problems, fmt.Sprintf("%s: missing required field `%s`", name, field))
			}
		}
		if status := strings.ToLower(stringOf(data["status"])); status != "" && !hasAnyPrefix(status, statusPrefix) {
			problems = append(problems, fmt.Sprintf("%s: status '%s' is not one of proposed / accepted / deprecated / superseded by ADR-NNNN", name, stringOf(data["status"])))
		}
		if impact := stringOf(data["interface-impact"]); impact != "" && !validImpact[strings.ToLower(impact)] {
			problems = append(problems, fmt.Sprintf("%s: interface-impact '%s' must be one of none / new / breaking", name, impact))
		}
		if raw, ok := data["affects"]; ok {
			problems = append(problems, checkAffects(name, raw, units)...)
		}
	}
	return problems, len(seen), nil
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func checkAffects(name string, raw any, units map[string]map[string]bool) []string {
	affects, ok := raw.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: `affects` must have components/bases/projects lists", name)}
	}
	var problems []string
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
		known := units[kind]
		if known == nil {
			continue
		}
		for _, v := range values {
			if v == "*" || known[v] {
				continue
			}
			problems = append(problems, fmt.Sprintf("%s: `affects.%s` names '%s', but no such %s exists", name, kind, v, strings.TrimSuffix(kind, "s")))
		}
	}
	return problems
}

// runADRLint validates the decision log under docs/adr/.
func runADRLint() (int, error) {
	if _, err := os.Stat(adrDir); err != nil {
		fmt.Printf("No %s/ directory found, nothing to lint.\n", adrDir)
		return 0, nil
	}
	problems, count, err := lintADRs(".")
	if err != nil {
		return 1, err
	}
	if len(problems) > 0 {
		fmt.Println("ADR lint found problems:")
		fmt.Println()
		for _, p := range problems {
			fmt.Printf("  - %s\n", p)
		}
		fmt.Println()
		fmt.Printf("Fix the files above. Each ADR must follow %s/%s. If you are unsure what a field means, see CONTRIBUTING.md.\n", adrDir, adrTemplate)
		return 1, nil
	}
	fmt.Printf("ADR lint passed: %d record(s) look good.\n", count)
	return 0, nil
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

func writeADRIndex(root string) (int, int, error) {
	records, _, err := readADRs(filepath.Join(root, adrDir))
	if err != nil {
		return 0, 0, err
	}
	type entry struct{ File, Title, Status string }
	var all []entry
	byUnit := map[string][]entry{}
	for _, rec := range records {
		e := entry{rec.File, rec.Title, stringOf(rec.Data["status"])}
		all = append(all, e)
		affects, _ := rec.Data["affects"].(map[string]any)
		for _, kind := range affectsKinds {
			values, _ := affectsList(affects, kind)
			for _, v := range values {
				byUnit[kind+"-"+v] = append(byUnit[kind+"-"+v], e)
			}
		}
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

	keys := make([]string, 0, len(byUnit))
	for k := range byUnit {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		kind, unit, _ := strings.Cut(k, "-")
		safe := unit
		if unit == "*" {
			safe = "_all_"
		}
		var body []string
		body = append(body, fmt.Sprintf("# ADRs affecting %s: %s", strings.TrimSuffix(kind, "s"), unit))
		for _, e := range byUnit[k] {
			body = append(body, line(e))
		}
		body = append(body, "", generatedNote, "")
		if err := os.WriteFile(filepath.Join(indexDir, kind+"-"+safe+".md"), []byte(strings.Join(body, "\n")), 0o644); err != nil {
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
	readme = append(readme, "", "## By brick")
	for _, k := range keys {
		kind, unit, _ := strings.Cut(k, "-")
		safe := unit
		if unit == "*" {
			safe = "_all_"
		}
		readme = append(readme, fmt.Sprintf("- %s `%s`: [%s-%s.md](%s-%s.md) (%d)",
			strings.TrimSuffix(kind, "s"), unit, kind, safe, kind, safe, len(byUnit[k])))
	}
	readme = append(readme, "")
	if err := os.WriteFile(filepath.Join(indexDir, "README.md"), []byte(strings.Join(readme, "\n")), 0o644); err != nil {
		return 0, 0, err
	}
	return len(all), len(byUnit), nil
}
