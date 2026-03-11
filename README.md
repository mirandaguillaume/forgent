# forgent — Forge agents from composable skill specs

A CLI for designing, building, and composing AI agents across frameworks.

Agents are compositions of **Skill Behaviors** — reusable behavioral units described by 6 facets:

| Facet | What it defines |
|-------|----------------|
| **Context** | Memory, inputs consumed, outputs produced |
| **Strategy** | Tools, approach, execution steps |
| **Guardrails** | Rules, limits, constraints |
| **Dependencies** | Skill composition and data flow |
| **Observability** | Traces, metrics, structured logging |
| **Security** | Filesystem, network, secrets, sandboxing |

Skills are defined in YAML, validated against a schema, and compiled to framework-specific formats.

## Install

### Binary download

Download the latest binary from [GitHub Releases](https://github.com/mirandaguillaume/forgent/releases).

### From source

```bash
go install github.com/mirandaguillaume/forgent/cmd/forgent@latest
```

Requires Go 1.22+.

## Commands

```bash
forgent init                      # Initialize a Forgent project
forgent skill create <name>       # Scaffold a new skill
forgent lint [path]               # Lint skills for best practices
forgent doctor [path]             # Full diagnostic (lint + deps + loops)
forgent score [path]              # Score design quality
forgent build                     # Build for Claude Code (default)
forgent build --target copilot    # Build for GitHub Copilot
forgent build --watch             # Watch and rebuild on changes
```

## Quick Start

```bash
mkdir my-agent && cd my-agent
forgent init
forgent skill create search-web --tools web_search,read_url
forgent lint
forgent build
```

This creates a skill YAML, validates it, and compiles it to Claude Code format (default target).

## Skill Anatomy

```yaml
skill: search-web
version: "1.0"

context:
  consumes: [user_query]
  produces: [search_results]
  memory: short-term

strategy:
  tools: [web_search, read_url]
  approach: "Search, filter, summarize"

guardrails:
  - "Max 5 search queries per invocation"
  - "timeout: 30s"

depends_on: []

observability:
  trace_level: standard
  metrics: [latency, token_usage]

security:
  filesystem: none
  network: full
  secrets: []

negotiation:
  file_conflicts: yield
  priority: 0
```

## Build Targets

`forgent build` compiles skill/agent YAML into framework-native formats.

| Target | Output | Status |
|--------|--------|--------|
| Claude Code | `.claude/` (SKILL.md + agent.md) | Available |
| GitHub Copilot | `.github/` (SKILL.md + agent.md + instructions) | Available |
| CrewAI | — | Planned |
| LangGraph | — | Planned |

```bash
forgent build --target claude           # default
forgent build --target copilot
forgent build --target claude -o out/
forgent build --watch                   # rebuilds on changes
```

## Development

```bash
git clone https://github.com/mirandaguillaume/forgent.git
cd forgent
go test ./...           # run tests
go build ./cmd/forgent  # compile
go vet ./...            # static analysis
```

## License

[Apache 2.0](LICENSE)
