---
title: "Composable Skill Behaviors: A Declarative Format for AI Agent Engineering"
weight: 2
---

<div style="text-align: center; margin-bottom: 2rem;">
<em>A format specification for defining, composing, and deploying AI agents through reusable behavioral units.</em>
<br><br>
<strong>Guillaume Miranda</strong> — March 2026
</div>

---

## Abstract

AI agents are becoming critical infrastructure in software engineering workflows. Yet most agent definitions today are monolithic prompt files — brittle, untestable, and locked to a single framework and a single model.

This paper presents the **Skill Behavior Model**, a declarative YAML format for agent engineering. Each **Skill Behavior** is a reusable unit governed by 5 facets — Context, Strategy, Guardrails, Observability, and Security. A skill is a pure interface: it declares what it consumes and what it produces. Skills compose into **Agents** — named compositions that declare their own I/O contract and orchestration strategy. The linter validates coherence between agent and skill interfaces statically.

The format is both framework-agnostic and LLM-agnostic: the same specification compiles to Claude Code [1], GitHub Copilot [2], or any other target without modification. We describe the model, its portability guarantees, and the design rationale behind each constraint.

---

## 1. Introduction

### 1.1 Monolithic Agent Prompts

The dominant pattern for defining AI agent behavior is a single long-form prompt file. A typical CI review agent might be defined as:

```markdown
You are a code reviewer. Read the diff, run tests, check types,
lint the code, write review comments, and assign a risk score.
Use bash for running commands and read for files...
```

This approach has three fundamental problems.

**Responsibility overload.** A single prompt handles linting, testing, reviewing, and scoring. When one concern changes (e.g., switching lint rules), the entire prompt must be re-validated.

**No reusability.** The linting behavior embedded in a CI agent cannot be extracted and reused by a security audit agent. Copy-paste becomes the de facto composition mechanism — a pattern that multi-agent orchestration research identifies as a key scaling bottleneck [3].

**Framework lock-in.** Prompts reference framework-specific tools (`Bash`, `Read` for Claude Code; `execute`, `search` for Copilot). Switching frameworks means rewriting every agent definition. Switching LLMs means re-tuning every prompt.

### 1.2 The Structured Gap

The coding agent ecosystem has grown rapidly. As of early 2026, developers choose between Claude Code [1], GitHub Copilot [2], OpenAI Codex, Gemini CLI, Cursor, Windsurf, Kiro, Aider, OpenCode, and others. Most of these tools now allow some form of agent customization — but the formats are fragmented and shallow.

Three families have emerged.


**Declarative markdown agents.** Claude Code, GitHub Copilot, Gemini CLI, and OpenCode define agents as markdown files with YAML frontmatter. A typical agent looks like:

```yaml
---
name: code-reviewer
description: Reviews code for quality
tools: Read, Grep, Glob
model: sonnet
---

You are a code reviewer. When invoked, analyze the code and provide
specific, actionable feedback on quality, security, and best practices.
```

The frontmatter declares tool access and model selection. The body is a free-form behavioral prompt. Each framework uses its own file location and field names, but the pattern is converging.

**Rules-based systems.** Cursor (`.cursor/rules/*.mdc`) and Windsurf (`.windsurf/rules/`) define conditional rules that guide a single built-in agent. Rules have activation conditions (file globs, always-apply flags) but no concept of separate behavioral agents or composition.

**Programmatic agents.** OpenAI Codex defines agents through its Python Agents SDK — no declarative file, agents are instantiated in code. Kiro (AWS) uses structured YAML configuration with "Powers" that bundle MCP tools, steering files, and hooks.

All three families share the same structural gap. Only two concerns are consistently structured across agent definitions:

| Concern | Structured? | Where it lives |
|---------|:-----------:|----------------|
| Tool access | Yes | Frontmatter (`tools:`) or config |
| Model selection | Yes | Frontmatter (`model:`) or config |
| Behavioral instructions | No | Free-form markdown body |
| Guardrails (timeouts, limits) | No | Buried in prose, if present |
| Security (filesystem, network) | No | Not declared |
| Skill composition / data flow | No | Not expressible |
| Observability (metrics, traces) | No | Not declared |
| I/O contract (consumes/produces) | No | Not declared |

