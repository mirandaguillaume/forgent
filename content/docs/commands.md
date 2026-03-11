---
title: Commands
weight: 3
---

Complete reference for all Forgent CLI commands.

## `forgent init`

Initialize a new Forgent project in the current directory.

```bash
forgent init [path]
```

Creates `forgent.yaml`, `skills/`, `agents/`, and an example skill template.

## `forgent skill create`

Scaffold a new skill YAML file.

```bash
forgent skill create <name> [flags]
```

| Flag | Description |
|------|-------------|
| `--tools` | Comma-separated list of tools (e.g. `read,write,web_search`) |

Skill names must be lowercase with hyphens or underscores (`a-z`, `0-9`, `-`, `_`).

```bash
forgent skill create search-web --tools web_search,read_url
```

## `forgent lint`

Lint skill files for design best practices.

```bash
forgent lint [path]
```

Default path: `skills/`. Checks 4 rules:

| Rule | Severity | Description |
|------|----------|-------------|
| `valid-schema` | error | Skill YAML must parse and validate |
| `has-guardrails` | warning | Skills should have at least one guardrail |
| `no-empty-tools` | warning | Skills should define tools |
| `security-needs-guardrails` | error | Skills with network/filesystem access must have guardrails |

## `forgent doctor`

Full diagnostic — lint + dependency analysis + loop detection.

```bash
forgent doctor [path]
```

| Flag | Description |
|------|-------------|
| `--skills` | Skills directory (default: `skills`) |
| `--agents` | Agents directory (default: `agents`) |

Reports:
- Lint issues across all skills
- Missing or circular dependencies
- Loop risks (self-referencing I/O, missing timeouts)
- Health score (0-100)

## `forgent score`

Score design quality of skills and agents.

```bash
forgent score [flags]
```

| Flag | Description |
|------|-------------|
| `--skills` | Skills directory (default: `skills`) |
| `--agents` | Agents directory (default: `agents`) |

### Skill scoring (5 facets)

| Facet | Weight | What it measures |
|-------|--------|-----------------|
| Context | 20 | Defined I/O contract (consumes/produces) |
| Strategy | 25 | Tools, approach, detailed steps |
| Guardrails | 20 | Timeout, limits, defense in depth |
| Observability | 15 | Trace level, metrics |
| Security | 20 | Least privilege, sandboxing |

### Agent scoring (4 dimensions)

| Dimension | Weight | What it measures |
|-----------|--------|-----------------|
| Description | 20 | Quality and length of description |
| Composition | 25 | Number and uniqueness of skills |
| Data Flow | 35 | Correct producer → consumer ordering |
| Orchestration | 20 | Strategy appropriateness |

## `forgent build`

Compile skills and agents into framework-specific output.

```bash
forgent build [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--target` | `-t` | `claude` | Target framework (`claude`, `copilot`) |
| `--skills` | `-s` | `skills` | Skills directory |
| `--agents` | `-a` | `agents` | Agents directory |
| `--output` | `-o` | (auto) | Output directory |
| `--watch` | `-w` | `false` | Watch for changes and rebuild |

```bash
# Build for Claude Code (default)
forgent build

# Build for GitHub Copilot
forgent build --target copilot

# Custom output directory
forgent build --target claude -o out/

# Watch mode — rebuilds on file changes
forgent build --watch
```

The build pipeline: parse YAML → validate → lint (fail on errors) → generate output → check word limits (warn if > 500 words per skill).
