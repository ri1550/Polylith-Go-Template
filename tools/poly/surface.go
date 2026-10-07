package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"strings"
)

// surface returns the public interface of one package: every exported
// declaration, rendered one per line without bodies or comments, sorted.
// files maps a file name to its source; _test.go files should not be passed.
//
// This is what "the brick's public interface" means in this workspace: the
// Go equivalent of a Polylith __init__.py or interface.clj is the set of
// exported identifiers in the brick's root package.
func surface(files map[string][]byte) ([]string, error) {
	fset := token.NewFileSet()
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)

	var lines []string
	for _, name := range names {
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if strings.HasSuffix(f.Name.Name, "_test") {
			continue
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() || (d.Recv != nil && !exportedReceiver(d.Recv)) {
					continue
				}
				d.Body, d.Doc = nil, nil
				lines = append(lines, render(fset, d))
			case *ast.GenDecl:
				if d.Tok == token.IMPORT {
					continue
				}
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if !s.Name.IsExported() {
							continue
						}
						s.Doc, s.Comment = nil, nil
						stripUnexportedFields(s.Type)
						lines = append(lines, "type "+render(fset, s))
					case *ast.ValueSpec:
						if !keepExportedNames(s) {
							continue
						}
						lines = append(lines, d.Tok.String()+" "+render(fset, s))
					}
				}
			}
		}
	}
	sort.Strings(lines)
	return lines, nil
}

// exportedReceiver reports whether a method's receiver type is exported,
// looking through pointers and type parameters.
func exportedReceiver(recv *ast.FieldList) bool {
	if recv == nil || len(recv.List) == 0 {
		return false
	}
	expr := recv.List[0].Type
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		case *ast.Ident:
			return e.IsExported()
		default:
			return false
		}
	}
}

// stripUnexportedFields removes unexported fields from a struct type so that
// private layout changes do not count as interface changes. Embedded fields
// are kept when their type name is exported.
func stripUnexportedFields(expr ast.Expr) {
	st, ok := expr.(*ast.StructType)
	if !ok || st.Fields == nil {
		return
	}
	var kept []*ast.Field
	for _, f := range st.Fields.List {
		f.Doc, f.Comment = nil, nil
		if len(f.Names) == 0 {
			if exportedTypeName(f.Type) {
				kept = append(kept, f)
			}
			continue
		}
		var names []*ast.Ident
		for _, n := range f.Names {
			if n.IsExported() {
				names = append(names, n)
			}
		}
		if len(names) > 0 {
			f.Names = names
			kept = append(kept, f)
		}
	}
	st.Fields.List = kept
}

func exportedTypeName(expr ast.Expr) bool {
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		case *ast.SelectorExpr:
			return e.Sel.IsExported()
		case *ast.Ident:
			return e.IsExported()
		default:
			return false
		}
	}
}

// keepExportedNames narrows a const/var spec to its exported names and reports
// whether any remain. Values are kept in step with the names when possible,
// since a changed constant value is a changed contract.
func keepExportedNames(s *ast.ValueSpec) bool {
	var names []*ast.Ident
	var values []ast.Expr
	paired := len(s.Values) == len(s.Names)
	for i, n := range s.Names {
		if n.IsExported() {
			names = append(names, n)
			if paired {
				values = append(values, s.Values[i])
			}
		}
	}
	if len(names) == 0 {
		return false
	}
	s.Names = names
	if paired {
		s.Values = values
	}
	s.Doc, s.Comment = nil, nil
	return true
}

// render prints a node on a single line. Comments are not printed because the
// node is printed without its file's comment map.
func render(fset *token.FileSet, node any) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return fmt.Sprintf("<unprintable: %v>", err)
	}
	var parts []string
	for _, l := range strings.Split(buf.String(), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			parts = append(parts, strings.Join(strings.Fields(l), " "))
		}
	}
	out := strings.Join(parts, "; ")
	out = strings.ReplaceAll(out, "{; ", "{ ")
	out = strings.ReplaceAll(out, "; }", " }")
	return out
}
