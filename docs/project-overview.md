# Project Overview

> Generated: 2026-03-27

## What is Forgent?

Forgent is a **skill compiler** for AI agents. You define agent skills once in YAML with explicit contracts (consumes/produces, guardrails, security facets), and Forgent compiles them into framework-native formats for multiple targets.

**Module path**: `github.com/mirandaguillaume/forgent`

## Core Concept

Skills are **pure interfaces** described by 5 behavioral facets:

| Facet | Defines |
|-------|---------|
| Context | Data I/O contract (consumes/produces) and memory type |
| Strategy | Tools, approach, steps, effort level |
| Guardrails | Rules, limits, constraints |
| Observability | Trace level and metrics |
| Security | Filesystem, network, secrets, sandboxing |

Agents are **compositions of skills** with orchestration strategies (sequential, parallel, parallel-then-merge, adaptive) organized in flat lists or named stages.

## Build Targets

| Target | Output | Description |
|--------|--------|-------------|
| `claude` | `.claude/skills/`, `.claude/agents/` | Claude Code SKILL.md + agent markdown |
| `copilot` | `.github/skills/`, `.github/agents/` | GitHub Copilot skill + agent + instructions |
| `forgent` | `.forgent/<name>/main.go` | Standalone executable Go binary with DAG |

## Tech Stack

- **Language**: Go 1.24
- **CLI**: Cobra
- **Testing**: testify
- **CI/CD**: GitHub Actions (test + Hugo site + release-please + goreleaser)

## Repository Structure

- `cmd/forgent/` — CLI entry point
- `pkg/` — Public API (model types, spec registry, DAG engine, analysis pipeline)
- `internal/` — Private implementation (commands, generators, scanner, importer, benchmarks)
- `skills/` — 18 skill YAML definitions (code review pipeline)
- `agents/` — 3 agent compositions (code-reviewer, ci-reviewer, staged-reviewer)
- `contracts/` — Output format specifications
- `templates/` — Scaffold templates for `forgent init`

## Key Commands

```bash
forgent init                    # Scaffold project
forgent skill create <name>     # Create skill YAML
forgent lint [path]             # Lint skills
forgent doctor [path]           # Full diagnostics + health score
forgent score [path]            # Design quality scoring
forgent build --target <t>      # Compile to target format
forgent build --watch           # Live rebuild on changes
forgent build --compact         # Single-file compact mode
forgent import <source>         # Import existing agent .md files
```

## Links

- [Architecture](./architecture.md) — detailed technical architecture
- [Source Tree](./source-tree-analysis.md) — annotated directory structure
- [Development Guide](./development-guide.md) — build, test, contribute
- [ROADMAP](./ROADMAP.md) — product roadmap
