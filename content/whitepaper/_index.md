---
title: "Composable Agent Skills: A SOLID Approach to AI Agent Engineering"
weight: 2
---

<div style="text-align: center; margin-bottom: 2rem;">
<em>A technical white paper on designing, validating, and deploying AI agents through composable skill specifications.</em>
<br><br>
<strong>Guillaume Miranda</strong> — March 2026
</div>

---

## Abstract

AI agents are becoming critical infrastructure in software engineering workflows. Yet most agent definitions today are monolithic prompt files — brittle, untestable, and locked to a single framework. This paper presents **Forgent**, an open-source CLI that treats agent design as a software engineering discipline. Agents are composed from **Skill Behaviors** — reusable units governed by 6 facets and validated against SOLID principles [1]. Skills are defined once in YAML and compiled to any target framework (Claude Code [2], GitHub Copilot [3], and more). We describe the model, the design constraints, the compilation pipeline, and the quality tools that make agent engineering reproducible.

---

## 1. The Problem

### 1.1 Monolithic Agent Prompts

The dominant pattern for defining AI agent behavior is a single long-form prompt file (`.md`, `.txt`, or inline string). A typical CI review agent might be defined as:

```markdown
You are a code reviewer. Read the diff, run tests, check types,
lint the code, write review comments, and assign a risk score.
Use bash for running commands and read for files...
```

This approach has three fundamental problems:

**Responsibility overload.** A single prompt handles linting, testing, reviewing, and scoring. When one concern changes (e.g., switching lint rules), the entire prompt must be re-validated.

**No reusability.** The linting behavior embedded in a CI agent cannot be extracted and reused by a security audit agent. Copy-paste becomes the de facto composition mechanism — a pattern that multi-agent orchestration research identifies as a key scaling bottleneck [4].

**Framework lock-in.** Prompts reference framework-specific tools (`Bash`, `Read` for Claude Code; `execute`, `search` for Copilot). Switching frameworks means rewriting every agent definition.

### 1.2 The Missing Abstraction

Software engineering solved analogous problems decades ago. Functions decompose monolithic programs. Interfaces enable composition. SOLID principles prevent design rot [1]. Agent engineering needs the same discipline — but the abstraction layer didn't exist.

---

## 2. The Forgent Approach

Forgent introduces a three-layer architecture:

```
┌─────────────────────────┐
│  Agent Composition      │  Orchestration of skills
│  (agents/*.agent.yaml)  │  sequential | parallel | adaptive
├─────────────────────────┤
│  Skill Behaviors        │  Reusable behavioral units
│  (skills/*.skill.yaml)  │  6 facets per skill
├─────────────────────────┤
│  Build Targets          │  Framework-specific output
│  (.claude/ | .github/)  │  Generated, never hand-written
└─────────────────────────┘
```

**Skills** are leaf nodes — each does exactly one thing and produces exactly one output. **Agents** are compositions — they wire skills into execution pipelines. **Build targets** are generated artifacts — compiled from the abstract specs into framework-native formats.

This separation means:
- Skills can be reused across agents
- Agents can be recomposed without touching skill definitions
- Framework migrations require zero changes to skill or agent specs

---

## 3. The Skill Behavior Model

Every skill is defined by **6 core facets** plus 3 optional documentation facets:

### 3.1 Core Facets

| # | Facet | Purpose | Example |
|---|-------|---------|---------|
| 1 | **Context** | I/O contract: what data flows in and out | `consumes: [git_diff, test_results]` → `produces: [review_comments]` |
| 2 | **Strategy** | Tools, approach, execution steps | `tools: [bash, read_file]`, `approach: test-first` |
| 3 | **Guardrails** | Rules, limits, constraints | `timeout: 5min`, `max_comments: 15` |
| 4 | **Dependencies** | Data flow between skills | `skill: tdd-runner provides test_results` |
| 5 | **Observability** | Traces and metrics | `trace_level: detailed`, `metrics: [tokens, latency]` |
| 6 | **Security** | Filesystem, network, secrets, sandboxing | `filesystem: read-only`, `network: none` |

### 3.2 Documentation Facets

| Facet | Purpose |
|-------|---------|
| **When to Use** | Triggers, exclusions, and especially-good-for conditions |
| **Anti-Patterns** | Common mistakes as excuse/reality pairs |
| **Examples** | Labeled code snippets demonstrating correct usage |

### 3.3 A Complete Skill

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

depends_on:
  - skill: tdd-runner
    provides: test_results
  - skill: ts-linter
    provides: lint_results

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

Note: `consumes` accepts a list (multiple inputs), but `produces` must contain **exactly one item**. This constraint — enforced at lint time — ensures each skill has a single responsibility.

---

## 4. SOLID Principles for Skills

Forgent enforces SOLID principles through lint rules, interface design, and architectural patterns.

