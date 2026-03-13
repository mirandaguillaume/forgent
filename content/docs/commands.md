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

## `forgent import`

Convert existing agent markdown files into Forgent skill YAML specs using LLM-assisted decomposition.

### Usage

```bash
forgent import <source>
```

### Sources

| Source | Example |
|--------|---------|
| Local file | `forgent import .claude/agents/review.md` |
| Local directory | `forgent import .claude/` |
| Copilot agent | `forgent import .github/agents/review.agent.md` |
| Any markdown | `forgent import my-agent.md` |
| Vercel registry | `forgent import vercel:code-reviewer` (planned) |

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--provider`, `-p` | `anthropic` | LLM provider (from `FORGENT_LLM_PROVIDER` env) |
| `--output`, `-o` | `.` | Output directory |
| `--min-score` | `60` | Minimum quality score (triggers LLM retry if below) |
| `--yes` | `false` | Skip confirmation, write directly |
| `--dry-run` | `false` | Preview without writing (default behavior) |
| `--force` | `false` | Write even if files exist or validation fails |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `FORGENT_API_KEY` | API key (priority 1) |
| `ANTHROPIC_API_KEY` | Anthropic API key (priority 2 for `--provider anthropic`) |
| `OPENAI_API_KEY` | OpenAI API key (priority 2 for `--provider openai`) |
| `FORGENT_LLM_PROVIDER` | Default provider name |

### Pipeline

The import command runs a 6-step pipeline:

1. **Resolve** — Read source file(s) from disk or registry
2. **Extract** — Parse YAML frontmatter (name, tools, model)
3. **Decompose** — Send to LLM for skill decomposition
4. **Validate** — Run `lint`, `doctor`, and `score` on generated YAML
5. **Preview** — Display decomposition with scores and warnings
6. **Write** — Save `skills/*.skill.yaml` and `agents/*.agent.yaml`

### Example

```
$ forgent import .claude/agents/ci-reviewer.md

  Analyzing: .claude/agents/ci-reviewer.md
  Provider: anthropic

  Decomposition:
    Input agent → 4 skills + 1 agent

    Skills:
      ts-linter                    score: 72/100
        consumes: [file_tree, source_code]
        produces: [lint_results]
        ⚠ lint: missing observability.metrics

      review-commenter             score: 85/100
        consumes: [git_diff, lint_results]
        produces: [review_comments]
        ✓ all checks pass

  Write 4 skill(s) + 1 agent(s)? [y/N]
```
