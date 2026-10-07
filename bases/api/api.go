// Package api is a thin entry point: it parses what the outside world hands it
// and delegates to components. It holds no business logic.
//
// EXAMPLE brick. Delete it once you have your own bases.
package api

import (
	"fmt"
	"io"

	"github.com/myorg/workspace/components/greeting"
)

// Run is what a project's main calls. It greets the first argument, or the
// world when there is none, and writes the result to out.
func Run(args []string, out io.Writer) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	_, err := fmt.Fprintln(out, greeting.Greet(name))
	return err
}
