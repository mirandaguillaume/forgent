# Forgent — Project Documentation Index

> Generated: 2026-03-27 | Scan: exhaustive | Go CLI monolith

## Project Overview

- **Type:** CLI monolith
- **Primary Language:** Go 1.24
- **Architecture:** Pipeline with plugin registry
- **Module:** `github.com/mirandaguillaume/forgent`

## Quick Reference

- **Framework:** Cobra (CLI), testify (testing)
- **Entry Point:** `cmd/forgent/main.go`
- **Architecture Pattern:** Parse → Lint → Analyze → Enrich → Generate
- **Build Targets:** Claude Code, GitHub Copilot, standalone Go runtime

## Generated Documentation

- [Project Overview](./project-overview.md) — what Forgent is, core concepts, tech stack summary
- [Architecture](./architecture.md) — domain model, package dependencies, build targets, DAG engine, analysis pipeline
- [Source Tree Analysis](./source-tree-analysis.md) — annotated directory tree with all critical folders
- [Development Guide](./development-guide.md) — prerequisites, build, test, CI/CD, contribution patterns

## Existing Documentation

- [README](../README.md) — installation, quick start, commands, skill anatomy
- [CLAUDE.md](../CLAUDE.md) — AI agent instructions and project context
- [ROADMAP](./ROADMAP.md) — 3-phase product roadmap
- [CHANGELOG](../CHANGELOG.md) — version history
- [Design Documents](./plans/) — 32 design docs and implementation plans (2026-03-10 to 2026-03-26)

## Getting Started

```bash
# Clone and build
git clone https://github.com/mirandaguillaume/forgent.git
cd forgent
go build ./cmd/forgent

# Create a project
forgent init
forgent skill create my-skill --tools web_search
forgent lint
forgent build --target claude

# Run tests
go test ./... -count=1 -race
```

## Key Packages

| Package | Purpose |
|---------|---------|
| `pkg/model` | SkillBehavior + AgentComposition domain types |
| `pkg/spec` | Generator plugin interface and registry |
| `pkg/dag` | Executable DAG engine (12 node kinds, auto-wiring) |
| `pkg/analysis` | Diff analysis pipeline (AST + pattern scanning) |
| `internal/generator/claude` | Claude Code target generator |
| `internal/generator/copilot` | GitHub Copilot target generator |
| `internal/generator/forgent` | Standalone Go runtime generator |
| `internal/scanner` | Intelligent codebase scanning (TF-IDF, entropy) |
| `internal/importer` | Agent .md → skill YAML conversion |
