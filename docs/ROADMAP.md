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

### Quick Wins

| Feature | Description |
|---------|-------------|
| Copilot model mapping | Map effort to GPT models (light=gpt-4o-mini, medium/heavy=gpt-4o) instead of Anthropic names |
| Auto effort inference | Determine effort from skill complexity (step count, tool count, guardrails) instead of manual declaration |
| Orphan step guard | Warn or skip when a skill is in `agent.Skills` but absent from `resolvedSkills` |

### Medium

| Feature | Description |
|---------|-------------|
| Parallel group generation | Group independent skills (no shared inputs) into a parallel block in generated agent markdown |
| Multi-level hierarchy | Agent → Agent → Skills (currently one level: orchestrator → skills) |
| `forgent import` batch mode | Process entire directories of agent .md files |

### Large

| Feature | Description |
|---------|-------------|
| DAG v2 | Race, fallback, map-reduce, and HITL (human-in-the-loop) node kinds |
| MCP Server for Copilot | `cmd/forgent-mcp/` binary exposing `dispatch_skill()` tool — multi-agent on platforms without native dispatch |
| `forgent test` | Behavioral testing for skills — validate agent behavior against expectations |
| Approval gate facet | Human-in-the-loop approval gates as a first-class skill facet |

## Internal Tools

### Benchmark Framework (`forgent bench`)
- Token overhead measurement
- Build determinism test
- Cross-target isomorphism validation
- Formal property tests (P10–P15)
- Pass@k consistency measurement
- LLM-as-judge evaluation (composed vs monolithic)
- SWE-bench lite wrapper
- Gremlins mutation testing integration
