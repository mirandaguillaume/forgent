---
title: Build Targets
weight: 4
---

Forgent compiles skill and agent YAML specs into framework-specific output. Each target generates files optimized for its framework's conventions.

## Available Targets

| Target | Output Dir | Status |
|--------|-----------|--------|
| Claude Code | `.claude/` | Available |
| GitHub Copilot | `.github/` | Available |
| CrewAI | — | Planned |
| LangGraph | — | Planned |

## Claude Code

```bash
forgent build --target claude
```

### Output structure

```
.claude/
  skills/
    search-web/
      SKILL.md          # Skill definition with frontmatter
    analyze-code/
      SKILL.md
  agents/
    research-pipeline.md  # Agent orchestration file
```

### Skill format

Each `SKILL.md` includes YAML frontmatter (`name`, `description`) and markdown sections ordered for LLM consumption:
1. **Guardrails** first (primacy bias — read first, remembered best)
2. Context, Dependencies, Strategy in the middle
3. **Security** last (recency bias — read last, top of mind)

### Agent format

Agent files include a `tools` field in frontmatter listing all framework-native tool names (e.g., `Glob`, `Grep`, `Read`, `Write`, `Edit`, `Bash`, `WebFetch`). Sequential agents include step-by-step execution instructions referencing skill file paths.

## GitHub Copilot

```bash
forgent build --target copilot
```

### Output structure

```
.github/
  skills/
    search-web/
      SKILL.md
    analyze-code/
      SKILL.md
  agents/
    research-pipeline.agent.md
  copilot-instructions.md       # Global instructions file
```

### Differences from Claude

- Agent files use `.agent.md` extension
- Tools use lowercase aliases (`read`, `edit`, `search`, `execute`)
- Skill descriptions truncated to 1024 characters
- Generates `copilot-instructions.md` with global context: available skills, agents, and guardrails

## Tool Mapping

Forgent maps generic tool names to framework-specific aliases:

| Generic | Claude Code | Copilot |
|---------|------------|---------|
| `read` | `Read` | `read` |
| `write` | `Write` | `edit` |
| `web_search` | `WebSearch` | `web` |
| `bash` | `Bash` | `execute` |
| `grep` | `Grep` | `search` |

## Adding a New Target

Implement the generator interfaces from `pkg/spec` and register:

```go
package mytarget

import "github.com/mirandaguillaume/forgent/pkg/spec"

type MyGenerator struct{}

func (g *MyGenerator) Target() string          { return "mytarget" }
func (g *MyGenerator) DefaultOutputDir() string { return ".mytarget" }
func (g *MyGenerator) ContextDir() string       { return "" }
// ... implement SkillGenerator and AgentGenerator interfaces

func init() {
    spec.Register("mytarget", func() spec.Generator {
        return &MyGenerator{}
    })
}
```

Then add a blank import in `internal/cmd/build.go`:

```go
import _ "github.com/mirandaguillaume/forgent/internal/generator/mytarget"
```

The new target is immediately available via `forgent build --target mytarget`.
