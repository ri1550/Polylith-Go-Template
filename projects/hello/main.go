// Project hello: the deployable artifact built from the api base.
//
// A project is wiring only: this file hands control to a base and holds no
// business logic. Put deployment infrastructure (a Dockerfile, deploy scripts)
// next to it. Build it with `go build -o bin/hello ./projects/hello`.
package main

import (
	"fmt"
	"os"

	"github.com/myorg/workspace/bases/api"
)

func main() {
	if err := api.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
