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

```bash
npm install -g forgent
```

Requires Node.js >= 20.

## Commands

```bash
forgent init                      # Initialize a Forgent project
forgent skill create <name>       # Scaffold a new skill
forgent lint [path]               # Lint skills for best practices
forgent doctor [path]             # Full diagnostic (lint + deps + loops)
forgent trace <file>              # Analyze JSONL trace files
forgent build [path]              # Generate skills/agents for a target framework
forgent score [path]              # Score design quality
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
name: search-web
version: "1.0"

context:
  consumes: [user_query]
  produces: [search_results]
  memory: short-term

strategy:
  tools: [web_search, read_url]
  approach: "Search, filter, summarize"

guardrails:
  rules:
    - "Max 5 search queries per invocation"
  limits:
    max_tokens: 4000
    timeout: 30

dependencies:
  depends_on: []
  provides: [search_results]

observability:
  traces: true
  metrics: [latency, token_usage]

security:
  network: [https://*]
  sandbox: strict
```

## Build Targets

`forgent build` compiles skill/agent YAML into framework-native formats.

| Target | Output | Status |
|--------|--------|--------|
| Claude Code | `.claude/` (SKILL.md + agent.md) | Available |
| CrewAI | — | Planned |
| LangGraph | — | Planned |
| OpenAI Agents SDK | — | Planned |

```bash
forgent build --target claude         # default
forgent build --target claude -o out/
```

## Development

```bash
git clone https://github.com/mirandaguillaume/forgent.git
cd forgent
npm install
npm test          # 166 tests
npm run build     # compile TypeScript
npm run dev       # run via tsx
```

## License

[Apache 2.0](LICENSE)
