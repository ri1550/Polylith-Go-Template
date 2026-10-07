# Everyday entry points. Each target is a few plain commands; read it if you
# want to run them by hand. `make all` is what CI runs, minus govulncheck.
.PHONY: hooks check vet test lint build adr-index adopt all

hooks:      ## turn the git hooks on for this clone (run once after cloning)
	git config core.hooksPath .githooks
	@echo "Hooks enabled. They run on every git commit."

check:      ## layout and boundaries, type check, decision log, tidy go.mod
	go run ./tools/poly check
	$(MAKE) --no-print-directory vet
	go run ./tools/poly adr lint
	go run ./tools/poly spec lint
	go mod tidy -diff

vet:        ## go vet, then the Polylith rules as a vet analyzer (file:line at the import)
	go vet ./...
	@tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
		go build -o "$$tmp/polyvet" ./tools/polyvet && go vet -vettool="$$tmp/polyvet" ./...

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

adopt:      ## one-time: make this clone yours (make adopt MODULE=github.com/org/repo [MAKERS="platform team"])
	go run ./tools/poly adopt --module "$(MODULE)" $(if $(MAKERS),--decision-makers "$(MAKERS)",)

all: check test lint
