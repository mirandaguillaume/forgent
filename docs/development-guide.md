# Development Guide

> Generated: 2026-03-27

## Prerequisites

- **Go 1.24+** (module uses `go 1.24.2`)
- **Git** for version control
- **Claude CLI** (optional, for benchmark tests)

## Setup

```bash
git clone https://github.com/mirandaguillaume/forgent.git
cd forgent
go mod download
```

## Build

```bash
# Main CLI binary
go build ./cmd/forgent

# Benchmark binary (internal tooling)
go build ./cmd/forgent-bench
```

## Test

```bash
# Run all tests
go test ./...

# Run with race detector (same as CI)
go test ./... -count=1 -race

# Run specific package tests
go test ./pkg/dag/...
go test ./internal/linter/...

# Static analysis
go vet ./...
```

## CI/CD Pipeline

### CI (`ci.yaml`)

Triggered on push to `main` and pull requests:

1. **test** — `go vet ./...` + `go test ./... -count=1 -race` + `go build ./cmd/forgent`
2. **hugo** — builds project website with Hugo (Hextra theme)
3. **deploy** — deploys to GitHub Pages (main branch only)

### Release (`release.yaml`)

Automated on push to `main`:

1. **release-please** — creates release PRs with conventional commit changelogs
2. **goreleaser** — builds cross-platform binaries on tag creation

## Project Conventions

### Package Organization

- `pkg/` — Public, importable packages. Stable API surface.
- `internal/` — Private implementation. Can change freely.
- `cmd/` — Binary entry points. Thin wrappers calling into `internal/cmd`.

### Test Patterns

- Table-driven tests with `testify` assertions
- Temp directory fixtures for file I/O tests
- Test files named `*_test.go` in same package (white-box) or `_test` package (black-box)
- Integration tests in `internal/importer/integration_test.go` and `internal/bench/`

### Generator Registration

Generators register via `init()` functions:

```go
func init() {
    spec.Register("claude", func() spec.Generator {
        return &claudeGenerator{}
    })
}
```

### Linter Rule Registration

Same `init()` pattern:

```go
func init() {
    Register(&noEmptyToolsRule{})
}
```

### YAML Conventions

- Skills: `<name>.skill.yaml`
- Agents: `<name>.agent.yaml`
- Templates in `templates/` directory
- Contracts in `contracts/` directory (markdown)

## Common Development Tasks

### Add a new CLI command

1. Create `internal/cmd/<command>.go` with Cobra command
2. Register in `init()` with `rootCmd.AddCommand()`
3. Add tests in `internal/cmd/<command>_test.go`

### Add a new build target

1. Create `internal/generator/<target>/` package
2. Implement `spec.Generator`, `spec.SkillGenerator`, `spec.AgentGenerator`
3. Register via `init()` → `spec.Register("<target>", factory)`
4. Add tests

### Add a new lint rule

1. Create rule struct implementing `linter.Rule` interface
2. Register via `init()` → `linter.Register(&myRule{})`
3. Add tests in `internal/linter/registry_test.go`

### Add a new skill

```bash
forgent skill create <name> --tools tool1,tool2
# Edit skills/<name>.skill.yaml
forgent lint
forgent build
```
