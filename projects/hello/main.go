// Project hello: the deployable built from the api base and the greeting component.
//
// A project is the composition root and nothing else (ADR-0009): main() reads
// the deployment's environment, constructs components, hands them to a base,
// and runs it. One file, no business logic. Deployment files (a Dockerfile,
// deploy scripts) sit next to it. Build with `go build -o bin/hello ./projects/hello`.
package main

import (
	"fmt"
	"os"

	"github.com/myorg/workspace/bases/api"
	"github.com/myorg/workspace/components/greeting"
)

func main() {
	if err := api.Run(os.Args[1:], os.Stdout, greeting.Greet); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
