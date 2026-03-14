# Whitepaper v2 — Extension Design

**Date:** 2026-03-15
**Approach:** Extend the existing whitepaper with new sections (no restructure)
**Level:** Semi-formal (DSPy-style — precise definitions, stated properties, light notation, no full proofs)

---

## Goals

1. **Formal foundations** — give the Skill Behavior Model a semi-formal algebraic grounding
2. **Vision étendue** — push toward frontiers: behavioral testing, marketplace, runtime enforcement, multi-agent
3. **Empirical evaluation** — benchmark composed vs monolithic agents on existing benchmarks

---

## Structure Changes

```
§1  Introduction                              [existing — no changes]
§2  The Skill Behavior Model                  [existing — no changes]
§3  Portability                               [existing — no changes]

§4  Formal Properties                         [NEW]
    §4.1 Definitions
    §4.2 Structural Properties
    §4.3 I/O Contract as a Type System
    §4.4 Composition Algebra
    §4.5 Linter Soundness

§5  Design Rationale                          [existing §4 — renumbered]

§6  Frontiers                                 [NEW]
    §6.1 Behavioral Testing
    §6.2 Skill Ecosystem & Marketplace
    §6.3 Runtime Enforcement
    §6.4 Multi-Agent Coordination

§7  Experience                                [existing §5 — renumbered + enriched]
    §7.1–§7.4                                 [existing subsections]
    §7.5 Empirical Evaluation                 [NEW]
    §7.6 Additional Case Studies              [NEW]
    §7.7 Developer Experience                 [NEW]
    §7.8 Illustrative Case: Import            [existing §5.5 — renumbered]

§8  Related Work                              [existing §6 — renumbered + enriched]
§9  Conclusion                                [existing §7 — renumbered + enriched]

Appendix A — I/O Type Reference              [existing — no changes]
References                                    [existing + ~10 new refs]
```

---

## §4 Formal Properties — Detailed Design

### §4.1 Definitions

Formal primitives:

- **Skill** `S = (C, P, F)` where `C = consumes` (set of type names), `P = produces` (singleton), `F = facets` (strategy, guardrails, observability, security, negotiation)
- **Agent** `A = (S₁, ..., Sₙ, Cₐ, Pₐ, σ)` where `Cₐ` = agent consumes, `Pₐ` = agent produces, `σ` = orchestration strategy
- **Data flow graph** `G(A)` = directed graph where edges represent consumes/produces relations between skills

### §4.2 Structural Properties

| Property | Intuition | Verified by |
|----------|-----------|-------------|
| **Acyclicity** | DAG has no cycles → no structural deadlock | `forgent doctor` — cycle detection |
| **Contract completeness** | Every `consumes` is satisfied by a `produces` or the agent's `consumes` | `forgent lint` — unmet dependencies |
| **Single responsibility** | `|P| = 1` for every skill | Lint rule SRP |
| **Build idempotence** | `build(spec) = build(spec)` — same input → same output | Deterministic generator property |
| **Additivity** | Adding skill S' doesn't modify existing skills' behavior if S' doesn't produce anything they consume | Composition without side-effects |
| **Monotonicity** | Adding a skill can only satisfy dependencies, never break them | Consequence of additivity |

### §4.3 I/O Contract as a Type System

- `consumes`/`produces` = input/output types
- The linter = type checker
- "Well-typed skills don't go wrong" (Milner analogy)
- Limitations: nominal types (string matching), not structural — linter verifies wiring, not semantics
- Gap: two skills can use the same type name with different expectations

### §4.4 Composition Algebra

**Category Skill:**
- Objects: I/O type names (`git_diff`, `lint_results`, `review_comments`, ...)
- Morphisms: skills `S : C₁ × C₂ × ... × Cₙ → P` (multi-input, single-output)
- Composition: `S₂ ∘ S₁` defined when `produces(S₁) ∈ consumes(S₂)`

**Composition properties:**

| Property | Statement | Practical consequence |
|----------|-----------|----------------------|
| Associativity | `(S₃ ∘ S₂) ∘ S₁ = S₃ ∘ (S₂ ∘ S₁)` | Grouping order doesn't affect result |
| Non-commutativity | `S₂ ∘ S₁ ≠ S₁ ∘ S₂` in general | Sequential order matters |
| Identity | Identity skill `id_T : T → T` exists for every type T | Transparent pass-through possible |
| Product | `S₁ ⊗ S₂ : C₁ × C₂ → P₁ × P₂` — parallel execution | `parallel` orchestration = tensor product |

