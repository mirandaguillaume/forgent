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
forgent doctor <file>                  # LLM-powered agent analysis
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

## Doctor — LLM-Powered Agent Analysis

`forgent doctor` analyzes any agent definition file and produces research-backed recommendations to improve it. It works on Claude Code agents, GitHub Copilot agents, and Forgent YAML skills — you don't need to use Forgent's build pipeline to benefit from it.

```bash
# Analyze a Claude Code agent
forgent doctor .claude/agents/code-reviewer.md

# Analyze a GitHub Copilot agent
forgent doctor .github/agents/dash.agent.md

# Analyze a Forgent skill
forgent doctor skills/review-commenter.skill.yaml

# Use a specific provider
forgent doctor agent.md --provider openrouter
forgent doctor agent.md --provider anthropic
```

Doctor checks 12 categories drawn from recent research (Liu et al. 2024, SkillsBench 2026, Mu et al. 2025, and others), including:

- **Guardrails** — are there explicit constraints (timeouts, output limits)?
- **Security** — are filesystem, network, and secrets access declared?
- **Ordering** — are critical instructions placed first to exploit primacy bias?
- **Decomposition** — should this be split into focused skills?
- **Redundancy** — are instructions duplicating what the model already infers?
- **Specificity** — are directives measurable or just vague ("be thorough")?

Each finding includes a confidence level (HIGH / MOD / LOW), a description of the current state, and a concrete suggested improvement.

**LLM provider fallback chain:** Ollama (local, no key needed) > `OPENROUTER_API_KEY` > `ANTHROPIC_API_KEY`. If no LLM is available, doctor falls back to static analysis only.

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
| `forgent lint` / `score` — static analysis | Done |
| `forgent doctor` — LLM-powered agent analysis (12 research-backed checks) | Done |
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
