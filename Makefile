# Everyday entry points. Each target is a few plain commands; read it if you
# want to run them by hand. `make all` is what CI runs, minus govulncheck.
.PHONY: hooks check test lint build adr-index all

hooks:      ## turn the git hooks on for this clone (run once after cloning)
	git config core.hooksPath .githooks
	@echo "Hooks enabled. They run on every git commit."

check:      ## layout and boundaries, type check, decision log, tidy go.mod
	go run ./tools/poly check
	go vet ./...
	go run ./tools/poly adr lint
	go mod tidy -diff

test:       ## run every test with the race detector
	go test -race -shuffle=on ./...

lint:       ## golangci-lint (required; install: https://golangci-lint.run/docs/welcome/install/)
	PATH="$$PATH:$$(go env GOPATH)/bin" golangci-lint run

build:      ## build every project into bin/
	@mkdir -p bin
	@found=0; for p in projects/*/; do \
		[ -d "$$p" ] || continue; found=1; \
		name=$$(basename "$$p"); \
		echo "building bin/$$name"; \
		go build -o "bin/$$name" "./$$p" || exit 1; \
	done; [ "$$found" = 1 ] || echo "no projects yet (projects/<name>/main.go)"

adr-index:  ## regenerate docs/adr/index/
	go run ./tools/poly adr index

all: check test lint
