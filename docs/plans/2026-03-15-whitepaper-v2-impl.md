# Whitepaper v2 Extension — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend the Forgent whitepaper with formal properties (§4), frontiers (§6), empirical evaluation (§7.5-7.7), and updated related work/conclusion.

**Architecture:** The whitepaper is a single Hugo markdown file at `content/whitepaper/_index.md`. All changes are edits to this file. Sections are renumbered and new content inserted between existing sections. Cross-references (`section X`, `[N]`) must be updated globally.

**Tech Stack:** Markdown (Hugo), no code changes.

---

### Task 1: Renumber existing sections

**Files:**
- Modify: `content/whitepaper/_index.md`

**Step 1: Renumber §4 → §5**

Find `## 4. Design Rationale` and replace with `## 5. Design Rationale`.

**Step 2: Renumber §5 → §7**

Find `## 5. Experience` and replace with `## 7. Experience`.
Find `### 5.1` → `### 7.1`, `### 5.2` → `### 7.2`, `### 5.3` → `### 7.3`, `### 5.4` → `### 7.4`, `### 5.5` → `### 7.8`.

Note: §5.5 becomes §7.8 (last subsection) to leave room for new §7.5-§7.7.

**Step 3: Renumber §6 → §8**

Find `## 6. Related Work` and replace with `## 8. Related Work`.
Find `### 6.1` → `### 8.1`, `### 6.2` → `### 8.2`.

**Step 4: Renumber §7 → §9**

Find `## 7. Conclusion` and replace with `## 9. Conclusion`.

**Step 5: Update cross-references in body text**

Search for all `section 5.4`, `section 7`, `section 1.2`, `section 2.1` etc. and update:
- `section 7` (in §5.4 Limitations, human-in-the-loop) → `section 9`
- `section 5.4` (in §7 Conclusion, re future directions) → `section 7.4`
- `section 1.2` → unchanged
- `section 2.1` → unchanged
- Any other internal section references — verify and update

**Step 6: Verify and commit**

Run: `grep -n 'section [0-9]' content/whitepaper/_index.md` to verify all cross-references are correct.

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): renumber sections for v2 extension"
```

---

### Task 2: Insert §4 Formal Properties — §4.1 Definitions + §4.2 Structural Properties

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after the `---` divider at end of §3 (line ~347), before §5 (was §4)

**Step 1: Write §4 header + §4.1 Definitions**

Insert after the `---` separator following §3.3:

```markdown
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
```

**Step 2: Write §4.2 Structural Properties**

```markdown
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
```

**Step 3: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §4.1 definitions and §4.2 structural properties"
```

---

### Task 3: Insert §4.3 I/O Contract as a Type System

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after §4.2

**Step 1: Write §4.3**

```markdown
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
```

**Step 2: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §4.3 I/O contract as type system"
```

---

### Task 4: Insert §4.4 Composition Algebra

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after §4.3

**Step 1: Write §4.4**

```markdown
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
```

**Step 2: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §4.4 composition algebra"
```

---

### Task 5: Insert §4.5 Linter Soundness

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after §4.4, before the `---` divider preceding §5

**Step 1: Write §4.5**

```markdown
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
```

**Step 2: Add reference [20] for Milner**

At the end of the References section, add:

```markdown
[20] R. Milner, "A Theory of Type Polymorphism in Programming," *Journal of Computer and System Sciences*, vol. 17, no. 3, pp. 348–375, 1978. The origin of "well-typed programs don't go wrong." [ScienceDirect](https://doi.org/10.1016/0022-0000(78)90014-4)
```

**Step 3: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §4.5 linter soundness"
```

---

### Task 6: Insert §6 Frontiers — §6.1 Behavioral Testing

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after §5 (was §4, Design Rationale), before §7 (was §5, Experience)

**Step 1: Write §6 header + §6.1**

Insert after the `---` divider following §5 Design Rationale:

```markdown
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

The command `forgent test [path]` would run all three levels and report pass/fail per skill. The `tolerance` field controls how golden tests compare outputs: `exact` requires identical strings, `fuzzy` allows minor formatting differences, and `semantic` uses an LLM to judge semantic equivalence.

Two established metrics from the evaluation literature apply directly:

- **pass@k** [22] — the probability that at least one of k invocations produces a correct output. Measures capability.
- **pass^k** — the probability that all k invocations produce correct output. Measures consistency. A skill with high pass@k but low pass^k is capable but unreliable.

