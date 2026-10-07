package api_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/myorg/workspace/bases/api"
)

// ExampleRun shows the calling convention: arguments, an output, and the
// greeter the base delegates to.
func ExampleRun() {
	shout := func(name string) string { return strings.ToUpper("hi " + name) }
	if err := api.Run([]string{"ada"}, os.Stdout, shout); err != nil {
		fmt.Println(err)
	}
	// Output: HI ADA
}

func TestRunPassesTheFirstArgumentToTheGreeter(t *testing.T) {
	var out bytes.Buffer
	echo := func(name string) string { return "<" + name + ">" }
	if err := api.Run([]string{"Ada", "ignored"}, &out, echo); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "<Ada>\n" {
		t.Errorf("got %q", got)
	}
}

func TestRunGreetsAnEmptyNameWithoutArguments(t *testing.T) {
	var out bytes.Buffer
	echo := func(name string) string { return "<" + name + ">" }
	if err := api.Run(nil, &out, echo); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "<>\n" {
		t.Errorf("got %q", got)
	}
}
