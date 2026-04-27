# Runtime vs Text Experiment Report

## Experiment Parameters

| Parameter | Value |
|-----------|-------|
| Date | 2026-04-16 |
| Model | haiku |
| Tasks | 15 |
| Total runs | 15 |
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
| Completion rate | 100% | 100% | +0% | [+0.0%, +0.0%] |
| Step ordering | 100% | 100% | +0% | [+0.0%, +0.0%] |
| Guardrail adherence | 93% | 78% | -14% | [-25.0%, -2.2%] |
| Token usage | 1410 | 3503 | +2093 | [+1008, +3356] |

*Positive delta = runtime outperforms text. Negative token delta = runtime is more efficient.*

## Engineering Properties

These metrics measure structural advantages that text instructions *cannot* provide by construction.

| Property | Text | Runtime | Delta | 95% CI |
|----------|------|---------|-------|--------|
| Wall-clock time | 9544ms | 54488ms | +44944ms | [+31786, +56781] |
| LLM calls | 1 | 7 | +6 | |
| Cross-contamination | 0% | 0% (isolated) | | |

*Cross-contamination: Jaccard similarity between outputs of steps that should be independent.*
*Fault tolerance: % of complete runs that succeed when 20% of LLM calls fail randomly.*

## Harness Detection Rate

How many soft-harness (text) failures would the hard-harness (runtime) have caught?

| Failure Type | Text Failures | Caught by Runtime | Detection Rate |
|-------------|---------------|-------------------|----------------|
| Completion | 0 | 0 | 0% |
| Ordering | 0 | 0 | 0% |
| Guardrails | 4 | 0 | 0% |
| **Any failure** | **4** | **0** | **0%** |

*Detection rate: of all runs where the text condition failed, what % would the runtime harness have caught (i.e., runtime succeeded where text failed).*

## Per-Task Results

| Task | Text Complete | Runtime Complete | Text Order | Runtime Order |
|------|--------------|------------------|------------|---------------|
| code-review | Yes | Yes | 100% | 100% |
| doc-generation | Yes | Yes | 100% | 100% |
| refactoring | Yes | Yes | 100% | 100% |
| security-audit | Yes | Yes | 100% | 100% |
| test-generation | Yes | Yes | 100% | 100% |

## Cost-Efficiency

| Metric | Text | Runtime |
|--------|------|--------|
| Quality score (weighted) | 97.8% | 93.5% |
| Mean tokens/run | 1410 | 3503 |
| Efficiency (quality per 1k tokens) | 0.694 | 0.267 |

*Quality = 40% completion + 30% ordering + 30% guardrails. Efficiency = quality / (tokens/1000).*

## Per-Step Breakdown

| Step | Text Complete | Runtime Complete | Text Keywords | Runtime Keywords |
|------|-------------|-----------------|--------------|----------------|
| assemble-doc | 100% | 100% | 100% | 100% |
| identify-responsibilities | 100% | 100% | 100% | 50% |
| design-interfaces | 100% | 100% | 100% | 75% |
| extract-functions | 100% | 100% | 100% | 92% |
| identify-cases | 100% | 100% | 100% | 100% |
| write-table-tests | 100% | 100% | 83% | 83% |
| threat-model | 100% | 100% | 92% | 75% |
| write-overview | 100% | 100% | 100% | 100% |
| add-property-tests | 100% | 100% | 100% | 100% |
| identify-bugs | 100% | 100% | 100% | 80% |
| style-review | 100% | 100% | 92% | 58% |
| remediation | 100% | 100% | 83% | 67% |
| compose-pipeline | 100% | 100% | 100% | 92% |
| vulnerability-scan | 100% | 100% | 100% | 92% |
| generate-fix | 100% | 100% | 100% | 100% |
| analyze-api | 100% | 100% | 100% | 100% |
| write-examples | 100% | 100% | 100% | 100% |

*Sorted by text completion rate (lowest first = most fragile steps).*

## Stability (Standard Deviation across runs)

| Task | Text σ(completion) | RT σ(completion) | Text σ(ordering) | RT σ(ordering) | Text σ(guardrail) | RT σ(guardrail) |
|------|-------------------|-----------------|-----------------|---------------|------------------|----------------|
| code-review | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| doc-generation | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| refactoring | 0.000 | 0.000 | 0.000 | 0.000 | 0.192 | 0.000 |
| security-audit | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.144 |
| test-generation | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |

*Lower σ = more stable. Runtime σ(ordering) should be 0.000 (guaranteed by topological sort).*

## Difficulty Gradient

Does the runtime advantage grow with task complexity?

| Task | Steps | Completion Δ | Ordering Δ |
|------|-------|-------------|------------|
| security-audit | 3 | +0% | +0% |
| test-generation | 3 | +0% | +0% |
| code-review | 3 | +0% | +0% |
| doc-generation | 4 | +0% | +0% |
| refactoring | 4 | +0% | +0% |

Spearman ρ (steps vs completion Δ): **0.000**
Spearman ρ (steps vs ordering Δ): **0.000**

*Positive ρ = runtime advantage increases with complexity.*

## Convergence Speed

| Condition | Runs to converge (±1%) |
|-----------|------------------------|
| Text | 3 |
| Runtime | 3 |

*Fewer runs = more predictable behavior.*

## Output Length Consistency

| Condition | Mean CV (coefficient of variation) |
|-----------|------------------------------------|
| Text | 0.000 |
| Runtime | 0.177 |

*Lower CV = more consistent output lengths. Runtime isolates step context.*

## Guardrail Violation Severity

| Condition | Mean violations/failed run | Max violations |
|-----------|--------------------------|----------------|
| Text | 1.0 | 4 |
| Runtime | 1.0 | 5 |