The bottom six rows are the gap. An agent's behavioral constraints, security posture, data dependencies, and I/O contract are either absent or scattered across unstructured prose. They cannot be validated statically, composed programmatically, or ported across frameworks. This gap exists in **every** coding agent framework today — not just one or two.

### 1.3 The Missing Abstraction

Software engineering solved analogous problems decades ago. Functions decompose monolithic programs. Interfaces enable composition. Design principles prevent rot [4]. Agent engineering needs the same discipline — a structured format that separates what an agent *does* from how a specific framework or model *executes* it.

---

## 2. The Skill Behavior Model

The Skill Behavior Model introduces a three-layer architecture:

```
┌─────────────────────────┐
│  Agent Composition      │  Orchestration of skills
│  (agents/*.agent.yaml)  │  sequential | parallel | adaptive
├─────────────────────────┤
│  Skill Behaviors        │  Reusable behavioral units
│  (skills/*.skill.yaml)  │  6 facets per skill
├─────────────────────────┤
│  Build Targets          │  Framework-specific output
│  (generated artifacts)  │  Compiled, never hand-written
└─────────────────────────┘
```

**Skills** are leaf nodes — each does exactly one thing and produces exactly one output. **Agents** are compositions — they wire skills into execution pipelines. **Build targets** are generated artifacts — compiled from the abstract specs into framework-native formats.

### 2.1 Skills

Every skill is defined by **5 core facets**:

| # | Facet | Purpose | Example |
|---|-------|---------|---------|
| 1 | **Context** | I/O contract: what data flows in and out | `consumes: [git_diff, test_results]` → `produces: [review_comments]` |
| 2 | **Strategy** | Tools, approach, execution steps | `tools: [read_file, grep]`, `approach: diff-first` |
| 3 | **Guardrails** | Rules, limits, constraints | `timeout: 5min`, `max_comments: 15` |
| 4 | **Observability** | Traces and metrics | `trace_level: detailed`, `metrics: [tokens, latency]` |
| 5 | **Security** | Filesystem, network, secrets | `filesystem: read-only`, `network: none` |

A **Negotiation** facet handles multi-agent conflict resolution (`file_conflicts: yield`, `priority: 3`).

Three **optional documentation facets** enrich the skill without affecting its behavior:

| Facet | Purpose |
|-------|---------|
| **when_to_use** | Triggers, exclusions, and especially-good-for conditions |
| **anti_patterns** | Common mistakes as excuse/reality pairs |
| **examples** | Labeled code snippets demonstrating correct usage |

#### A complete skill

```yaml
skill: review-commenter
version: "1.0.0"

context:
  consumes: [git_diff, test_results, lint_results]
  produces: [review_comments]
  memory: conversation

strategy:
  tools: [read_file, grep, search]
  approach: diff-first
  steps:
    - read the git diff
    - cross-reference with test results and lint results
    - identify risky changes
    - write actionable review comments

guardrails:
  - max_comments: 15
  - timeout: 5min
  - no_approve_without_tests

observability:
  trace_level: detailed
  metrics: [tokens, latency, comment_count]

security:
  filesystem: read-only
  network: none
  secrets: []

negotiation:
  file_conflicts: yield
  priority: 3
```

**Key constraint:** `consumes` accepts a list (multiple inputs), but `produces` must contain **exactly one item**. A skill that `produces: [lint_results, type_errors]` is two skills pretending to be one. This constraint enforces the Single Responsibility Principle at the format level — each skill does one thing, and the format itself makes violations structurally visible.

Skills are pure interfaces. They declare their I/O contract (`consumes`/`produces`) but have no knowledge of which other skills provide their inputs. Data flow is not declared in the skill — it emerges from the composition declared in the agent. This enforces the Law of Demeter [5]: a skill only accesses data it explicitly declares in its `consumes` contract. The linter validates that every skill's inputs are satisfied by either another skill's outputs or the agent's own `consumes`.

### 2.2 Agents

An agent is a named composition of skills with its own I/O contract and orchestration strategy:

