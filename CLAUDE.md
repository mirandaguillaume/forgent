# Forgent — Forge agents from composable skill specs

## Context

Standalone Go CLI that forges AI agents from composable skill specs across frameworks (Claude Code, GitHub Copilot, and more).

Core concept: agents are **compositions of Skill Behaviors** — reusable behavioral units with 5 facets (Context, Strategy, Guardrails, Observability, Security). Skills are pure interfaces (consumes/produces). Agents declare their own I/O contract and orchestration.

## Tech Stack

- Go 1.22+
- CLI: Cobra (github.com/spf13/cobra)
- YAML: gopkg.in/yaml.v3
- Testing: testify (github.com/stretchr/testify)
- Output: fatih/color
- File watching: fsnotify (github.com/fsnotify/fsnotify)

## Commands

```bash
forgent init                           # Initialize Forgent project
forgent skill create <name>           # Scaffold a new skill
forgent lint [path]                    # Lint skills for best practices
forgent doctor [path]                  # Full diagnostic (lint + dependency + loop analysis)
forgent score [path]                   # Score design quality
forgent build --target claude          # Generate skills/agents for Claude Code
forgent build --target copilot         # Generate skills/agents for GitHub Copilot
forgent build --target forgent         # Generate standalone Go runtime
forgent build --compact                # Reduce structural overhead
forgent build --watch                  # Watch and rebuild on changes
forgent import <source>               # Import agent .md files as Forgent skill specs
forgent import <source> --yes         # Skip confirmation, write directly
forgent bench <repo-path>             # Benchmark agent composition quality
```

## Dev Commands

```bash
go test ./...                # Run all tests
go build ./cmd/forgent       # Compile binary
go vet ./...                 # Static analysis
```

## Architecture

```
cmd/
  forgent/
    main.go                  # CLI entry point
pkg/
  model/                     # SkillBehavior, AgentComposition, validation
  spec/                      # TargetGenerator interface + registry
  dag/                       # Executable DAG engine (auto-wiring, layers, router, retry)
  analysis/                  # Formal property analysis
internal/
  cmd/                       # CLI command handlers (Cobra)
  analyzer/                  # Dependency checker, loop detector, score, ordering
  linter/                    # Lint rules
  yaml/                      # YAML loader
  scanner/                   # Codebase scanner
  enricher/                  # Skill enricher (codebase_index injection)
  llm/                       # LLM provider interface (Anthropic, OpenRouter)
  builder/                   # Build orchestration
  bench/                     # Benchmark framework (token, isomorphism, SWE-bench, H2H)
  importer/                  # Agent .md → skill spec decomposition pipeline
  generator/
    claude/                  # Claude Code generator (skill, agent, toolmap)
    copilot/                 # GitHub Copilot generator (skill, agent, instructions, toolmap)
    forgent/                 # Go runtime generator (standalone binary)
templates/                   # Skill/agent YAML templates
```

## Build Targets

Generators implement `pkg/spec.TargetGenerator` and register via `init()`.

| Target | Output Dir | Files |
|--------|-----------|-------|
| claude | `.claude/` | `skills/<name>/SKILL.md`, `agents/<name>.md` |
| copilot | `.github/` | `skills/<name>/SKILL.md`, `agents/<name>.agent.md`, `copilot-instructions.md` |
| forgent | user-specified | standalone Go binary (main.go + go.mod) |
