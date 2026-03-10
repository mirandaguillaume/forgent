# AX CLI - Agent Experience CLI

## Problem

The AI agent ecosystem in 2026 is highly fragmented. Each framework (Claude Code, CrewAI, LangGraph, OpenAI Agents SDK, AG2) has its own abstractions, patterns, and mental models. Developers who master one framework are lost when switching to another. There is no standardized tooling for the **Agent Experience (AX)** — the developer experience of building, debugging, composing, and maintaining AI agents.

## Vision

`ax` is a standalone CLI that improves AX across all agent frameworks through a composition-first approach. Agents are not monolithic entities but **compositions of Skill Behaviors** — reusable, testable, portable behavioral units.

## Core Concept: Skill Behavior Model

A Skill Behavior is the atomic unit of agent capability. It defines not just WHAT an agent can do, but HOW it does it, WHEN it triggers, and WHAT limits apply.

### The 6 Facets

```
+------------------ Skill Behavior Model ------------------+
|                                                          |
|  +----------+  +----------+  +----------+               |
|  | Context  |  | Strategy |  |  Guards  |               |
|  | in/out   |  | tools    |  | limits   |               |
|  | memory   |  | approach |  | rules    |               |
|  +----------+  +----------+  +----------+               |
|                                                          |
|  +----------+  +----------+  +----------+               |
|  |   Deps   |  | Observe  |  | Security |               |
|  | requires |  | traces   |  | perms    |               |
|  | provides |  | metrics  |  | sandbox  |               |
|  +----------+  +----------+  +----------+               |
|                                                          |
|  version: semver | negotiation: yield|override|merge     |
+----------------------------------------------------------+
```

#### 1. Context (Memory/State)
- `consumes`: data the skill needs as input (e.g., git_diff, file_tree)
- `produces`: data the skill generates as output (e.g., review_comments, risk_score)
- `memory`: persistence model — `short-term` (single run), `conversation` (session), `long-term` (across sessions)

#### 2. Strategy (How)
- `tools`: which tools the skill can use
- `approach`: the behavioral pattern (e.g., diff-first, depth-first, breadth-first)
- `steps`: optional ordered steps within the strategy

#### 3. Guardrails (Limits)
- Rules that constrain behavior (e.g., no_approve_without_tests)
- Quantitative limits (max_comments, timeout, max_tokens)

#### 4. Dependencies (Composition)
- `depends_on`: other skills that must run before this one
- `provides`: what this skill makes available to downstream skills
- Enables DAG-based orchestration of skill execution

#### 5. Observability (Tracing)
- Structured traces of decisions, tool calls, and outcomes
- Metrics: token usage, latency, success rate, cost
- Exportable to OpenTelemetry-compatible backends

#### 6. Security (Permissions)
- Filesystem access scope (read-only, specific paths, full access)
- Network access (none, allowlist, full)
- Secret access (which env vars/vaults the skill can read)
- Sandbox level (none, container, VM)

### Versioning
Skills follow semver. Breaking changes to `consumes`/`produces` require a major version bump. Strategy changes are minor. Guardrail adjustments are patches.

### Negotiation
When multiple skills want to modify the same resource:
- `yield`: defer to higher-priority skill
- `override`: take precedence
- `merge`: attempt to merge changes (with conflict resolution strategy)

## Skill Definition Example

```yaml
skill: code-review
version: 1.2.0

context:
  consumes: [git_diff, file_tree, test_results]
  produces: [review_comments, risk_score]
  memory: conversation

strategy:
  tools: [read_file, grep, git_diff]
  approach: diff-first
  steps:
    - analyze_diff
    - check_patterns
    - write_review

guardrails:
  - no_approve_without_tests
  - max_comments: 10
  - timeout: 5min

depends_on:
  - skill: test-coverage
    provides: test_results

observability:
  trace_level: detailed
  metrics: [tokens, latency, decisions]

security:
  filesystem: read-only
  network: none
  secrets: []

negotiation:
  file_conflicts: yield
  priority: 2
```

## Agent as Skill Composition

An agent is simply a named composition of skills with an orchestration strategy:

```yaml
agent: senior-reviewer
skills:
  - code-review
  - security-audit
  - test-coverage-check
orchestration: parallel-then-merge
```

Orchestration strategies:
- `sequential`: skills run in dependency order
- `parallel`: independent skills run concurrently
- `parallel-then-merge`: parallel execution, then merge results
- `adaptive`: orchestrator decides based on context

## CLI Architecture

```
+---------------------------------------------+
|                  ax CLI                      |
+-------------+-------------+-----------------+
|  Diagnostic |  Compose    |  Portability    |
|  doctor     |  compose    |  translate      |
|  lint       |  test       |  generate       |
|  trace      |  bench      |                 |
+-------------+-------------+-----------------+
|           Skill Behavior Model              |
+---------------------------------------------+
|           Framework Adapters                |
|  Claude Code | CrewAI | LangGraph | Custom  |
+---------------------------------------------+
```

### Commands

| Command | Layer | Description |
|---------|-------|-------------|
| `ax init` | Setup | Initialize an AX project, detect existing frameworks |
| `ax skill create <name>` | Creation | Scaffold a new skill-behavior from template |
| `ax skill test <name>` | Testing | Simulate a skill with mock inputs |
| `ax compose <agent.yaml>` | Composition | Compose skills into an agent, validate DAG |
| `ax doctor <path>` | Diagnostic | Analyze an existing agent (traces, costs, loops) |
| `ax lint <path>` | Quality | Check AX best practices (tool descriptions, error handling) |
| `ax translate <skill> --from <fw> --to <fw>` | Portability | Convert a skill between frameworks |
| `ax trace <agent>` | Observability | Real-time execution trace |
| `ax bench <task> --frameworks <list>` | Benchmark | Compare same task across frameworks |

## Tech Stack

- **Language**: TypeScript (Node.js)
- **CLI framework**: Commander.js + Ink (for interactive TUI)
- **Package distribution**: npm
- **Skill format**: YAML (with JSON Schema validation)
- **Tracing**: OpenTelemetry-compatible
- **Testing**: Vitest

## Non-Goals (YAGNI)

- Visual/GUI editor for skills (CLI-first)
- Runtime agent execution (ax describes and analyzes, it doesn't run agents)
- Framework-specific IDE plugins (out of scope for v1)
- Cloud/SaaS version (local CLI only)

## Open Questions

1. Should `ax` include a skill registry/marketplace, or rely on npm/git for distribution?
2. How deep should framework adapters go? Full code generation vs. skeleton + manual wiring?
3. Should negotiation be defined at the skill level or at the composition/agent level?

## References

- [Agent Experience (AX) - Nordic APIs](https://nordicapis.com/what-is-agent-experience-ax/)
- [5 Key Trends Shaping Agentic Development in 2026](https://thenewstack.io/5-key-trends-shaping-agentic-development-in-2026/)
- [LangGraph vs CrewAI vs OpenAI Agents SDK 2026](https://particula.tech/blog/langgraph-vs-crewai-vs-openai-agents-sdk-2026)
- [AI Agent Frameworks Compared 2026](https://arsum.com/blog/posts/ai-agent-frameworks/)