```yaml
agent: ci-reviewer
description: "Runs linting, type-checking, tests, coverage, then reviews and scores"
skills:
  - ts-linter
  - type-checker
  - tdd-runner
  - coverage-reporter
  - review-commenter
  - risk-scorer
orchestration: sequential
consumes: [git_diff, file_tree, source_code]
produces: [review_comments, risk_score]
```

The agent declares what it consumes from the outside world and what it produces as final output. Skills within the agent wire together through their `consumes`/`produces` contracts — `tdd-runner` produces `test_results`, `review-commenter` consumes `test_results`. The linter validates this coherence statically: every skill's inputs must be satisfied by either another skill's outputs or the agent's `consumes`.

Agents compose skills — they never inherit from other agents. There is no agent hierarchy, no "base agent" pattern. This is composition over inheritance applied at the agent level: behavior is assembled from atomic parts, not specialized from a parent.

Skills declare what they produce. The agent declares the execution order and its own I/O boundary. The framework reads both and orchestrates. No skill asks another skill for data — the framework resolves the data flow and routes outputs. This is Tell, Don't Ask [6] applied to agent design.

### 2.3 Orchestration

The `consumes`/`produces` contracts between skills form a directed acyclic graph (DAG):

```
ci-reviewer agent data flow:

ts-linter ──────────┐
type-checker        │
tdd-runner ─────────┼──→ review-commenter
coverage-reporter   │
                    └──→ risk-scorer
```

`ts-linter` produces `lint_results`. `review-commenter` consumes `lint_results`. The data flow emerges from the skill interfaces — no explicit wiring is needed. The graph is statically verifiable before any agent runs.

Four orchestration strategies are defined:

| Strategy | Behavior | Use case |
|----------|----------|----------|
| `sequential` | Skills run in declared order, outputs feed forward | CI pipelines, review workflows |
| `parallel` | All skills run concurrently | Independent analyses |
| `parallel-then-merge` | Run in parallel, merge results | Multi-source aggregation |
| `adaptive` | Dynamic execution based on intermediate results | Complex decision trees |

Three validation rules apply to the DAG:

| Rule | What it catches |
|------|----------------|
| **Missing dependencies** | A skill consumes data that no other skill in the agent produces |
| **Circular dependencies** | Skills A → B → C → A |
| **Unmet context** | A skill requires input that no enricher provides |

These rules are statically checkable — no agent needs to run for violations to be detected.

---

## 3. Portability

### 3.1 Framework Independence

Skills use abstract tool names in their `strategy.tools` field: `read_file`, `grep`, `search`, `execute`. These are behavioral capabilities, not framework-specific APIs.

When compiling to a target, the abstract names are mapped:

| Abstract | Claude Code | GitHub Copilot |
|----------|-------------|----------------|
| `read_file` | `Read` | `read` |
| `grep` | `Grep` | `search` |
| `execute` | `Bash` | `execute` |
| `search` | `Glob` | `search` |

The skill author writes `tools: [read_file, grep]`. The compilation step translates. Switching from Claude Code to GitHub Copilot requires zero changes to any skill or agent specification — only a different build target.

### 3.2 LLM Independence

The format describes **behavior**, not **prompt instructions**. A skill's `strategy.steps` are declarative:

```yaml
steps:
  - read the git diff
  - cross-reference with test results
  - identify risky changes
  - write actionable review comments
```

These steps describe *what* the skill does, not *how* a specific model should interpret them. The same `review-commenter` skill works whether the underlying LLM is Claude, GPT, Gemini, or a future model. The compilation step may adapt the generated prompt to a model's strengths, but the skill specification itself remains unchanged.

This separation means that model upgrades — or model switches — do not require rewriting skill definitions. The behavioral contract is stable; the execution adapts.

### 3.3 Write Once, Deploy Anywhere

The same YAML specifications produce different outputs depending on the target:

| Target | Output directory | Skill format | Agent format |
|--------|-----------------|--------------|--------------|
| Claude Code | `.claude/` | `skills/<name>/SKILL.md` | `agents/<name>.md` |
| GitHub Copilot | `.github/` | `skills/<name>/SKILL.md` | `agents/<name>.agent.md` |

