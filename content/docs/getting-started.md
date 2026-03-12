---
title: Getting Started
weight: 1
---

Get Forgent installed and build your first agent in 5 minutes.

## Installation

### Binary download

Download the latest binary from [GitHub Releases](https://github.com/mirandaguillaume/forgent/releases) for your platform (Linux, macOS, Windows).

### From source

```bash
go install github.com/mirandaguillaume/forgent/cmd/forgent@latest
```

Requires Go 1.22+.

## Quick Start

{{% steps %}}

### Initialize a project

```bash
mkdir my-agent && cd my-agent
forgent init
```

This creates:
- `forgent.yaml` — project config
- `skills/` — directory for skill definitions
- `agents/` — directory for agent compositions
- `skills/example.skill.yaml` — a starter skill template

### Create a skill

```bash
forgent skill create search-web --tools web_search,read_url
```

This scaffolds a `skills/search-web.skill.yaml` with pre-filled tools and sensible defaults. Edit it to define your skill's behavior:

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
  - "Max 5 queries per invocation"
  - "timeout: 30s"

observability:
  trace_level: standard
  metrics: [latency, token_usage]

security:
  filesystem: none
  network: full
  secrets: []
```

### Validate your design

```bash
forgent lint
forgent score
```

`lint` checks for common issues (missing guardrails, empty tools). `score` rates your skill design quality on a 0-100 scale across 5 facets.

### Build for your framework

```bash
forgent build
```

This compiles your skills and agents into Claude Code format (default) under `.claude/`. To target GitHub Copilot:

```bash
forgent build --target copilot
```

{{% /steps %}}

## What's Next?

- [Concepts](../concepts) — understand the Skill Behavior Model
- [Commands](../commands) — full reference for all CLI commands
- [Build Targets](../build-targets) — learn about supported output formats
