# Epic 10 — SWE-Bench Experiment Results

**Story:** 10-12 (SWE-bench Validation)
**Status:** Infrastructure complete — experiment run pending (API budget approval required)
**Date:** 2026-05-13

---

## Executive Summary

Story 10-12 delivered the full SWE-bench experiment infrastructure:
- `resolve-issue.yaml` — 4-step DAG (read-issue → explore-repo → generate-patch → validate-patch)
  with complexity and security specialist tool_nodes
- `--level swebench-experiment` command in `forgent-bench`
- `experiment.BootstrapCI()` for statistical significance testing

The experiment run (300 tasks × 2 conditions) requires API budget approval before execution.

---

## Invocation

### Smoke test (5 tasks, no harness):
```bash
./forgent-bench bench . \
  --level swebench-experiment \
  --swe-size quick5 \
  --output /tmp/swebench-smoke.json
```

### Lite run (30 tasks, with harness):
```bash
./forgent-bench bench . \
  --level swebench-experiment \
  --swe-size sample \
  --harness \
  --output /tmp/swebench-lite.json \
  --resume /tmp/swebench-lite.checkpoint.jsonl
```

### Full run (300 tasks, lite dataset, with harness):
```bash
./forgent-bench bench . \
  --level swebench-experiment \
  --swe-size lite \
  --harness \
  --conditions text,runtime,specialists \
  --output /tmp/swebench-full.json \
  --resume /tmp/swebench-full.checkpoint.jsonl
```

### Bootstrap CI (post-run):
```go
// After collecting resolved[]bool per condition from results:
lo, hi := experiment.BootstrapCI(resolvedText, resolvedRuntime, 10_000)
fmt.Printf("Delta runtime-text: 95%% CI [%.1f%%, %.1f%%]\n", lo*100, hi*100)
// CI not containing 0 → statistically significant
```

---

## Results

> **PENDING** — experiment run not yet executed.
>
> Fill in this section after the run completes:

| Condition   | Tasks | Patched | Applied | Resolved | Resolve Rate |
|-------------|-------|---------|---------|----------|-------------|
| text        |   —   |    —    |    —    |    —     |     —       |
| runtime     |   —   |    —    |    —    |    —     |     —       |
| specialists |   —   |    —    |    —    |    —     |     —       |

**Delta (runtime vs text):** — pp  
**Bootstrap 95% CI:** [—, —]  
**Statistically significant:** —

---

## FR-SWE Scientific Claim

**Claim:** Forgent-runtime and Forgent-specialists outperform Forgent-text on SWE-bench,
proving that structural enforcement (DAG execution, typed contracts, loop bounds) improves
agent effectiveness beyond prompt engineering alone.

**Target:** +5-10 pp runtime vs text (resolve rate delta).

**Dataset:** `princeton-nlp/SWE-bench_Verified` (500 tasks) or `princeton-nlp/SWE-bench_Lite` (300 tasks).

**Note on absolute score:** 6-18% is realistic depending on harness integration depth and
LLM capability. The scientific claim is the **delta between conditions**, not the absolute score.

---

## Notes

- Raw `predictions-{condition}.jsonl` files are NOT committed (policy: no large data in git).
- Checkpoint files enable resume: `--resume /tmp/swebench-full.checkpoint.jsonl`.
- Docker must be running and `swebench` Python package installed for `--harness`.
  Install: `pip install swebench` (or `uv pip install swebench`) in a venv.