**Orchestrations as categorical constructions:**

| Orchestration | Construction | Notation |
|---------------|-------------|----------|
| `sequential` | Morphism composition | `S₃ ∘ S₂ ∘ S₁` |
| `parallel` | Tensor product | `S₁ ⊗ S₂ ⊗ S₃` |
| `parallel-then-merge` | Product + composition | `merge ∘ (S₁ ⊗ S₂ ⊗ S₃)` |
| `adaptive` | Conditional coproduct | `S₁ + S₂` (dynamic choice) |

**Build functor** `B : Skill → Target`:
- Maps each I/O type to its target framework representation
- Maps each morphism (skill) to a generated prompt file
- Preserves composition: `B(S₂ ∘ S₁) = B(S₂) ∘ B(S₁)` — pipeline build = build of pipeline
- Build idempotence follows: the functor is deterministic

**Honest limitations:**
- The category is "loose" — morphisms are LLM transformations, not pure functions (non-determinism)
- Types are nominal (string matching), not structural
- Static verification guarantees structural coherence (wiring), not semantic correctness (output content)
- Algebraic model captures structure, not runtime behavior — analogous to type systems: well-typed ≠ correct

### §4.5 Linter Soundness

**Definition:** The linter is *sound* if: when it reports no errors, certain classes of defects are guaranteed absent.

**Soundness theorem (structural).** If `forgent lint` and `forgent doctor` report no diagnostics for agent `A`, then:
1. **I/O completeness** — For every skill `Sᵢ ∈ A`, for every `t ∈ consumes(Sᵢ)`, there exists either `Sⱼ ∈ A` such that `produces(Sⱼ) = t`, or `t ∈ consumes(A)`
2. **Acyclicity** — The dependency graph `G(A)` is a DAG
3. **Single responsibility** — `|produces(Sᵢ)| = 1` for every skill `Sᵢ`

**What soundness does NOT guarantee:**

| Not guaranteed | Why |
|----------------|-----|
| Semantic correctness | `review_comments` could contain anything — the type is a name, not a schema |
| Output quality | A "sound" skill can produce mediocre output (LLM non-determinism) |
| Termination | The LLM can run indefinitely — the format declares `timeout` but doesn't enforce it |
| Runtime security | `filesystem: read-only` is declarative — the framework may ignore the constraint |
| I/O injection absence | A malicious skill can produce output that manipulates downstream skills |

**Completeness.** The linter is NOT complete — there are structural defects it doesn't detect (e.g., empty `approach` field). Completeness is asymptotic: each new lint rule reduces the gap.

---

## §6 Frontiers — Detailed Design

### §6.1 Behavioral Testing

Three levels of testing for skills:

| Level | Mechanism | What it verifies |
|-------|-----------|------------------|
| **Schema testing** | Assertions on output structure (JSON schema, regex, format checks) | Skill produces output in correct format |
| **Golden testing** | Fixed input → output compared to approved reference (with semantic tolerance) | Behavioral stability across versions |
| **LLM-as-judge** | A second LLM evaluates output quality against criteria declared in the skill | Semantic quality (relevance, exhaustiveness, actionability) |

YAML extension:
```yaml
skill: review-commenter
testing:
  schema: review-comments.schema.json
  golden:
    - input: fixtures/simple-diff.txt
      expected: fixtures/simple-review.golden.md
      tolerance: semantic  # exact | fuzzy | semantic
  judge:
    criteria:
      - "Comments are actionable (not vague)"
      - "Comments reference specific lines"
      - "No false positives on style-only changes"
```

Command: `forgent test [path]` — run all three levels, report pass/fail.

Reference existing benchmarks and evaluation methodologies:
- SkillsBench (arXiv:2602.12670) — benchmark for skill efficacy evaluation
- Anthropic's agent eval recommendations — code/model/human graders
- pass@k and pass^k metrics for consistency measurement

### §6.2 Skill Ecosystem & Marketplace

- **Registry** — centralized or federated index of published, versioned skills
- **Resolution** — `forgent install user/review-commenter@1.2` or git-based resolution
- **Semantic versioning** — breaking change = change in `consumes`/`produces` (the I/O contract IS the API)
- **Compatibility** — skill v2 compatible with v1 if `consumes(v2) ⊆ consumes(v1)` and `produces(v2) = produces(v1)` (input contravariance, output covariance — exactly like LSP)
- **Trust & curation** — quality scores (`forgent score`), verified authors, passing tests

