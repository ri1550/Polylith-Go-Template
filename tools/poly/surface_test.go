package main

import (
	"reflect"
	"testing"
)

func TestSurfaceListsOnlyExportedDeclarations(t *testing.T) {
	src := `package x

// Greet says hello.
func Greet(name string) string { return "Hello, " + clean(name) }

func clean(s string) string { return s }

type Config struct {
	Name    string ` + "`json:\"name\"`" + `
	timeout int
	Logger
}

type private struct{ A int }

const (
	Version = "1"
	secret  = "x"
)

var Default, fallback = Config{}, Config{}

func (c *Config) Validate() error { return nil }
func (p private) Hidden() {}
func (c Config) helper() {}

type Store[T any] struct{ items []T }

func (s *Store[T]) Put(v T) {}
`
	got, err := surface(map[string][]byte{"x.go": []byte(src)})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"const Version = \"1\"",
		"func (c *Config) Validate() error",
		"func (s *Store[T]) Put(v T)",
		"func Greet(name string) string",
		"type Config struct { Name string `json:\"name\"`; Logger }",
		"type Store[T any] struct{}",
		"var Default", // the initializer is not part of the contract
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("surface mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestSurfaceIncludesUnexportedTypesReachableFromExportedSignatures(t *testing.T) {
	src := `package x

type builder struct{ n string; Verbose bool }

// NewBuilder returns a builder callers can use.
func NewBuilder() *builder { return &builder{} }

func (b *builder) Name(n string) *builder { b.n = n; return b }

type orphan struct{}

func (o orphan) Unreachable() {}
`
	got, err := surface(map[string][]byte{"x.go": []byte(src)})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"func (b *builder) Name(n string) *builder",
		"func NewBuilder() *builder",
		"type builder struct{ Verbose bool }",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("surface mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestSurfaceIgnoresBodiesCommentsVarInitializersAndTestFiles(t *testing.T) {
	a := map[string][]byte{"x.go": []byte("package x\nimport \"errors\"\nvar ErrNope = errors.New(\"a\")\nfunc F() int { return 1 }\n")}
	b := map[string][]byte{
		"x.go":      []byte("package x\nimport \"errors\"\n// ErrNope is documented now.\nvar ErrNope = errors.New(\"b\")\n// F is documented now.\nfunc F() int {\n\treturn 2\n}\n"),
		"x_test.go": []byte("package x_test\nfunc Helper() {}\n"),
	}
	sa, err := surface(a)
	if err != nil {
		t.Fatal(err)
	}
	sb, err := surface(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sa, sb) {
		t.Errorf("implementation-only changes altered the surface: %q vs %q", sa, sb)
	}
}

func TestDiffLines(t *testing.T) {
	removed, added := diffLines([]string{"a", "b"}, []string{"b", "c"})
	if !reflect.DeepEqual(removed, []string{"a"}) || !reflect.DeepEqual(added, []string{"c"}) {
		t.Errorf("got removed=%q added=%q", removed, added)
	}
}
