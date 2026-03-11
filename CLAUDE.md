# Forgent — Forge agents from composable skill specs

## Context

Standalone TypeScript CLI that forges AI agents from composable skill specs across frameworks (Claude Code, CrewAI, LangGraph, OpenAI Agents SDK).

Core concept: agents are **compositions of Skill Behaviors** — reusable behavioral units with 6 facets (Context, Strategy, Guardrails, Dependencies, Observability, Security).

## Tech Stack

- TypeScript 5.x, Node.js 20+, ESM
- CLI: Commander.js
- Validation: ajv (JSON Schema)
- YAML: yaml (npm)
- Testing: vitest
- Output: chalk

## Commands (MVP)

```bash
forgent init                    # Initialize Forgent project
forgent skill create <name>     # Scaffold a new skill
forgent lint [path]             # Lint skills for best practices
forgent doctor [path]           # Full diagnostic (lint + dependency + loop analysis)
forgent trace <file>            # Analyze JSONL trace files
forgent build [path]            # Generate skills/agents for a target framework
forgent score [path]            # Score design quality
```

## Dev Commands

```bash
npm run dev                # Run CLI via tsx
npm test                   # Run vitest
npm run build              # Compile TypeScript
npm run lint               # Type-check only
```

## Architecture

```
src/
  index.ts                 # CLI entry point
  commands/                # CLI command handlers
  model/                   # Skill Behavior Model types + schema
  analyzers/               # Dependency checker, loop detector, trace parser
  linters/                 # AX quality rules
  utils/                   # YAML loader
templates/                 # Skill/agent YAML templates
tests/                     # Mirrors src/ structure
docs/plans/                # Design doc + implementation plan
```

## Implementation Plan

See `docs/plans/2026-03-10-ax-cli-implementation-plan.md` — 13 tasks, TDD, bite-sized.

Use `superpowers:executing-plans` or `superpowers:subagent-driven-development` to execute.
