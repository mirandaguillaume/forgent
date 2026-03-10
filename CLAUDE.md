# AX CLI — Agent Experience CLI

## Context

Standalone TypeScript CLI that improves Agent Experience (AX) across AI agent frameworks (Claude Code, CrewAI, LangGraph, OpenAI Agents SDK).

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
ax init                    # Initialize AX project
ax skill create <name>     # Scaffold a new skill
ax lint [path]             # Lint skills for AX best practices
ax doctor [path]           # Full diagnostic (lint + dependency + loop analysis)
ax trace <file>            # Analyze JSONL trace files
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