### 4.1 Single Responsibility Principle (SRP)

**Rule:** Each skill produces exactly one output.

A skill that `produces: [lint_results, type_errors]` violates SRP — it's two skills pretending to be one. Forgent's linter catches this:

```
ERROR  singleProducesOutput  review-commenter
  Skill produces 2 outputs but should have exactly 1 (SRP)
```

Three lint rules enforce SRP:

| Rule | What it checks |
|------|---------------|
| `singleProducesOutput` | `produces` list has exactly 1 entry |
| `producesMatchesDescription` | Strategy approach doesn't contain conjunctions ("and", "then") suggesting multiple responsibilities |
| `skillNameMatchesOutput` | Skill name doesn't contain conjunction patterns (`-and-`, `-then-`) |

**Before SRP:** 1 skill `pr-reviewer` producing `[review_comments, risk_score]`

**After SRP:** 2 skills — `review-commenter` producing `[review_comments]` and `risk-scorer` producing `[risk_score]`

### 4.2 Open/Closed Principle (OCP)

**Rule:** The system is open for extension, closed for modification.

Lint rules self-register via Go's `init()` pattern:

```go
// rules_srp.go
func init() {
    Register(&singleProducesOutputRule{})
    Register(&producesMatchesDescriptionRule{})
}
```

Adding a new lint rule requires creating a new file with an `init()` function — zero modifications to existing code. Same pattern applies to build targets: implement the generator interfaces, register with `init()`, done.

### 4.3 Liskov Substitution Principle (LSP)

**Rule:** Any generator implementing the required interfaces can replace another.

The Claude generator doesn't support `GenerateInstructions` (Claude Code has no global instructions file). Instead of returning nil, it simply doesn't implement the `InstructionsGenerator` interface. Callers use type assertions:

```go
if ig, ok := gen.(spec.InstructionsGenerator); ok {
    content := ig.GenerateInstructions(skills, agents)
    // write content...
}
```

No nil checks, no special cases, no surprises.

### 4.4 Interface Segregation Principle (ISP)

**Rule:** No client should be forced to depend on methods it doesn't use.

The original monolithic `TargetGenerator` interface had 9 methods. Generators that didn't support instructions still had to implement `GenerateInstructions() *string` returning nil. After ISP:

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
    GenerateAgent(...) string
    AgentPath(name string) string
}

type InstructionsGenerator interface {  // Optional
    GenerateInstructions(...) string
    InstructionsPath() string
}
```

4 focused interfaces instead of 1 monolithic one. Generators implement only what they support.

### 4.5 Dependency Inversion Principle (DIP)

**Rule:** High-level modules depend on abstractions, not concrete implementations.

The loop detector needs to check if a skill has a timeout guardrail. Instead of depending on the concrete guardrail format:

```go
type GuardrailChecker interface {
    HasCapability(skill SkillBehavior, capability string) bool
}
```

The analyzer depends on this interface, not on the specific guardrail YAML structure. Testing is trivial — inject a mock checker.

---

## 5. Agent Composition

An agent is a named list of skills with an orchestration strategy:

```yaml
agent: ci-reviewer
skills:
  - ts-linter
  - type-checker
  - tdd-runner
  - coverage-reporter
  - review-commenter
  - risk-scorer
orchestration: sequential
description: "Runs linting, type-checking, tests, coverage, then reviews and scores"
```

### 5.1 Orchestration Strategies

| Strategy | Behavior | Use case |
|----------|----------|----------|
| `sequential` | Skills run in order, outputs feed forward | CI pipelines, review workflows |
| `parallel` | All skills run concurrently | Independent analyses |
| `parallel-then-merge` | Run in parallel, merge results | Multi-source aggregation |
| `adaptive` | Dynamic execution based on intermediate results | Complex decision trees |

### 5.2 Data Flow Graph

Dependencies between skills form a directed acyclic graph (DAG). Forgent's `doctor` command validates this graph:

```
ci-reviewer agent data flow:

ts-linter ──────────┐
type-checker        │
tdd-runner ─────────┼──→ review-commenter
coverage-reporter   │
                    └──→ risk-scorer
