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
| Completion rate | 100% | 33% | -67% | [-86.7%, -40.0%] |
| Step ordering | 100% | 33% | -67% | [-86.7%, -40.0%] |
| Guardrail adherence | 95% | 31% | -64% | [-85.0%, -40.0%] |
| Token usage | 1477 | 1932 | +455 | [-1026, +2270] |

*Positive delta = runtime outperforms text. Negative token delta = runtime is more efficient.*

## Engineering Properties

These metrics measure structural advantages that text instructions *cannot* provide by construction.

| Property | Text | Runtime | Delta | 95% CI |
|----------|------|---------|-------|--------|
| Wall-clock time | 9990ms | 50390ms | +40400ms | [+29578, +52951] |
| LLM calls | 1 | 8 | +7 | |
| Cross-contamination | 0% | 0% (isolated) | | |

*Cross-contamination: Jaccard similarity between outputs of steps that should be independent.*
*Fault tolerance: % of complete runs that succeed when 20% of LLM calls fail randomly.*

## Harness Detection Rate

How many soft-harness (text) failures would the hard-harness (runtime) have caught?

| Failure Type | Text Failures | Caught by Runtime | Detection Rate |
|-------------|---------------|-------------------|----------------|
| Completion | 0 | 0 | 0% |
| Ordering | 0 | 0 | 0% |
| Guardrails | 3 | 0 | 0% |
| **Any failure** | **3** | **0** | **0%** |

*Detection rate: of all runs where the text condition failed, what % would the runtime harness have caught (i.e., runtime succeeded where text failed).*

## Per-Task Results

| Task | Text Complete | Runtime Complete | Text Order | Runtime Order |
|------|--------------|------------------|------------|---------------|
| refactoring | Yes | No | 100% | 0% |
| security-audit | Yes | No | 100% | 0% |
| test-generation | Yes | Yes | 100% | 100% |
| code-review | Yes | No | 100% | 0% |
| doc-generation | Yes | Yes | 100% | 100% |

## Cost-Efficiency

| Metric | Text | Runtime |
|--------|------|--------|
| Quality score (weighted) | 98.5% | 32.7% |
| Mean tokens/run | 1477 | 1932 |
| Efficiency (quality per 1k tokens) | 0.667 | 0.169 |

*Quality = 40% completion + 30% ordering + 30% guardrails. Efficiency = quality / (tokens/1000).*

## Per-Step Breakdown

| Step | Text Complete | Runtime Complete | Text Keywords | Runtime Keywords |
|------|-------------|-----------------|--------------|----------------|
| identify-bugs | 100% | 0% | 93% | 0% |
| write-overview | 100% | 100% | 100% | 100% |
| extract-functions | 100% | 100% | 100% | 100% |
| identify-cases | 100% | 100% | 100% | 100% |
| style-review | 100% | 0% | 100% | 0% |
| analyze-api | 100% | 100% | 100% | 100% |
| assemble-doc | 100% | 100% | 100% | 100% |
| identify-responsibilities | 100% | 100% | 100% | 100% |
| design-interfaces | 100% | 100% | 100% | 100% |
| compose-pipeline | 100% | 100% | 92% | 100% |
| vulnerability-scan | 100% | 0% | 100% | 0% |
| add-property-tests | 100% | 100% | 100% | 100% |
| generate-fix | 100% | 0% | 100% | 0% |
| threat-model | 100% | 0% | 75% | 0% |
| write-examples | 100% | 100% | 100% | 100% |
| remediation | 100% | 0% | 75% | 0% |
| write-table-tests | 100% | 100% | 100% | 100% |

*Sorted by text completion rate (lowest first = most fragile steps).*

## Stability (Standard Deviation across runs)

| Task | Text σ(completion) | RT σ(completion) | Text σ(ordering) | RT σ(ordering) | Text σ(guardrail) | RT σ(guardrail) |
|------|-------------------|-----------------|-----------------|---------------|------------------|----------------|
| code-review | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| doc-generation | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| refactoring | 0.000 | 0.577 | 0.000 | 0.577 | 0.000 | 0.385 |
| security-audit | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| test-generation | 0.000 | 0.577 | 0.000 | 0.577 | 0.000 | 0.577 |

*Lower σ = more stable. Runtime σ(ordering) should be 0.000 (guaranteed by topological sort).*

## Difficulty Gradient

Does the runtime advantage grow with task complexity?

| Task | Steps | Completion Δ | Ordering Δ |
|------|-------|-------------|------------|
| code-review | 3 | -100% | -100% |
| security-audit | 3 | -100% | -100% |
| test-generation | 3 | -67% | -67% |
| doc-generation | 4 | +0% | +0% |
| refactoring | 4 | -67% | -67% |

Spearman ρ (steps vs completion Δ): **0.761**
Spearman ρ (steps vs ordering Δ): **0.761**

*Positive ρ = runtime advantage increases with complexity.*

## Convergence Speed

| Condition | Runs to converge (±1%) |
|-----------|------------------------|
| Text | 3 |
| Runtime | 12 |

*Fewer runs = more predictable behavior.*

## Output Length Consistency

| Condition | Mean CV (coefficient of variation) |
|-----------|------------------------------------|
| Text | 0.000 |
| Runtime | 0.297 |

*Lower CV = more consistent output lengths. Runtime isolates step context.*

## Guardrail Violation Severity

| Condition | Mean violations/failed run | Max violations |
|-----------|--------------------------|----------------|
| Text | 1.0 | 3 |
| Runtime | 1.0 | 10 |

