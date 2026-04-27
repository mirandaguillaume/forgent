# Runtime vs Text Experiment — Methodology

## Research Question

Does structural enforcement via a DAG runtime outperform text-instruction-only orchestration on complex multi-step coding tasks?

## Design

**Controlled experiment** with two conditions:

1. **Text instruction (control):** All task steps are given as a single monolithic prompt. The LLM must follow the step sequence, respect guardrails, and produce all outputs in one response.

2. **Runtime DAG (treatment):** Steps are encoded as DAG nodes with typed consumes/produces contracts. The Forgent engine enforces ordering via topological sort, parallelizes independent steps, and provides per-step isolation.

**Controlled variable:** Orchestration method (text vs DAG).
**Held constant:** LLM model, provider, temperature, task definitions, guardrail constraints.

## Task Definitions

Tasks are defined as YAML files in `tasks/`. Each task has:
- Multiple ordered steps with explicit I/O contracts
- Guardrail constraints (positive: must include; negative `!`: must not include)
- Expected keywords per step for completion verification

Categories: code-review, documentation, security-audit, refactoring, test-generation.

## Metrics

| Metric | Definition | Measurement |
|--------|-----------|-------------|
| **Task completion rate** | % of tasks where all steps produced output | Check each step's output for presence and keywords |
| **Step ordering accuracy** | % of steps executed in correct order | Compare step sequence in response against expected order |
| **Guardrail adherence** | % of tasks where all constraints respected | Check response against positive/negative constraints |
| **Token usage** | Total tokens consumed | Estimate from character count (4 chars/token) |

## Statistical Analysis

- **Bootstrap confidence intervals** (1000 resamples, fixed seed=42 for reproducibility)
- **95% confidence level** (2.5th and 97.5th percentiles)
- Delta = runtime - text (positive = runtime outperforms)

## Reproducibility

To reproduce this experiment:

1. Install Forgent and build `forgent-bench`
2. Set up an LLM provider (Anthropic API key or Ollama)
3. Run: `forgent-bench . --level experiment --tasks-dir internal/bench/experiment/tasks/ --provider <name> --model <model> --runs 30 --output results.md`
4. Compare output against published results

The experiment is deterministic given the same:
- Task definition files
- LLM provider and model
- Number of runs
- Bootstrap seed (hardcoded: 42)

Note: LLM non-determinism (temperature > 0) introduces variance. Use temperature=0 or report results across N=30+ runs for confidence intervals.

## Limitations

- Token counting is approximate (character-based, not tokenizer-based)
- Guardrail checking is keyword-based, not semantic
- Step completion is verified by keyword presence, not quality of output
- The text condition uses a single-response format; multi-turn text would be a different control
