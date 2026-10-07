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

	"github.com/myorg/workspace/tools/polylith"
)

// The rule vocabulary lives in tools/polylith, shared with the polyvet analyzer.
type (
	Kind = polylith.Kind
	Pkg  = polylith.Pkg
)

// Kinds, re-exported for brevity.
const (
	KindComponent   = polylith.KindComponent
	KindBase        = polylith.KindBase
	KindProject     = polylith.KindProject
	KindDevelopment = polylith.KindDevelopment
	KindTools       = polylith.KindTools
	KindOther       = polylith.KindOther
)

var (
	classify   = polylith.Classify
	importRule = polylith.ImportRule
)

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
