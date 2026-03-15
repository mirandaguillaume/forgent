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

Most AI agent definitions today are monolithic prompt files — brittle, untestable, and locked to one framework.

- **Skills as interfaces** — each skill declares `consumes`/`produces` and is governed by 5 facets (Context, Strategy, Guardrails, Observability, Security)
- **Agents as compositions** — a named list of skills with its own I/O contract and orchestration strategy, validated statically by the linter
- **Write once, deploy anywhere** — the same YAML generates output for Claude Code [1], GitHub Copilot [2], or any other target without modification

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
- P ∈ T is a single produced type name (the skill's output)
- F is a record of facets: strategy, guardrails, observability, security, negotiation

An **agent** is a tuple A = (S₁, ..., Sₙ, Cₐ, Pₐ, σ) where:
- S₁, ..., Sₙ are the agent's skills
- Cₐ ⊆ T is the agent's consumed types (external inputs)
- Pₐ ⊆ T is the agent's produced types (final outputs)
- σ ∈ {sequential, parallel, parallel-then-merge, adaptive} is the orchestration strategy

The **data flow graph** G(A) is a directed graph where:
- Nodes are the skills S₁, ..., Sₙ
- An edge (Sᵢ, Sⱼ) exists when P(Sᵢ) ∈ C(Sⱼ) — skill i produces a type that skill j consumes

T is the universe of type names — free-form strings like `git_diff`, `lint_results`, `review_comments`. Types are nominal: two types match if and only if their names are identical strings. There is no structural subtyping.

### 4.2 Structural Properties

The Skill Behavior Model guarantees six structural properties when an agent passes validation (`forgent lint` + `forgent doctor`):

| Property | Formal statement | Verified by |
|----------|-----------------|-------------|
| **Acyclicity** | G(A) is a directed acyclic graph | `forgent doctor` — cycle detection |
| **Contract completeness** | ∀ Sᵢ ∈ A, ∀ t ∈ C(Sᵢ): ∃ Sⱼ ∈ A where P(Sⱼ) = t, or t ∈ Cₐ | `forgent lint` — unmet dependencies |
| **Single responsibility** | ∀ Sᵢ ∈ A: |{P(Sᵢ)}| = 1 | Lint rule SRP |
| **Build idempotence** | build(spec) = build(spec) — deterministic generation | Generator implementation |
| **Additivity** | Adding S' to A does not alter any existing Sᵢ's behavior if P(S') ∉ ⋃ C(Sᵢ) | Structural independence of skills |
| **Monotonicity** | If A passes validation and A' = A ∪ {S'}, then every dependency satisfied in A remains satisfied in A' | Consequence of additivity |

Acyclicity, contract completeness, and single responsibility are checked by tooling. Build idempotence is a property of the deterministic generator. Additivity and monotonicity follow from the model's design: skills have no shared mutable state and no implicit dependencies.

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

Skills and their compositions can be characterized as a category.

**The category Skill.** Define a category **Skill** where:
- **Objects** are elements of T — the I/O type names
- **Morphisms** are skills: S : C₁ × C₂ × ... × Cₙ → P
- **Composition** S₂ ∘ S₁ is defined when P(S₁) ∈ C(S₂) — the output of S₁ feeds an input of S₂

The composition satisfies four properties:

| Property | Statement | Practical consequence |
|----------|-----------|----------------------|
| **Associativity** | (S₃ ∘ S₂) ∘ S₁ = S₃ ∘ (S₂ ∘ S₁) | The order of grouping does not affect the pipeline |
| **Non-commutativity** | S₂ ∘ S₁ ≠ S₁ ∘ S₂ in general | Sequential order matters — `tdd-runner` before `review-commenter`, not the reverse |
| **Identity** | For every type T, an identity skill id_T : T → T exists (a pass-through that forwards its input unchanged) | Agents can include transparent relay skills |
| **Product** | S₁ ⊗ S₂ : C₁ × C₂ → P₁ × P₂ — parallel execution of independent skills | The `parallel` orchestration strategy is the tensor product |

The four orchestration strategies (section 2.3) correspond to categorical constructions:

| Orchestration | Construction | Notation |
|---------------|-------------|----------|
| `sequential` | Morphism composition | S₃ ∘ S₂ ∘ S₁ |
| `parallel` | Tensor product | S₁ ⊗ S₂ ⊗ S₃ |
| `parallel-then-merge` | Product then composition | merge ∘ (S₁ ⊗ S₂ ⊗ S₃) |
| `adaptive` | Coproduct (conditional choice) | S₁ + S₂ |

**The build functor.** A build target defines a functor B : **Skill** → **Target** that:
- Maps each type T ∈ T to its framework-specific representation (e.g., a section header in the generated prompt)
- Maps each morphism S to a generated artifact (a `.md` skill file, a `.mdc` rule, etc.)
- Preserves composition: B(S₂ ∘ S₁) = B(S₂) ∘ B(S₁) — the build of a pipeline equals the pipeline of builds

Build idempotence follows from the functor being deterministic: same input specification → same output artifacts.

**Honest limitations.** The category is *loose* in three ways:

1. **Non-deterministic morphisms.** Skills are executed by LLMs. The same skill applied to the same input may produce different outputs across invocations. The algebraic model captures *structural* composition, not *behavioral* determinism.

2. **Nominal type matching.** Types are strings, not schemas. Two skills using `review_comments` with different internal expectations will compose structurally but may fail semantically. The category verifies wiring, not meaning.

3. **Structure ≠ correctness.** The algebraic model guarantees that a well-formed agent has no structural defects — analogous to a well-typed program having no type errors. It does not guarantee that the agent produces correct, useful, or safe outputs. The gap between structural soundness and semantic correctness is inherent to any specification that runs on non-deterministic executors.

### 4.5 Linter Soundness

The linter is *sound* with respect to structural properties: when it reports no diagnostics, certain classes of defects are guaranteed absent.

**Theorem (Structural Soundness).** If `forgent lint` and `forgent doctor` report no diagnostics for an agent A = (S₁, ..., Sₙ, Cₐ, Pₐ, σ), then:

1. **I/O completeness.** For every skill Sᵢ ∈ A, for every type t ∈ C(Sᵢ), either there exists Sⱼ ∈ A such that P(Sⱼ) = t, or t ∈ Cₐ. Every skill's inputs are satisfied.

2. **Acyclicity.** The data flow graph G(A) contains no directed cycles. No skill transitively depends on its own output.

3. **Single responsibility.** For every skill Sᵢ ∈ A, Sᵢ produces exactly one type. No skill is a merged responsibility.

The proof follows from construction: the linter explicitly checks each property and reports a diagnostic for any violation. If no diagnostic is reported, no violation exists.

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

---

## 5. Design Rationale

The Skill Behavior Model applies established software design principles to agent engineering.

**Single Responsibility Principle.** Each skill produces exactly one output. This is enforced as a linter rule rather than a schema constraint — a skill with `produces: [lint_results, type_errors]` will parse but will be flagged as a violation. The rule forces decomposition: one `pr-reviewer` producing two outputs becomes two skills — `review-commenter` and `risk-scorer` — each testable and reusable independently.

**Explicit dependency injection.** A skill has no ambient access to shared state — its only inputs are what it explicitly declares in its `consumes` contract [5]. Data flow between skills is inferred from their `consumes`/`produces` interfaces and validated by the linter. No global state, no shared context outside the declared contract. The data flow graph is fully explicit and statically analyzable.

**Composition over Inheritance.** Agents are flat lists of skills — no "base agents", no inheritance, no override mechanisms. Behavior is assembled, not specialized. This eliminates the fragile base class problem: changing a skill affects only agents that explicitly include it.

**Tell, Don't Ask.** Skills declare what they produce. The framework reads the declarations and routes data [6]. No skill queries another skill's state — skills are decoupled from each other and only know their own contract.

**LLM Attention Optimization.** Generated output orders sections to exploit primacy and recency biases [7][8]. Guardrails go first (primacy), security goes last (recency). This is a generation concern — the YAML format is unordered, the generated artifacts are deliberately structured.

**Framework Independence.** Tool names are abstract behavioral capabilities. The mapping to concrete APIs happens at generation time. Skill authors never write framework-specific code; framework migrations are zero-cost at the specification level.

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

These specifications answer different questions about agent systems:

**AGENTS.md** [12] is a free-form markdown standard for guiding coding agents, stewarded by the Linux Foundation. It provides project-level context (build steps, conventions, architecture) but has no structured facets, no typed I/O contracts, and no composition model. It tells an agent *about the project* — the Skill Behavior Model tells an agent *what to do and under what constraints*.

**SOUL.md** [9] defines agent identity — personality, tone, values. The Skill Behavior Model defines agent capabilities. They answer different questions and are composable: an agent can have both a soul (who it is) and skills (what it can do).

**Open Agent Specification** [14] is a declarative, framework-agnostic YAML format for defining agent workflows, introduced by Oracle in October 2025. It models workflows as directed graphs of typed nodes (LLMNode, APINode, ToolNode) with explicit data flow edges — analogous to ONNX for ML models. The Skill Behavior Model operates at a different granularity: it defines the *behavioral content* of each node, not the *workflow between nodes*. The two are potentially complementary — skills could serve as the behavioral building blocks within an Agent Spec workflow. The NIST AI Agent Standards Initiative [16] is working toward standardizing similar concerns at the policy level.

**Programmatic frameworks** such as LangGraph and CrewAI define agents in code (Python); these frameworks are surveyed in [15]. The Skill Behavior Model defines agents in data (YAML). The declarative approach enables static analysis and non-programmer access to agent design. The trade-off is expressiveness — programmatic frameworks can encode error handling, conditional branching, loops, and state machines that a declarative format cannot.

**DSPy** [17] defines typed input/output signatures for LLM modules — `question -> answer` or `context, question -> rationale, answer`. This is the closest prior art to the Skill Behavior Model's `consumes`/`produces` pattern. The key difference is scope: DSPy signatures define I/O for individual LLM calls within a Python program and support automatic prompt optimization, while the Skill Behavior Model defines I/O for behavioral units that generate entire prompt specifications. DSPy optimizes prompts programmatically; the Skill Behavior Model composes them declaratively.

**Microsoft's Semantic Kernel** [18] and **AutoGen** [19] take different approaches to agent composition. Semantic Kernel defines "plugins" as typed function collections with input/output schemas — closer to API contracts than behavioral specifications. AutoGen defines multi-agent conversations with typed message protocols. Both are programmatic (Python/.NET) frameworks. The Skill Behavior Model shares the goal of structured I/O but differs in being declarative, framework-agnostic, and targeting prompt generation rather than runtime orchestration.

**ADL** [13] (Agent Description Language) is a declarative DSL for chatbot agents with typed slots and dialogue flows. It shares the goal of replacing free-form prompts with structured specifications, but targets conversational agents rather than coding agents. The Skill Behavior Model focuses on tool-using skills with I/O contracts rather than dialogue management.

**Communication protocols** (MCP, A2A, ACP) [10][11] define how agents discover each other, negotiate, and exchange messages. The Skill Behavior Model defines what each agent does internally. They operate at different layers and are composable: skills define behavior, protocols define communication.

---

## 9. Conclusion

The Skill Behavior Model brings to agent engineering what interfaces and design principles brought to software engineering: structured decomposition, explicit contracts, and static validation. Each skill is a single-responsibility unit with a declared I/O contract. Agents are flat compositions of skills. The format is framework-agnostic, LLM-agnostic, and statically validatable.

The format is deliberately minimal. It does not prescribe an implementation language, a runtime, or a specific LLM. It defines a behavioral contract — what an agent skill does, what it needs, what it produces, and what constraints it operates under. Implementations generate framework-native artifacts from this specification.

Future directions include new facets for emerging concerns (cost budgets, latency targets, human-in-the-loop gates as discussed in section 7.4), behavioral testing infrastructure, integration with agent communication protocols (MCP, A2A), and community-driven extension of the facet registry.

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
