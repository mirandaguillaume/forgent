# AX Build — Design Doc

## Goal

Add `ax build` command that generates Claude Code-compatible skill and agent files from AX YAML definitions, with scientifically optimized prompt structure.

## Command

```bash
ax build [skills-dir] [--output <dir>]
```

- Default input: `skills/` + `agents/`
- Default output: `.claude/` (project root)

## Skill Generation → `.claude/skills/<name>/SKILL.md`

### Frontmatter

```yaml
---
name: <skill>
description: <approach>-based skill consuming <consumes> to produce <produces>
---
```

### Body — Optimized ordering (critical constraints at start and end)

Based on research showing LLMs attend most to beginning and end of context:
- [Du et al., 2025](https://arxiv.org/abs/2510.05381): Performance degrades 13.9%-85% with context length
- [Liu et al., 2023](https://arxiv.org/abs/2307.03172): "Lost in the Middle" — info in middle is ignored
- Reasoning degradation starts at ~3000 tokens

```markdown
# <Skill Name>

## Guardrails            ← first (primacy bias)
- ...

## Context
Consumes: ...
Produces: ...
Memory: ...

## Strategy
Approach: ...
Tools: ...
### Steps
1. ...

## Security              ← last (recency bias)
- Filesystem: ...
- Network: ...
- Secrets: ...
```

**Word budget**: ~200-400 words per skill. Warning if > 500 words.

## Agent Generation → `.claude/agents/<name>.md`

### Frontmatter

```yaml
---
name: <agent>
description: <description>
model: inherit
---
```

### Body

```markdown
You are <agent>, orchestrating skills in <orchestration> mode.

## Skills
Use these skills: <skill-1>, <skill-2>, ...

## Orchestration
<sequential|parallel|parallel-then-merge|adaptive> execution.
```

## Pre-build Validation

- Run `lint` before generating
- Refuse build if lint errors exist (warnings OK)
- Warn if generated skill exceeds 500 words

## Files to Create

- `src/commands/build.ts` — build command logic
- `src/generators/skill-generator.ts` — AX YAML → SKILL.md
- `src/generators/agent-generator.ts` — agent YAML → agent .md
- `tests/generators/skill-generator.test.ts`
- `tests/generators/agent-generator.test.ts`
- `tests/commands/build.test.ts`
- Modify `src/index.ts` — wire `ax build`