Both targets generate markdown skill files optimized for each framework's conventions. New targets can be added without modifying any existing specification — the format is open for extension.

Generated skill files use a deliberate section ordering based on LLM attention research. Liu et al. [7] demonstrated that language models perform best when critical information is placed at the beginning or end of the input context — the "lost in the middle" effect. Peysakhovich & Lerer [8] confirmed that primacy and recency biases are widespread across LLM architectures. The generated output therefore places:

1. **Guardrails** first — primacy bias ensures constraints are remembered
2. **Context, Strategy** in the middle — the execution plan
3. **Security** last — recency bias keeps access controls top of mind

---

## 4. Design Rationale

The Skill Behavior Model applies established software design principles to agent engineering.

**Single Responsibility Principle.** Each skill produces exactly one output. This is not a convention — it is a structural constraint of the format. A skill with `produces: [lint_results, type_errors]` is invalid. This forces decomposition: one `pr-reviewer` producing two outputs becomes two skills — `review-commenter` and `risk-scorer` — each testable and reusable independently.

**Law of Demeter.** A skill only accesses data declared in its `consumes` contract [5]. Data flow between skills is inferred from their `consumes`/`produces` interfaces and validated by the linter. No global state, no shared context outside the declared contract. The data flow graph is fully explicit and statically analyzable.

**Composition over Inheritance.** Agents are flat lists of skills — no "base agents", no inheritance, no override mechanisms. Behavior is assembled, not specialized. This eliminates the fragile base class problem: changing a skill affects only agents that explicitly include it.

**Tell, Don't Ask.** Skills declare what they produce. The framework reads the declarations and routes data [6]. No skill queries another skill's state — skills are decoupled from each other and only know their own contract.

**LLM Attention Optimization.** Generated output orders sections to exploit primacy and recency biases [7][8]. Guardrails go first (primacy), security goes last (recency). This is a compilation concern — the YAML format is unordered, the generated artifacts are deliberately structured.

**Framework Independence.** Tool names are abstract behavioral capabilities. The mapping to concrete APIs happens at compilation time. Skill authors never write framework-specific code; framework migrations are zero-cost at the specification level.

---

## 5. Experience

### 5.1 Reusability

In practice, skills compose across agents without modification. The `tdd-runner` skill — which produces `test_results` — is consumed by `review-commenter`, `risk-scorer`, and `coverage-reporter` across different agents. Each consumer declares `test_results` in its `consumes`; the `tdd-runner` skill is unaware of its consumers. Adding a new consumer requires zero changes to `tdd-runner`.

### 5.2 Cross-Framework Deployment

