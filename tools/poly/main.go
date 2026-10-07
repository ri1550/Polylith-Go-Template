// Command poly is the Polylith tooling for this workspace.
//
// There is no official Polylith tool for Go, so the handful of commands the
// process needs live here, in the same module, checked by the same compiler
// and tests as everything else. See ADR-0008 for the decision and AGENTS.md
// for the rules these commands enforce.
//
// Run it from the workspace root with `go run ./tools/poly <command>`.
package main

import (
	"fmt"
	"os"
)

const usage = `poly: Polylith workspace tooling for this repository.

Usage (from the workspace root):
  go run ./tools/poly check                  validate brick boundaries and the workspace layout
  go run ./tools/poly deps                   show what each brick uses and what each project pulls in
  go run ./tools/poly diff [--since <ref>]   show bricks and projects changed since the last stable-* tag
  go run ./tools/poly interface [--mode pre-commit|ci]
                                             report public interface (exported API) changes and
                                             whether they are recorded in an ADR
  go run ./tools/poly adr lint               validate the decision log under docs/adr/
  go run ./tools/poly adr index              regenerate docs/adr/index/

See AGENTS.md for the rules these commands enforce.
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var (
		code int
		err  error
	)
	switch args[0] {
	case "check":
		code, err = runCheck()
	case "deps":
		code, err = runDeps()
	case "diff":
		code, err = runDiff(args[1:])
	case "interface":
		code, err = runInterface(args[1:])
	case "adr":
		if len(args) < 2 {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
		switch args[1] {
		case "lint":
			code, err = runADRLint()
		case "index":
			code, err = runADRIndex()
		default:
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "poly: unknown command %q\n\n%s", args[0], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "poly %s: %v\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(code)
}
