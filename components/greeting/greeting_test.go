package greeting_test

import (
	"fmt"
	"testing"

	"github.com/myorg/workspace/components/greeting"
)

// ExampleGreet is the usage layer: go doc shows it, go test runs it (ADR-0011).
func ExampleGreet() {
	fmt.Println(greeting.Greet("Ada"))
	fmt.Println(greeting.Greet("   "))
	// Output:
	// Hello, Ada!
	// Hello, world!
}

func TestGreetUsesTheName(t *testing.T) {
	if got := greeting.Greet("Ada"); got != "Hello, Ada!" {
		t.Errorf("got %q", got)
	}
}

func TestGreetDefaultsToWorldWhenBlank(t *testing.T) {
	if got := greeting.Greet("   "); got != "Hello, world!" {
		t.Errorf("got %q", got)
	}
}
