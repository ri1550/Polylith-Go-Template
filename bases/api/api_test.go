package api_test

import (
	"bytes"
	"testing"

	"github.com/myorg/workspace/bases/api"
)

func TestRunDelegatesToGreeting(t *testing.T) {
	var out bytes.Buffer
	if err := api.Run([]string{"Ada"}, &out); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "Hello, Ada!\n" {
		t.Errorf("got %q", got)
	}
}

func TestRunGreetsTheWorldWithoutArguments(t *testing.T) {
	var out bytes.Buffer
	if err := api.Run(nil, &out); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "Hello, world!\n" {
		t.Errorf("got %q", got)
	}
}
