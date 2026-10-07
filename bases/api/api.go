// Package api is a thin entry point: it turns command-line arguments into a
// call on the greeter it was given and writes the result. It holds no
// business logic and constructs nothing; the project wires it (ADR-0009).
//
// EXAMPLE brick. Delete it once you have your own bases.
package api

import (
	"fmt"
	"io"
)

// Greeter is what the base needs from the outside: any function that greets
// a name. The project passes greeting.Greet; tests pass a fake.
type Greeter func(name string) string

// Run greets the first argument, or the world when there is none, using
// greet, and writes the result to out.
func Run(args []string, out io.Writer, greet Greeter) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	_, err := fmt.Fprintln(out, greet(name))
	return err
}
