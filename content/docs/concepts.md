---
title: Concepts
weight: 2
---

Forgent is built on one core idea: **agents are compositions of Skill Behaviors**.

## Skill Behavior Model

A **Skill Behavior** is a reusable behavioral unit that describes *what* an agent skill does and *how* it should behave. Each skill is a pure interface defined in YAML with **5 facets**:

| Facet | What it defines |
|-------|----------------|
| **Context** | Memory type, data consumed and produced (the I/O contract) |
| **Strategy** | Tools, approach, execution steps |
| **Guardrails** | Rules, limits, constraints (timeouts, max tokens) |
| **Observability** | Trace level, metrics to collect |
| **Security** | Filesystem access, network access, secrets, sandboxing |

### Why 5 facets?

Each facet addresses a distinct concern in agent design:

- **Context** defines the I/O contract — what data flows in and out
- **Strategy** describes the execution plan — tools and steps
- **Guardrails** prevent runaway behavior — timeouts, limits, constraints
- **Observability** makes behavior visible — traces and metrics
- **Security** enforces least privilege — minimal access by default

Skills are pure interfaces: they declare `consumes` and `produces` but have no knowledge of which other skills provide their inputs. Data flow emerges from the composition declared in the agent.

## Agent Composition

An **Agent** is a named composition of skills with its own I/O contract and orchestration strategy:

```yaml
agent: research-pipeline
description: "Search, analyze, and report"
skills:
  - search-web
  - analyze-code
  - write-report
orchestration: sequential
consumes: [query, source_code]
produces: [research_report]
```

The agent declares what it `consumes` from the outside world and what it `produces` as final output. The linter validates coherence: every skill's inputs must be satisfied by either another skill's outputs or the agent's `consumes`, and the agent's `produces` must match what its skills actually produce.

### Orchestration Strategies

| Strategy | Behavior |
|----------|----------|
| `sequential` | Execute skills in order, passing outputs forward |
| `parallel` | Execute all skills concurrently |
| `parallel-then-merge` | Run in parallel, then merge results |
| `adaptive` | Choose execution order dynamically based on context |

## Build Targets

Skills and agents are **framework-agnostic** — they describe behavior, not implementation. The `forgent build` command compiles them into framework-specific formats:

```
YAML Spec → forgent build → Framework Output
                ↓
         ┌──────┴──────┐
    .claude/       .github/
   (Claude Code)  (Copilot)
```

This separation means you write your skill specs once and deploy to any supported framework. See [Build Targets](../build-targets) for details.

## Generator Interfaces

New build targets are added by implementing focused interfaces from `pkg/spec`:

```go
type Generator interface {
    Target() string
    DefaultOutputDir() string
    ContextDir() string
}

type SkillGenerator interface {
    GenerateSkill(skill model.SkillBehavior) string
    SkillPath(name string) string
}

type AgentGenerator interface {
    GenerateAgent(agent model.AgentComposition, skills []model.SkillBehavior, outputDir string) string
    AgentPath(name string) string
}

type InstructionsGenerator interface {  // Optional
    GenerateInstructions(skills []model.SkillBehavior, agents []model.AgentComposition) string
    InstructionsPath() string
}
```

Interfaces follow the Interface Segregation Principle — generators implement only what they support. Third parties can import `pkg/spec` and `pkg/model` to build their own generators without depending on Forgent's internal implementation.

## Design Quality

Forgent includes built-in analysis tools:

- **Lint** — checks for common design issues (missing guardrails, empty tools, security gaps)
- **Score** — rates skill quality across 5 weighted facets (0-100 scale)
- **Doctor** — full diagnostic: lint + dependency analysis + loop detection
