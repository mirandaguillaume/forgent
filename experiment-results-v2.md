# Runtime vs Text Experiment Report

## Experiment Parameters

| Parameter | Value |
|-----------|-------|
| Date | 2026-04-15 |
| Model | haiku |
| Tasks | 390 |
| Total runs | 390 |
| Bootstrap resamples | 1000 |
| Confidence level | 95% |

## Methodology

Each task is executed under two conditions:

1. **Text instruction**: All steps given as a single monolithic prompt. The LLM must follow the step sequence, respect guardrails, and produce all outputs in one response.
2. **Runtime (DAG)**: Steps are encoded as DAG nodes with consumes/produces contracts. The Forgent engine enforces ordering, parallelizes independent steps, and provides per-step isolation.

The only variable is orchestration method. Same LLM, same model, same task definitions.

## Aggregate Results

| Metric | Text | Runtime | Delta | 95% CI |
|--------|------|---------|-------|--------|
| Completion rate | 15% | 9% | -6% | [-11.0%, -1.5%] |
| Step ordering | 14% | 9% | -5% | [-8.9%, -0.0%] |
| Guardrail adherence | 13% | 6% | -7% | [-10.6%, -3.3%] |
| Token usage | 635 | 647 | +12 | [-433, +580] |

*Positive delta = runtime outperforms text. Negative token delta = runtime is more efficient.*

## Engineering Properties

These metrics measure structural advantages that text instructions *cannot* provide by construction.

| Property | Text | Runtime | Delta | 95% CI |
|----------|------|---------|-------|--------|
| Wall-clock time | 4962ms | 18005ms | +13043ms | [+2782, +26007] |
| LLM calls | 1 | 3 | +2 | |
| Cross-contamination | 0% | 0% (isolated) | | |

*Cross-contamination: Jaccard similarity between outputs of steps that should be independent.*
*Fault tolerance: % of complete runs that succeed when 20% of LLM calls fail randomly.*

## Harness Detection Rate

How many soft-harness (text) failures would the hard-harness (runtime) have caught?

| Failure Type | Text Failures | Caught by Runtime | Detection Rate |
|-------------|---------------|-------------------|----------------|
| Completion | 330 | 0 | 0% |
| Ordering | 360 | 6 | 2% |
| Guardrails | 360 | 0 | 0% |
| **Any failure** | **360** | **0** | **0%** |

*Detection rate: of all runs where the text condition failed, what % would the runtime harness have caught (i.e., runtime succeeded where text failed).*

## Per-Task Results

| Task | Text Complete | Runtime Complete | Text Order | Runtime Order |
|------|--------------|------------------|------------|---------------|
| code-review | Yes | Yes | 100% | 100% |
| complex-migration | Yes | Yes | 71% | 100% |
| conflicting-constraints | No | No | 0% | 0% |
| parallel-analysis | No | No | 0% | 0% |
| refactoring | No | No | 0% | 0% |
| security-audit | No | No | 0% | 0% |
| test-generation | No | No | 0% | 0% |
| constraint-overload | No | No | 0% | 0% |
| doc-generation | No | No | 0% | 0% |
| gsd-implementation | No | No | 0% | 0% |
| iterative-refinement | No | No | 0% | 0% |
| long-input-analysis | No | No | 0% | 0% |
| multi-agent-review | No | No | 0% | 0% |

## Cost-Efficiency

| Metric | Text | Runtime |
|--------|------|--------|
| Quality score (weighted) | 14.1% | 8.3% |
| Mean tokens/run | 635 | 647 |
| Efficiency (quality per 1k tokens) | 0.223 | 0.128 |

*Quality = 40% completion + 30% ordering + 30% guardrails. Efficiency = quality / (tokens/1000).*

## Per-Step Breakdown

| Step | Text Complete | Runtime Complete | Text Keywords | Runtime Keywords |
|------|-------------|-----------------|--------------|----------------|
| identify-bugs | 100% | 100% | 97% | 87% |
| style-review | 100% | 100% | 90% | 78% |
| analyze-breaking | 100% | 100% | 100% | 100% |
| generate-fix | 100% | 100% | 100% | 100% |
| dependency-order | 100% | 100% | 100% | 100% |
| data-transform | 100% | 100% | 95% | 88% |
| rollback-plan | 100% | 100% | 76% | 70% |
| zero-downtime | 100% | 100% | 90% | 80% |
| validation-queries | 100% | 100% | 100% | 100% |
| runbook | 100% | 100% | 87% | 90% |

*Sorted by text completion rate (lowest first = most fragile steps).*

## Stability (Standard Deviation across runs)

| Task | Text σ(completion) | RT σ(completion) | Text σ(ordering) | RT σ(ordering) | Text σ(guardrail) | RT σ(guardrail) |
|------|-------------------|-----------------|-----------------|---------------|------------------|----------------|
| code-review | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| complex-migration | 0.000 | 0.407 | 0.073 | 0.407 | 0.107 | 0.232 |
| conflicting-constraints | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| constraint-overload | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| doc-generation | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| gsd-implementation | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| iterative-refinement | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| long-input-analysis | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| multi-agent-review | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| parallel-analysis | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| refactoring | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| security-audit | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| test-generation | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |

*Lower σ = more stable. Runtime σ(ordering) should be 0.000 (guaranteed by topological sort).*

## Difficulty Gradient

Does the runtime advantage grow with task complexity?

| Task | Steps | Completion Δ | Ordering Δ |
|------|-------|-------------|------------|
| code-review | 3 | +0% | +0% |
| complex-migration | 7 | -80% | -59% |

Spearman ρ (steps vs completion Δ): **0.000**
Spearman ρ (steps vs ordering Δ): **0.000**

*Positive ρ = runtime advantage increases with complexity.*

## Convergence Speed

| Condition | Runs to converge (±1%) |
|-----------|------------------------|
| Text | 367 |
| Runtime | 352 |

*Fewer runs = more predictable behavior.*

## Failure Mode Correlation (text condition)

| Pair | Pearson r |
|------|-----------|
| Completion ↔ Ordering | 0.677 |
| Completion ↔ Guardrails | 0.677 |
| Ordering ↔ Guardrails | 1.000 |

*High r = failure modes are correlated (same root cause). Low r = independent failure modes (harness must protect each separately).*

## Output Length Consistency

| Condition | Mean CV (coefficient of variation) |
|-----------|------------------------------------|
| Text | 0.000 |
| Runtime | 0.238 |

*Lower CV = more consistent output lengths. Runtime isolates step context.*

## Guardrail Violation Severity

| Condition | Mean violations/failed run | Max violations |
|-----------|--------------------------|----------------|
| Text | 1.0 | 10 |
| Runtime | 1.0 | 10 |

