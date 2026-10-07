// Command polyvet is the Polylith rules as a go vet analyzer, so violations
// are reported at the offending import with a file and line, the way any
// other vet finding is. The rules themselves live in tools/polylith and are
// the same ones `poly check` enforces; `poly check` keeps the cross-package
// rules (unique brick names, every package loads) and the summary, this
// analyzer keeps the per-package ones close to the editor.
//
// Run it with the tool built first, since -vettool needs a binary:
//
//	go build -o /tmp/polyvet ./tools/polyvet && go vet -vettool=/tmp/polyvet ./...
//
// -vettool replaces the standard vet analyzers for that invocation, so keep
// running the plain `go vet ./...` as well (the Makefile and hook do both).
package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/myorg/workspace/tools/polylith"
)

var analyzer = &analysis.Analyzer{
	Name: "polylith",
	Doc:  "enforce the Polylith layout and dependency rules of this workspace (see AGENTS.md > Polylith rules)",
	Run:  run,
}

func main() { singlechecker.Main(analyzer) }

func run(pass *analysis.Pass) (any, error) {
	if len(pass.Files) == 0 {
		return nil, nil
	}
	dir := filepath.Dir(pass.Fset.Position(pass.Files[0].Pos()).Filename)
	root, module, ok := moduleOf(dir)
	if !ok {
		return nil, nil // not inside a module; nothing to say
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return nil, nil
	}
	rel = filepath.ToSlash(rel)
	kind, brick := polylith.Classify(rel)
	from := &polylith.Pkg{ImportPath: pass.Pkg.Path(), Rel: rel, Name: pass.Pkg.Name(), Kind: kind, Brick: brick}

	// go vet analyzes a package twice when it has tests: once with its
	// non-test files and once including them. Check each file in exactly
	// one of those passes so nothing is reported twice.
	testPass := false
	for _, f := range pass.Files {
		if strings.HasSuffix(pass.Fset.Position(f.Pos()).Filename, "_test.go") {
			testPass = true
			break
		}
	}
	importsBase := false
	for _, f := range pass.Files {
		isTest := strings.HasSuffix(pass.Fset.Position(f.Pos()).Filename, "_test.go")
		if !isTest {
			from.GoFiles++
		}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil || !strings.HasPrefix(path, module+"/") {
				continue
			}
			trel := strings.TrimPrefix(path, module+"/")
			tkind, tbrick := polylith.Classify(trel)
			to := &polylith.Pkg{ImportPath: path, Rel: trel, Kind: tkind, Brick: tbrick}
			if tkind == polylith.KindBase {
				importsBase = true
			}
			if isTest != testPass {
				continue
			}
			if msg := polylith.ImportRule(from, to); msg != "" {
				pass.Reportf(imp.Pos(), "%s imports %s: %s", rel, trel, msg)
			}
		}
	}
	if !testPass {
		for _, msg := range polylith.LayoutProblems(from, importsBase) {
			pass.Reportf(pass.Files[0].Package, "%s", msg)
		}
	}
	return nil, nil
}

var (
	modMu    sync.Mutex
	modCache = map[string][2]string{} // dir -> {root, module path}
)

// moduleOf finds the go.mod that governs dir and returns the module root and path.
func moduleOf(dir string) (root, module string, ok bool) {
	modMu.Lock()
	defer modMu.Unlock()
	if v, hit := modCache[dir]; hit {
		return v[0], v[1], v[1] != ""
	}
	for d := dir; ; d = filepath.Dir(d) {
		if m := modulePath(filepath.Join(d, "go.mod")); m != "" {
			modCache[dir] = [2]string{d, m}
			return d, m, true
		}
		if filepath.Dir(d) == d {
			modCache[dir] = [2]string{"", ""}
			return "", "", false
		}
	}
}

func modulePath(gomod string) string {
	f, err := os.Open(gomod)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); strings.HasPrefix(line, "module ") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "module ")), "\"")
		}
	}
	return ""
}