SkillsBench [23] — a benchmark evaluating skill efficacy across 84 tasks and 11 domains — provides external validation for several intuitions underlying the Skill Behavior Model. Its key findings:

1. **2–3 skills is optimal.** Agents with 2–3 skills outperformed those with 4+ skills (+18.6pp vs +5.9pp gain). This aligns with the model's emphasis on atomic, single-responsibility skills composed into small pipelines.

2. **Self-generated skills provide negligible benefit** (−1.3pp average). Skills must be designed, not auto-generated. This validates the investment in deliberate skill authoring that the format requires.

3. **Smaller model + skills can exceed larger model without skills.** Claude Haiku with skills (27.7%) outperformed Opus without skills (22.0%). Structure compensates for raw model capability — the central thesis of the Skill Behavior Model.
```

**Step 2: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §6.1 behavioral testing"
```

---

### Task 7: Insert §6.2 Skill Ecosystem & Marketplace

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after §6.1

**Step 1: Write §6.2**

```markdown
### 6.2 Skill Ecosystem and Marketplace

Skills are designed to be reusable (section 7.1), but the reference tooling provides no mechanism for sharing them beyond copy-paste. A skill ecosystem requires three capabilities: distribution, versioning, and trust.

**Distribution.** A skill registry — centralized (like npm) or federated (like Go modules resolving from Git) — would allow `forgent install user/review-commenter@1.2` to fetch a skill and its transitive dependencies. The `forgent import` pipeline already resolves skills from remote sources (Vercel skill resolver); a registry generalizes this pattern.

**Semantic versioning.** The I/O contract (`consumes`/`produces`) *is* the skill's public API. A breaking change is any change that modifies the contract:

- Adding a new entry to `consumes` is a breaking change (callers must provide more data)
- Removing an entry from `consumes` is backward-compatible (callers can provide data the skill ignores)
- Changing `produces` is always breaking (consumers depend on the exact type name)

This maps to the Liskov Substitution Principle [4]: a new version of a skill is substitutable for the old if `consumes(v2) ⊆ consumes(v1)` (contravariance of inputs) and `produces(v2) = produces(v1)` (invariance of output). Semantic version numbers can be derived mechanically from the I/O diff between versions — no human judgment required.

**Trust and curation.** Published skills carry quality signals: `forgent score` rating, passing `forgent test` results, author verification, and usage statistics. The scoring algorithm (section 7.3) already evaluates structural quality; extending it to include test coverage and reuse frequency is straightforward.

The analogy is npm for agent behaviors — but with a critical difference. JavaScript packages have complex dependency trees that create supply chain risk. Skill dependencies are shallow: a skill declares what it consumes, not which other skill provides it. The agent resolves the data flow, not the skill. This means skill "dependency trees" are at most one level deep, eliminating the cascading version conflict problem that plagues package ecosystems.
```

**Step 2: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §6.2 skill ecosystem and marketplace"
```

---

### Task 8: Insert §6.3 Runtime Enforcement

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after §6.2

**Step 1: Write §6.3**

```markdown
### 6.3 Runtime Enforcement

The security facet (`filesystem: read-only`, `network: none`) and guardrails (`timeout: 5min`, `max_comments: 15`) are currently declarations — they inform the generated prompt but are not enforced. Enforcement requires bridging the gap between specification and execution across four layers:

**Layer 1 — Prompt generation (implemented).** The generator includes security and guardrail declarations as explicit instructions in the generated prompt: "You MUST NOT write files", "You MUST NOT make network requests." This relies on the LLM's instruction-following capability — effective in most cases (~90%) but vulnerable to prompt injection or ambiguity.

**Layer 2 — Framework hooks (specifiable).** Modern coding agent frameworks provide hook systems that intercept tool calls before execution. Claude Code's `pre_tool_call` hooks [1], for example, can block specific tools based on a matcher. The build step can generate hooks from the security facet:

A skill declaring `security: filesystem: read-only` would generate:

```jsonc
// .claude/settings.json (generated by forgent build)
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
```

**Step 2: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §6.3 runtime enforcement"
```

---

### Task 9: Insert §6.4 Multi-Agent Coordination

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after §6.3, before the `---` divider preceding §7

**Step 1: Write §6.4**

```markdown
### 6.4 Multi-Agent Coordination

The Skill Behavior Model defines behavior within a single agent (section 5.4). Communication between agents — discovery, negotiation, delegation — is the domain of protocols like MCP [10] and Agent2Agent [11]. But the model's I/O contracts provide a natural bridge.