Analogy: npm/crates.io/PyPI for agent behaviors. Semantic versioning directly derivable from I/O contracts.

### §6.3 Runtime Enforcement

Four enforcement layers:

| Layer | Mechanism | Status |
|-------|-----------|--------|
| **Generation** (L1) | Generated prompt includes constraints as explicit instructions | Already implemented |
| **Framework hooks** (L2) | Use native framework hooks to validate tool calls against security facet | Specifiable now |
| **Sandbox** (L3) | OS-level enforcement — containers, WASM, seccomp profiles generated from security facet | Future |
| **Guardrails runtime** (L4) | Post-execution validation of guardrails (timeout watchdog, output validators) | Specifiable now |

**Concrete YAML spec for hooks generation (L2):**

The build step generates framework hooks from the security facet:
```yaml
# Skill declaration
security:
  filesystem: read-only
  network: none
```

Generates for Claude Code:
```jsonc
// .claude/settings.json
{
  "hooks": {
    "pre_tool_call": [{
      "matcher": { "tool_name": "Write" },
      "command": "echo 'BLOCK: skill review-commenter has filesystem: read-only' && exit 1"
    }, {
      "matcher": { "tool_name": "WebFetch" },
      "command": "echo 'BLOCK: skill review-commenter has network: none' && exit 1"
    }]
  }
}
```

**Concrete YAML spec for guardrails enforcement (L4):**
```yaml
guardrails:
  - timeout: 5min
  - max_comments: 15
  - no_approve_without_tests
```

Generates:
- Timeout watchdog (kill process after 5min)
- Post-hook that validates output comment count ≤ 15
- Post-hook that checks test results presence before approval

### §6.4 Multi-Agent Coordination

- Agent exposes its I/O contract (`consumes`/`produces`) as an MCP tool interface
- Agent can consume outputs from another agent via `agent_output:<agent_name>` type
- `negotiation` facet extends toward coordination protocol:
  - Claim/release of resources
  - Priority-based arbitration
  - Delegation (agent passes sub-task to another)
- A2A bridge: agent I/O contract generates an Agent Card for A2A discovery

---

## §7 Experience — Enrichments

### §7.5 Empirical Evaluation

Evaluate composed agent vs monolithic equivalent on existing benchmarks:
- Take ci-reviewer composed (6 skills) and monolithic equivalent
- Run on same tasks (subset of SWE-bench or SkillsBench)
- Compare: pass@k, token usage, consistency (pass^k)
- Same model, same tasks, fair comparison
- Report honestly with limitations (small N, single agent type)

Token overhead becomes a sub-result of the evaluation, not a standalone section.

### §7.6 Additional Case Studies

1. **ci-reviewer dogfood** — Forgent's own 6-skill agent, deployed to Claude Code and Copilot. Factoring, scoring, cross-target reuse.
2. **Open-source agent import** — Take a public `.agent.md`, import with `forgent import`, show decomposition. Before/after comparison.

### §7.7 Developer Experience

Qualitative observations on authoring experience:
- Scaffolding time (with `forgent skill create`)
- Feedback loop (lint → fix → lint)
- Cost of formalism vs. reuse benefit
- Learning curve observations

---

## §8 Related Work — Additions

New references to add:
- SkillsBench (arXiv:2602.12670) — skill efficacy benchmark
- SWE-bench / SWE-bench Verified — coding agent benchmark
- Terminal-Bench — CLI automation benchmark
- DPAI Arena (JetBrains) — developer productivity benchmark
- Anthropic agent eval guide — grader types and methodology
- Survey: "Evaluation and Benchmarking of LLM Agents" (KDD 2025)

---

## §9 Conclusion — Updates

Strengthen the conclusion with:
- Reference to formal properties (§4) as contribution
- Reference to SkillsBench findings validating the model's intuitions
- Updated future directions referencing concrete frontier designs (§6)

---

## Implementation Notes

- Existing sections (§1–§3) remain untouched
- Existing §4 (Design Rationale) renumbered to §5
- Existing §5 (Experience) renumbered to §7, subsections renumbered, new subsections added
- Existing §6 (Related Work) renumbered to §8, enriched
- Existing §7 (Conclusion) renumbered to §9, enriched
- All internal cross-references must be updated
- Reference numbering must be updated for new citations
