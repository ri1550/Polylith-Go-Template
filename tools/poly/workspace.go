package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Kind says where a package sits in the Polylith layout.
type Kind string

const (
	KindComponent   Kind = "component"
	KindBase        Kind = "base"
	KindProject     Kind = "project"
	KindDevelopment Kind = "development"
	KindTools       Kind = "tools"
	KindOther       Kind = "other" // Go code outside the layout; `poly check` rejects it
)

// Pkg is one Go package in the workspace.
type Pkg struct {
	ImportPath string
	Rel        string   // directory relative to the module root, forward slashes
	Name       string   // package clause name
	Imports    []string // regular and test imports, deduplicated
	Kind       Kind
	Brick      string // brick or project name (the directory under components/, bases/ or projects/)
	GoFiles    int    // number of non-test Go files in the package
	Err        string // why the package failed to load, if it did
}

// IsRoot reports whether the package is a brick's or project's root package
// (components/<name>, bases/<name>, projects/<name>) rather than a subpackage.
func (p *Pkg) IsRoot() bool {
	return p.Brick != "" && strings.Count(p.Rel, "/") == 1
}

// Workspace is the set of packages in the module, classified.
type Workspace struct {
	Module string
	Pkgs   []*Pkg
}

// ByImport indexes the packages by import path.
func (w *Workspace) ByImport() map[string]*Pkg {
	m := make(map[string]*Pkg, len(w.Pkgs))
	for _, p := range w.Pkgs {
		m[p.ImportPath] = p
	}
	return m
}

// classify maps a module-relative directory to its kind and brick name.
func classify(rel string) (Kind, string) {
	parts := strings.Split(rel, "/")
	brick := ""
	if len(parts) > 1 {
		brick = parts[1]
	}
	switch parts[0] {
	case "components":
		return KindComponent, brick
	case "bases":
		return KindBase, brick
	case "projects":
		return KindProject, brick
	case "development":
		return KindDevelopment, brick
	case "tools":
		return KindTools, brick
	}
	return KindOther, ""
}

type listEntry struct {
	ImportPath   string
	Dir          string
	Name         string
	Imports      []string
	TestImports  []string
	XTestImports []string
	GoFiles      []string
	Module       *struct{ Path, Dir string }
	Error        *struct{ Err string }
}

// loadWorkspace runs `go list` over the whole module and classifies every package.
func loadWorkspace() (*Workspace, error) {
	cmd := exec.Command("go", "list", "-e",
		"-json=ImportPath,Dir,Name,Imports,TestImports,XTestImports,GoFiles,Module,Error", "./...")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %v\n%s", err, stderr.String())
	}

	ws := &Workspace{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var e listEntry
		if err := dec.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("parsing go list output: %w", err)
		}
		if e.Module == nil {
			continue
		}
		if ws.Module == "" {
			ws.Module = e.Module.Path
		}
		rel, err := filepath.Rel(e.Module.Dir, e.Dir)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		kind, brick := classify(rel)
		loadErr := ""
		if e.Error != nil {
			loadErr = strings.TrimSpace(e.Error.Err)
		}
		ws.Pkgs = append(ws.Pkgs, &Pkg{
			ImportPath: e.ImportPath,
			Rel:        rel,
			Name:       e.Name,
			Imports:    dedupe(e.Imports, e.TestImports, e.XTestImports),
			Kind:       kind,
			Brick:      brick,
			GoFiles:    len(e.GoFiles),
			Err:        loadErr,
		})
	}
	sort.Slice(ws.Pkgs, func(i, j int) bool { return ws.Pkgs[i].Rel < ws.Pkgs[j].Rel })
	return ws, nil
}

func dedupe(lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, l := range lists {
		for _, s := range l {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	sort.Strings(out)
	return out
}

// git runs a git command and returns its stdout.
func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// gitWithStdin runs a git command with the given stdin and returns its stdout.
func gitWithStdin(stdin string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func gitLines(args ...string) ([]string, error) {
	out, err := git(args...)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines, nil
}

// brickOfPath maps a repository path (components/<brick>/file.go) to the kind
// and brick it belongs to. Files directly under a layout directory, such as
// components/.keep, belong to no brick.
func brickOfPath(path string) (Kind, string) {
	if strings.Count(path, "/") < 2 {
		return KindOther, ""
	}
	return classify(path)
}