```

`ts-linter` produces `lint_results`. `review-commenter` consumes `lint_results`. The dependency is declared explicitly in the skill spec, and Forgent verifies that the producing skill is present in the agent's skill list.

### 5.3 Validation

The `doctor` command performs three checks:

| Check | What it catches |
|-------|----------------|
| **Missing dependencies** | A skill consumes data that no other skill in the agent produces |
| **Circular dependencies** | Skills A → B → C → A |
| **Unmet context** | A skill requires `codebase_index` but no enricher provides it |

---

## 6. Cross-Framework Compilation

### 6.1 Write Once, Deploy Anywhere

The `forgent build` command compiles YAML specs into framework-native output:

```bash
forgent build --target claude    # → .claude/skills/*/SKILL.md + .claude/agents/*.md
forgent build --target copilot   # → .github/skills/*/SKILL.md + .github/agents/*.agent.md
```

Both targets produce markdown skill files — but optimized for each framework's conventions:

| Aspect | Claude Code | GitHub Copilot |
|--------|------------|----------------|
| Agent extension | `.md` | `.agent.md` |
| Tool names | `Bash`, `Read`, `Grep` | `execute`, `read`, `search` |
| Global instructions | None (not needed) | `copilot-instructions.md` |
| Description limit | Unlimited | 1024 chars |

### 6.2 Skill File Structure

Generated `SKILL.md` files use a deliberate ordering based on LLM attention patterns. Research by Liu et al. [5] demonstrated that language models perform best when critical information is placed at the beginning or end of the input context, with significant performance degradation for information in the middle — the "lost in the middle" effect. Subsequent work confirmed that serial position effects (primacy and recency biases) are widespread across LLM architectures [6]:

1. **Guardrails** first — primacy bias ensures constraints are read and remembered
2. **Context, Dependencies, Strategy** in the middle — the execution plan
3. **Security** last — recency bias keeps access controls top of mind

### 6.3 Adding a New Target

```go
package crewai

import "github.com/mirandaguillaume/forgent/pkg/spec"

type CrewAIGenerator struct{}

func (g *CrewAIGenerator) Target() string          { return "crewai" }
func (g *CrewAIGenerator) DefaultOutputDir() string { return "crewai_output" }
func (g *CrewAIGenerator) ContextDir() string       { return "" }
// ... implement SkillGenerator and AgentGenerator

func init() {
    spec.Register("crewai", func() spec.Generator { return &CrewAIGenerator{} })
}
```

One file, one `init()`, zero modifications to existing code.

---

## 7. Quality Assurance Pipeline

### 7.1 Lint

Static analysis with 8 rules (extensible via OCP registry):

| Rule | Severity | What it checks |
|------|----------|---------------|
| `singleProducesOutput` | Error | `produces` has exactly 1 item |
| `producesMatchesDescription` | Error | Approach doesn't suggest multiple responsibilities |
| `skillNameMatchesOutput` | Error | Name doesn't contain conjunction patterns |
| `noEmptyTools` | Warning | Strategy has at least one tool |
| `hasGuardrails` | Warning | Guardrails list is not empty |
| `securityNeedsGuardrails` | Warning | Skills with write access or network have guardrails |
| `observableOutputs` | Info | Metrics reference produced outputs |
| `hasWhenToUse` | Info | When-to-use facet is populated |

```bash
$ forgent lint skills/
✓ 6 skills checked: 0 errors, 0 warnings, 6 info
```

### 7.2 Score

Rates design quality on a 0-100 scale across 5 weighted facets:

| Facet | Weight | What it measures |
|-------|--------|-----------------|
| Strategy | 25% | Tools defined, steps specified, approach clarity |
| Guardrails | 25% | Rules present, timeouts, limits |
| Observability | 20% | Trace level, metrics coverage |
| Security | 20% | Least privilege (restrictive defaults score higher) |
| Context | 10% | I/O contract completeness |

```bash
$ forgent score skills/
  review-commenter    87/100
  risk-scorer         82/100
  ts-linter           76/100
  ...
  ci-reviewer (agent) 100/100
```

### 7.3 Doctor

Full diagnostic combining lint, dependency analysis, and loop detection:

```bash
$ forgent doctor
Health Score: 100/100
  ✓ Lint: 0 errors
  ✓ Dependencies: all satisfied
  ✓ Loops: no circular risks detected
