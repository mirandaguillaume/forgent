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

AI agent definitions are predominantly monolithic prompt files — unstructured, untestable, and coupled to a single framework. This paper presents the Skill Behavior Model, a declarative format for defining AI agents as compositions of reusable behavioral units called *skills*. Each skill declares a typed I/O contract (`consumes`/`produces`) and is governed by five facets: Context, Strategy, Guardrails, Observability, and Security. Agents are flat compositions of skills with their own I/O boundary and orchestration strategy.

We formalize the model's structural properties and prove fifteen results: well-formedness is decidable in linear time (P1), compatible skill substitution preserves agent validity — justifying contract-based semantic versioning (P2), independent skills are safely parallelizable with a canonical layer decomposition and computable maximum parallelism (P3, Corollaries 3.1–3.2), a URI-based data flow protocol guarantees resolution completeness (P4), output immutability under a write-once axiom (P5), cycle-free resolution (P6), skill addition under isolation conditions preserves well-formedness (P7), valid schedules exist for every well-formed agent (P8), artifacts generated for different frameworks are structurally isomorphic (P9), every skill is reachable from source to sink (P10), exclusively-connected skills can be fused while preserving well-formedness (P11), pure execution environments provide data isolation (P12), permission containment (P13), parallel independence (P14), and conflict-free merge under disjoint write sets (P15). The linter provides structural soundness — well-validated agents are free from missing dependencies, circular references, responsibility overload, and orphan outputs. We characterize the gap between structural soundness and semantic correctness, establish a Petri net correspondence and connections to linear logic, formalize a worktree-based isolation model with three-level merge strategies, and describe four frontier directions: behavioral testing, a skill ecosystem with contract-derived versioning, runtime enforcement through a four-layer ladder, and multi-agent coordination via protocol bridges.

The format is framework-agnostic and LLM-agnostic: the same YAML specification generates output for Claude Code [1], GitHub Copilot [2], or other targets without modification.

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

**No reusability.** The linting behavior embedded in a CI agent cannot be extracted and reused by a security audit agent. Copy-paste becomes the de facto composition mechanism. Surveys of multi-agent architectures consistently note the lack of modular reuse mechanisms as a barrier to scaling [3].

**Framework lock-in.** Prompts reference framework-specific tools (`Bash`, `Read` for Claude Code; `execute`, `search` for Copilot). Switching frameworks means rewriting every agent definition. Switching LLMs means re-tuning every prompt.

### 1.2 The Structured Gap

The coding agent ecosystem has grown rapidly. As of early 2026, developers choose between Claude Code [1], GitHub Copilot [2], OpenAI Codex, Gemini CLI, Cursor, Windsurf, Cline, Roo Code, Augment Code, Kiro, Aider, OpenCode, Amazon Q Developer, Zed AI, Continue, and others. Most of these tools now allow some form of agent customization — but the formats are fragmented and shallow.

Four families have emerged.


**Declarative markdown agents.** Claude Code, GitHub Copilot, Gemini CLI, OpenCode, Amazon Q Developer, Zed AI, and Continue define agents as markdown files with YAML frontmatter. A typical agent looks like:

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

**Rules-based systems.** Cursor (`.cursor/rules/*.mdc`), Windsurf (`.windsurf/rules/`), Cline (`.clinerules/`), Roo Code (`.roo/rules/`), and Augment Code (`.augment/rules/`) define conditional rules that guide a single built-in agent. Rules have activation conditions (file globs, always-apply flags) but no concept of separate behavioral agents or composition.

**Config-based systems.** Kiro (AWS) uses structured YAML configuration with "Powers" that bundle MCP tools, steering files, and hooks. Aider uses `.aider.conf.yml` for model and behavior configuration without a standalone agent concept.

**Programmatic agents.** OpenAI Codex defines agents through its Python Agents SDK — no declarative file, agents are instantiated in code.

