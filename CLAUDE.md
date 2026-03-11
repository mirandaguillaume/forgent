# Forgent — Forge agents from composable skill specs

## Context

Standalone Go CLI that forges AI agents from composable skill specs across frameworks (Claude Code, GitHub Copilot, and more).

Core concept: agents are **compositions of Skill Behaviors** — reusable behavioral units with 6 facets (Context, Strategy, Guardrails, Dependencies, Observability, Security).

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
forgent trace <file>                   # Analyze JSONL trace files
forgent score [path]                   # Score design quality
forgent build --target claude          # Generate skills/agents for Claude Code
forgent build --target copilot         # Generate skills/agents for GitHub Copilot
forgent build --watch                  # Watch and rebuild on changes
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
internal/
  cmd/                       # CLI command handlers (Cobra)
  analyzer/                  # Dependency checker, loop detector, trace parser, score, ordering
  linter/                    # Lint rules
  yaml/                      # YAML loader
  generator/
    claude/                  # Claude Code generator (skill, agent, toolmap)
    copilot/                 # GitHub Copilot generator (skill, agent, instructions, toolmap)
templates/                   # Skill/agent YAML templates
```

## Build Targets

Generators implement `pkg/spec.TargetGenerator` and register via `init()`.

| Target | Output Dir | Files |
|--------|-----------|-------|
| claude | `.claude/` | `skills/<name>/SKILL.md`, `agents/<name>.md` |
| copilot | `.github/` | `skills/<name>/SKILL.md`, `agents/<name>.agent.md`, `copilot-instructions.md` |