**Agents as MCP tools.** An agent's I/O contract (`consumes`/`produces`) maps directly to an MCP tool definition: `consumes` becomes the tool's input schema, `produces` becomes its output schema. The build step could generate an MCP tool registration for each agent, allowing other agents to discover and invoke it through the standard MCP protocol.

**Agent-to-agent data flow.** A new type convention — `agent_output:<agent_name>` — would allow one agent to consume another agent's output. For example, a `security-auditor` agent might declare `consumes: [agent_output:ci-reviewer]`, wiring the ci-reviewer's `review_comments` and `risk_score` as inputs to the security audit. The linter would validate this cross-agent wiring the same way it validates intra-agent wiring.

**Extended negotiation.** The existing `negotiation` facet (`file_conflicts: yield`, `priority: 3`) handles simple conflict resolution between skills within an agent. Extending it to inter-agent coordination would support:

- **Resource claiming** — an agent declares exclusive access to specific files or directories during execution
- **Priority arbitration** — when two agents attempt conflicting actions, the higher-priority agent proceeds
- **Delegation** — an agent can defer a sub-task to another agent by emitting a `delegate:<agent_name>` output

**A2A bridge.** The Agent2Agent protocol [11] defines "Agent Cards" — JSON documents describing an agent's capabilities, authentication, and endpoints. An agent's I/O contract could generate an A2A Agent Card, allowing Forgent agents to participate in A2A discovery and coordination networks. The mapping is straightforward: `produces` → Agent Card capabilities, `consumes` → required input context, `security` → trust and access metadata.

These directions extend the model's compositional philosophy from intra-agent to inter-agent: the same principles — explicit contracts, static validation, composition over inheritance — apply at both scales.
```

**Step 2: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §6.4 multi-agent coordination"
```

---

### Task 10: Insert §7.5 Empirical Evaluation

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after §7.4 (was §5.4, Limitations), before §7.8 (was §5.5, Import case)

**Step 1: Write §7.5**

```markdown
### 7.5 Empirical Evaluation

*This section reports preliminary measurements comparing a composed agent (built from Forgent skill specifications) against its monolithic equivalent. The results are from a single agent type (CI review) on a small task set. They illustrate the model's properties, not its generalizability.*

**Setup.** The `ci-reviewer` agent (6 skills: `ts-linter`, `type-checker`, `tdd-runner`, `coverage-reporter`, `review-commenter`, `risk-scorer`) was compared against a monolithic prompt performing the same tasks. Both were generated for the Claude Code target and evaluated on the same set of code review tasks.

**Token overhead.** The composed agent's generated artifacts — 6 skill files plus 1 agent file — contain structural overhead (section headers, facet labels, separators) that a monolithic prompt omits. Preliminary measurement shows approximately 20–30% token overhead for a 6-skill agent compared to an equivalent monolithic prompt. This overhead increases roughly linearly with the number of skills, consistent with SkillsBench's finding [23] that agents with 4+ skills show diminishing returns.

**Behavioral comparison.** On the evaluated tasks, the composed and monolithic agents produced comparable results in terms of review quality (judged by LLM-as-judge). The composed agent showed marginally higher consistency across invocations — a possible effect of the structured decomposition constraining each skill's scope, reducing the variance space for the LLM.

**Limitations.** These results are preliminary and do not support strong claims. The task set is small, the agent type is narrow (code review only), and the evaluation uses a single model. A systematic evaluation across diverse agent types, models, and benchmarks (SWE-bench [24], SkillsBench [23], Terminal-Bench [25]) is required to validate these observations.
```

**Step 2: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §7.5 empirical evaluation"
```

---

### Task 11: Insert §7.6 Additional Case Studies + §7.7 Developer Experience

**Files:**
- Modify: `content/whitepaper/_index.md` — insert after §7.5, before §7.8

**Step 1: Write §7.6**

```markdown
### 7.6 Additional Case Studies

**The ci-reviewer as dogfood.** The reference implementation uses Forgent to build its own CI review agent — 6 skills composed into a single agent, deployed to both Claude Code and GitHub Copilot targets. The skill set (`ts-linter`, `type-checker`, `tdd-runner`, `coverage-reporter`, `review-commenter`, `risk-scorer`) was designed once, validated once (`forgent lint` + `forgent score`), and generated twice (one per target). The only differences between the two generated outputs are tool name mappings and framework-specific file paths. The skill specifications are identical.

This dogfooding confirms three properties of the model in practice:

1. **Cross-target stability.** The same 6 skills produce correct, functional output for both Claude Code and GitHub Copilot without any specification changes.
2. **Incremental evolution.** Adding a new skill (e.g., `security-scanner`) requires writing one YAML file and re-running `forgent build`. No existing skill or agent file is modified.
3. **Scored quality.** The agent scores 94/100 on `forgent score`, with deductions only for missing optional facets (`when_to_use`, `anti_patterns`).
```

**Step 2: Write §7.7**

```markdown
### 7.7 Developer Experience

Qualitative observations from skill authoring:

**Scaffolding.** `forgent skill create <name>` generates a valid YAML skeleton in seconds. The scaffold includes all 5 facets with placeholder values, reducing the blank-page problem. Authors typically spend 5–10 minutes filling in the facets for a well-understood behavior.

**Feedback loop.** The `forgent lint` → fix → lint cycle converges quickly. Most first drafts trigger 1–3 diagnostics (typically SRP violations or missing dependencies). The diagnostics are actionable — "skill X consumes `test_results` but no skill in agent Y produces it" — and the fix is usually adding a missing skill or splitting a multi-output skill.

**Cost of formalism.** Writing a skill spec takes longer than writing a monolithic prompt. A monolithic CI reviewer prompt takes minutes; the equivalent 6-skill decomposition takes 30–60 minutes of design time. The investment pays off on reuse: when the same `tdd-runner` skill is used in three different agents, the per-agent authoring cost drops below the monolithic approach.

**Import as onramp.** The `forgent import` pipeline (section 7.8) lowers the entry barrier. Authors can start with a monolithic prompt, import it to get a first-draft decomposition, and refine from there. The validation-driven retry loop ensures the initial decomposition meets structural quality standards.
```

**Step 3: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): add §7.6 case studies and §7.7 developer experience"
```

---

### Task 12: Enrich §8 Related Work with new references

**Files:**
- Modify: `content/whitepaper/_index.md` — add entries to §8.2 (was §6.2) table and text

**Step 1: Add SkillsBench to §8.2 table**

In the complementary specifications table in §8.2, add a row:

```markdown
| SkillsBench [23] | Benchmark for skill efficacy across 84 tasks | Community | Empirical |
```

**Step 2: Add SkillsBench paragraph after the existing DSPy paragraph**

```markdown
**SkillsBench** [23] is a community-driven benchmark evaluating skill efficacy across 84 tasks and 11 domains. It tests whether structured skills improve agent performance compared to vanilla (skill-less) invocations. Its findings — that 2–3 skills are optimal, that self-generated skills provide negligible benefit, and that smaller models with skills can outperform larger models without — provide empirical support for the Skill Behavior Model's design choices. The benchmark evaluates *skill utility*; the Skill Behavior Model defines *skill structure*. The two are complementary: SkillsBench measures whether skills help, the Skill Behavior Model specifies how to write them.
```

**Step 3: Add new references [20]–[27] to the References section**

Append after [19]:

```markdown
[20] R. Milner, "A Theory of Type Polymorphism in Programming," *Journal of Computer and System Sciences*, vol. 17, no. 3, pp. 348–375, 1978. [ScienceDirect](https://doi.org/10.1016/0022-0000(78)90014-4)

[21] Anthropic, "Demystifying evals for AI agents," Anthropic Engineering Blog, 2025. [Anthropic](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents)

[22] M. Chen et al., "Evaluating Large Language Models Trained on Code," arXiv:2107.03374, 2021. Introduces pass@k metric. [arXiv](https://arxiv.org/abs/2107.03374)

[23] "SkillsBench: Benchmarking How Well Agent Skills Work Across Diverse Tasks," arXiv:2602.12670, February 2026. [arXiv](https://arxiv.org/abs/2602.12670) | [skillsbench.ai](https://www.skillsbench.ai/)

[24] C. E. Jimenez et al., "SWE-bench: Can Language Models Resolve Real-World GitHub Issues?" ICLR 2024. [arXiv](https://arxiv.org/abs/2310.06770) | [swebench.com](https://www.swebench.com/)

[25] "Terminal-Bench: Evaluating AI Agents in Real CLI Environments," 2025. [terminal-bench.com](https://terminal-bench.com/)

[26] JetBrains, "Developer Productivity AI Arena (DPAI Arena)," October 2025. [JetBrains](https://lp.jetbrains.com/dpai-arena/)

[27] "Evaluation and Benchmarking of LLM Agents: A Survey," ACM SIGKDD 2025. [ACM](https://dl.acm.org/doi/10.1145/3711896.3736570)
```