All four families share the same structural gap. Only two concerns are consistently structured across agent definitions:

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
│  (skills/*.skill.yaml)  │  5 facets per skill
├─────────────────────────┤
│  Build Targets          │  Framework-specific output
│  (generated artifacts)  │  Generated, never hand-written
└─────────────────────────┘
```

**Skills** are leaf nodes — each does exactly one thing and produces exactly one output. **Agents** are compositions — they wire skills into execution pipelines. **Build targets** are generated artifacts — produced from the abstract specs into framework-native formats.

### 2.1 Skills

Every skill is defined by **5 core facets**:

| # | Facet | Purpose | Example |
|---|-------|---------|---------|
| 1 | **Context** | I/O contract: what data flows in and out | `consumes: [git_diff, test_results]` → `produces: [review_comments]` |
| 2 | **Strategy** | Tools, approach, execution steps | `tools: [read_file, grep]`, `approach: diff-first` |
| 3 | **Guardrails** | Rules, limits, constraints | `timeout: 5min`, `max_comments: 15` |
| 4 | **Observability** | Traces and metrics | `trace_level: detailed`, `metrics: [tokens, latency]` |
| 5 | **Security** | Filesystem, network, secrets | `filesystem: read-only`, `network: none` |

Beyond the 5 core facets, a **Negotiation** facet handles multi-agent conflict resolution (`file_conflicts: yield`, `priority: 3`). It is separate because it governs inter-agent coordination, not the skill's own behavior.

Three **optional documentation facets** enrich the skill without affecting its runtime behavior:

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

**Key constraint:** `consumes` accepts a list (multiple inputs), but `produces` should contain **exactly one item**. A skill that `produces: [lint_results, type_errors]` is two skills pretending to be one. The reference linter flags multi-output skills as violations — the YAML schema does not prevent multiple entries, but the tooling treats this as a design rule that enforces the Single Responsibility Principle. Each skill does one thing, and the linter makes violations visible. The constraint is intentionally asymmetric: agents aggregate outputs from multiple skills, so `agent.produces` is a list.

The `version` field is declarative metadata — the reference tooling does not enforce version compatibility or resolve version conflicts between skills. It exists to support manual governance and documentation, not automated dependency management.

The `memory` field declares what conversational context a skill expects: `short-term` (no context beyond the current invocation), `conversation` (prior turns in the same session), or `long-term` (persistent state across sessions). Memory is a hint to the generator — it affects prompt structure in targets that support multi-turn context (e.g., Claude Code's `memory` frontmatter field) but has no effect on targets that do not.

Skills are pure interfaces. They declare their I/O contract (`consumes`/`produces`) but have no knowledge of which other skills provide their inputs. Data flow is not declared in the skill — it emerges from the composition declared in the agent. This enforces the Law of Demeter [5]: a skill only accesses data it explicitly declares in its `consumes` contract. The linter validates that every skill's inputs are satisfied by either another skill's outputs or the agent's own `consumes`.

#### I/O Type Catalog

The `consumes` and `produces` fields use **semantic type names** — free-form strings, not a closed enum. These names are human-readable labels, not typed schemas — there is no type system that validates the shape or structure of the data flowing between skills. Two skills wire together when their names match; the linter validates that every input is satisfied but cannot prevent mismatches where two skills use the same name with different expectations. The trade-off is deliberate: semantic names are easy to author and understand, while typed schemas would add authoring cost for a benefit that matters most at scale. Teams can define domain-specific types (`security_audit`, `api_docs`, `migration_plan`) alongside the built-in ones.

Types fall into three categories based on their role in the data flow:

- **Input** — provided by the outside world (the agent's `consumes`). These are the raw materials the agent receives.
- **Intermediate** — produced by one skill and consumed by another within the same agent. These are internal data products.
- **Output** — the agent's final deliverables (the agent's `produces`). These are what the caller receives.

The reference implementation ships with the following types (see Appendix A for detailed descriptions and examples):

| Type name | Category | Description |
|-----------|----------|-------------|
| `git_diff` | Input | Unified diff of pending changes |
| `source_code` | Input | Raw source files for analysis |
| `file_tree` | Input | Repository file path listing |
| `lint_results` | Intermediate | Linter diagnostics with locations |
| `type_errors` | Intermediate | Type-checker diagnostics |
| `test_results` | Intermediate | Test runner pass/fail output |
| `coverage_report` | Intermediate | Line and branch coverage data |
| `review_comments` | Output | Actionable PR review comments |
| `risk_score` | Output | Numeric risk score (0–10) with justification |
| `approval_gate` | Intermediate | Human decision: approve, reject, or request changes with questions |
| `human_feedback` | Input | Free-form human input injected mid-pipeline |

These types are conventions, not constraints. A team building a security-focused agent might define `vulnerability_scan`, `dependency_audit`, or `compliance_report`. A documentation agent might use `api_schema` and `generated_docs`. The linter validates the wiring — that every `consumes` is satisfied by a `produces` — regardless of type names.

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
| `adaptive` | Dynamic execution based on intermediate results (e.g., skip `coverage-reporter` if `tdd-runner` reports zero test files). Defined for completeness; not yet implemented in the reference tooling. | Complex decision trees |

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

When generating output for a target, the abstract names are mapped to framework-specific tool APIs. The full catalog of recognized abstract tools:

| Abstract tool | Aliases | Claude Code | GitHub Copilot |
|---------------|---------|-------------|----------------|
| `read_file` | `read` | `Read` | `read` |
| `write_file` | `write` | `Write` | `edit` |
| `edit_file` | `edit` | `Edit` | `edit` |
| `grep` | — | `Grep` | `search` |
| `search` | `find`, `glob` | `Glob` | `search` |
| `bash` | `shell`, `exec`, `terminal` | `Bash` | `execute` |
| `web_fetch` | `http`, `fetch` | `WebFetch` | `web` |
| `web_search` | — | `WebSearch` | `web` |
| `todo` | — | `TodoWrite` | `todo` |
| `task` | — | `Task` | `agent` |

Aliases are normalized at generation time — `read_file` and `read` both resolve to the same framework tool. When a target does not support a given tool (e.g., Cursor has no `task` equivalent), the tool is silently omitted from the generated output. The skill remains portable; the generated artifact adapts.

The generation step also **infers tools from the security facet**. A skill with `filesystem: read-only` automatically gains `read_file`, `grep`, and `search` in the generated output, even if the author did not list them explicitly. A skill with `network: full` gains `web_fetch` and `web_search`. This inference ensures that the generated agent has the minimum tools required by its declared security posture.

The skill author writes `tools: [read_file, grep]`. The generation step translates. Switching from Claude Code to GitHub Copilot requires zero changes to any skill or agent specification — only a different build target.

Tool name mapping is the first layer of portability — and the easiest. The harder layers — different tool capabilities, context window sizes, and model behaviors — remain framework-specific concerns. The generation step can adapt to these differences (e.g., generating shorter prompts for smaller context windows), but the specification itself does not encode them. Portability at the specification level is solved; portability at the execution level is an ongoing challenge delegated to the generator.

### 3.2 LLM Independence

The format describes **behavior**, not **prompt instructions**. A skill's `strategy.steps` are declarative:

```yaml
steps:
  - read the git diff
  - cross-reference with test results
  - identify risky changes
  - write actionable review comments
```

These steps describe *what* the skill does, not *how* a specific model should interpret them. The same `review-commenter` skill works whether the underlying LLM is Claude, GPT, Gemini, or a future model. The generation step may adapt the generated prompt to a model's strengths, but the skill specification itself remains unchanged.

This separation means that model upgrades — or model switches — do not require rewriting skill definitions. The behavioral contract is stable; the execution adapts.

### 3.3 Write Once, Deploy Anywhere

The same YAML specifications produce different outputs depending on the target. The four framework families described in section 1.2 each have different implications for the generation step: declarative targets get high-fidelity output preserving all facets; rules-based targets get conditional rules with activation patterns; config-based and programmatic targets get best-effort mappings with reduced fidelity.

| Target | Output directory | Skill format | Agent format | Extra artifacts | Family | Status |
|--------|-----------------|--------------|--------------|-----------------|--------|--------|
| Claude Code [1] | `.claude/` | `skills/<name>/SKILL.md` | `agents/<name>.md` | — | Declarative | **Available** |
| GitHub Copilot [2] | `.github/` | `skills/<name>/SKILL.md` | `agents/<name>.agent.md` | `copilot-instructions.md` | Declarative | **Available** |
| Gemini CLI | `.gemini/` | `skills/<name>/SKILL.md` | `agents/<name>.md` | `GEMINI.md` | Declarative | Planned |
| OpenCode | `.opencode/` | `skills/<name>/SKILL.md` | `agents/<name>.md` | `opencode.json` | Declarative | Planned |
| Amazon Q Developer | `.amazonq/` | `rules/<name>.md` | `cli-agents/<name>.json` | — | Declarative | Possible |
| Zed AI | `.rules` | Rules library `.md` | — | — | Declarative | Possible |
| Continue | `.continue/` | Prompt files `.md` | `config.yaml` agents | — | Declarative | Possible |
| Cursor | `.cursor/rules/` | `<name>.mdc` | — | — | Rules-based | Possible |
| Windsurf | `.windsurf/rules/` | `<name>.md` | — | — | Rules-based | Possible |
| Cline | `.clinerules/` | `<name>.md` | — | — | Rules-based | Possible |
| Roo Code | `.roo/rules/` | `<name>.md` | Mode config | — | Rules-based | Possible |
| Augment Code | `.augment/rules/` | `<name>.md` | — | — | Rules-based | Possible |
| Codex (OpenAI) | — | — | Agents SDK (Python) | `AGENTS.md` | Programmatic | Possible |
| Kiro (AWS) | `.kiro/` | YAML specs | YAML config | Steering files | Config-based | Possible |
| Aider | — | — | — | `.aider.conf.yml` | Config-based | Possible |

New targets can be added without modifying any existing specification — the format is open for extension.

Generated skill files use a deliberate section ordering motivated by LLM attention research. Liu et al. [7] demonstrated that language models perform best when critical information is placed at the beginning or end of the input context — the "lost in the middle" effect. Peysakhovich & Lerer [8] confirmed that primacy and recency biases are widespread across LLM architectures. These studies focus on information retrieval rather than instruction following, but the principle is plausible for agent prompts. Motivated by these findings, the generated output places:

1. **Guardrails** first — primacy bias ensures constraints are remembered
2. **Context, Strategy** in the middle — the execution plan
3. **Security** last — recency bias keeps access controls top of mind

---

## 4. Formal Properties

The preceding sections define the Skill Behavior Model informally. This section gives it semi-formal grounding — precise definitions, stated properties, and an algebraic characterization of composition. The goal is not full formalization but sufficient rigor to reason about structural guarantees.

### 4.1 Definitions

A **skill** is a triple S = (C, P, F) where:
- C ⊆ T is a finite set of consumed type names (the skill's inputs)
- P ∈ T is a single produced type name (the skill's output). Note: the YAML schema permits `produces` as a list, so a skill *file* can declare multiple outputs. The formal model assumes P is a singleton; the linter enforces this constraint (Single Responsibility, section 4.2). Skills violating SRP are syntactically valid YAML but are rejected by the linter before entering the formal model.
- F is a record of the remaining facets: strategy, guardrails, observability, security, negotiation (the Context facet's I/O contract is captured by C and P)

An **agent** is a tuple A = (S₁, ..., Sₙ, Cₐ, Pₐ, σ) where:
- S₁, ..., Sₙ are the agent's skills (n ≥ 1)
- Cₐ ⊆ T is the agent's consumed types (external inputs)
- Pₐ ⊆ {P(S₁), ..., P(Sₙ)} is the agent's produced types (final outputs) — every agent output must be produced by some skill
- σ ∈ {sequential, parallel, parallel-then-merge, adaptive} is the orchestration strategy

**Edge cases.** A *source skill* has C = ∅: it requires no input and may generate content from its prompt alone (e.g., a skill that produces a boilerplate template). Source skills have no incoming edges in G(A) other than from ⊤ₐ. An agent with n = 0 is degenerate (no skills, no outputs) and is rejected by validation as vacuous. An agent with Cₐ = ∅ (no external inputs) is valid if all skills' inputs are satisfied internally — all skills form a closed system rooted in source skills. An agent with Pₐ = ∅ (no declared outputs) is degenerate — it performs computation but produces no deliverable — and should be flagged by the linter as a warning.

The **data flow graph** G(A) is a directed graph where:
- Nodes are the skills S₁, ..., Sₙ, a virtual source node ⊤ₐ representing the agent's external inputs Cₐ, and a virtual sink node ⊥ₐ representing the agent's final outputs Pₐ
- An edge (Sᵢ, Sⱼ) exists when P(Sᵢ) ∈ C(Sⱼ) — skill i produces a type that skill j consumes
- An edge (⊤ₐ, Sⱼ) exists when some t ∈ C(Sⱼ) is in Cₐ — the skill consumes an external input
- An edge (Sᵢ, ⊥ₐ) exists when P(Sᵢ) ∈ Pₐ — the skill's output is an agent-level output

T is the universe of type names — free-form strings like `git_diff`, `lint_results`, `review_comments`. Types are nominal: two types match if and only if their names are identical strings. There is no structural subtyping.

### 4.2 Structural Properties

The Skill Behavior Model relies on three categories of structural properties.

**Well-formedness invariants** (checked by tooling — an agent that violates any of these is rejected before entering the formal model):

| Invariant | Formal statement | Verified by |
|-----------|-----------------|-------------|
| **Acyclicity** | G(A) is a directed acyclic graph | Diagnostic checker — cycle detection |
| **Contract completeness** | ∀ Sᵢ ∈ A, ∀ t ∈ C(Sᵢ): ∃ Sⱼ ∈ A where P(Sⱼ) = t, or t ∈ Cₐ | Linter — unmet dependencies |
| **Single responsibility** | P(Sᵢ) is a singleton for every skill (the YAML schema permits lists; the linter rejects multi-output skills before they enter the formal model, where P ∈ T is definitionally a singleton) | Lint rule SRP |
| **Producer uniqueness** | ∀ t ∈ T: \|{Sᵢ ∈ A : P(Sᵢ) = t}\| ≤ 1 — each type has at most one producer | Linter — duplicate producer detection |
| **Output coverage** | Pₐ ⊆ {P(S₁), ..., P(Sₙ)} — every agent output is produced by some skill | Linter — orphan output detection |
| **No dead inputs** | ∀ t ∈ Cₐ: ∃ Sᵢ ∈ A where t ∈ C(Sᵢ) — every declared external input is consumed by at least one skill | Linter — unused input warning |
| **No dead skills** | ∀ Sᵢ ∈ A: P(Sᵢ) ∈ Pₐ or ∃ Sⱼ ∈ A where P(Sᵢ) ∈ C(Sⱼ) — every skill's output is either an agent output or consumed by another skill | Linter — dead skill warning |

These are *preconditions*, not theorems: the tooling verifies them and rejects non-conforming agents (or warns for the last two). Their role is analogous to typing judgments — the type checker verifies them, but they are not "proved" from the definitions.

**Derived properties** (provable from the definitions and the well-formedness invariants — see Propositions 7–8 in section 4.4):

| Property | Statement | Proved in |
|----------|-----------|-----------|
| **Structural additivity** | Adding a skill S' under conditions H1–H3 preserves the incoming edges and well-formedness of existing skills | Proposition 7 |
| **Monotonicity** | Adding any skill preserves existing dependency satisfaction (unconditionally) | Corollary of Proposition 7 |

**Implementation contract** (property of the build tooling, not of the formal model):

| Property | Statement | Verified by |
|----------|-----------|-------------|
| **Build determinism** | The build function B is empirically deterministic: same specification → same artifacts. Depends on absence of non-deterministic sources in the generator code. | Implementation discipline + empirical testing |

Producer uniqueness ensures deterministic data flow: if two skills produced the same type, consumers would face ambiguous resolution. The derived properties (additivity, monotonicity) follow from the graph structure; their proofs appear in section 4.4.

### 4.3 The I/O Contract as a Type System

The `consumes`/`produces` declarations form a lightweight type system. Each skill's signature can be written as:

```
S : C₁ × C₂ × ... × Cₙ → P
```

where C₁, ..., Cₙ are the consumed types and P is the produced type. The linter acts as a type checker: it verifies that every input type is provided by some source (another skill's output or the agent's external input). In the terminology of type theory:

**Well-typed agents don't have structural defects.** This is the analogue of Milner's "well-typed programs don't go wrong" [20] — if the linter accepts an agent, a specific class of defects (missing data, circular dependencies, responsibility overload) is eliminated before any LLM executes.

The analogy has deliberate limits:

| Type system property | Skill Behavior Model | Traditional type systems |
|---------------------|---------------------|------------------------|
| Type identity | Nominal (string equality) | Nominal or structural |
| Subtyping | None | Subtype hierarchies |
| Parametric types | None | Generics, type variables |
| Type inference | None needed — types are declared | Hindley-Milner, bidirectional |
| Soundness | Structural only (section 4.5) | Semantic (progress + preservation) |

The type system is intentionally shallow. Types are names, not schemas — `review_comments` is a label, not a structural description of what review comments contain. Two skills can use the same type name with different expectations, and the linter cannot detect this mismatch. This is the trade-off: shallow types are trivial to author and compose, while deep types (JSON Schema, Protocol Buffers) would add authoring cost for a benefit that matters most at scale.

### 4.4 Composition Algebra

Skills compose into agents through a directed acyclic graph (DAG). This section formalizes the composition operations and proves properties of the resulting structure.

The data flow graph G(A) (defined in section 4.1) is the foundation of the composition algebra.

Two composition operations build the DAG:

**Sequential composition.** S₂ ∘ S₁ connects the output of S₁ to an input of S₂. This is defined when P(S₁) ∈ C(S₂) and creates an edge (S₁, S₂) in the DAG. Note that S₂ may have additional inputs beyond P(S₁) — these are satisfied by other skills or by the agent's external inputs.

**Parallel composition.** S₁ ⊗ S₂ places two skills with no data dependency between them. This is defined when P(S₁) ∉ C(S₂) and P(S₂) ∉ C(S₁). Both skills draw from the agent's available inputs independently. The precondition is symmetric, so ⊗ is commutative: S₁ ⊗ S₂ = S₂ ⊗ S₁. It is also associative: if S₁ ⊗ S₂ and (S₁ ⊗ S₂) ⊗ S₃ are both defined, pairwise independence guarantees S₁ ⊗ (S₂ ⊗ S₃) is also defined and produces the same DAG.

**Important caveat.** The operators ∘ and ⊗ are *notational shorthands* for edge/independence relations in the DAG, not algebraic operations closed on skills. The "result" of S₂ ∘ S₁ is not a new skill but a DAG fragment. The notation is useful for describing DAG shapes concisely, but does not constitute a closed algebra with carrier set and internal operations.

**Edge count bound.** By producer uniqueness, each consumed type resolves to at most one producer. Therefore the number of inter-skill edges satisfies |E| ≤ ∑ᵢ |C(Sᵢ)|. This bound is used implicitly in the complexity analysis of Proposition 1.

The four orchestration strategies (section 2.3) correspond to DAG shapes:

| Orchestration | DAG shape | Notation |
|---------------|-----------|----------|
| `sequential` | Linear chain | S₃ ∘ S₂ ∘ S₁ |
| `parallel` | Independent nodes | S₁ ⊗ S₂ ⊗ S₃ |
| `parallel-then-merge` | Fan-in | Sₘ ∘ (S₁ ⊗ S₂ ⊗ S₃), where Sₘ consumes the outputs of S₁, S₂, S₃ |
| `adaptive` | Conditional branching | Runtime-selected among {S₁, S₂} based on input content |

Note: the `sequential` notation S₃ ∘ S₂ ∘ S₁ denotes a strict chain where P(S₁) ∈ C(S₂) and P(S₂) ∈ C(S₃). In practice, the `sequential` orchestration allows accumulation — S₃ may also consume P(S₁) directly. The strict chain is a special case.

**Proposition 1 (Decidability).** Given an agent specification A, determining whether G(A) is well-formed (acyclic, contract-complete, single-responsibility, producer-unique) is decidable in O(|S| + |E|) time, where |S| is the number of skills and |E| is the number of data flow edges.

*Proof.* Acyclicity is detected by topological sort (O(|S| + |E|)). Contract completeness is verified by building a hash map from produced types to skills (O(|S|)), then checking each consumed type against the map (O(∑|C(Sᵢ)|) = O(|E|)). Single responsibility and producer uniqueness are checked per skill (O(|S|)). □

**Proposition 2 (Compatible substitution).** Let A be a well-formed agent containing skill S. Let S' be a skill with C(S') ⊆ C(S) and P(S') = P(S). Then A' = A[S ↦ S'] is well-formed.

*Proof.* We verify each well-formedness property.

*Acyclicity.* Let E(A) denote the edge set of G(A). An edge (Sᵢ, Sⱼ) exists iff P(Sᵢ) ∈ C(Sⱼ). In G(A'):
- Outgoing edges from S': {(S', Sⱼ) : P(S') ∈ C(Sⱼ)} = {(S, Sⱼ) : P(S) ∈ C(Sⱼ)} (since P(S') = P(S)). Identical to S's outgoing edges.
- Incoming edges to S': {(Sᵢ, S') : P(Sᵢ) ∈ C(S')} ⊆ {(Sᵢ, S) : P(Sᵢ) ∈ C(S)} (since C(S') ⊆ C(S)). A subset of S's incoming edges.
- All other edges are unchanged.

Therefore E(A') ⊆ E(A). A subgraph of a DAG is acyclic. ✓

*Contract completeness.* We must show: ∀ Sⱼ ∈ A', ∀ t ∈ C(Sⱼ): t is produced by some skill in A' or t ∈ Cₐ.
- Case Sⱼ ≠ S': C(Sⱼ) is unchanged. If t was produced by S in A, it is now produced by S' in A' (since P(S') = P(S)). If t was produced by some Sₖ ≠ S, the producer is unchanged. ✓
- Case Sⱼ = S': For every t ∈ C(S'), we have t ∈ C(S) (since C(S') ⊆ C(S)). The type t was satisfied in A — its producer is unchanged in A'. ✓

*Single responsibility.* P(S') = P(S), which is a singleton because A is well-formed. ✓

*Producer uniqueness.* S was the unique producer of P(S) in A. In A', S' produces P(S') = P(S). No other skill changed. Therefore S' is the unique producer of that type. ✓ □

*Corollary (Contravariant substitutability).* The substitution condition — C(S') ⊆ C(S) for inputs, P(S') = P(S) for output — is *contravariant* in inputs and *invariant* in output. A skill that requires *fewer* inputs is always substitutable. A skill that requires *more* inputs may break contract completeness (the new inputs may have no producer). This matches the Liskov Substitution Principle [4] and directly justifies the semantic versioning scheme in section 6.2.

**Proposition 3 (Parallelizability).** In a well-formed agent A, skills S₁ and S₂ are *independent* if there is no directed path between them in G(A). Independent skills can be scheduled in any relative order or concurrently: every such scheduling respects the data flow constraints of G(A).

*Proof.* Independence means S₁ and S₂ are *incomparable* in the partial order induced by G(A) — neither is an ancestor of the other. By the order extension principle (Szpilrajn's theorem), any partial order can be extended to a total order. Since S₁ and S₂ are incomparable, both extensions (S₁ < S₂ and S₂ < S₁) are consistent with the partial order. Therefore, valid topological orderings (i.e., valid schedules) exist with S₁ before S₂, and valid orderings with S₂ before S₁.

Data flow correctness is scheduling-independent: each consumed type t requires ∃ Sⱼ : P(Sⱼ) = t — a structural property of the DAG, not a property of any particular execution order. At runtime, a scheduler need only ensure that a skill executes after all of its predecessors in G(A) have completed; independent skills have no such ordering constraint between them. ✓

*Consequence.* The set of maximal independent groups is computable by topological layer decomposition: skills at the same layer of the DAG (same longest-path distance from any source) are pairwise independent. This directly yields the `parallel` orchestration strategy. □

**The build mapping.** A build target defines a deterministic mapping B from agent specifications to generated artifacts (see Proposition 9, cross-target structural isomorphism):
- B maps each type t ∈ T to its framework-specific representation
- B maps each skill S to a generated artifact (a prompt file, a rule file, etc.)
- B maps the DAG structure to orchestration logic in the generated agent file

**A note on categorical structure.** The notation above (∘ for sequential, ⊗ for parallel) suggests a categorical reading — skills as morphisms, types as objects, the build mapping as a functor. This analogy is useful vocabulary but requires care. Multi-input skills require a *multicategory* rather than an ordinary category, and the morphisms are LLM transformations (non-deterministic) rather than pure functions. The propositions above are stated and proved directly on the DAG structure rather than relying on categorical machinery — the proofs are self-contained.

**Limitations.**

1. **Specification, not execution.** The algebra describes how skills *wire together*, not how they *behave*. Two invocations of the same skill may produce different outputs. The composition properties hold at the structural level (data flows correctly) but not at the behavioral level (outputs may vary).

2. **Nominal type matching.** Types are strings, not schemas. Two skills using `review_comments` with different expectations compose structurally but may fail semantically. The algebra verifies wiring, not meaning.

3. **No shared state.** The model assumes skills communicate only through their declared I/O contracts. If two skills read the same file (a shared implicit dependency), their outputs may conflict. The DAG captures declared dependencies, not implicit ones.

**Proposition 7 (Structural additivity).** Let A be a well-formed agent and S' = (C', P', F') a skill satisfying:
- (H1) P' ∉ C(Sᵢ) for all Sᵢ ∈ A — no existing skill consumes S''s output
- (H2) P' ≠ P(Sⱼ) for all Sⱼ ∈ A — no producer uniqueness violation
- (H3) P' ∉ C' — no self-loop

If the dependencies of S' are satisfied in A' = A ∪ {S'} (i.e., ∀ t ∈ C': ∃ Sⱼ ∈ A with P(Sⱼ) = t, or t ∈ Cₐ), then:
1. For every Sᵢ ∈ A, the set of *incoming* edges to Sᵢ in G(A') is identical to that in G(A).
2. A' is well-formed.

*Proof.*

*Part 1 (Incoming edge preservation).* An incoming edge (Sₓ, Sᵢ) exists iff P(Sₓ) ∈ C(Sᵢ). The only new node is S'. The edge (S', Sᵢ) exists iff P' ∈ C(Sᵢ). By H1, P' ∉ C(Sᵢ) for all existing Sᵢ. Therefore no new incoming edge is created for any existing skill. ✓

Note: *outgoing* edges from existing skills may increase — if P(Sₖ) ∈ C', the edge (Sₖ, S') is created. This does not affect what data Sₖ receives, only who consumes its output.

*Part 2 (Well-formedness of A').*

*Acyclicity.* S' has no outgoing edges to existing nodes (by H1, P' ∉ C(Sᵢ)). Therefore every path through S' terminates at S'. No path from S' can reach any node of A, so no cycle through both S' and nodes of A can exist. By H3, P' ∉ C', so S' has no self-loop. Therefore G(A') is acyclic. ✓

*Contract completeness.* For existing Sᵢ: their inputs are unchanged and their producers are unchanged (no skill was removed). For S': the hypothesis guarantees every t ∈ C' has a producer in A' or t ∈ Cₐ. ✓

*Single responsibility.* P' is a singleton (S' passed the linter). Existing skills are unchanged. ✓

*Producer uniqueness.* By H2, P' ≠ P(Sⱼ) for all existing Sⱼ. So S' produces a type that no other skill produces. ✓

*Output coverage.* Pₐ is unchanged (adding a skill does not modify the agent's declared outputs). Pₐ ⊆ {P(S₁),...,P(Sₙ)} ⊆ {P(S₁),...,P(Sₙ), P'} = {P(S) : S ∈ A'}. ✓ □

*Corollary (Stability of Pₐ under substitution).* If A is well-formed and A' = A[S ↦ S'] satisfies the conditions of Proposition 2 (P(S') = P(S)), then Pₐ(A') = Pₐ(A) — the agent's output set is unchanged by compatible substitution.

**Corollary 7.1 (Monotonicity).** Let A be a well-formed agent and S' any skill. Let A' = A ∪ {S'}. Then every dependency satisfied in A remains satisfied in A': for every Sᵢ ∈ A and every t ∈ C(Sᵢ), if t was produced by some Sⱼ ∈ A or t ∈ Cₐ, the same holds in A'.

*Proof.* A ⊆ A', so every skill Sⱼ ∈ A is also in A'. Cₐ is unchanged. Therefore every producer of t in A is still present in A'. □

Note: monotonicity is *unconditional* — it does not require H1, H2, or H3. It guarantees that existing *dependencies* are preserved, not that A' is well-formed. The new skill may introduce cycles (violating acyclicity) or duplicate producers (violating producer uniqueness).

**Proposition 8 (Existence of a valid schedule).** If A is well-formed, there exists at least one total ordering of the skills that respects all data flow constraints — i.e., every skill is scheduled after all of its predecessors in G(A).

*Proof.* G(A) is a DAG (acyclicity). Every finite DAG admits at least one topological ordering (Kahn's algorithm or DFS-based topological sort). A topological ordering is a valid schedule: if (Sᵢ, Sⱼ) ∈ E(A), then Sᵢ appears before Sⱼ, so Sⱼ's input from Sᵢ is available when Sⱼ executes. □

**Proposition 9 (Cross-target structural isomorphism).** Let T = {t₁, t₂, ...} be the set of build targets (e.g., Claude Code, GitHub Copilot). For each target t, the build function Bₜ maps a well-formed agent specification A to a set of generated artifacts. The structural content of the artifacts is invariant across targets: for any two targets t₁, t₂ and any well-formed specification A, the generated artifacts Bₜ₁(A) and Bₜ₂(A) encode the same data flow graph, the same I/O contracts, and the same skill ordering.

**Definition (Structural extraction).** For any set of generated artifacts Art, the *structural extraction function* φ(Art) = (N, E, C_ext, P_ext) extracts:
- N = the set of skill names appearing in the artifacts
- E = {(sᵢ, sⱼ) : sⱼ's artifact references a type produced by sᵢ} — the data flow edges
- C_ext = the set of types declared as external inputs
- P_ext = the set of types declared as agent outputs

φ is well-defined because every target generator embeds skill names, consumes/produces declarations, and execution order in a parseable format (Markdown sections with structured headers). The extraction is syntactic, not semantic — it reads declared types, not runtime behavior.

Formally: ∀ t₁, t₂ ∈ T, ∀ A ∈ F_linter: φ(Bₜ₁(A)) = φ(Bₜ₂(A))

*Proof.* Each target generator Bₜ is a template function:

```
Bₜ(A) = { templateₜ(Sᵢ) : Sᵢ ∈ A } ∪ { agent_templateₜ(A) }
```

The template varies the *presentation* (file paths, Markdown structure, framework-specific syntax) but interpolates the same *structural data*: skill name, C(Sᵢ), P(Sᵢ), tools, guardrails, and the DAG-derived execution order. No template introduces, removes, or reorders skills. No template modifies the consumes/produces declarations.

Therefore φ extracts the same (N, E, C_ext, P_ext) regardless of target: φ ∘ Bₜ₁ = φ ∘ Bₜ₂. □

*Epistemic status.* This proof is an argument by inspection of the template implementations, not a deduction from axioms. To make it fully formal, one would need to axiomatize the class of "structural-preserving templates" and prove that all implemented generators belong to this class. In practice, the property is verifiable by testing: generate for both targets and compare φ-extractions.

*Consequence.* Cross-target isomorphism justifies the portability claim (section 3): migrating from one framework to another is a change of presentation, not a change of behavior. It also means that well-formedness invariants verified on the specification carry over to all generated artifacts — the linter validates once, and the guarantee holds for every target.

*Limitations.* The isomorphism is *structural*, not *behavioral*. Two targets may interpret the same structural content differently at runtime. For example, Claude Code may enforce `filesystem: read-only` via hooks while GitHub Copilot may ignore it. The structural content is preserved; its enforcement depends on the target framework.

**Build determinism** (implementation contract). The build function is empirically deterministic: same specification → same artifacts across executions. This is an implementation property, not a formal theorem — it depends on the absence of non-deterministic sources (timestamps, RNG, unsorted map iteration) in the generator code. The implementation guards against non-determinism by sorting all map iterations and using a fixed topological sort algorithm with lexicographic tie-breaking on skill names.

**Proposition 10 (Reachability).** In a well-formed agent A, every skill Sᵢ is reachable from the virtual source ⊤ₐ in G(A), and the virtual sink ⊥ₐ is reachable from every skill Sᵢ.

*Proof.*

*Forward reachability (⊤ₐ ⇝ Sᵢ).* Let Sᵢ ∈ A be arbitrary. If C(Sᵢ) ∩ Cₐ ≠ ∅, then an edge (⊤ₐ, Sᵢ) exists and Sᵢ is directly reachable. Otherwise, every t ∈ C(Sᵢ) is produced by some Sⱼ ∈ A (contract completeness), giving an edge (Sⱼ, Sᵢ). Apply the same argument to Sⱼ. This backward walk terminates because G(A) is acyclic — the walk visits strictly earlier nodes in any topological order. It must reach a skill Sₖ with C(Sₖ) ∩ Cₐ ≠ ∅ or C(Sₖ) = ∅ (a source skill, reachable from ⊤ₐ by convention). ✓

*Backward reachability (Sᵢ ⇝ ⊥ₐ).* By the no-dead-skills invariant, every skill Sᵢ satisfies: (a) P(Sᵢ) ∈ Pₐ, giving an edge (Sᵢ, ⊥ₐ) directly; or (b) ∃ Sⱼ with P(Sᵢ) ∈ C(Sⱼ), giving an edge (Sᵢ, Sⱼ). In case (b), apply the same argument to Sⱼ. This forward walk terminates by acyclicity and must reach a skill Sₘ with P(Sₘ) ∈ Pₐ. Therefore Sᵢ → ... → Sₘ → ⊥ₐ. ✓ □

*Consequence.* Reachability means G(A) is connected in the strong sense: every skill lies on at least one directed path from ⊤ₐ to ⊥ₐ. There are no isolated subgraphs and no dangling branches — the structural analogue of "no dead code."

**Corollary 3.1 (Layer decomposition).** Let A be a well-formed agent. Define the *layer* of a skill Sᵢ as:

```
layer(Sᵢ) = length of the longest directed path from ⊤ₐ to Sᵢ in G(A)
```

The skills of A admit a unique partition into layers L₀, L₁, ..., Lₖ where Lⱼ = {Sᵢ : layer(Sᵢ) = j}. Two skills in the same layer are pairwise independent (no directed path between them).

*Proof.* The partition exists and is unique because `layer` is a well-defined function — Proposition 10 guarantees every skill is reachable from ⊤ₐ, so the longest-path distance is finite; acyclicity bounds it by n.

Suppose for contradiction that Sₐ, Sᵦ ∈ Lⱼ and there is a directed path Sₐ → ... → Sᵦ of length ≥ 1. Let π be a longest path from ⊤ₐ to Sₐ (of length j). Then π extended by Sₐ → ... → Sᵦ is a path from ⊤ₐ to Sᵦ of length ≥ j + 1 > j, contradicting layer(Sᵦ) = j.

The decomposition is computable in O(n + |E|) time by topological traversal. □

*Relationship to P3.* Proposition 3 states that independent skills can be scheduled concurrently. Corollary 3.1 strengthens this: the maximal independent groups form a canonical layered structure. The `parallel` and `parallel-then-merge` orchestration strategies correspond directly to this layer decomposition — each layer is a parallel batch, and layers execute sequentially.

**Corollary 3.2 (Dilworth width and maximum parallelism).** The *width* of G(A) — the size of the largest antichain — equals max₀≤j≤k |Lⱼ|. By Dilworth's theorem, this also equals the minimum number of chains (sequential paths) needed to cover all skills.

*Proof.* Each layer Lⱼ is an antichain (Corollary 3.1). For any antichain Q, if it contained skills from different layers Lᵢ and Lⱼ with i < j, the presence of a path between them would contradict independence. Therefore every antichain is contained within a single layer, and the largest antichain is the largest layer: width = max |Lⱼ|. □

*Consequence.* The width is the **maximum degree of parallelism** — the maximum number of skills that can execute simultaneously. An agent with width 1 is inherently sequential; an agent with width n has n skills that can run in a single parallel batch at the widest point.

**Proposition 11 (Skill fusion).** Let A be a well-formed agent containing skills S₁ and S₂ such that:
- (F1) P(S₁) ∈ C(S₂) — S₂ consumes S₁'s output
- (F2) P(S₁) ∉ C(Sⱼ) for all Sⱼ ∈ A with Sⱼ ≠ S₂ — no other skill consumes S₁'s output
- (F3) P(S₁) ∉ Pₐ — S₁'s output is not an agent-level output

Then the *fused skill* S₁₂ = (C(S₁) ∪ (C(S₂) \ {P(S₁)}), P(S₂), F₁₂) can replace S₁ and S₂ in A, yielding A' = (A \ {S₁, S₂}) ∪ {S₁₂}, and A' is well-formed.

*Proof.* We verify each well-formedness invariant.

*Acyclicity.* G(A') is obtained from G(A) by contracting the edge (S₁, S₂). By F2 and F3, S₁ had no outgoing edges except (S₁, S₂). Edge contraction in a DAG preserves acyclicity: any cycle in the contracted graph would imply a cycle in the original. ✓

*Contract completeness.* For S₁₂: every t ∈ C(S₁₂) = C(S₁) ∪ (C(S₂) \ {P(S₁)}) was consumed by S₁ or S₂ in A and had a producer there. That producer remains in A'. For other skills Sⱼ: if Sⱼ consumed P(S₂), S₁₂ still produces it. No skill consumed P(S₁) except S₂ (by F2). ✓

*Single responsibility.* S₁₂ produces P(S₂), a singleton. ✓

*Producer uniqueness.* S₁₂ produces P(S₂), the same type S₂ produced. P(S₁) is now internal — no longer a produced type in A'. ✓

*Output coverage.* P(S₁) ∉ Pₐ (by F3), so removing S₁ as a producer doesn't affect coverage. S₁₂ still produces P(S₂). ✓

*No dead skills.* S₁₂ produces P(S₂). Since S₂ was not dead in A, either P(S₂) ∈ Pₐ or some skill consumed P(S₂). The same holds for S₁₂. ✓

*No dead inputs.* C(S₁₂) ⊇ C(S₁), so external inputs consumed by S₁ are preserved. C(S₂) \ {P(S₁)} ⊆ C(S₁₂), so external inputs consumed by S₂ are preserved. ✓ □

*Consequence.* Fusion is the converse of Single Responsibility decomposition: if an intermediate type is consumed by exactly one downstream skill and is not an agent output, the two skills can be merged. This is the structural analogue of *inlining* a private function called from exactly one site.

*Limitation.* Fusion is structural — it merges I/O contracts but says nothing about merging behavioral facets. Combining two prompts may introduce responsibility overload or conflicting guardrails. Fusion preserves *well-formedness* but may degrade *design quality*. The `score` command can detect this regression.

**Proposition dependency table.** The table below summarizes which well-formedness invariants each proposition depends on:

| Proposition | Depends on |
|-------------|-----------|
| **P1** (Decidability) | Definitions only |
| **P2** (Compatible substitution) | Acyclicity, contract completeness, SRP, producer uniqueness |
| **P3** (Parallelizability) | Acyclicity, contract completeness |
| **P4** (Resolution completeness) | Contract completeness, producer uniqueness |
| **P5** (Output immutability) | Execution axiom (write-once) |
| **P6** (Acyclic resolution) | Acyclicity |
| **P7** (Structural additivity) | Acyclicity, contract completeness, SRP, producer uniqueness, output coverage (under H1–H3) |
| **P8** (Valid schedule existence) | Acyclicity |
| **P9** (Cross-target isomorphism) | Definitions only |
| **P10** (Reachability) | Acyclicity, contract completeness, no dead skills |
| **Cor. 3.1** (Layer decomposition) | Acyclicity, contract completeness, no dead skills (via P10) |
| **Cor. 3.2** (Dilworth width) | Acyclicity, contract completeness, no dead skills (via Cor. 3.1) |
| **P11** (Skill fusion) | Acyclicity, contract completeness, SRP, producer uniqueness, output coverage, no dead skills (under F1–F3) |

Acyclicity is the most-used invariant — it appears in every proposition except P1, P4, P5, and P9. Contract completeness is the second most-used, appearing whenever a proof traces data flow backward through the graph.

### 4.5 Linter Soundness

The linter is *sound* with respect to structural properties: when it reports no diagnostics, certain classes of defects are guaranteed absent. This claim is straightforward — it follows directly from the linter's construction — but stating it explicitly clarifies the boundary between what is verified and what is not.

**Structural soundness (by construction).** If the linter and diagnostic checker report no diagnostics for an agent A = (S₁, ..., Sₙ, Cₐ, Pₐ, σ), then all five well-formedness invariants of section 4.2 hold:

1. **I/O completeness.** For every skill Sᵢ ∈ A, for every type t ∈ C(Sᵢ), either there exists Sⱼ ∈ A such that P(Sⱼ) = t, or t ∈ Cₐ. Every skill's inputs are satisfied.

2. **Acyclicity.** The data flow graph G(A) contains no directed cycles. No skill transitively depends on its own output.

3. **Single responsibility.** For every skill Sᵢ ∈ A, Sᵢ produces exactly one type. No skill is a merged responsibility.

4. **Producer uniqueness.** For every type t produced in A, at most one skill produces it. Data flow resolution is unambiguous.

5. **Output coverage.** For every type t ∈ Pₐ, there exists Sᵢ ∈ A such that P(Sᵢ) = t. Every declared agent output is produced by some skill.

This holds because the linter explicitly checks each property and reports a diagnostic for any violation. The claim is analogous to stating that a type checker that verifies property P guarantees P when it succeeds — true by construction, but useful for delineating the guarantee boundary.

**What soundness does not guarantee.** The following properties are explicitly outside the scope of structural soundness:

| Property | Why it is not guaranteed |
|----------|------------------------|
| **Semantic correctness** | The type `review_comments` is a name, not a schema. A skill that produces `review_comments` containing random text is structurally sound. |
| **Output quality** | A sound skill can produce mediocre output. LLM non-determinism means the same skill may produce excellent results on one invocation and poor results on another. |
| **Termination** | The format declares `timeout: 5min` as a guardrail, but the linter does not enforce termination. A skill may run indefinitely if the framework does not enforce the declared timeout. |
| **Runtime security** | `filesystem: read-only` is a declaration, not a sandbox. The linter verifies the declaration exists; it cannot verify that the framework enforces it. |
| **Data flow injection** | A malicious or misconfigured skill can produce output that, when consumed by a downstream skill, causes unintended behavior. The format assumes trusted authorship. |

**Completeness.** The linter is *not* complete — there exist structural defects it does not detect. For example, a skill with an empty `approach` field is syntactically valid but semantically vacuous. Each new lint rule narrows the completeness gap. Completeness is an asymptotic goal, not a binary property: the linter catches more defects as its rule set grows.

The relationship between soundness and completeness mirrors that of traditional type systems: soundness is a hard guarantee (no false negatives for checked properties), while completeness is an engineering trade-off (some defects are not checked).

### 4.6 Data Flow Protocol

The composition algebra (section 4.4) specifies *what data flows where* — which skills produce and consume which types. This section specifies *how the data travels* — the concrete mechanism by which a skill's output reaches its consumers.

**The problem.** In current practice, data flow between skills is implicit: the LLM's context window carries everything. When `tdd-runner` produces `test_results` and `review-commenter` consumes it, the mechanism is that both run within the same conversation and the LLM retains prior output. This has three deficiencies:

1. **No isolation.** All skills share one context window. A skill can access data it does not declare in `consumes`, violating the explicit dependency model.
2. **No efficiency.** The full output of every skill persists in the context, even for consumers that need only a subset. For large outputs (coverage reports, full test logs), this wastes context budget.
3. **No cross-boundary flow.** Data cannot flow between agents or between invocations without ad-hoc file sharing.

**Definition (Skill URI).** A *skill URI* is a reference to a skill's output:

```
skill://<skill_name>/<type_name>[?invocation=<id>]
```

A second URI scheme handles external inputs (types in Cₐ):

```
input://<type_name>
```

Examples:
- `skill://tdd-runner/test_results` — the output of `tdd-runner` from the current invocation
- `skill://tdd-runner/test_results?invocation=abc123` — a specific historical invocation
- `skill://ci-reviewer/review_comments` — an agent-level output (for cross-agent flow)
- `input://git_diff` — an external input provided to the agent

When a skill declares `consumes: [test_results]`, the runtime resolves this to `skill://tdd-runner/test_results` — the skill within the same agent whose `produces` matches the consumed type. Resolution is deterministic because contract completeness (section 4.2) guarantees a producer exists, and producer uniqueness (section 4.2) guarantees it is the only one.

**Resolution strategies.** A *resolver* R maps a skill URI to the actual data. Three strategies are defined:

| Resolver | Mechanism | Trade-off |
|----------|-----------|-----------|
| **Context** | Data is present in the LLM's active context window | Zero-latency but no isolation. Default for conversational agents. |
| **File** | Output is written to `<base>/<skill_name>/<type_name>.md` and the URI resolves to a file read | Isolation and persistence, but requires filesystem access. Suited for CI pipelines. |
| **API** | Output is fetched via MCP tool call or HTTP endpoint | Full isolation and cross-agent support, but adds network latency. Suited for distributed systems. |

The build step selects the resolver based on the target framework. For Claude Code (conversational agent), the default is `context`. For CI environments, `file` is preferred. For multi-agent deployments (section 6.4), `api` enables cross-boundary resolution.

**YAML extension.** A `data_flow` field in the agent specification declares the protocol:

```yaml
agent: ci-reviewer
data_flow:
  resolver: file
  base_path: .forgent/outputs/
  format: markdown
```

Skills can also declare output hints:

```yaml
skill: coverage-reporter
produces: [coverage_report]
output:
  size_hint: large       # signals the resolver to prefer file over context
  retention: invocation  # discard after invocation ends
```

**Formal properties.**

**Proposition 4 (Resolution completeness and determinism).** If agent A is well-formed, then every skill URI generated by the runtime is resolvable, and the resolution is deterministic (each URI maps to exactly one producer).

*Proof.* A consumer Sⱼ with t ∈ C(Sⱼ) generates a URI. Two cases:
- If t is produced internally: the URI is `skill://Sᵢ/t` where P(Sᵢ) = t. Contract completeness guarantees such Sᵢ exists. Producer uniqueness guarantees it is the only one. Resolution is deterministic.
- If t ∈ Cₐ: the URI is `input://t`, which resolves to the agent's external input.

The resolver R maps each URI to actual data (via context, file, or API). Proposition 4 guarantees *structural* resolvability — the URI designates a valid, unambiguous producer. *Effective* resolvability (the data is accessible at runtime) depends on the resolver implementation, which is outside the formal model. □

Note: Proposition 4 combines contract completeness and producer uniqueness applied to the URI scheme. Its value is not a new theorem but the explicit guarantee that the URI abstraction layer preserves the structural properties of section 4.2.

**Execution model (axiom).** Each URI `skill://S/t` receives at most one write during an invocation.

This is the *minimal* axiom required for Proposition 5. It is weaker than requiring topological execution or single execution per skill — it permits lazy evaluation, retry strategies, and caching, as long as the resolver guarantees write-once semantics per URI. Under a strict topological execution model, this axiom is derivable: acyclicity ensures each skill is visited once, and producer uniqueness ensures no other skill writes to the same URI. The axiom is stated explicitly to support resolver implementations that may deviate from strict topological execution.

**Proposition 5 (Immutability).** Under the execution model axiom, a skill's output is immutable within a single invocation: once `skill://S/t` is written, its value does not change.

*Proof.* The axiom guarantees at most one write per URI. Therefore, once `skill://S/t` is written, no subsequent write occurs — the value is stable for the remainder of the invocation. □

*Consequence.* Immutability guarantees that parallel consumers of the same URI receive consistent data — no race conditions, no dirty reads.

**Proposition 6 (Acyclic resolution).** URI resolution cannot produce cycles: no skill URI's resolution transitively depends on the consumer's own output.

*Proof.* Resolving `skill://Sᵢ/t` for consumer Sⱼ traces the reverse of edge (Sᵢ, Sⱼ) in G(A). Sᵢ may in turn resolve URIs from its own producers, tracing further upstream along reverse edges. Since G(A) is acyclic, its reverse is also acyclic. The resolution chain is finite and cycle-free. □

Note: Proposition 6 is a direct consequence of the acyclicity invariant (section 4.2) applied to the reverse graph. Its value is making explicit that the URI resolution mechanism inherits the DAG's structural guarantee.

### 4.7 Petri Net Correspondence

The data flow graph G(A) admits a natural interpretation as a Petri net. This correspondence is not deep — it follows mechanically from the definitions — but it connects the Skill Behavior Model to a mature body of theory and tooling for concurrent systems.

**Definition (Petri net image).** Given an agent A = (S₁, ..., Sₙ, Cₐ, Pₐ, σ), define the Petri net N(A) = (P, Tr, F⁻, F⁺, M₀) as follows:

| Skill Model | Petri Net |
|---|---|
| Type t ∈ T | Place pₜ |
| Skill Sᵢ | Transition trᵢ |
| t ∈ C(Sᵢ) | Input arc (pₜ, trᵢ) ∈ F⁻ |
| P(Sᵢ) = t | Output arc (trᵢ, pₜ) ∈ F⁺ |
| t ∈ Cₐ | M₀(pₜ) = 1 |
| t ∉ Cₐ | M₀(pₜ) = 0 |

All arc weights are 1. The virtual source ⊤ₐ and sink ⊥ₐ are not represented explicitly — the initial marking and the identification of output places (those corresponding to Pₐ) serve their roles.

**Property transfer.** The well-formedness invariants impose strong structural constraints on N(A):

- **Producer uniqueness** guarantees each place has at most one incoming transition. Combined with SRP (each transition produces one output), the net is a *marked graph* — a restrictive and well-understood subclass.

- **Acyclicity** of G(A) makes N(A) an *acyclic* net. This rules out feedback and iteration, and makes reachability, liveness, and boundedness decidable in polynomial time.

- **Write-once semantics (P5)** — each place receives at most one token — makes N(A) *1-safe* (1-bounded). Since the net is also acyclic, this follows automatically.

- **Contract completeness** ensures every transition is *eventually enabled*: all input places receive tokens.

- **A valid schedule (P8)** corresponds to a *firing sequence* that fires every transition exactly once.

In short, N(A) is an acyclic, 1-safe marked graph in which every transition fires exactly once. This is a nearly trivial class of Petri net — and that is the point. The well-formedness invariants constrain the model tightly enough that the corresponding net is always well-behaved.

**Connection to linear logic.** The consume-and-produce discipline mirrors the resource semantics of linear logic. Types are linear propositions; a skill with C(Sᵢ) = {t₁, ..., tₖ} and P(Sᵢ) = t corresponds to the sequent t₁ ⊗ ... ⊗ tₖ ⊢ t, where ⊗ is the multiplicative conjunction (tensor). Composing skills whose output and input share a type is the cut rule. The write-once invariant is the no-contraction property: resources cannot be duplicated. This connection suggests the model could be given a type-theoretic foundation in which well-formedness is enforced by the type system itself.

**Connection to Kahn Process Networks.** If we relax the single-invocation assumption and allow skills to execute repeatedly over streams, N(A) becomes a Kahn Process Network (KPN): deterministic sequential processes over unbounded FIFO channels. The KPN determinism property — the network's I/O function is independent of scheduling — follows from the same structural constraints (no shared state, single producer per channel). This suggests an extension path toward streaming and reactive agent architectures.

**Practical implications.** Mature Petri net tools — invariant computation, reachability graphs, structural reduction — can be applied directly to N(A). For well-formed agents the analysis is trivially fast, but the tools become useful at the *diagnostic* stage: when an agent fails a well-formedness check, computing its S-invariants can localize defects (uncovered places, dead transitions) more precisely than the syntactic checks of section 4.5 alone.

### 4.8 Execution Purity

A skill specification declares not only *what* a skill computes but *what it needs* to compute it. This section formalizes the principle that a skill should execute in an environment containing exactly its declared requirements — no more, no less.

**Definition (Execution environment).** For a skill S = (C, P, F) within agent A, the *execution environment* of S is:

```
Env(S) = (D, Π, G)
```

where:

- D = {resolved(t) : t ∈ C(S)} — the data visible to S, obtained by resolving each consumed type through the data flow protocol (section 4.6)
- Π = F.security — the permission set (filesystem access, network access, secrets)
- G = F.guardrails — the constraint set (timeout, limits, behavioral constraints)

An execution is *pure* if S can access exactly D, exercises at most Π, and is bounded by G.

**Proposition 12 (Isolation).** Let S ∈ A be a skill whose consumed types are resolved via a non-shared resolver (file or api). Then the data accessible to S during execution is exactly D = {resolved(t) : t ∈ C(S)}. For any type t' ∉ C(S), the resolved value resolved(t') is inaccessible to S.

*Proof.* Under the file resolver, each type t is materialized to a distinct path and S receives only the paths corresponding to C(S). Under the api resolver, S receives a scoped endpoint exposing only its declared inputs. In neither case does S obtain a reference to data outside C(S). □

*Caveat.* Proposition 12 fails under the *context* resolver. Because all context-resolved types share the LLM's context window, a skill can attend to data produced for a sibling skill. This is a deliberate trade-off: the context resolver optimizes for simplicity and low latency at the cost of isolation. Systems requiring strict purity must use file or api resolution.

**Proposition 13 (Environment containment).** For every skill S in a well-formed agent A: Π(S) ⊆ Π(A). A skill cannot declare permissions exceeding those of its containing agent.

*Proof.* Π(A) is the union of all permissions the agent is authorized to exercise. A skill with Π(S) ⊄ Π(A) would require the runtime to grant capabilities the agent itself does not possess. The linter rejects such specifications as ill-formed. □

**Proposition 14 (Parallel independence).** If S₁ ⊗ S₂ (parallel execution), then Env(S₁) and Env(S₂) share no mutable state. Their write targets are disjoint.

*Proof.* By producer uniqueness, P(S₁) ≠ P(S₂), so each skill writes to a distinct output type. By the write-once axiom, each output URI receives at most one write. Their input sets D₁ and D₂ may overlap, but overlapping inputs are read-only references, not mutable state. □

**Observation (Environment determinism).** For a well-formed agent A and skill S ∈ A, Env(S) is fully determined by the specification and the resolved input values. Two executions of S with identical inputs receive identical environments. The LLM's output may vary, but the environment is fixed.

**Observation (Non-escalation).** The guardrails set G is monotonically restrictive: adding a constraint to G can only narrow the set of permitted behaviors, never expand it. If G' ⊃ G, then the set of executions satisfying G' is a subset of those satisfying G.

**Observation (Static verifiability).** The linter can verify coherence of Env(S) before execution: every type in C(S) has a producer in G(A), and Π(S) ⊆ Π(A). No runtime information is required.

#### 4.8.1 Worktree Isolation Model

The pure execution environment Env(S) describes *what* a skill should see. When skills have side effects on the filesystem — modifying source files, not just producing typed data — a concrete isolation mechanism is needed. Git worktrees provide one.

**Definition (Worktree execution).** For a skill Sᵢ executing within agent A:

```
Exec(Sᵢ) :
  1. Wᵢ ← git worktree create (ephemeral branch from HEAD)
  2. Sᵢ executes in Wᵢ with Env(Sᵢ) = (D, Π, G)
  3. P(Sᵢ) ← extract output from Wᵢ (structured data or diff)
  4. Merge Wᵢ → main working tree
  5. git worktree remove Wᵢ
```

Each skill operates in its own filesystem copy. The worktree is ephemeral — created before execution, destroyed after merge. Rollback is trivial: `git worktree remove` discards all changes.

**Definition (Write set).** The *write set* of a skill S, denoted W(S) ⊆ Files, is the set of filesystem paths that S may modify during execution. This extends the security facet:

```yaml
security:
  filesystem: read-write
  write_set: ["src/**/*.go", "tests/**/*.go"]
```

**Merge strategies.** When multiple worktrees must be reconciled, three strategies form a safety spectrum:

| Strategy | Precondition | Guarantee | Mechanism |
|----------|-------------|-----------|-----------|
| **Disjoint write sets** | W(S₁) ∩ W(S₂) = ∅ | Conflict-free, commutative | Independent patches, any merge order |
| **Three-way merge** | None required | Best-effort | Git three-way merge; non-overlapping hunks merge automatically, overlapping hunks fail |
| **Sequential fallback** | None required | Always safe | Skills execute in topological order; each sees predecessor's results |

**Proposition 15 (Conflict-free parallel merge).** If S₁ ⊗ S₂ and W(S₁) ∩ W(S₂) = ∅, then merging their worktrees W₁ and W₂ into the main working tree is conflict-free and commutative: merge(W₁, W₂) = merge(W₂, W₁).

*Proof.* Both worktrees branch from the same HEAD. Each modifies a disjoint subset of files. A three-way merge between HEAD, W₁, and W₂ applies non-overlapping hunks — since the modified file sets are disjoint, every hunk is non-overlapping. The merge result is independent of the order in which W₁ and W₂ are merged. □

**Automatic degradation.** The scheduler can use write sets to choose the orchestration strategy automatically:

```
If W(S₁) ∩ W(S₂) = ∅  →  parallel execution (merge guaranteed)
If W(S₁) ∩ W(S₂) ≠ ∅  →  sequential fallback (always safe)
```

This connects to the existing `negotiation` facet: `file_conflicts: yield` indicates a skill that accepts sequential demotion when write sets overlap. The write set makes the negotiation *static* — the scheduler resolves conflicts before execution, not during.

**Container-based isolation.** Worktrees isolate the filesystem but not the network or process space. For full isolation, the execution environment can be a container:

```yaml
security:
  filesystem: read-write
  write_set: ["src/**/*.go"]
  sandbox: container          # ← L3 enforcement
```

The container receives: (1) a mounted volume with only the files in W(S) ∪ {resolved files for C(S)}, (2) network access per Π(S), (3) resource limits per G (CPU, memory, timeout). This maps directly to the L3 enforcement layer of section 6.3. The LLM CLI (Claude Code, Copilot CLI, etc.) runs inside the container with exactly Env(S) — nothing more.

**Enforcement reality.** The table below maps purity properties to enforcement mechanisms:

| Layer | Isolation (P12) | Containment (P13) | Independence (P14) | Write set |
|-------|------------------|--------------------|---------------------|-----------|
| L1 — Generated prompt | Aspirational | Statically checked | Structural (by spec) | Declared only |
| L2 — Framework hooks | Partial | Partial | Structural | Hook-enforced |
| L3 — Worktree/Container | **Enforced** | **Enforced** | **Enforced** | **Enforced** |
| L4 — Guardrails runtime | Enforced + G | Enforced + G | Enforced + G | Enforced + G |

At L1, purity is specification discipline: the generated prompt instructs the LLM to respect boundaries, but nothing prevents violation. At L3 (worktrees or containers), the runtime actively constrains the execution environment to match Env(S), closing the gap between declared and actual purity.

---

## 5. Design Rationale

The model's design choices map to established software engineering principles. This section names those connections; the mechanisms themselves are defined in sections 2–4.

| Principle | Application in the model |
|-----------|-------------------------|
| **Single Responsibility** [4] | One output per skill (`\|P(S)\| = 1`). The linter enforces this as a rule, not a schema constraint. |
| **Explicit dependency injection** [5] | Skills have no ambient access — only declared `consumes`. Data flow is inferred from contracts and validated statically. |
| **Composition over inheritance** | Agents are flat skill lists. No "base agent" pattern, no override mechanism. Changing a skill affects only agents that include it. |
| **Tell, Don't Ask** [6] | Skills declare outputs. The framework routes data. No skill queries another's state. |
| **Attention optimization** [7][8] | Generated output orders sections for LLM primacy/recency biases: guardrails first, security last. |
| **Framework independence** | Tool names are abstract capabilities. Mapping to concrete APIs happens at generation time. |

---

## 6. Frontiers

*The Skill Behavior Model is a foundation. This section describes four directions that extend it beyond static specification into testing, sharing, enforcement, and multi-agent coordination. Each direction is designed concretely — with proposed formats and mechanisms — but none is implemented in the reference tooling. They represent the model's natural evolution, not speculative features.*

### 6.1 Behavioral Testing

Static validation (section 4.5) guarantees structural soundness but not behavioral correctness. A skill that passes all lint rules may still produce useless output. Behavioral testing closes this gap.

Three levels of behavioral testing are proposed, ordered by increasing sophistication:

| Level | Mechanism | What it verifies |
|-------|-----------|------------------|
| **Schema testing** | Assertions on output structure (JSON Schema, regex, format checks) | The skill produces output in the correct format |
| **Golden testing** | Fixed input → output compared to an approved reference, with configurable tolerance | Behavioral stability across versions and model changes |
| **LLM-as-judge** | A second LLM evaluates output quality against criteria declared in the skill | Semantic quality: relevance, exhaustiveness, actionability |

These levels mirror the three grader types recommended by Anthropic's agent evaluation guide [21]: code-based graders (schema testing), model-based graders (LLM-as-judge), and human graders (golden test authoring and approval).

A proposed YAML extension declares tests inline with the skill:

```yaml
skill: review-commenter
# ... existing facets ...

testing:
  schema: review-comments.schema.json
  golden:
    - input: fixtures/simple-diff.txt
      expected: fixtures/simple-review.golden.md
      tolerance: semantic  # exact | fuzzy | semantic
  judge:
    criteria:
      - "Comments are actionable, not vague"
      - "Comments reference specific lines in the diff"
      - "No false positives on style-only changes"
```

A `test` command would run all three levels and report pass/fail per skill. The `tolerance` field controls how golden tests compare outputs: `exact` requires identical strings, `fuzzy` allows minor formatting differences, and `semantic` uses an LLM to judge semantic equivalence.

Two established metrics from the evaluation literature apply directly:

- **pass@k** [22] — the probability that at least one of k invocations produces a correct output. Measures capability.
- **pass^k** — the probability that all k invocations produce correct output. Measures consistency. A skill with high pass@k but low pass^k is capable but unreliable.

SkillsBench [23] — a benchmark evaluating skill efficacy across 84 tasks and 11 domains — provides external evidence relevant to the Skill Behavior Model's design choices. **Important caveat:** SkillsBench defines "skills" as atomic LLM capabilities (tool use, planning, retrieval), not as declarative YAML behavioral units. The findings below are analogies, not direct validations — they support the *intuition* that structured decomposition improves agent performance, but they test a different notion of "skill" than the one defined in this paper. Key findings:

1. **2–3 skills is optimal.** Agents with 2–3 skills outperformed those with 4+ skills (+18.6pp vs +5.9pp gain). This aligns with the model's emphasis on atomic, single-responsibility skills composed into small pipelines.

2. **Self-generated skills provide negligible benefit** (−1.3pp average). Skills must be designed, not auto-generated. This validates the investment in deliberate skill authoring that the format requires.

3. **Smaller model + skills can exceed larger model without skills.** Claude Haiku with skills (27.7%) outperformed Opus without skills (22.0%). Structure compensates for raw model capability — the central thesis of the Skill Behavior Model.

### 6.2 Skill Ecosystem and Marketplace

Skills are designed to be reusable (section 7.1), but sharing them beyond copy-paste requires additional infrastructure. A skill ecosystem requires three capabilities: distribution, versioning, and trust.

**Distribution.** A skill registry — centralized (like npm) or federated (like Go modules resolving from Git) — would allow commands like `install user/review-commenter@1.2` to fetch a skill and its transitive dependencies. An import pipeline that resolves skills from remote sources provides the pattern; a registry generalizes it.

**Semantic versioning.** The I/O contract (`consumes`/`produces`) *is* the skill's public API. A breaking change is any change that modifies the contract:

- Adding a new entry to `consumes` is a breaking change (callers must provide more data)
- Removing an entry from `consumes` is backward-compatible (callers can provide data the skill ignores)
- Changing `produces` is always breaking (consumers depend on the exact type name)

This maps to the Liskov Substitution Principle [4]: a new version of a skill is substitutable for the old if `consumes(v2) ⊆ consumes(v1)` (contravariance of inputs) and `produces(v2) = produces(v1)` (invariance of output). Semantic version numbers can be derived mechanically from the I/O diff between versions — no human judgment required.

**Trust and curation.** Published skills carry quality signals: structural quality scores, passing behavioral tests, author verification, and usage statistics. The scoring algorithm (section 7.3) evaluates structural quality; extending it to include test coverage and reuse frequency is straightforward.

The analogy is npm for agent behaviors — but with a critical difference. JavaScript packages have complex dependency trees that create supply chain risk. Skill dependencies are shallow: a skill declares what it consumes, not which other skill provides it. The agent resolves the data flow, not the skill. This means skill "dependency trees" are at most one level deep, eliminating the cascading version conflict problem that plagues package ecosystems.

### 6.3 Runtime Enforcement

The security facet (`filesystem: read-only`, `network: none`) and guardrails (`timeout: 5min`, `max_comments: 15`) are currently declarations — they inform the generated prompt but are not enforced. Enforcement requires bridging the gap between specification and execution across four layers:

**Layer 1 — Prompt generation (implemented).** The generator includes security and guardrail declarations as explicit instructions in the generated prompt: "You MUST NOT write files", "You MUST NOT make network requests." This relies on the LLM's instruction-following capability — effective in most cases but vulnerable to prompt injection or ambiguity.

**Layer 2 — Framework hooks (specifiable).** Modern coding agent frameworks provide hook systems that intercept tool calls before execution. Claude Code's `pre_tool_call` hooks [1], for example, can block specific tools based on a matcher. The build step can generate hooks from the security facet:

A skill declaring `security: filesystem: read-only` would generate:

```jsonc
// .claude/settings.json (generated by the build step)
{
  "hooks": {
    "pre_tool_call": [{
      "matcher": { "tool_name": "Write" },
      "command": "echo 'BLOCK: review-commenter has filesystem: read-only' && exit 1"
    }, {
      "matcher": { "tool_name": "Edit" },
      "command": "echo 'BLOCK: review-commenter has filesystem: read-only' && exit 1"
    }]
  }
}
```

A skill declaring `security: network: none` would generate hooks blocking `WebFetch` and `WebSearch`. This provides deterministic enforcement at the framework level — no LLM compliance required.

**Layer 3 — OS-level sandboxing (future).** The most robust enforcement maps security declarations to OS-level constraints: `filesystem: read-only` → read-only mount, `network: none` → network namespace isolation, `secrets: []` → filtered environment variables. Container profiles, WASM sandboxes, or seccomp filters could be generated from the security facet. This is the ideal end state but requires a runtime layer that does not yet exist.

**Layer 4 — Guardrails enforcement (specifiable).** Beyond security, guardrails like `timeout` and `max_comments` can be enforced through post-execution validation:

```yaml
guardrails:
  - timeout: 5min
  - max_comments: 15
  - no_approve_without_tests
```

Generates:
- A timeout watchdog that terminates the agent process after 5 minutes
- A post-execution hook that counts review comments in the output and fails if the count exceeds 15
- A post-execution hook that verifies `test_results` are present before allowing an approval decision

The enforcement ladder — prompt → hooks → sandbox → guardrails runtime — represents increasing investment for increasing guarantees. Layer 1 is free. Layer 2 requires generating framework-specific hook configurations. Layer 3 requires a dedicated runtime. Layer 4 requires output validation logic. Each layer is independently useful; together, they transform declarations into constraints.

### 6.4 Multi-Agent Coordination

The Skill Behavior Model defines behavior within a single agent (section 7.4). Communication between agents — discovery, negotiation, delegation — is the domain of protocols like MCP [10] and Agent2Agent [11]. But the model's I/O contracts provide a natural bridge.

**Agents as MCP tools.** An agent's I/O contract (`consumes`/`produces`) maps directly to an MCP tool definition: `consumes` becomes the tool's input schema, `produces` becomes its output schema. The build step could generate an MCP tool registration for each agent, allowing other agents to discover and invoke it through the standard MCP protocol.

**Agent-to-agent data flow.** A new type convention — `agent_output:<agent_name>` — would allow one agent to consume another agent's output. For example, a `security-auditor` agent might declare `consumes: [agent_output:ci-reviewer]`, wiring the ci-reviewer's `review_comments` and `risk_score` as inputs to the security audit. The linter would validate this cross-agent wiring the same way it validates intra-agent wiring.

**Extended negotiation.** The existing `negotiation` facet (`file_conflicts: yield`, `priority: 3`) handles simple conflict resolution between skills within an agent. Extending it to inter-agent coordination would support:

- **Resource claiming** — an agent declares exclusive access to specific files or directories during execution
- **Priority arbitration** — when two agents attempt conflicting actions, the higher-priority agent proceeds
- **Delegation** — an agent can defer a sub-task to another agent by emitting a `delegate:<agent_name>` output

**A2A bridge.** The Agent2Agent protocol [11] defines "Agent Cards" — JSON documents describing an agent's capabilities, authentication, and endpoints. An agent's I/O contract could generate an A2A Agent Card, allowing agents defined in this model to participate in A2A discovery and coordination networks. The mapping is straightforward: `produces` → Agent Card capabilities, `consumes` → required input context, `security` → trust and access metadata.

These directions extend the model's compositional philosophy from intra-agent to inter-agent: the same principles — explicit contracts, static validation, composition over inheritance — apply at both scales.

---

## 7. Experience

*The Skill Behavior Model is early-stage. The observations below come from the reference implementation and its test suite, not from large-scale production adoption. They illustrate the model's properties, not its maturity.*

### 7.1 Reusability

In practice, skills compose across agents without modification. The `tdd-runner` skill — which produces `test_results` — is consumed by `review-commenter`, `risk-scorer`, and `coverage-reporter` across different agents. Each consumer declares `test_results` in its `consumes`; the `tdd-runner` skill is unaware of its consumers. Adding a new consumer requires zero changes to `tdd-runner`.

### 7.2 Cross-Framework Deployment

The same set of 6 skills and 1 agent has been generated for both Claude Code and GitHub Copilot targets. The skill specifications are identical across both. The only differences are in the generated output: tool name mappings, file paths, and framework-specific conventions (e.g., Copilot's `copilot-instructions.md` global file, which Claude Code does not use).

### 7.3 Static Validation

The DAG structure enables validation before any agent executes. Missing dependencies, circular references, and unmet context are caught at specification time. This is analogous to type checking in programming languages — errors are found before runtime, not during a costly LLM invocation.

### 7.4 Limitations

The Skill Behavior Model has the following known limitations and scope boundaries:

**Agent identity.** The format describes what an agent *can do*, not *who it is*. Personality, tone, and values are the domain of complementary formats like SOUL.md [9]. The two are composable — an agent can have both a skill specification (capabilities) and a soul specification (identity).

**Inter-agent communication.** The format defines behavior within a single agent. Communication between agents — discovery, negotiation, message passing — is the domain of protocols like MCP [10], Agent2Agent [11], and ACP. The Skill Behavior Model can coexist with these protocols: skills define what each agent does, protocols define how agents talk to each other.

**Authoring cost.** The format trades authoring speed for composability. A monolithic prompt takes seconds to write; a full skill spec with 5 facets requires deliberate design. The CLI scaffolds a valid skeleton (`forgent skill create`), and the investment pays off when the same skill is reused across multiple agents or deployed to multiple frameworks. For one-off agents with no reuse intent, a monolithic prompt may remain simpler.

**Runtime enforcement.** The format defines static declarations, not runtime enforcement. Whether a framework actually restricts filesystem access to `read-only` or enforces a `timeout: 5min` guardrail is framework-specific. The specification makes the author's intent explicit and machine-readable; enforcement is delegated to the target platform.

**Error handling and recovery.** The format does not define what happens when a skill fails — no retry policies, fallback skills, or error propagation rules. If `tdd-runner` fails to produce `test_results`, skills that consume it (e.g., `review-commenter`) have no declared recovery path. Error handling is delegated entirely to the target framework's runtime. This is a deliberate scope boundary: the format defines the success path, not the failure path.

**Human-in-the-loop.** The format has no mechanism for declaring human approval gates within a skill pipeline — for example, requiring a human to approve before a `risk-scorer` result triggers a merge block. Human-in-the-loop gates are a future facet direction discussed in section 9.

**Non-determinism.** Skills are executed by LLMs, which are inherently non-deterministic. The same `review-commenter` skill may produce different comments on the same input across invocations. The format defines behavioral *intent*, not deterministic *output*. Static validation ensures structural correctness (all inputs are satisfied), but it cannot guarantee behavioral consistency at runtime.

**Behavioral testing.** The format supports static validation (dependency checking, loop detection, linting) but not behavioral testing — there is no built-in mechanism to assert that a skill's actual output matches its declared `produces` type. Testing that `review-commenter` actually produces useful review comments requires external validation outside the spec.

**Compositional overhead.** Decomposing a monolithic prompt into multiple skills adds structural overhead — each generated skill file includes section headers, facet labels, and formatting that a monolithic prompt would omit. For agents with many skills, this overhead can consume a meaningful fraction of the target model's context window, potentially degrading performance. The trade-off is explicit: compositional clarity and reusability at the cost of token budget.

**Data flow security.** The current security facet declares filesystem and network access but does not address data flow injection — a malicious or misconfigured skill could produce output that, when consumed by a downstream skill, causes unintended behavior. The format assumes skills are authored by trusted teams. Extending the security model to validate inter-skill data flow is a potential future direction.

### 7.5 Empirical Evaluation

*This section reports preliminary measurements comparing a composed agent (built from skill specifications) against its monolithic equivalent. The results are from a single agent type (CI review) on a small task set. They illustrate the model's properties, not its generalizability.*

**Setup.** The `ci-reviewer` agent (6 skills: `ts-linter`, `type-checker`, `tdd-runner`, `coverage-reporter`, `review-commenter`, `risk-scorer`) was compared against a monolithic prompt performing the same tasks. Both were generated for the Claude Code target and evaluated on the same set of code review tasks.

**Token overhead.** The composed agent's generated artifacts — 6 skill files plus 1 agent file — contain structural overhead (section headers, facet labels, separators) that a monolithic prompt omits. Preliminary measurement shows approximately 20–30% token overhead for a 6-skill agent compared to an equivalent monolithic prompt. This overhead increases roughly linearly with the number of skills, consistent with SkillsBench's finding [23] that agents with 4+ skills show diminishing returns.

**Behavioral comparison.** On the evaluated tasks, the composed and monolithic agents produced comparable results in terms of review quality (judged by LLM-as-judge). The composed agent showed marginally higher consistency across invocations — a possible effect of the structured decomposition constraining each skill's scope, reducing the variance space for the LLM.

**Limitations.** These results are preliminary and do not support strong claims. The task set is small, the agent type is narrow (code review only), and the evaluation uses a single model. A systematic evaluation across diverse agent types, models, and benchmarks (SWE-bench [24], SkillsBench [23], Terminal-Bench [25]) is required to validate these observations.

### 7.6 Additional Case Studies

**The ci-reviewer as dogfood.** The reference implementation uses Forgent to build its own CI review agent — 6 skills composed into a single agent, deployed to both Claude Code and GitHub Copilot targets. The skill set (`ts-linter`, `type-checker`, `tdd-runner`, `coverage-reporter`, `review-commenter`, `risk-scorer`) was designed once, validated once (`forgent lint` + `forgent score`), and generated twice (one per target). The only differences between the two generated outputs are tool name mappings and framework-specific file paths. The skill specifications are identical.

This dogfooding confirms three properties of the model in practice:

1. **Cross-target stability.** The same 6 skills produce correct, functional output for both Claude Code and GitHub Copilot without any specification changes.
2. **Incremental evolution.** Adding a new skill (e.g., `security-scanner`) requires writing one YAML file and re-running `forgent build`. No existing skill or agent file is modified.
3. **Scored quality.** The agent scores 94/100 on `forgent score`, with deductions only for missing optional facets (`when_to_use`, `anti_patterns`).

### 7.7 Developer Experience

Qualitative observations from skill authoring:

**Scaffolding.** `forgent skill create <name>` generates a valid YAML skeleton in seconds. The scaffold includes all 5 facets with placeholder values, reducing the blank-page problem. Authors typically spend 5–10 minutes filling in the facets for a well-understood behavior.

**Feedback loop.** The `forgent lint` → fix → lint cycle converges quickly. Most first drafts trigger 1–3 diagnostics (typically SRP violations or missing dependencies). The diagnostics are actionable — "skill X consumes `test_results` but no skill in agent Y produces it" — and the fix is usually adding a missing skill or splitting a multi-output skill.

**Cost of formalism.** Writing a skill spec takes longer than writing a monolithic prompt. A monolithic CI reviewer prompt takes minutes; the equivalent 6-skill decomposition takes 30–60 minutes of design time. The investment pays off on reuse: when the same `tdd-runner` skill is used in three different agents, the per-agent authoring cost drops below the monolithic approach.

**Import as onramp.** The `forgent import` pipeline (section 7.8) lowers the entry barrier. Authors can start with a monolithic prompt, import it to get a first-draft decomposition, and refine from there. The validation-driven retry loop ensures the initial decomposition meets structural quality standards.

### 7.8 Illustrative Case: Importing a Monolithic Agent

*The following is a qualitative illustration (N=1) of the import pipeline applied to a single open-source agent definition. It demonstrates the mechanics of decomposition and validation feedback, not the generalizability of the approach. A systematic evaluation across diverse agent definitions is left for future work.*

We applied the `forgent import` pipeline to the Claude-SPARC Automated Development System — a 1 260-line monolithic Markdown prompt that orchestrates multi-agent software development through 5 phases (Specification, Pseudocode, Architecture, Refinement, Completion). The pipeline sends the prompt to an LLM, parses the response into skill specs, and validates them with the existing linter, scorer, and dependency checker. If any skill scores below a configurable threshold, the pipeline retries with the validation feedback appended to the prompt.

**Without quality gate.** The LLM produced 7 skills, all violating the Single Responsibility Principle (multiple `produces` per skill). The dependency checker flagged 9 unmet `consumes` entries. **With quality gate (min-score 80).** The retry prompt included the linter feedback. The LLM split the 7 skills into 15 atomic skills, each with a single `produces`. All dependency edges were satisfied. The agent scored 94/100.

Three observations:

1. **Static validation as LLM feedback.** The linter's SRP and unmet-dependency diagnostics, when appended to the retry prompt, were sufficient to guide the LLM toward a structurally correct decomposition without human intervention.

2. **Composability under decomposition.** The 15-skill graph maintained clean `consumes`/`produces` edges — the same property that enables reuse in hand-authored skills also held for LLM-generated ones.

3. **Format as quality gate.** The structural constraints of the spec (single `produces`, declared dependencies, scored facets) caught defects that a free-form prompt review would likely miss.

---

## 8. Related Work

### 8.1 The Coding Agent Ecosystem

The following table maps how each major coding agent framework handles customization as of early 2026:

| Framework | Context file | Agent format | Skills | Structured fields |
|-----------|-------------|--------------|--------|-------------------|
| Claude Code [1] | `CLAUDE.md` | `.claude/agents/*.md` | `.claude/skills/*/SKILL.md` | name, description, tools, model, memory, hooks |
| GitHub Copilot [2] | Repo instructions | `.github/agents/*.agent.md` | `.github/skills/*/SKILL.md` | name, description, tools, model |
| Gemini CLI | `GEMINI.md` | `.gemini/agents/*.md` | — | name, description, tools, model, temperature, max_turns, timeout |
| OpenCode | `opencode.json` | `.opencode/agents/*.md` | — | description, mode, model, tools, permission |
| Amazon Q Developer | `.amazonq/rules/*.md` | `.amazonq/cli-agents/*.json` | — | name, description, tools, permissions |
| Zed AI | `.rules` | Rules library | — | model, temperature, rules |
| Continue | `config.yaml` | `config.yaml` agents | Prompt files `.md` | models, rules, tools (MCP) |
| Codex (OpenAI) | `AGENTS.md` | Agents SDK (Python) | — | Programmatic only |
| Kiro (AWS) | Steering files | YAML config | "Powers" | name, description, prompt, tools, mcpServers |
| Cursor | `.cursor/rules/*.mdc` | — (rules only) | — | description, globs, alwaysApply |
| Windsurf | `.windsurf/rules/` | — (rules only) | — | Activation mode |
| Cline | `.clinerules/*.md` | — (rules only) | — | Globs (frontmatter), activation conditions |
| Roo Code | `.roo/rules/*.md` | Mode config | — | Modes, rules, AGENTS.md support |
| Augment Code | `.augment/rules/*.md` | — (rules only) | — | Agent-requested activation |
| Aider | `.aider.conf.yml` | — | — | Config only |

A convergence is visible: the dominant pattern is markdown with YAML frontmatter declaring `name`, `description`, `tools`, and `model`. But across **all** frameworks, the same concerns remain unstructured: guardrails, security, observability, and I/O contracts.

### 8.2 Complementary Specifications

| Approach | What it defines | Composition | Validation |
|----------|----------------|:-----------:|:----------:|
| AGENTS.md [12] | Project context for agents | Hierarchical override | None |
| SOUL.md [9] | Agent identity (personality, tone) | Per-agent | None |
| Open Agent Spec [14] | Workflow graphs (nodes, edges, data flow) | Graph-based | Schema |
| MCP / A2A [10][11] | Inter-agent communication protocols | Message-based | Schema |
| LangGraph / CrewAI | Programmatic agent orchestration (Python) | Code-level | Runtime |
| DSPy Signatures [17] | Typed I/O for LLM modules | Programmatic | Optimizer |
| Semantic Kernel [18] | Plugins with typed functions | Programmatic | Schema |
| AutoGen [19] | Multi-agent conversations | Programmatic | Runtime |
| NIST AI Agent Standards [16] | Governance, security, monitoring policies | Policy-level | Audit |
| **Skill Behavior Model** | **Behavioral capabilities and constraints** | **Declarative YAML** | **Static** |
| SkillsBench [23] | Benchmark for skill efficacy across 84 tasks | Community | Empirical |

These specifications answer different questions about agent systems:

**AGENTS.md** [12] is a free-form markdown standard for guiding coding agents, stewarded by the Linux Foundation. It provides project-level context (build steps, conventions, architecture) but has no structured facets, no typed I/O contracts, and no composition model. It tells an agent *about the project* — the Skill Behavior Model tells an agent *what to do and under what constraints*.

**SOUL.md** [9] defines agent identity — personality, tone, values. The Skill Behavior Model defines agent capabilities. They answer different questions and are composable: an agent can have both a soul (who it is) and skills (what it can do).

**Open Agent Specification** [14] is a declarative, framework-agnostic YAML format for defining agent workflows, introduced by Oracle in October 2025. It models workflows as directed graphs of typed nodes (LLMNode, APINode, ToolNode) with explicit data flow edges — analogous to ONNX for ML models. The Skill Behavior Model operates at a different granularity: it defines the *behavioral content* of each node, not the *workflow between nodes*. The two are potentially complementary — skills could serve as the behavioral building blocks within an Agent Spec workflow. The NIST AI Agent Standards Initiative [16] is working toward standardizing similar concerns at the policy level.

**Programmatic frameworks** such as LangGraph and CrewAI define agents in code (Python); these frameworks are surveyed in [15]. The Skill Behavior Model defines agents in data (YAML). The declarative approach enables static analysis and non-programmer access to agent design. The trade-off is expressiveness — programmatic frameworks can encode error handling, conditional branching, loops, and state machines that a declarative format cannot.

**DSPy** [17] defines typed input/output signatures for LLM modules — `question -> answer` or `context, question -> rationale, answer`. This is the closest prior art to the Skill Behavior Model's `consumes`/`produces` pattern. The key difference is scope: DSPy signatures define I/O for individual LLM calls within a Python program and support automatic prompt optimization, while the Skill Behavior Model defines I/O for behavioral units that generate entire prompt specifications. DSPy optimizes prompts programmatically; the Skill Behavior Model composes them declaratively.

**Microsoft's Semantic Kernel** [18] and **AutoGen** [19] take different approaches to agent composition. Semantic Kernel defines "plugins" as typed function collections with input/output schemas — closer to API contracts than behavioral specifications. AutoGen defines multi-agent conversations with typed message protocols. Both are programmatic (Python/.NET) frameworks. The Skill Behavior Model shares the goal of structured I/O but differs in being declarative, framework-agnostic, and targeting prompt generation rather than runtime orchestration.

**ADL** [13] (Agent Description Language) is a declarative DSL for chatbot agents with typed slots and dialogue flows. It shares the goal of replacing free-form prompts with structured specifications, but targets conversational agents rather than coding agents. The Skill Behavior Model focuses on tool-using skills with I/O contracts rather than dialogue management.

**SkillsBench** [23] benchmarks skill efficacy across 84 tasks and 11 domains. Its "skills" are atomic LLM capabilities (tool use, planning, retrieval) — a different notion than the declarative behavioral units defined in this paper. Despite this distinction, SkillsBench's structural findings — that 2–3 skills compose optimally, that self-generated skills underperform designed ones, and that structure compensates for model size — provide indirect support for the Skill Behavior Model's design intuitions. The relationship is analogical, not validating.

**Communication protocols** (MCP, A2A, ACP) [10][11] define how agents discover each other, negotiate, and exchange messages. The Skill Behavior Model defines what each agent does internally. They operate at different layers and are composable: skills define behavior, protocols define communication.

---

## 9. Conclusion

The Skill Behavior Model brings to agent engineering what interfaces and design principles brought to software engineering: structured decomposition, explicit contracts, and static validation. Each skill is a single-responsibility unit with a declared I/O contract. Agents are flat compositions of skills. The format is framework-agnostic, LLM-agnostic, and statically validatable.

Section 4 formalizes these properties through fifteen results. The core nine establish structural well-formedness: decidability (P1), compatible substitution (P2), parallelizability with layer decomposition and Dilworth width (P3, Corollaries 3.1–3.2), resolution completeness (P4), output immutability (P5), cycle-free resolution (P6), structural additivity (P7), valid schedule existence (P8), and cross-target structural isomorphism (P9). Two additional propositions strengthen the graph structure: full reachability from source to sink (P10) and well-formedness-preserving skill fusion (P11). Three propositions formalize execution purity: data isolation under non-shared resolvers (P12), permission containment (P13), and parallel independence of mutable state (P14). A final proposition guarantees conflict-free parallel merge under disjoint write sets (P15). The model admits a Petri net correspondence (section 4.7) connecting it to linear logic and Kahn Process Networks. A worktree-based isolation model (section 4.8.1) provides concrete L3 enforcement with a three-level merge strategy spectrum.

Section 6 maps the model's natural extensions: behavioral testing (schema, golden, LLM-as-judge), a skill ecosystem with contract-derived semantic versioning, runtime enforcement through a four-layer ladder (prompt → hooks → sandbox → guardrails), and multi-agent coordination via MCP/A2A bridges. Each direction builds on the model's core principle — explicit, machine-readable contracts — rather than introducing new abstractions.

Early empirical evidence (section 7.5) suggests that composed agents perform comparably to monolithic equivalents with 20–30% token overhead. SkillsBench [23] — testing a different notion of "skill" (LLM capabilities, not declarative YAML units) — provides indirect support for the model's core intuition: structured decomposition improves agent performance, and structure compensates for model capability.

The format is deliberately minimal. It does not prescribe an implementation language, a runtime, or a specific LLM. It defines a behavioral contract — what an agent skill does, what it needs, what it produces, and what constraints it operates under. Implementations generate framework-native artifacts from this specification.

The gap between static specification and runtime behavior remains the model's central tension. Soundness guarantees structure, not semantics. Behavioral testing, runtime enforcement, and empirical evaluation are the paths toward closing that gap — each moving the boundary of what can be verified before an agent runs.

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

[17] O. Khattab et al., "DSPy: Compiling Declarative Language Model Calls into Self-Improving Pipelines," ICLR 2024. [arXiv](https://arxiv.org/abs/2310.03714)

[18] Microsoft, "Semantic Kernel — Integrate cutting-edge LLM technology quickly and easily into your apps," 2023. [GitHub](https://github.com/microsoft/semantic-kernel)

[19] D. Wu et al., "AutoGen: Enabling Next-Gen LLM Applications via Multi-Agent Conversation," arXiv:2308.08155, 2023. [arXiv](https://arxiv.org/abs/2308.08155)

[20] R. Milner, "A Theory of Type Polymorphism in Programming," *Journal of Computer and System Sciences*, vol. 17, no. 3, pp. 348–375, 1978. The origin of "well-typed programs don't go wrong." [ScienceDirect](https://doi.org/10.1016/0022-0000(78)90014-4)

[21] Anthropic, "Demystifying evals for AI agents," Anthropic Engineering Blog, 2025. [Anthropic](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents)

[22] M. Chen et al., "Evaluating Large Language Models Trained on Code," arXiv:2107.03374, 2021. Introduces pass@k metric. [arXiv](https://arxiv.org/abs/2107.03374)

[23] "SkillsBench: Benchmarking How Well Agent Skills Work Across Diverse Tasks," arXiv:2602.12670, February 2026. [arXiv](https://arxiv.org/abs/2602.12670) | [skillsbench.ai](https://www.skillsbench.ai/)

[24] C. E. Jimenez et al., "SWE-bench: Can Language Models Resolve Real-World GitHub Issues?" ICLR 2024. [arXiv](https://arxiv.org/abs/2310.06770) | [swebench.com](https://www.swebench.com/)

[25] "Terminal-Bench: Evaluating AI Agents in Real CLI Environments," 2025. [terminal-bench.com](https://terminal-bench.com/)

[26] JetBrains, "Developer Productivity AI Arena (DPAI Arena)," October 2025. [JetBrains](https://lp.jetbrains.com/dpai-arena/)

[27] "Evaluation and Benchmarking of LLM Agents: A Survey," ACM SIGKDD 2025. [ACM](https://dl.acm.org/doi/10.1145/3711896.3736570)

---

## Appendix A — I/O Type Reference

Detailed descriptions and examples for each type in the reference I/O catalog (section 2.1).

---

<dl>

<dt><code>git_diff</code></dt>
<dd>

**Category:** Input
**Description:** The unified diff of pending changes, typically obtained from `git diff`, `git diff --staged`, or a pull request payload. Represents the scope of modifications under review.
**Typical producer:** Agent input (CI hook, PR event, user invocation).
**Typical consumers:** `review-commenter`, `risk-scorer`.
**Example:**
```diff
diff --git a/src/auth/login.ts b/src/auth/login.ts
index 3a4b5c6..7d8e9f0 100644
--- a/src/auth/login.ts
+++ b/src/auth/login.ts
@@ -12,3 +12,5 @@ export async function login(credentials: Credentials) {
+  if (!credentials.email) {
+    throw new ValidationError("Email is required");
+  }
```

</dd>

<dt><code>source_code</code></dt>
<dd>

**Category:** Input
**Description:** Raw source files relevant to the task, read from the working tree or a checkout. May be the full repository or a filtered subset (e.g., only files matching `src/**/*.ts`). Provides the code context that analysis skills operate on.
**Typical producer:** Agent input (filesystem read).
**Typical consumers:** `ts-linter`, `type-checker`, `tdd-runner`, `coverage-reporter`.
**Example:**
```typescript
// src/auth/login.ts
export async function login(credentials: Credentials): Promise<Session> {
  const user = await findUser(credentials.email);
  if (!user || !await verify(credentials.password, user.hash)) {
    throw new AuthError("Invalid credentials");
  }
  return createSession(user);
}
```

</dd>

<dt><code>file_tree</code></dt>
<dd>

**Category:** Input
**Description:** A listing of file paths in the repository, providing structural context. Enables skills to understand project layout, detect naming conventions, and identify relevant files without reading their contents.
**Typical producer:** Agent input (directory scan).
**Typical consumers:** `ts-linter`, `type-checker`, `tdd-runner`, `coverage-reporter`.
**Example:**
```
src/
  auth/
    login.ts
    middleware.ts
    session.ts
  api/
    handler.go
    routes.go
  tests/
    auth.test.ts
    api.test.go
```

</dd>

<dt><code>lint_results</code></dt>
<dd>

**Category:** Intermediate
**Description:** Linter output containing errors, warnings, and their file locations. Produced by a linting skill after analyzing source code against configured rules. Structured as a list of diagnostics with severity, file path, line number, and message.
**Typical producer:** `ts-linter` (or any language-specific linting skill).
**Typical consumers:** `review-commenter`, `risk-scorer`.
**Example:**
```
src/auth/login.ts:3:1    error    no-unused-vars       'logger' is defined but never used
src/api/handler.go:45:12 warning  sql-injection-risk    SQL query uses string concatenation
src/auth/session.ts:8:5  warning  no-explicit-any       Unexpected 'any' type
```

</dd>

<dt><code>type_errors</code></dt>
<dd>

**Category:** Intermediate
**Description:** Type-checker output containing type mismatches, missing imports, and interface violations. Produced by a type-checking skill after static analysis. Each diagnostic includes the file path, position, error code, and a human-readable message.
**Typical producer:** `type-checker`.
**Typical consumers:** `review-commenter`.
**Example:**
```
src/auth/login.ts:15:23 - error TS2345: Argument of type 'string' is not assignable
                          to parameter of type 'HashedPassword'.
src/api/routes.go:8:2   - error: cannot use "admin" (untyped string constant)
                          as type Role in assignment
```

</dd>

<dt><code>test_results</code></dt>
<dd>

**Category:** Intermediate
**Description:** Test runner output including pass/fail counts, failure details, and execution duration. Produced by a testing skill after running the project's test suite (or a filtered subset). Provides evidence of functional correctness.
**Typical producer:** `tdd-runner`.
**Typical consumers:** `review-commenter`, `risk-scorer`, `coverage-reporter`.
**Example:**
```
Tests: 42 passed, 2 failed, 1 skipped (45 total)
Duration: 12.3s

FAIL TestAuth/expired_token (auth_test.go:78)
  Expected: ErrSessionExpired
  Got:      nil

FAIL TestAPI/rate_limit (api_test.go:156)
  Expected status: 429
  Got status: 200
```

</dd>

<dt><code>coverage_report</code></dt>
<dd>

**Category:** Intermediate
**Description:** Code coverage data including line and branch percentages, per-file breakdown, and uncovered regions. Produced by a coverage skill after analyzing test execution traces. Highlights areas of the codebase that lack test coverage.
**Typical producer:** `coverage-reporter`.
**Typical consumers:** `review-commenter`.
**Example:**
```
Overall coverage: 78.3% (lines), 65.1% (branches)

File                      Lines    Branches
src/auth/login.ts         92.0%    85.7%
src/auth/middleware.ts     45.2%    30.0%   ← below threshold
src/api/handler.go        88.5%    72.3%
src/auth/refresh.ts       12.0%    0.0%    ← critical gap
```

</dd>

<dt><code>review_comments</code></dt>
<dd>

**Category:** Output
**Description:** Actionable review comments anchored to specific files and lines, ready to post on a pull request. Each comment identifies the issue, explains why it matters, and suggests a concrete fix. Produced by a review skill that synthesizes lint results, type errors, test results, and diff analysis.
**Typical producer:** `review-commenter`.
**Typical consumers:** Agent output (posted to PR, displayed to user).
**Example:**
```markdown
**src/api/handler.go:45** — SQL injection risk
The query is built with string concatenation. Use parameterized queries instead:
`db.Query("SELECT * FROM users WHERE id = ?", userID)`

**src/auth/middleware.ts:12** — Missing error handling
The `verifyToken()` call can throw but is not wrapped in try/catch.
Consider adding error handling or using a middleware error boundary.
```

</dd>

<dt><code>risk_score</code></dt>
<dd>

**Category:** Output
**Description:** A numeric score from 0 (no risk) to 10 (critical risk) summarizing the overall risk of the change, accompanied by a brief justification. Produced by a scoring skill that weighs diff size, test coverage delta, lint severity, and affected subsystems.
**Typical producer:** `risk-scorer`.
**Typical consumers:** Agent output (used for merge gating, reviewer prioritization).
**Example:**
```
Score: 7/10
Justification: Large diff (420 lines) touching authentication logic
with no new tests. Lint reports 1 SQL injection warning. Coverage
for modified files dropped from 82% to 71%.
```

</dd>

<dt><code>approval_gate</code></dt>
<dd>

**Category:** Intermediate
**Description:** A human decision injected at a defined point in the pipeline. The skill that produces an `approval_gate` pauses execution and requests human review before downstream skills proceed. The decision can be an approval, a rejection, or a request for changes — including questions that the human needs answered before proceeding. Questions can be open-ended or carry predefined answer options, reducing reviewer effort and producing machine-readable responses. When the decision includes questions, the pipeline may loop back to a prior skill to address them before re-requesting approval.
**Typical producer:** A gate skill that presents prior results to a human reviewer.
**Typical consumers:** Any downstream skill that should only run after human approval (e.g., a deployment skill, a merge-trigger skill). A skill that addresses questions raised in a rejected gate.
**Example:**
```yaml
decision: request_changes
reviewer: "alice@example.com"
timestamp: "2026-03-12T14:30:00Z"
questions:
  - text: "What's the rollback strategy if the auth migration fails?"
    options:
      - "Revert migration via down script"
      - "Feature flag — disable new auth path"
      - "Manual DB restore from backup"
  - text: "Have we tested this against the staging LDAP server?"
    options:
      - "Yes, all tests pass"
      - "No, needs staging deployment first"
notes: "The approach looks reasonable but I need answers before approving."
```

Questions can be open-ended (no `options` field) or structured with predefined choices. Structured questions reduce reviewer effort and produce machine-readable answers that downstream skills can act on programmatically.

</dd>

<dt><code>human_feedback</code></dt>
<dd>

**Category:** Input
**Description:** Free-form human input injected into the pipeline at any point. Unlike `approval_gate` (which is a structured decision with a clear outcome), `human_feedback` carries unstructured guidance — clarifications, priority adjustments, or domain-specific instructions that the agent cannot infer from code alone. This type enables semi-automated workflows where human expertise supplements automated analysis.
**Typical producer:** Agent input (user prompt, Slack message, PR comment).
**Typical consumers:** Any skill that benefits from human context (e.g., `review-commenter` adjusting focus based on reviewer priorities).
**Example:**
```markdown
Focus the review on the authentication changes in src/auth/.
The API changes in src/api/ are already reviewed — skip those.
Also flag any changes to error messages, we have i18n constraints.
```

</dd>

</dl>
