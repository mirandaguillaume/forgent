# Story 10-6: Violation Telemetry per Node

**Epic:** 10 — Node-Level Behavioral Contracts
**Status:** done

## Story

As a **developer running a Forgent experiment**,
I want each node boundary to log attempted and prevented violations across all four enforcement types (guardrail, file permission, output contract, network),
So that I can distinguish between agents that comply naturally (Theory B) and agents coerced by structural enforcement (Theory A).

## Acceptance Criteria

**AC1:** Given a step violation occurs (guardrail, file permission, output contract, or network),
When the runtime records it,
Then the violation log includes:
- `node_id` (step ID)
- `violation_type` (one of: `guardrail`, `file_permission`, `output_contract`, `network`)
- `attempted: true` (the step tried the operation)
- `prevented: true|false` (the structural enforcement blocked it before effect, vs. the violation was observed post-hoc)
- `attempt_number` (1-indexed: which retry of the step triggered this violation, with `1` meaning the first execution)
- `reason` (stable string from the source enforcer, e.g. `"step is read-only"`, `"required marker missing: verdict"`)

**AC2:** Given a step completes without any violation across all four types,
When the runtime records it,
Then NO violation entry is emitted for that node (sparse telemetry — empty == clean).

**AC3:** Given an experiment run completes,
When `TaskResult` is constructed,
Then it includes a `NodeViolations []NodeViolation` field listing every violation across all steps and all attempts in chronological order,
And it includes a `ContractComplianceRate float64` field — the fraction of node boundaries (steps) that had zero violations on their first attempt (range 0.0 to 1.0; 1.0 means all steps complied on first try).

**AC4:** Given a step violates a guardrail on attempt 1, retries, and succeeds on attempt 2,
When the experiment completes,
Then `NodeViolations` contains exactly one entry for that step with `attempt_number: 1`, `prevented: false` (guardrail violations are observed post-hoc, not prevented),
And the step contributes 0 to `ContractComplianceRate`'s numerator (had a violation on first attempt).

**AC5:** Given a step is denied on file permission BEFORE the operation runs (read-only step attempts file_write),
When the violation is recorded,
Then `prevented: true` (the structural enforcement actually blocked the action),
And the step still counts toward `ContractComplianceRate`'s denominator (every step boundary counts).

**AC6:** Given a step has multiple attempts and violations occur on multiple attempts,
When the violations are aggregated,
Then each attempt's violation is recorded as a separate `NodeViolation` with the correct `attempt_number`,
And `ContractComplianceRate` only checks the FIRST attempt — subsequent attempt violations don't double-penalise.

**AC7:** Given a step has BOTH a file permission violation AND a guardrail violation on the same attempt,
When recorded,
Then BOTH are emitted as separate `NodeViolation` entries,
And the step counts as ONE failure in `ContractComplianceRate`'s numerator (multiple violations on the same attempt = still one non-compliant step).