The same set of 6 skills and 1 agent has been compiled to both Claude Code and GitHub Copilot targets. The skill specifications are identical across both. The only differences are in the generated output: tool name mappings, file paths, and framework-specific conventions (e.g., Copilot's `copilot-instructions.md` global file, which Claude Code does not use).

### 5.3 Static Validation

The DAG structure enables validation before any agent executes. Missing dependencies, circular references, and unmet context are caught at specification time. This is analogous to type checking in programming languages — errors are found before runtime, not during a costly LLM invocation.

### 5.4 Limitations

The Skill Behavior Model deliberately does not cover two concerns:

**Agent identity.** The format describes what an agent *can do*, not *who it is*. Personality, tone, and values are the domain of complementary formats like SOUL.md [9]. The two are composable — an agent can have both a skill specification (capabilities) and a soul specification (identity).

**Inter-agent communication.** The format defines behavior within a single agent. Communication between agents — discovery, negotiation, message passing — is the domain of protocols like MCP [10], Agent2Agent [11], and ACP. The Skill Behavior Model can coexist with these protocols: skills define what each agent does, protocols define how agents talk to each other.

---

## 6. Related Work

### 6.1 The Coding Agent Ecosystem

The following table maps how each major coding agent framework handles customization as of early 2026:

| Framework | Context file | Agent format | Skills | Structured fields |
|-----------|-------------|--------------|--------|-------------------|
| Claude Code [1] | `CLAUDE.md` | `.claude/agents/*.md` | `.claude/skills/*/SKILL.md` | name, description, tools, model, memory, hooks |
| GitHub Copilot [2] | Repo instructions | `.github/agents/*.agent.md` | `.github/skills/*/SKILL.md` | name, description, tools, model |
| Gemini CLI | `GEMINI.md` | `.gemini/agents/*.md` | — | name, description, tools, model, temperature, max_turns, timeout |
| OpenCode | `opencode.json` | `.opencode/agents/*.md` | — | description, mode, model, tools, permission |
| Codex (OpenAI) | `AGENTS.md` | Agents SDK (Python) | `SKILL.md` | Programmatic only |
| Kiro (AWS) | Steering files | YAML config | "Powers" | name, description, prompt, tools, mcpServers |
| Cursor | `.cursor/rules/*.mdc` | — (rules only) | — | description, globs, alwaysApply |
| Windsurf | `.windsurfrules` | — (rules only) | — | Activation mode |
| Aider | `.aider.conf.yml` | — | — | Config only |

A convergence is visible: the dominant pattern is markdown with YAML frontmatter declaring `name`, `description`, `tools`, and `model`. But across **all** frameworks, the same concerns remain unstructured: guardrails, security, observability, and I/O contracts.

### 6.2 Complementary Specifications

| Approach | What it defines | Composition | Validation |
|----------|----------------|:-----------:|:----------:|
| AGENTS.md [12] | Project context for agents | Hierarchical override | None |
| SOUL.md [9] | Agent identity (personality, tone) | Per-agent | None |
| Open Agent Spec [14] | Workflow graphs (nodes, edges, data flow) | Graph-based | Schema |
| MCP / A2A [10][11] | Inter-agent communication protocols | Message-based | Schema |
| LangGraph / CrewAI [15] | Programmatic agent orchestration (Python) | Code-level | Runtime |
| NIST AI Agent Standards [16] | Governance, security, monitoring policies | Policy-level | Audit |
| **Skill Behavior Model** | **Behavioral capabilities and constraints** | **Declarative YAML** | **Static** |

These specifications answer different questions about agent systems:

**AGENTS.md** [12] is a free-form markdown standard for guiding coding agents, stewarded by the Linux Foundation. It provides project-level context (build steps, conventions, architecture) but has no structured facets, no typed I/O contracts, and no composition model. It tells an agent *about the project* — the Skill Behavior Model tells an agent *what to do and under what constraints*.

**SOUL.md** [9] defines agent identity — personality, tone, values. The Skill Behavior Model defines agent capabilities. They answer different questions and are composable: an agent can have both a soul (who it is) and skills (what it can do).

**Open Agent Specification** [14] is a declarative, framework-agnostic YAML format for defining agent workflows, introduced by Oracle in October 2025. It models workflows as directed graphs of typed nodes (LLMNode, APINode, ToolNode) with explicit data flow edges — analogous to ONNX for ML models. The Skill Behavior Model operates at a different granularity: it defines the *behavioral content* of each node, not the *workflow between nodes*. The two are potentially complementary — skills could serve as the behavioral building blocks within an Agent Spec workflow. The NIST AI Agent Standards Initiative [16] is working toward standardizing similar concerns at the policy level.

**Programmatic frameworks** (LangGraph, CrewAI) [15] define agents in code (Python). The Skill Behavior Model defines agents in data (YAML). The declarative approach enables static analysis and non-programmer access to agent design. The trade-off is expressiveness — programmatic frameworks can encode arbitrary logic that a declarative format cannot.

**Communication protocols** (MCP, A2A, ACP) [10][11] define how agents discover each other, negotiate, and exchange messages. The Skill Behavior Model defines what each agent does internally. They operate at different layers and are composable: skills define behavior, protocols define communication.

---

## 7. Conclusion

The Skill Behavior Model brings to agent engineering what interfaces and design principles brought to software engineering: structured decomposition, explicit contracts, and static validation. Each skill is a single-responsibility unit with a declared I/O contract. Agents are flat compositions of skills. The format is framework-agnostic, LLM-agnostic, and statically validatable.

The format is deliberately minimal. It does not prescribe an implementation language, a runtime, or a specific LLM. It defines a behavioral contract — what an agent skill does, what it needs, what it produces, and what constraints it operates under. Implementations compile this contract to framework-native artifacts.

Future directions include new facets for emerging concerns (cost budgets, latency targets, human-in-the-loop gates), integration with agent communication protocols (MCP, A2A), and community-driven extension of the facet registry.

---

## References

[1] Anthropic, "Equipping agents for the real world with Agent Skills," Anthropic Engineering Blog, 2025. [Anthropic](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills) | [Docs](https://code.claude.com/docs/en/skills)

[2] GitHub, "Custom agents for GitHub Copilot," GitHub Changelog, October 2025. [GitHub Blog](https://github.blog/changelog/2025-10-28-custom-agents-for-github-copilot/) | [Docs](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-custom-agents)

[3] "The Orchestration of Multi-Agent Systems: Architectures, Protocols, and Enterprise Adoption," arXiv:2601.13671, January 2026. [arXiv](https://arxiv.org/abs/2601.13671)

[4] R. C. Martin, "Design Principles and Design Patterns," 2000. The SOLID acronym was coined by M. Feathers circa 2004. [Wikipedia](https://en.wikipedia.org/wiki/SOLID)

[5] K. J. Lieberherr, I. M. Holland, and A. J. Riel, "Object-Oriented Programming: An Objective Sense of Style," OOPSLA '88. The Law of Demeter: "Only talk to your immediate friends." [ACM](https://dl.acm.org/doi/10.1145/62084.62113)

[6] A. Hunt and D. Thomas, *The Pragmatic Programmer*, Addison-Wesley, 1999. "Tell, Don't Ask" principle. [Pragmatic Bookshelf](https://pragprog.com/titles/tpp20/the-pragmatic-programmer-20th-anniversary-edition/)

[7] N. F. Liu, K. Lin, J. Hewitt, A. Paranjape, M. Bevilacqua, F. Petroni, and P. Liang, "Lost in the Middle: How Language Models Use Long Contexts," *Transactions of the Association for Computational Linguistics*, vol. 12, pp. 157–173, 2024. [arXiv](https://arxiv.org/abs/2307.03172)

[8] "Serial Position Effects of Large Language Models," arXiv:2406.15981, 2024. [arXiv](https://arxiv.org/abs/2406.15981)

[9] A. Mars, "soul.md — The best way to build a personality for your agent," GitHub, 2025. [GitHub](https://github.com/aaronjmars/soul.md) | [soul.md](https://soul.md/)

[10] Anthropic, "Model Context Protocol," 2024. Now governed by the Linux Foundation. [MCP](https://modelcontextprotocol.io/)

[11] Google Cloud, "Agent2Agent Protocol," April 2025. Governed by the Linux Foundation. [A2A](https://github.com/google/A2A)

[12] "AGENTS.md — A simple, open format for guiding coding agents," 2025. Stewarded by the Agentic AI Foundation under the Linux Foundation. [agents.md](https://agents.md/) | [GitHub](https://github.com/agentsmd/agents.md)

[13] S. Zeng et al., "ADL: A Declarative Language for Agent-Based Chatbots," arXiv:2504.14787, 2025. [arXiv](https://arxiv.org/abs/2504.14787)

[14] "Open Agent Specification (Agent Spec): A Unified Representation for AI Agents," arXiv:2510.04173, October 2025. A declarative, framework-agnostic YAML format for defining agents and workflows. [arXiv](https://arxiv.org/abs/2510.04173) | [Oracle Blog](https://blogs.oracle.com/ai-and-datascience/introducing-open-agent-specification)

[15] "Agentic AI: A Comprehensive Survey of Architectures, Applications, and Future Directions," arXiv:2510.25445, 2025. [arXiv](https://arxiv.org/abs/2510.25445)

[16] NIST, "AI Agent Standards Initiative," Center for AI Standards and Innovation (CAISI), February 2026. [NIST](https://www.nist.gov/caisi/ai-agent-standards-initiative)
