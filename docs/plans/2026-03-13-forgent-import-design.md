# Design: `forgent import` — Import agents and skills from any source

## Context

Developers have existing agent definitions (Claude Code, Copilot, Cursor, etc.) written as free-form markdown files with no consistent structure. They want to convert these into composable Forgent skill YAML specs without starting from scratch. The `forgent import` command bridges the gap between hand-written agents and structured skill specifications.

## Decisions

- **Single command** for all sources: local files, directories, and registries (Vercel)
- **LLM-assisted parsing** for all inputs (no heuristic fallback) — agent markdown is too unstructured for reliable regex parsing
- **Decomposition**: the LLM can split a monolithic agent into N skills + 1 agent composition
- **Dry-run by default**: shows what would be generated, requires `--yes` to write
- **Provider configurable** via env var `FORGENT_LLM_PROVIDER` (default: anthropic)
- **Validated by Forgent tools**: lint, doctor, score run on generated YAML — same quality bar as hand-written specs
- **LLM retry loop**: if validation fails or score < threshold, feedback is sent back to the LLM for one retry

## CLI Interface

```bash
# Local import
forgent import .claude/agents/review.md           # Single file
forgent import .claude/                            # Directory (auto-detect skills/ and agents/)
forgent import .github/agents/review.agent.md     # Copilot format
forgent import my-agent.md                        # Any markdown file

# Registry import
forgent import vercel:code-reviewer               # From Vercel skills registry

# Options
forgent import .claude/ --yes                     # Skip confirmation, write directly
forgent import .claude/ --dry-run                 # Alias: explicit dry-run (default behavior)
forgent import .claude/ --provider anthropic      # LLM provider (default from env)
forgent import .claude/ --output skills/          # Output directory (default: skills/ and agents/)
forgent import .claude/ --min-score 70            # Minimum quality score (default: 60, triggers LLM retry)
forgent import .claude/ --force                   # Write even if validation fails
```

### Source detection

- Path to file/directory → local import
- `vercel:` prefix → Vercel skills registry
- Extensible: `github:user/repo` in the future

## Pipeline

```
Input (.md file or registry reference)
    │
    ├─ 1. RESOLVE SOURCE
    │     ├─ Local file → read from disk
    │     ├─ Local directory → glob for *.md, *.agent.md, SKILL.md
    │     └─ vercel:name → fetch from Vercel API/CDN
    │
    ├─ 2. EXTRACT FRONTMATTER (deterministic)
    │     ├─ Parse YAML frontmatter (name, description, tools, model)
    │     └─ Reverse tool mapping (Read → read_file, Bash → bash, etc.)
    │
    ├─ 3. LLM DECOMPOSITION
    │     ├─ Send frontmatter + full markdown body to LLM
    │     ├─ Prompt: "Analyze this agent definition and produce Forgent YAML"
    │     ├─ LLM returns: array of skill YAMLs + optional agent YAML
    │     └─ Each YAML conforms to SkillBehavior / AgentComposition schema
    │
    ├─ 4. VALIDATE (reuses forgent lint, doctor, score)
    │     ├─ Parse each returned YAML (yaml.Unmarshal)
    │     ├─ Run model.ValidateSkill() / model.ValidateAgent()
    │     ├─ Run linter.LintSkill() on each skill (best-practice rules)
    │     ├─ Run analyzer.DetectLoopRisks() on each skill
    │     ├─ Run analyzer.ScoreSkill() on each skill (0-100 quality score)
    │     ├─ If agent present:
    │     │     ├─ Run analyzer.CheckDependencies(agent, skills)
    │     │     ├─ Run analyzer.CheckSkillOrdering(agent, skillMap)
    │     │     └─ Run analyzer.ScoreAgent(agent, resolvedSkills)
    │     ├─ Collect warnings for partial/inferred fields
    │     └─ If score < threshold or lint errors → warn, suggest --force to write anyway
    │
    ├─ 5. PREVIEW (dry-run, default)
    │     ├─ Display decomposition summary
    │     ├─ Show each skill: name, consumes, produces, score, lint warnings
    │     ├─ Show agent: skills list, orchestration, score, dependency issues
    │     └─ Prompt for confirmation (unless --yes)
    │
    └─ 6. WRITE
          ├─ Write skills/*.skill.yaml
          ├─ Write agents/*.agent.yaml
          ├─ Check for conflicts with existing files
          └─ Display result summary
```

## LLM Prompt Strategy

The LLM receives:
1. The full Forgent skill YAML schema (all facets, all enums, all constraints)
2. A reference example (the `review-commenter` skill)
3. The agent markdown to import (frontmatter + body)
4. Instructions to decompose if the agent handles multiple responsibilities

The LLM returns a JSON array:
```json
{
  "skills": [
    { "yaml": "skill: ts-linter\nversion: ...\n..." },
    { "yaml": "skill: review-commenter\n..." }
  ],
  "agent": {
    "yaml": "agent: ci-reviewer\nskills: [ts-linter, review-commenter]\n..."
  }
}
```

If the input is already a single-responsibility skill, the LLM returns 1 skill and no agent.

