# ADR index
Generated view of the decision log. Do not edit by hand; run `go run ./tools/poly adr index`.

## All ADRs
- [ADR-0001: Record decisions using MADR](../0001-record-decisions-using-madr.md) (accepted)
- [ADR-0002: Adopt the Polylith architecture](../0002-adopt-polylith-architecture.md) (accepted)
- [ADR-0003: Adopt Best Simple System for Now as the guiding principle](../0003-adopt-bssn-as-guiding-principle.md) (accepted)
- [ADR-0004: Use a single Go module for the whole workspace](../0004-use-a-single-go-module.md) (accepted)
- [ADR-0005: Capture context in the codebase](../0005-capture-context-in-the-codebase.md) (accepted)
- [ADR-0006: Enforce the context system with automated checks](../0006-enforce-the-process-with-automated-checks.md) (accepted)
- [ADR-0007: Enforce code quality with automated checks](../0007-enforce-code-quality-with-automated-checks.md) (accepted)
- [ADR-0008: Build the Polylith tooling in the repository](../0008-build-polylith-tooling-in-repo.md) (accepted)
- [ADR-0009: Projects are the composition root](../0009-projects-are-the-composition-root.md) (accepted)
- [ADR-0010: Make the context system's gates exact](../0010-make-the-gates-exact.md) (accepted)

## By brick or project
Each view includes the workspace-wide ADRs (`affects: ["*"]`).
- base `_all_`: [bases-_all_.md](bases-_all_.md) (7)
- base `api`: [bases-api.md](bases-api.md) (7)
- component `_all_`: [components-_all_.md](components-_all_.md) (7)
- component `greeting`: [components-greeting.md](components-greeting.md) (7)
- project `_all_`: [projects-_all_.md](projects-_all_.md) (8)
- project `hello`: [projects-hello.md](projects-hello.md) (8)