**AC8:** Given the experiment runs in mixed mode (some tasks have step-level enforcers, some don't),
When telemetry is aggregated across the experiment's `TaskResult` slice,
Then `ContractComplianceRate` is computable per-task and the per-experiment mean is meaningful (no NaN, no division by zero on tasks with zero steps — a zero-step task is already rejected by `Validate()`, but defensively `ContractComplianceRate = 1.0` if `len(StepResults) == 0`).

**AC9:** Given `go test ./...` and `go test -race ./internal/bench/experiment/...`,
Then all existing tests pass without modification (race-clean).

## Tasks/Subtasks

- [ ] T1: `NodeViolation` data model
  - [ ] T1.1: Define `NodeViolation` struct in `internal/bench/experiment/design.go` near `TaskResult`:
        ```go
        type NodeViolation struct {
            NodeID        string             // step ID
            ViolationType string             // "guardrail" | "file_permission" | "output_contract" | "network"
            Attempted     bool               // always true if recorded
            Prevented     bool               // true = blocked before effect; false = observed post-hoc
            AttemptNumber int                // 1-indexed
            Reason        string             // stable wording from the enforcer
            // Path / Target carries the offending resource for file/network types
            Target        string             // e.g. "src/main.go" for file, "api.stripe.com" for network, "" for guardrail
        }
        ```
  - [ ] T1.2: Define exported violation type constants:
        `ViolationTypeGuardrail = "guardrail"`,
        `ViolationTypeFilePermission = "file_permission"`,
        `ViolationTypeOutputContract = "output_contract"`,
        `ViolationTypeNetwork = "network"`.
  - [ ] T1.3: Add fields to `TaskResult`:
        - `NodeViolations []NodeViolation` (chronological)
        - `ContractComplianceRate float64` (0.0–1.0)
  - [ ] T1.4: Document the semantics in a Go doc comment above `TaskResult`:
        - "first attempt" means `AttemptNumber == 1`
        - `ContractComplianceRate = (steps with zero attempt-1 violations) / (total steps)`
        - Steps with violations on attempt ≥2 but compliant on attempt 1 still count as compliant (the enforcement worked structurally OR the LLM self-corrected on retry — both are acceptable per Theory B)

- [ ] T2: Track `attempt_number` in the per-step closure
  - [ ] T2.1: Inside `runFn` in `runtime_runner.go` per-step closure, introduce a local
        `attempt int` counter starting at 1. Increment when a guardrail-retry path triggers
        re-execution (search for the existing `for retry := 0; retry < maxAttempts; retry++`
        loop or equivalent).
  - [ ] T2.2: Pass the current `attempt` to every audit recording call site that may
        produce a violation:
        - The guardrail violation path (currently writes to `result.GuardrailScore` —
          extend to also append a `NodeViolation` with `attempt_number = attempt`).
        - The file permission denial path (story 10-3 — already records via `sandbox.record`).
        - The output contract violation path (story 10-4 — already records via `sandbox.record`).
        - The network policy denial path (story 10-5 — already records via `sandbox.record`).
  - [ ] T2.3: Threading model: rather than threading `attempt` through every helper, store
        it in a per-step context key or as a new field on `SandboxFS` (`currentAttempt int`)
        that the closure mutates between attempts. Pick the lower-churn option — likely a
        new method `sandbox.SetAttempt(stepID, n)` called at the top of each retry iteration,
        and `record(...)` reads it from a `map[stepID]int`.

- [ ] T3: Aggregator — convert audit trail + guardrail eval to `NodeViolations`
  - [ ] T3.1: Add a function `aggregateViolations(task TaskDefinition, sandbox *SandboxFS, guardrailFailures []guardrailFailure) []NodeViolation`
        in `runtime_runner.go` (or a new `telemetry.go` file).
  - [ ] T3.2: Walk `sandbox.Log()` (existing method on `SandboxFS`):
        - For each entry with `Allowed: false`, derive the `ViolationType` from the
          `Action` field:
          - `Action: "read" | "write" | "delete"` → `ViolationTypeFilePermission`
          - `Action: "slot_write"` → `ViolationTypeOutputContract`
          - `Action: "network"` → `ViolationTypeNetwork`
        - `Prevented = true` for all sandbox-level denials (the action did not happen).
        - Read `attempt_number` from the per-stepID attempt map populated in T2.
  - [ ] T3.3: Walk the guardrail evaluation results (a new structure populated in T2.2's
        guardrail path):
        - One `NodeViolation` per guardrail failure with `ViolationType: "guardrail"`,
          `Prevented: false` (guardrail violations are observed AFTER the LLM responds —
          structural enforcement at this layer is retry-based, not action-blocking).
  - [ ] T3.4: Sort the resulting `[]NodeViolation` by `(StepIndex, AttemptNumber)` so the
        slice is reproducible across runs (for deterministic tests). Use the step's
        position in `task.Steps` as `StepIndex`.

- [ ] T4: Compute `ContractComplianceRate`
  - [ ] T4.1: Add `computeContractComplianceRate(stepCount int, violations []NodeViolation) float64`:
        - For each step, check if ANY violation has `attempt_number == 1` for that step
        - `compliant = stepCount - len(stepsWithFirstAttemptViolation)`
        - If `stepCount == 0` → return `1.0` (defensive — no violations possible)
        - Otherwise → `float64(compliant) / float64(stepCount)`
  - [ ] T4.2: Wire into `evaluateRuntimeOutputs`: after the existing GuardrailScore loop,
        call `aggregateViolations` + `computeContractComplianceRate`, populate the new
        `TaskResult` fields.
  - [ ] T4.3: Document the orthogonality with `GuardrailScore`:
        - `GuardrailScore` measures "did the final output respect the guardrails?" (post-retry)
        - `ContractComplianceRate` measures "did the LLM comply on the first try?" (attempt-1
          only)
        - Both are useful: high GuardrailScore + low ContractComplianceRate = enforcement
          succeeded but agent needed coercion (Theory A); high both = agent complied
          naturally (Theory B); low GuardrailScore = enforcement insufficient OR LLM
          fundamentally non-compliant (third bucket needing investigation).

- [ ] T5: Tests
  - [ ] T5.1: `TestNodeViolation_GuardrailViolationOnFirstAttempt` — step's guardrail
        fails on attempt 1, succeeds on attempt 2. Expect:
        - `len(NodeViolations) == 1`
        - `NodeViolations[0].ViolationType == "guardrail"`
        - `NodeViolations[0].AttemptNumber == 1`
        - `NodeViolations[0].Prevented == false`
        - `ContractComplianceRate == 0.0` (single step, attempt-1 violation)
  - [ ] T5.2: `TestNodeViolation_FilePermissionViolation` — step with
        `FileAccess: read-only` (story 10-3), LLM emits `file_write`. Expect:
        - One `NodeViolation` with `ViolationType: "file_permission"`, `Prevented: true`,
          `Target: "src/main.go"` (or whatever path the LLM tried).
  - [ ] T5.3: `TestNodeViolation_OutputContractViolation` — step with REQUIRED marker
        (story 10-4), LLM output omits the marker. Expect:
        - One `NodeViolation` with `ViolationType: "output_contract"`, `Prevented: true`,
          `Target: "slots/<key>.md"`.
  - [ ] T5.4: `TestNodeViolation_NetworkViolation` — step with
        `ForbiddenDomains: ["api.stripe.com"]` (story 10-5), HTTP client makes blocked
        request. Expect:
        - One `NodeViolation` with `ViolationType: "network"`, `Prevented: true`,
          `Target: "api.stripe.com"`.
  - [ ] T5.5: `TestNodeViolation_NoViolation_EmptySlice` — step completes cleanly. Expect:
        - `len(NodeViolations) == 0`
        - `ContractComplianceRate == 1.0`
  - [ ] T5.6: `TestNodeViolation_MultipleAttemptsRecorded` — guardrail violation on
        attempts 1 and 2, success on 3. Expect:
        - `len(NodeViolations) == 2` with `attempt_number` 1 and 2
        - `ContractComplianceRate == 0.0` (the attempt-1 violation made it non-compliant)
  - [ ] T5.7: `TestNodeViolation_MultipleViolationTypesSameAttempt` — step has BOTH
        a guardrail violation AND a file permission violation on attempt 1 (e.g., LLM
        attempted a `file_write` to a read-only step AND its text response violated a
        guardrail). Expect:
        - `len(NodeViolations) == 2` (one per type)
        - Step counts as ONE failure in `ContractComplianceRate` (set membership, not
          violation count)
  - [ ] T5.8: `TestContractComplianceRate_MixedSteps` — task with 4 steps:
        - Step 1: clean
        - Step 2: guardrail violation on attempt 1, retry success
        - Step 3: clean
        - Step 4: file permission violation on attempt 1
        Expect: `ContractComplianceRate == 0.5` (2 of 4 compliant on first attempt)
  - [ ] T5.9: `TestNodeViolation_ChronologicalOrder` — multi-step task with violations
        on different steps and attempts; verify the slice is sorted by step index then
        attempt number for reproducible test output.
  - [ ] T5.10: `TestContractComplianceRate_ZeroSteps_DefensiveOne` — call the helper
        with `stepCount == 0` directly; expect `1.0` (avoids NaN even though Validate
        rejects zero-step tasks at config load).

- [ ] T6: Run `go test ./... -count=1` and `go test -race ./internal/bench/experiment/...`
       — no regressions, race-clean.

- [ ] T7: Documentation
  - [ ] T7.1: Add a section to the package doc of `internal/bench/experiment/` explaining
        the four violation types, the `Prevented` semantics (true for sandbox-level denials,
        false for guardrail post-hoc detection), and how to read `ContractComplianceRate`
        in the context of Theory A vs Theory B.
  - [ ] T7.2: Add a comment in `report.go` (or wherever `TaskResult` is rendered) noting
        that `NodeViolations` and `ContractComplianceRate` are now part of the report's
        output schema. Story 10-8 (per-node experiment metrics) will format these for
        the experiment-level rollup.

## Dev Notes

### Architecture

**Single audit channel feeds telemetry.** Stories 10-3, 10-4, 10-5 all record violations
via `sandbox.record(stepID, action, path, allowed, reason)` and the existing guardrail-
retry path mutates `GuardrailScore`. This story does NOT add a new channel; it adds an
AGGREGATOR that walks the existing audit log and the guardrail evaluation, joins them by
step ID and attempt number, and produces a typed `[]NodeViolation`.

**Why an aggregator pattern instead of inline emission?** Inline emission would require
every enforcement site (sandbox.go's denial paths, the guardrail loop in runFn, the
QualityGate middleware) to know about the typed violation struct. Aggregation keeps the
enforcement sites enforcement-only and lets telemetry evolve independently. The cost is
one walk over `sandbox.Log()` per task — measured in microseconds at experiment scale.

**`Prevented` semantics:**
- File permission denial (10-3): the operation was blocked before it could touch state
  → `Prevented: true`
- Output contract failure (10-4): the slot write was blocked before it persisted
  → `Prevented: true`
- Network policy denial (10-5): the request was blocked at the proxy before the host
  was contacted → `Prevented: true`
- Guardrail violation: the LLM produced a response that violates the guardrail; the
  retry path then re-prompts. The original response WAS produced (we observed it).
  → `Prevented: false`

This distinction matters for Theory A vs Theory B analysis. `Prevented: true` events
indicate the harness DID intervene structurally; `Prevented: false` events indicate the
LLM either self-corrected on retry or the guardrail check fired post-hoc.

### Tracking attempt_number

The runtime currently has retry logic in two places:
1. **`runFn`-internal guardrail retry loop** — increments inline within the closure for
   guardrail-driven retries.
2. **`pkg/dag.runWithRetry`** — increments at the DAG level for "node returned an error"
   retries. The QualityGate middleware's post-check returns errors that flow into this
   path.

Both should populate the same `attempt` counter for telemetry. The cleanest seam: add a
`SandboxFS.SetCurrentAttempt(stepID string, n int)` method called at the top of each
attempt loop iteration AND from a hook at the start of every retry in `runWithRetry`.

Avoid threading `attempt` through every recording call signature — that's a churn-heavy
change for a cross-cutting concern. The map lookup adds ~10ns per recording, irrelevant
at experiment scale.

### Backward compatibility (NFR40)

- Existing tests that don't exercise the new fields will see their `TaskResult.NodeViolations`
  default to `nil` (zero value). Any test that JSON-marshals `TaskResult` will get
  `"NodeViolations":null` — confirm `json:"omitempty"` is set if existing snapshots
  matter.
- `TaskResult.ContractComplianceRate` defaults to `0.0` for old code paths that don't
  call the new aggregator. T4.2 wires it into the existing `evaluateRuntimeOutputs` so
  the field is populated for all real experiment runs. Direct unit-test constructions of
  `TaskResult` literals stay untouched (the field is just zero).
- `GuardrailScore` is unchanged. `ContractComplianceRate` is added alongside it, not
  replacing it.

### Files likely modified

- `internal/bench/experiment/design.go` — `NodeViolation` struct, type constants, new
  `TaskResult` fields
- `internal/bench/experiment/runtime_runner.go` — `attempt_number` threading, aggregation,
  rate computation; possibly extracted to a new `telemetry.go`
- `internal/bench/experiment/sandbox.go` — `SetCurrentAttempt` method + per-step attempt
  map (`map[stepID]int`)
- `internal/bench/experiment/report.go` (or wherever `TaskResult` is JSON-marshalled) —
  add the new fields to the output schema with `json` tags

### What NOT to do

- **Don't change `TaskResult.GuardrailScore` semantics.** It stays as "fraction of guardrails
  respected by the FINAL output" (post-retry). `ContractComplianceRate` is the parallel
  "first-attempt compliance" metric.
- **Don't add a new audit log channel.** Reuse `sandbox.Log()`. The aggregator is the
  one new code path; the source of truth is the existing audit trail.
- **Don't compute `ContractComplianceRate` per attempt.** It's a per-step boolean
  (compliant on first attempt: yes/no), aggregated by mean across the task's steps.
  Per-attempt compliance is implicit in `NodeViolations` itself.
- **Don't emit violations for retries that succeed.** The current retry path does NOT
  emit a violation when retry leads to success — that's a feature: `GuardrailScore`
  captures the FINAL state. `NodeViolations` captures the historical record but does
  NOT downgrade compliance for self-correcting agents.
- **Don't expose internal attempt-counter wiring as a public API.** `SetCurrentAttempt`
  is package-internal even though the audit log is package-public. The attempt counter
  is an implementation detail of the recording flow.

### Previous story intelligence

- **10-0 introduced `Validate()` empty/duplicate StepID enforcement.** Telemetry indexing
  by `StepID` is safe — no collisions possible in valid configs.
- **10-0 fixed `evaluateRuntimeOutputs` to look up StepResults by step ID.** This story
  extends the same defensive pattern: aggregator joins by step ID, never by index.
- **10-3 introduces `step.FileAccess`** with audit trail entries for permission denials.
  The `Action` field values (`"read"`, `"write"`, `"delete"`) are the discriminator for
  `ViolationTypeFilePermission`. Pin the Action wording — telemetry depends on it.
- **10-4 introduces output-contract REQUIRED markers** with audit entries for slot-write
  blocks. The `Action: "slot_write"` value discriminates `ViolationTypeOutputContract`.
- **10-5 introduces network policy** with audit entries for proxy denials. The
  `Action: "network"` value discriminates `ViolationTypeNetwork`.
- **The four `Action` values become a closed set after this story.** Add a doc comment
  to `AccessEntry.Action` listing them as the canonical discriminator. Future stories
  that introduce new violation kinds must update the discriminator and the
  `ViolationType` constants.

### Project context reference

- **Sprint status:** `_bmad-output/implementation-artifacts/sprint-status.yaml` —
  `10-6-violation-telemetry: ready-for-dev` after this file lands.
- **Epic 10 plan:** `_bmad-output/planning-artifacts/epics.md` lines 399-420 (Story 7-6
  in the original numbering).
- **Recent commits:**
  - `a3296d9 feat(10-0): epic 9 debt cleanup` — established the audit-log-as-source-of-truth
    pattern with `sandbox.record()` + `AccessEntry`
  - `46d041a feat(epic9): close epic 9 (DX) + retro + deferred-work triage and quick wins`
- **Test infrastructure:** 155 tests in `internal/bench/experiment/` after 10-0;
  327 race-clean across affected packages. New tests should keep both green and add the
  10 new T5.1–T5.10 cases.

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

_empty_

### Completion Notes List

- `AttemptNumber` added to `AccessEntry` (backward-compatible zero default for pre-tracking entries)
- `SetCurrentAttempt` on `SandboxFS` is the single seam for attempt tracking — no param threading through call sites
- `classifyAction` is the discriminator between `Prevented: true` (file/network/slot_write) and `Prevented: false` (guardrail): structural enforcement vs. post-hoc detection
- `mockNetworkRequestProvider` requires `Proxy: http.ProxyFromEnvironment` explicitly — `&http.Transport{}` with nil Proxy does NOT fall back to env vars (unlike `http.DefaultTransport`)
- Two pre-existing failures in `internal/doctor/...` (TestDrift_NoProjectRoot, TestDetectProjectRoot_NoMarkers) confirmed unrelated
- 208 tests pass, race-clean

### File List

- `internal/bench/experiment/design.go` — ViolationType* constants, NodeViolation struct, extended TaskResult (NodeViolations + ContractComplianceRate)
- `internal/bench/experiment/sandbox.go` — AttemptNumber in AccessEntry, currentAttempt map on SandboxFS, SetCurrentAttempt, updated recordLocked
- `internal/bench/experiment/telemetry.go` (new) — aggregateViolations, classifyAction, computeContractComplianceRate
- `internal/bench/experiment/runtime_runner.go` — SetCurrentAttempt at start of retry loop, guardrail violation recording, aggregation wired in runRuntimeTask
- `internal/bench/experiment/violation_telemetry_test.go` (new) — T5.1–T5.10 (10 tests)

### Review Findings

- [x] [Review][Patch] Guardrail pattern stored in `Path`/`Target` instead of empty string [runtime_runner.go]
- [x] [Review][Patch] CCR defaults to `0.0` on DAG-build failure path instead of defensive `1.0` [runtime_runner.go]
- [x] [Review][Patch] T5.4 network test: no HTTP client timeout — can hang in CI [violation_telemetry_test.go]
- [x] [Review][Defer] Unknown step IDs (e.g., "dag") sort to position 0, colliding with first step [telemetry.go] — deferred, pre-existing stepID "dag" design
- [x] [Review][Defer] AttemptNumber=0 violations excluded from CCR penalty — latent, unreachable in current paths [telemetry.go] — deferred, pre-existing
- [x] [Review][Defer] DAG-level retry resets AttemptNumber — second DAG invocation's violations look like attempt-1 [runtime_runner.go] — deferred, bench tasks don't use DAG-level node retries
- [x] [Review][Defer] Sandboxless tasks report CCR=1.0, indistinguishable from measured compliance [telemetry.go] — deferred, design gap requiring nullable float or extra field

## Change Log

| Date | Change | Notes |
|---|---|---|
| 2026-04-28 | Story 10-6 created from Epic 10 plan (Story 7-6 in epics.md) | Aggregator over the audit-log-as-source-of-truth established by stories 10-3, 10-4, 10-5. Adds `NodeViolation` struct, `TaskResult.NodeViolations` + `ContractComplianceRate`, attempt_number tracking via `SandboxFS.SetCurrentAttempt`. Decoupled aggregation pattern lets enforcement sites stay enforcement-only and telemetry evolve independently. |
