package greeting_test

import (
	"testing"

	"github.com/myorg/workspace/components/greeting"
)

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
