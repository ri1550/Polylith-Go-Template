// Package greeting turns a name into a friendly greeting.
//
// Blank input is normalized here, so every caller gets the same fallback
// instead of each re-implementing it. Everything unexported is implementation
// and free to change while the tests and the Example keep passing.
//
// EXAMPLE brick showing the interface/implementation split. Delete it once
// you have your own components.
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
