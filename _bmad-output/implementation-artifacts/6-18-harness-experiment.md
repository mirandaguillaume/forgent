# Story 6-18: Harness Experiment — Wire Middleware Guards into Runtime Runner

**Status:** done
**Epic:** 6 — Runtime Harness
**Sprint:** 7

## Goal

Wire QualityGateMiddleware and BudgetMiddleware into the experiment runtime runner so that
programmatic enforcement is attached to DAG nodes on every run.

**Design decision (2026-04-28, code review):** The original plan gated middlewares behind a
`--harness` flag (`HarnessEnabled bool` in `ExperimentConfig`). During implementation, the
team concluded that always-on enforcement is the right product position — _"the harness IS
the runtime's value proposition"_. A flag-gated harness would mean the baseline condition
runs without enforcement, which defeats the point of proving structural contracts over
keyword guardrails. The backward-compat AC ("without --harness, behavior is identical") was
deliberately abandoned in favour of always-on.

## Changes

### `internal/bench/experiment/runtime_runner.go`
- Added imports for `pkg/budget` and `pkg/qualitygate`
- In `buildTaskDAG`, for every LLM step (always, no flag check):
  - Attaches `QualityGateMiddleware` with produces/consumes template contracts
  - Attaches `BudgetMiddleware` with default policy (MaxTokens=8192, Timeout=60s)
  - Sets `MaxRetries=2` on the node so quality gate can trigger retries
- Static and assert steps are NOT affected (no middlewares attached)

## Acceptance Criteria

- [x] `go build ./cmd/forgent-bench` compiles
- [x] `go test ./... -count=1 -short` passes
- [x] Middlewares wired to LLM nodes only (static/assert excluded)
- [x] Text runner is unmodified
- [x] `compare.go` and `report.go` are unmodified
- [x] Always-on by design — no flag required

### Review Findings (2026-04-28)

**Decision resolved:**
- [x] [Review][Decision] always-on vs --harness flag → **B: always-on kept, spec amended.**

**Patches applied:**
- [x] [Review][Patch] `BudgetMiddleware` mutates shared inputs map across retries → copy before inject [`pkg/budget/middleware.go:33`]
- [x] [Review][Patch] `classifyValidationReason` has no direct tests — wording is 10-6 telemetry API → 3 pinning tests added [`internal/bench/experiment/output_contract_test.go`]
- [x] [Review][Patch] `checkMarker` plain-list asymmetry: `"- "` didn't accept `* item` from LLM → extended [`pkg/qualitygate/gate.go:checkMarker`]

**Deferred (conflict with Epic 10):**
- [x] [Review][Defer] `consumesRetryMiddleware`: delete slot before confirming producerOut[key] → conflicts semantically with 10-6 (slot_clear audit semantics)
- [x] [Review][Defer] `len(consumes) > 0` guard too broad → potential line conflict with 10-5 middleware wiring section
- [x] [Review][Defer] `context.Background()` hardcoded in `runRuntimeTask` → structural change, deferred post-MVP
- [x] [Review][Defer] Goroutine leak: `provider.Complete` has no `ctx` param → requires interface change across `llm.Provider`, deferred post-MVP