```

### 7.4 Bench

Measures how well the generated index helps an LLM navigate a codebase:

**Proxy level** — Checks what percentage of source files are reachable from the generated index without calling an LLM:

```bash
$ forgent bench .
Source files: 84
Sampled: 84
Reachable: 84 (100.0%)
Index: 18 entries, 1712 bytes
```

**Agent level** — Sends navigation tasks to a real LLM and measures hit rate:

```bash
$ forgent bench . --level agent
Tasks: 20 | Hits: 18 | Misses: 2 | Hit Rate: 90.0%
Avg Latency: 1.2s | Total Cost: $0.04
```

---

## 8. Comparison with Other Approaches

| Approach | Reusability | Validation | Multi-framework | Composition |
|----------|:-----------:|:----------:|:---------------:|:-----------:|
| Raw prompt files | None | None | No | Copy-paste |
| SOUL.md | Agent-level | None | Partial | No skills |
| Framework configs | Per-framework | Basic | No | Framework-specific |
| LangGraph/CrewAI | Code-level | Runtime | Per-framework | Programmatic |
| **Forgent** | **Skill-level** | **Static + SOLID** | **Yes** | **Declarative YAML** |

**vs. Raw prompts:** Forgent adds structure, validation, and reusability. A lint rule catches what a prompt review might miss.

**vs. SOUL.md:** SOUL.md [7] defines agent identity (personality, tone, values). Forgent defines agent capabilities (what it can do, with what constraints). They're complementary — SOUL.md answers "who is this agent?", Forgent answers "what can it do and how?".

**vs. Framework-specific configs:** Forgent's YAML specs are framework-agnostic. Switching from Claude Code to GitHub Copilot is `--target copilot`, not a rewrite.

**vs. Programmatic frameworks:** LangGraph and CrewAI define agents in code (Python) [8]. Forgent defines agents in data (YAML). The declarative approach enables static analysis, cross-framework compilation, and non-programmer access to agent design.

---

## 9. Getting Started

### 9.1 Install

```bash
go install github.com/mirandaguillaume/forgent/cmd/forgent@latest
```

### 9.2 Initialize a Project

```bash
forgent init
```

### 9.3 Create a Skill

```bash
forgent skill create my-analyzer --tools bash,read_file,grep
```

Edit the generated YAML. Remember: **one `produces` output per skill**.

### 9.4 Compose an Agent

Create `agents/my-pipeline.agent.yaml`:

```yaml
agent: my-pipeline
skills: [my-analyzer, my-reporter]
orchestration: sequential
description: "Analyze and report"
```

### 9.5 Validate and Build

```bash
forgent lint                    # Check for design issues
forgent score                   # Rate design quality
forgent doctor                  # Full diagnostic
forgent build --target claude   # Generate for Claude Code
forgent build --target copilot  # Generate for GitHub Copilot
```

---

## 10. Conclusion

Agent engineering needs the same rigor we apply to software engineering. Forgent brings that rigor through:

1. **A structured model** — 6 facets that cover every aspect of agent behavior
2. **SOLID enforcement** — lint rules that catch design violations before deployment
3. **Declarative composition** — agents as YAML, not code
4. **Framework independence** — write once, compile to any target
5. **Static analysis** — lint, score, doctor, bench — before your agent runs a single prompt

The result is agent definitions that are reusable, testable, and maintainable — the same properties we demand from production code.

---

<div style="text-align: center; margin-top: 2rem;">

**Forgent** — [github.com/mirandaguillaume/forgent](https://github.com/mirandaguillaume/forgent)

*Forge agents from composable skill specs.*

</div>

---

## References

[1] R. C. Martin, "Design Principles and Design Patterns," 2000. Later expanded in *Clean Architecture: A Craftsman's Guide to Software Structure and Design*, Prentice Hall, 2017. The SOLID acronym was coined by M. Feathers circa 2004. [Wikipedia](https://en.wikipedia.org/wiki/SOLID)

[2] Anthropic, "Equipping agents for the real world with Agent Skills," Anthropic Engineering Blog, 2025. Agent Skills specification and Claude Code skills documentation. [Anthropic](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills) | [Docs](https://code.claude.com/docs/en/skills)

[3] GitHub, "Custom agents for GitHub Copilot," GitHub Changelog, October 2025. Custom agent `.agent.md` specification and Copilot coding agent documentation. [GitHub Blog](https://github.blog/changelog/2025-10-28-custom-agents-for-github-copilot/) | [Docs](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-custom-agents)

[4] "The Orchestration of Multi-Agent Systems: Architectures, Protocols, and Enterprise Adoption," arXiv:2601.13671, January 2026. Survey of orchestration patterns and composition challenges in multi-agent systems. [arXiv](https://arxiv.org/abs/2601.13671)

[5] N. F. Liu, K. Lin, J. Hewitt, A. Paranjape, M. Bevilacqua, F. Petroni, and P. Liang, "Lost in the Middle: How Language Models Use Long Contexts," *Transactions of the Association for Computational Linguistics*, vol. 12, pp. 157–173, 2024. [arXiv](https://arxiv.org/abs/2307.03172) | [ACL Anthology](https://aclanthology.org/2024.tacl-1.9/)

[6] "Serial Position Effects of Large Language Models," arXiv:2406.15981, 2024. Confirms primacy and recency biases across multiple LLM architectures and task types. [arXiv](https://arxiv.org/abs/2406.15981)

[7] A. Mars, "soul.md — The best way to build a personality for your agent," GitHub, 2025. Framework for defining AI agent identity through structured markdown files. [GitHub](https://github.com/aaronjmars/soul.md) | [soul.md](https://soul.md/)

[8] "Agentic AI: A Comprehensive Survey of Architectures, Applications, and Future Directions," arXiv:2510.25445, 2025. Comprehensive survey covering LangGraph, CrewAI, AutoGen, and other programmatic agent frameworks. [arXiv](https://arxiv.org/abs/2510.25445)
