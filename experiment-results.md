# Runtime vs Text Experiment Report

## Experiment Parameters

| Parameter | Value |
|-----------|-------|
| Date | 2026-04-14 |
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
| Completion rate | 93% | 100% | +7% | [+4.9%, +10.0%] |
| Step ordering | 96% | 100% | +4% | [+2.8%, +4.5%] |
| Guardrail adherence | 87% | 93% | +6% | [+3.4%, +8.0%] |
| Token usage | 3723 | 11698 | +7975 | [+6919, +8999] |

*Positive delta = runtime outperforms text. Negative token delta = runtime is more efficient.*

## Engineering Properties

These metrics measure structural advantages that text instructions *cannot* provide by construction.

| Property | Text | Runtime | Delta | 95% CI |
|----------|------|---------|-------|--------|
| Wall-clock time | 25993ms | 79761ms | +53768ms | [+46591, +60739] |
| LLM calls | 1 | 5 | +4 | |
| Cross-contamination | 0% | 0% (isolated) | | |

*Cross-contamination: Jaccard similarity between outputs of steps that should be independent.*
*Fault tolerance: % of complete runs that succeed when 20% of LLM calls fail randomly.*

## Per-Task Results

| Task | Text Complete | Runtime Complete | Text Order | Runtime Order |
|------|--------------|------------------|------------|---------------|
| gsd-implementation | Yes | Yes | 100% | 100% |
| iterative-refinement | No | Yes | 80% | 100% |
| long-input-analysis | Yes | Yes | 100% | 100% |
| security-audit | Yes | Yes | 100% | 100% |
| complex-migration | Yes | Yes | 86% | 100% |
| conflicting-constraints | Yes | Yes | 100% | 100% |
| constraint-overload | Yes | Yes | 100% | 100% |
| multi-agent-review | Yes | Yes | 100% | 100% |
| parallel-analysis | Yes | Yes | 100% | 100% |
| refactoring | Yes | Yes | 100% | 100% |
| test-generation | Yes | Yes | 100% | 100% |
| code-review | Yes | Yes | 100% | 100% |
| doc-generation | Yes | Yes | 100% | 100% |

