# Epic 10 Experiment Results — 2026-04-29

> **Status: PENDING** — T2 (corpus augmentation) and T3 (Primary Safety Metric code) are complete.
> T5 (actual experiment run) is blocked on API credit availability. This document will be filled
> in after the run completes. Numbers below are placeholders marked with `[TBD]`.

## Run Configuration

- **Date:** [TBD — to be filled after run]
- **Model:** claude-sonnet-4-6 (matches baseline model family — 2026-04-16 run used Sonnet)
- **Provider:** Anthropic
- **RunsPerTask:** 3
- **Conditions:** text, runtime, ranked
- **Tasks:** code-review, doc-generation, refactoring, security-audit, test-generation (5-task overlap)
- **Baseline:** experiment-sonnet-baseline.json (converted from experiment-sonnet-baseline.{text,runtime}.json via `forgent-bench convert-baseline`)
- **Epic 10 stories active:** 10-1 through 10-8, 10-10, 10-11 (all done)

### Reproduction Command

```bash
# Step 1: Convert legacy baseline (one-time)
forgent-bench convert-baseline \
  experiment-sonnet-baseline.text.json \
  experiment-sonnet-baseline.runtime.json \
  experiment-sonnet-baseline-converted.json

# Step 2: Run the 5-task overlap experiment
forgent-bench bench . \
  --level experiment \
  --tasks-dir internal/bench/experiment/tasks \
  --only code-review,doc-generation,refactoring,security-audit,test-generation \
  --retriever ranked \
  --runs 3 \
  --baseline experiment-sonnet-baseline-converted.json \
  --output epic-10-experiment-results.json
```

Note: raw JSON checkpoints stay LOCAL (feedback_no_datasets policy). Only this `.md` file is committed.

## Headline Numbers

| Metric | Baseline (2026-04-16, keyword) | New (structural) | Delta |
|--------|-------------------------------|------------------|-------|
| Detection Rate (overall) | 0% | [TBD] | [TBD] |
| Detection Rate (runtime) | N/A | [TBD] | — |
| ContractComplianceRate (text) | [from baseline JSON] | [TBD] | [TBD] |
| ContractComplianceRate (runtime) | [from baseline JSON] | [TBD] | [TBD] |
| ContractComplianceRate (ranked) | N/A | [TBD] | — |

## Theory A vs Theory B (per violation type)

| Violation Type | Dominant Classification | Notes |
|----------------|------------------------|-------|
| guardrail | [TBD] | Keyword-based — already in baseline |
| file_permission | [TBD] | New in Epic 10 (10-3) |
| output_contract | [TBD] | New in Epic 10 (10-4) |
| network | [TBD] | New in Epic 10 (10-5) |
| loop_bound_exceeded | [TBD] | New in Epic 10 (10-7) |

## Statistically Meaningful Deltas

[TBD — Bootstrap CI 95% non-zero list to be filled after run completes]

## FR45 Conclusion

[TBD — Plain English narrative: yes/no/mixed, with specific numbers]

> **Note on partial runs:** If API credits are exhausted mid-run, the partial checkpoint
> at `epic-10-experiment-results.checkpoint.jsonl` will be used. The conclusion will be
> marked "tentative" with the completed task count vs total.

## Limitations

- **Scope:** 5-task overlap only (vs originally planned 16 tasks). 11 unbaselined tasks deferred to T5b.
- **Baseline mismatch:** Legacy baseline (2026-04-16) has no NodeViolations data. CCR is proxied from GuardrailScore via `convert-baseline`. Step-level deltas are approximate.
- **Model parity:** Baseline used Sonnet (exact version unknown from 2026-04-16). Rerun targets claude-sonnet-4-6. Minor model version differences may contribute to delta noise.
- **Theory B risk:** If the 5 baseline tasks produce no enforcement events under structural enforcement (agents naturally compliant with file/network/output contracts), the Primary Safety Metric section will be empty and FR45 evidence will be inconclusive for these tasks.
