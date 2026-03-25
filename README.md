# forgent — Define agent skills once in YAML, compile to any framework

Forgent is a skill compiler. You describe agent skills in YAML with explicit contracts (consumes/produces, guardrails, security). Forgent compiles them into framework-native formats for Claude Code, GitHub Copilot, or a standalone Go runtime. One source of truth, multiple targets.

## Install

### Binary download

Download the latest binary from [GitHub Releases](https://github.com/mirandaguillaume/forgent/releases).

### From source

```bash
go install github.com/mirandaguillaume/forgent/cmd/forgent@latest
```

Requires Go 1.22+.

## Quick Start

```bash
mkdir my-agent && cd my-agent
forgent init
forgent skill create search-web --tools web_search,read_url
forgent lint
forgent build
```

This creates a skill YAML, validates it, and compiles it to Claude Code format (default target).

## Commands

```bash
forgent init                           # Initialize a Forgent project
forgent skill create <name>            # Scaffold a new skill
forgent lint [path]                    # Lint skills for best practices
forgent doctor [path]                  # Full diagnostic (lint + deps + loops)
forgent score [path]                   # Score design quality
forgent build --target claude          # Build for Claude Code (default)
forgent build --target copilot         # Build for GitHub Copilot
forgent build --target forgent         # Generate standalone Go runtime
forgent build --compact                # Reduce structural overhead
forgent build --watch                  # Watch and rebuild on changes
forgent import <source>               # Import agent .md files as Forgent skill specs
forgent import <source> --yes          # Skip confirmation, write directly
```

## Skill Anatomy

Skills are pure interfaces (consumes/produces) described by 5 facets:

| Facet | What it defines |
|-------|----------------|
| **Context** | Memory, inputs consumed, outputs produced |
| **Strategy** | Tools, approach, execution steps, effort level |
| **Guardrails** | Rules, limits, constraints |
| **Observability** | Traces, metrics, structured logging |
| **Security** | Filesystem, network, secrets, sandboxing |

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
  effort: medium

guardrails:
  - "Max 5 search queries per invocation"
  - "timeout: 30s"

observability:
  trace_level: standard
  metrics: [latency, token_usage]

security:
  filesystem: none
  network: full
  secrets: []
```

## Build Targets

`forgent build` compiles skill/agent YAML into framework-native formats.

| Target | Output | Status |
|--------|--------|--------|
| Claude Code | `.claude/` (SKILL.md + agent.md) | Available |
| GitHub Copilot | `.github/` (SKILL.md + agent.md + instructions) | Available |
| Forgent | standalone Go binary (runtime + prompt) | Available |

## Advanced Features

Forgent includes an executable DAG engine (`pkg/dag`) for auto-wiring skill dependencies and a Go runtime target (`--target forgent`) that compiles skills into standalone binaries.

## Development

```bash
git clone https://github.com/mirandaguillaume/forgent.git
cd forgent
go test ./...                  # run tests
go build ./cmd/forgent         # compile
go build ./cmd/forgent-bench   # bench binary (internal)
go vet ./...                   # static analysis
```

## Roadmap

| Feature | Status |
|---------|--------|
| Skill YAML format + validation | Done |
| `forgent lint` / `doctor` / `score` | Done |
| `forgent build` — Claude Code + Copilot targets | Done |
| `forgent build --watch` — file watcher | Done |
| SRP lint rules (single produces, name matches output) | Done |
| `forgent import` — LLM-powered agent decomposition | Done |
| LLM providers — Anthropic + OpenRouter | Done |
| `--compact` mode (reduces overhead) | Done |
| Executable DAG engine (`pkg/dag`) | Done |
| `forgent build --target forgent` — Go runtime generator | Done |
| DAG v2 (race, fallback, map-reduce, HITL) | Planned |
| `forgent import` — batch directory processing | Planned |
| Approval gate facet (human-in-the-loop) | Planned |
| `forgent test` — behavioral testing for skills | Planned |

## License

[Apache 2.0](LICENSE)