## Provider Architecture

```go
// internal/llm/provider.go
type Provider interface {
    Complete(prompt string) (string, error)
}

// internal/llm/anthropic.go
type AnthropicProvider struct { apiKey string }

// internal/llm/registry.go
func GetProvider(name string) (Provider, error)
```

Provider is resolved from:
1. `--provider` flag
2. `FORGENT_LLM_PROVIDER` env var
3. Default: `anthropic`

API key from:
1. `FORGENT_API_KEY` env var
2. Provider-specific env var (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`)

## Reverse Tool Mapping

Reuse existing toolMap from generators, inverted:

```go
// internal/importer/toolmap.go
var reverseClaudeMap = map[string]string{
    "Read": "read_file", "Write": "write_file", "Edit": "edit_file",
    "Grep": "grep", "Glob": "search", "Bash": "bash",
    "WebFetch": "web_fetch", "WebSearch": "web_search",
    "TodoWrite": "todo", "Task": "task",
}
```

Auto-detect source framework from:
- `.claude/` path → Claude tool names
- `.github/` path → Copilot tool names
- Otherwise → try both mappings

## Output Example

```
$ forgent import .claude/agents/ci-reviewer.md

  Analyzing: .claude/agents/ci-reviewer.md
  Provider: anthropic (claude-sonnet-4-20250514)

  Decomposition:
    Input agent handles 4 responsibilities → 4 skills + 1 agent

    Skills:
      ts-linter                                          score: 72/100
        consumes: [file_tree, source_code]
        produces: [lint_results]
        security: filesystem=read-only, network=none
        ⚠ lint: missing observability.metrics

      type-checker                                       score: 68/100
        consumes: [file_tree, source_code]
        produces: [type_errors]
        security: filesystem=read-only, network=none
        ⚠ lint: missing guardrails.timeout

      review-commenter                                   score: 85/100
        consumes: [git_diff, lint_results, type_errors]
        produces: [review_comments]
        security: filesystem=read-only, network=none
        ✓ all lint rules pass

      risk-scorer                                        score: 78/100
        consumes: [git_diff, lint_results]
        produces: [risk_score]
        security: filesystem=read-only, network=none
        ✓ all lint rules pass

    Agent:
      ci-reviewer                                        score: 82/100
        skills: [ts-linter, type-checker, review-commenter, risk-scorer]
        orchestration: sequential
        consumes: [git_diff, file_tree, source_code]
        produces: [review_comments, risk_score]
        ✓ dependencies satisfied
        ✓ no circular dependencies
        ✓ skill ordering valid

  Write 4 skills + 1 agent? [y/N]
```

## File Structure

```
internal/
  importer/
    importer.go          # Main import pipeline (RunImport)
    source.go            # Source detection and resolution
    frontmatter.go       # YAML frontmatter extraction
    toolmap.go           # Reverse tool mapping
    prompt.go            # LLM prompt construction
  llm/
    provider.go          # Provider interface
    anthropic.go         # Anthropic/Claude provider
    registry.go          # Provider registry
internal/
  cmd/
    import.go            # Cobra command definition
```

## Validation Integration

The import pipeline reuses Forgent's existing analysis tools to ensure imported skills meet the same quality bar as hand-written specs:

| Tool | Function | What it checks |
|------|----------|---------------|
| **lint** | `linter.LintSkill(skill)` | Best-practice rules (missing facets, empty fields, naming) |
| **loop** | `analyzer.DetectLoopRisks(skill, checker)` | Self-referencing data, missing timeouts |
| **score** | `analyzer.ScoreSkill(skill)` | Quality score (0-100) across 8 facets |
| **deps** | `analyzer.CheckDependencies(agent, skills)` | Missing skills, circular deps, unmet context |
| **order** | `analyzer.CheckSkillOrdering(agent, skillMap)` | Sequential data-flow ordering |
| **agent score** | `analyzer.ScoreAgent(agent, skills)` | Agent quality (0-100) across 4 dimensions |

All functions take parsed `model.SkillBehavior` / `model.AgentComposition` structs — the same types the pipeline produces after `yaml.Unmarshal`.

### LLM retry on validation failure

If the LLM-generated YAML fails validation or scores below a configurable threshold (default: 60/100), the pipeline retries once:

1. Collect all lint warnings, validation errors, and low-score facets
2. Append them to the original prompt as structured feedback
3. Ask the LLM to fix the issues
4. Re-validate the corrected output
5. If still failing → show warnings in preview, let user decide

This creates a feedback loop: `LLM → validate → feedback → LLM → validate → preview`.

## Error Handling

- No API key → clear error message with setup instructions
- LLM returns invalid YAML → retry once with error feedback, then fail with raw output
- LLM returns YAML that fails validation → show validation errors, write anyway with `--force`
- File conflicts → prompt to overwrite, skip, or rename
- Network errors (registry) → retry once, then fail

## Testing

- Unit tests for frontmatter extraction, reverse tool mapping, source detection
- Integration test with mock LLM provider returning known YAML
- Golden file tests: known .md input → expected .skill.yaml output
