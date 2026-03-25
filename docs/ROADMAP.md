# Forgent Roadmap

## Done

### Multi-Agent Dispatch
- `strategy.effort` field (`light` | `medium` | `heavy`) on skill YAML
- `EffortToModel` mapping: light=haiku, medium=sonnet, heavy=opus
- Orchestrator agent format: `tools: Task`, each skill dispatched as independent subagent
- Both targets: Claude Code (`Task()`) and Copilot (`task`)

### Staged Agent Pipelines
- `stages` field on AgentComposition for multi-stage orchestration
- Mutual exclusivity validation between flat and staged agents
- Claude and Copilot generators support staged pipelines

### Compact Mode
- `--compact` flag reduces structural overhead from 117% to 14%
- Single-file agent output with inlined skills

### Executable DAG Engine (`pkg/dag`)
- Core types: Node, NodeKind (Task/Router/Merge), auto-wiring via consumes/produces
- Layer decomposition (Kahn's + longest-path) — topologies T1–T10
- Cycle detection (DFS 3-colour), topology hint validation
- Layer-parallel execution engine with router support, retry, timeout

### Forgent Runtime Target
- `forgent build --target forgent` generates a standalone Go binary
- Prompt template builder from skill specs
- OpenRouter support with `--input`/`--output` flags

## Next

### Phase 1 — Format Maturity

| Feature | Size | Description |
|---------|------|-------------|
| Typed contracts | Medium | Schema for consumes/produces — compatibility validation between skills, primitive types (string, json, file...) |
| Skill documentation fields | Quick | Promote `when_to_use`, `examples`, `anti_patterns` to first-class fields — stripped at build time (dev-only, not in generated output) |
| Semantic versioning | Medium | `version` with compatibility rules, breaking change detection on contracts |

### Phase 2 — Compiler Quality

| Feature | Size | Description |
|---------|------|-------------|
| Auto effort inference | Quick | Determine effort from skill complexity (step count, tool count, guardrails) instead of manual declaration |
| Orphan step guard | Quick | Warn or skip when a skill is in `agent.Skills` but absent from `resolvedSkills` |
| `forgent test` | Large | Behavioral testing — validate that a compiled agent produces expected results |
| Parallel group generation | Medium | Group independent skills (no shared inputs) into a parallel block in generated markdown |

### Phase 3 — Extension

| Feature | Size | Description |
|---------|------|-------------|
| DAG v2 | Large | Race, fallback, map-reduce, HITL node kinds |
| Cyclic graphs (DG) | Large | Allow cycles for feedback/refinement loops — requires termination control (max iterations, exit conditions) |
| Multi-level hierarchy | Medium | Agent → Agent → Skills (currently one level: orchestrator → skills) |
| `forgent import` batch mode | Medium | Process entire directories of agent .md files |
| New build targets | Large | Cursor, Windsurf, or others based on demand |

### Deferred

| Feature | Reason |
|---------|--------|
| MCP Server for Copilot | Premature — format must stabilize first |
| Copilot model mapping | Trivial — will be done when touching the Copilot generator |

## Internal Tools

### Benchmark Framework (`forgent-bench`)
- Token overhead measurement
- Build determinism test
- Cross-target isomorphism validation
- Formal property tests (P10–P15)
- Pass@k consistency measurement
- LLM-as-judge evaluation (composed vs monolithic)
- SWE-bench lite wrapper
- Gremlins mutation testing integration
