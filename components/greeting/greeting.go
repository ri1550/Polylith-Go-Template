// Package greeting turns a name into a friendly greeting.
//
// Public interface (the contract other bricks depend on): Greet.
// Everything unexported is implementation and free to change as long as the
// tests keep passing.
//
// EXAMPLE brick demonstrating the interface/implementation split. Delete it
// once you have your own components.
package greeting

import "strings"

// Greet returns a friendly greeting for name. Blank input greets the world.
func Greet(name string) string {
	return "Hello, " + clean(name) + "!"
}

// clean is unexported: private, not part of the public interface.
func clean(name string) string {
	if s := strings.TrimSpace(name); s != "" {
		return s
	}
	return "world"
}
