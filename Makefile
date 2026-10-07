# Everyday entry points. Each target is one or two plain commands; read the
# Makefile if you want to run them by hand.
.PHONY: hooks check test lint build all adr-index

hooks:      ## turn the git hooks on for this clone (run once after cloning)
	git config core.hooksPath .githooks
	@echo "Hooks enabled. They run on every git commit."

check:      ## workspace checks: compile, vet, brick boundaries, decision log
	go build ./...
	go vet ./...
	go run ./tools/poly check
	go run ./tools/poly adr lint

test:       ## run every test
	go test ./...

lint:       ## golangci-lint (install: https://golangci-lint.run/docs/welcome/install/)
	golangci-lint run

build:      ## build every project into bin/
	@mkdir -p bin
	@for p in projects/*/; do \
		name=$$(basename $$p); \
		echo "building bin/$$name"; \
		go build -o bin/$$name ./$$p || exit 1; \
	done

adr-index:  ## regenerate docs/adr/index/
	go run ./tools/poly adr index

all: check test lint