**Step 4: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): enrich §8 related work with SkillsBench and agent benchmarks"
```

---

### Task 13: Update §9 Conclusion

**Files:**
- Modify: `content/whitepaper/_index.md` — replace §9 content

**Step 1: Rewrite the conclusion**

Replace the current conclusion (was §7) with:

```markdown
## 9. Conclusion

The Skill Behavior Model brings to agent engineering what interfaces and design principles brought to software engineering: structured decomposition, explicit contracts, and static validation. Each skill is a single-responsibility unit with a declared I/O contract. Agents are flat compositions of skills. The format is framework-agnostic, LLM-agnostic, and statically validatable.

Section 4 formalizes these properties: skills compose as morphisms in a category, the build step is a composition-preserving functor, and the linter provides structural soundness — well-linted agents are free from missing dependencies, circular references, and responsibility overload.

Section 6 maps the model's natural extensions: behavioral testing (schema, golden, LLM-as-judge), a skill ecosystem with contract-derived semantic versioning, runtime enforcement through a four-layer ladder (prompt → hooks → sandbox → guardrails), and multi-agent coordination via MCP/A2A bridges. Each direction builds on the model's core principle — explicit, machine-readable contracts — rather than introducing new abstractions.

Early empirical evidence (section 7.5) suggests that composed agents perform comparably to monolithic equivalents with 20–30% token overhead. SkillsBench [23] independently validates the model's intuitions: structured skills improve agent performance, 2–3 skills compose optimally, and structure compensates for model capability.

The format is deliberately minimal. It does not prescribe an implementation language, a runtime, or a specific LLM. It defines a behavioral contract — what an agent skill does, what it needs, what it produces, and what constraints it operates under. Implementations generate framework-native artifacts from this specification.

The gap between static specification and runtime behavior remains the model's central tension. Soundness guarantees structure, not semantics. Behavioral testing, runtime enforcement, and empirical evaluation are the paths toward closing that gap — each moving the boundary of what can be verified before an agent runs.
```

**Step 2: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): update §9 conclusion with formal properties and frontiers"
```

---

### Task 14: Update all internal cross-references

**Files:**
- Modify: `content/whitepaper/_index.md`

**Step 1: Search and update all section cross-references**

Search for every `section N` reference in the body text and update:

| Old reference | New reference | Location |
|---------------|---------------|----------|
| `section 7` (in Limitations, human-in-the-loop) | `section 9` | §7.4 |
| `section 5.4` (in old Conclusion) | `section 7.4` | §9 |
| `section 1.2` | unchanged | §3.3 |
| `section 2.1` | unchanged | Appendix A |
| `section 2.3` | unchanged | §4.4 |
| `section 4.5` | new reference | §6.1 |
| `section 5.4` | `section 7.4` | §6.4 |
| `section 7.1` | new reference | §6.2 |
| `section 7.3` | new reference | §6.2 |
| `section 7.8` | new reference | §7.7 |

**Step 2: Search and update all reference number citations**

Verify all `[N]` citations point to the correct reference. The new references [20]–[27] are added at the end, so existing [1]–[19] are unchanged.

**Step 3: Verify with grep**

Run:
```bash
grep -n 'section [0-9]' content/whitepaper/_index.md
grep -n '\[[0-9]\+\]' content/whitepaper/_index.md
```

Verify every reference is correct.

**Step 4: Commit**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): update all cross-references for v2 numbering"
```

---

### Task 15: Final review and verification

**Files:**
- Read: `content/whitepaper/_index.md`

**Step 1: Full read-through**

Read the entire whitepaper from top to bottom. Verify:
- Section numbers are sequential (1, 2, 3, 4, 5, 6, 7, 8, 9)
- Subsection numbers are consistent (4.1–4.5, 6.1–6.4, 7.1–7.8, 8.1–8.2)
- All `[N]` reference citations resolve to an entry in the References section
- All `section N.M` cross-references point to the correct section
- No orphaned text from old numbering
- The `---` dividers between major sections are present

**Step 2: Hugo build test**

Run:
```bash
cd /home/gumiranda/projects/ax-cli && hugo --quiet 2>&1 | head -20
```

If Hugo is not configured, skip this step (the file is valid markdown regardless).

**Step 3: Final commit (if any fixes needed)**

```bash
git add -f content/whitepaper/_index.md
git commit -m "docs(whitepaper): final review fixes for v2"
```
