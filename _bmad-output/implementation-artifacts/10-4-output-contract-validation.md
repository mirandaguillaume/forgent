# Story 10-4: Output Contract Validation via Template

**Epic:** 10 — Node-Level Behavioral Contracts
**Status:** done

## Story

As a **developer defining a Forgent task**,
I want to declare a required output structure via `template` or `template_file` in my step's `produces` contract with explicit `<!-- REQUIRED: name -->` markers,
So that the runtime validates the LLM output BEFORE writing it to the slot, recording violations for telemetry.

## Acceptance Criteria

**AC1:** Given a step's `produces` declares a `template_file` (or inline `template`) with `<!-- REQUIRED: verdict -->` markers,
When the LLM output is missing one or more REQUIRED markers,
Then the runtime returns a contract violation BEFORE writing to `slots/*.md`,
And the slot file is NOT written for the failing step,
And the violation is recorded in the audit trail with `Allowed: false` and `Reason` containing the missing marker name.

**AC2:** Given all REQUIRED markers are present in the LLM output,
When the runtime validates,
Then the output IS written to the slot normally,
And the audit trail records the successful write with `Allowed: true`.

**AC3:** Given a step's `produces` has no template declared (inline `Template == ""` and `TemplateFile == ""`),
When the runtime executes,
Then no validation occurs — full backward compatibility (NFR40),
And existing structural-marker validation (`## headings`, `- **` bold lists) continues to work for templates that use those styles.

**AC4:** Given `template_file` is used (depends on Story 10-2's `TaskDir` resolution + Story 10-0's TOCTOU/symlink fix),
When the runtime resolves the path,
Then resolution is relative to the task YAML directory and the post-read symlink check from Story 10-0 still applies.

**AC5:** Given a `ConsumesTemplate` validation fails on retry (Task 5.4 from 10-0 deferred),
When the runtime retries the step,
Then the upstream slot's stale value is cleared from the DAG store before re-execution,
So the producer node re-runs and produces a fresh value rather than the consumer re-failing on the same frozen bad input.

**AC6:** Given a step that produces a `recommendation` slot (or any named output) wired into the recommendation-builder DAG,
When the DAG terminal-output extraction runs (Task 5.2 from 10-0 deferred),
Then the output is extracted by named key from the produced map, not by `len(steps)-1` position,
And an explicit error is returned when the named key is missing from the produced map.

**AC7:** Given an LLM emits the REQUIRED marker as `<!-- required: VERDICT -->` (case variation),
When the runtime validates,
Then the validation is case-insensitive on the marker NAME (not the surrounding HTML comment syntax),
So agents do not fail validation purely on case typos.

**AC8:** Given `go test ./... -count=1` and `go test -race ./internal/bench/experiment/... ./pkg/qualitygate/... ./pkg/dag/...`,
Then all existing tests pass without modification (race-clean).

## Tasks/Subtasks

- [x] T1: Add REQUIRED-marker extraction to qualitygate
  - [x] T1.1: `extractRequiredMarkers` in `pkg/qualitygate/gate.go` recognises
        `<!-- REQUIRED: name -->` (and comma-separated lists), lowercased + de-duplicated.
  - [x] T1.2: Distinct error wording — `quality gate: required marker %q missing from output`
        (consumes side mirrors with `... missing from input`). Heuristic-marker tests stay green.
  - [x] T1.3: Grammar documented inline on `extractRequiredMarkers` AND in the package
        docstring at the top of `gate.go`.

- [x] T2: Validate-before-write in the runtime runner (AC #1, #2)
  - [x] T2.1: **Option B chosen** — inline `qualitygate.ValidateProduces(producesTemplate, lastResponse)`
        in `runFn` (LLM kind) BEFORE the `slots/{key}.md` write loop. Templates resolved ONCE
        up-front (before the runFn closure) and captured; middleware post-check kept for
        defense-in-depth.
  - [x] T2.2: On failure, `sandbox.record(stepID, "slot_write", "slots/{key}.md", false, reason)`
        with stable telemetry-parseable wording (`"required marker missing: <name>"` or
        `"output missing expected structure: <marker>"` via `classifyValidationReason`).
  - [x] T2.3: Validation error returned from `runFn` and surfaces through the existing
        `runWithRetry`. Middleware post-check kept (defense-in-depth).
  - [x] T2.X (additional fix uncovered while writing T5.6): `runRuntimeTask` now populates
        `result.ActionLog` and `result.ActionMetrics` on the FAILURE path too — story 10-6
        telemetry needs the audit trail even when the task did not complete.

- [x] T3: Recommendation-builder named extraction (AC #6) — from 10-0 Task 5.2
  - [x] T3.1: Verified `internal/doctor/orchestrator.go:102` already does the named
        lookup `outputs["recommendations"]` with explicit error on miss. No `len(steps)-1`
        pattern exists anywhere in the codebase. No code change required.
  - [x] T3.2: Same — already correct.
  - [x] T3.3: Added `TestRunDAG_MissingNamedKey_ReturnsError` (custom DAG with wrong key →
        error mentions both `recommendations` and `missing`) plus
        `TestRunDAG_NamedExtractionFromOutputMap` (positive case).

- [x] T4: ConsumesTemplate retry slot clearing (AC #5) — from 10-0 Task 5.4
  - [x] T4.1: Identified — consumes pre-check failure happens in qualitygate middleware and
        propagates up WITHOUT triggering upstream re-execution (layer-based DAG only retries
        the failing node).
  - [x] T4.2: Implemented `consumesRetryMiddleware` in `runtime_runner.go`. Sits OUTSIDE
        qualitygate so it observes the pre-check error and recovers. On consumes-validation
        failure: for each consumed slot key, clear `slots/{key}.md` from sandbox (audit
        `slot_clear`), re-invoke the producer's runFn (registered in `producerByKey` —
        inner unwrapped runFn so re-invocation does not recurse), refresh the consumer
        inputs, retry once. Producer is invoked with EMPTY inputs deliberately: LLM-step
        prompts derive from step metadata + sandbox `file_read`, not from inputs map content.
  - [x] T4.3: `TestRuntime_ConsumesTemplate_RetryClearsUpstreamSlot` — 2-step task,
        producer's first response is bad (no heading), consumer's middleware fires,
        producer re-invoked (per-step provider call count == 2 for producer), consumer
        succeeds on retry, audit trail contains `slot_clear` entry by the consumer step.

- [x] T5: Tests (all 11)
  - [x] T5.1–T5.5: `pkg/qualitygate/gate_test.go` — REQUIRED-marker missing/present/case-
        insensitive/multi-name/absent-template (plus an extra `TestQualityGate_RequiredAndHeuristicCoexist`
        verifying both styles run when both declared).
  - [x] T5.6, T5.7, T5.10, T5.11: `internal/bench/experiment/output_contract_test.go`.
  - [x] T5.8, T5.9: `internal/doctor/orchestrator_test.go`.

- [x] T6: `go test ./... -count=1` + `go test -race ./internal/bench/experiment/... ./pkg/qualitygate/... ./pkg/dag/...`
       — 338 race-clean across affected packages. Two PRE-EXISTING failures in unrelated
       packages confirmed on baseline (verified via `git stash` + re-run):
       `internal/doctor/context.TestDetectProjectRoot_NoMarkers` and
       `internal/doctor/static.TestDrift_NoProjectRoot`. Both are environment-sensitive
       (TempDir under /tmp without project-root markers) and not touched by 10-4.

- [x] T7: Documentation
  - [x] T7.1: Package docstring at top of `pkg/qualitygate/gate.go` documents both marker
        styles (heuristic structural vs explicit REQUIRED) and when each fires.
  - [x] T7.2: No `templates/` subdirectory or `CONTRIBUTING.md` exists in the repo —
        deferred to a follow-up doc story per the story's own instruction.

### Review Findings

_Generated 2026-04-28 by `bmad-code-review` (Blind Hunter + Edge Case Hunter + Acceptance Auditor)._

**Decision-needed (2) — resolved**

- [x] [Review][Decision] REQUIRED-marker substring vs word-boundary match → **A: word-boundary kept, docstring updated, boundary pinned in tests.** [`pkg/qualitygate/gate.go:14,48`]
- [x] [Review][Decision] Recovery-exhausted error wrapping → **A: `!refreshedAny` path wrapped with `"consumes-retry: no upstream producer for recovery: %w"`.** [`internal/bench/experiment/runtime_runner.go:consumesRetryMiddleware`]

**Patch (12) — applied**

- [x] [Review][Patch] gofmt regression on `var middlewares` declaration — fixed indentation. [`internal/bench/experiment/runtime_runner.go`]
- [x] [Review][Patch] Guard producerByKey against duplicate keys — fail-fast with clear error. [`internal/bench/experiment/runtime_runner.go:buildTaskDAG`]
- [x] [Review][Patch] Restrict producer registry to LLM-kind producers only. [`internal/bench/experiment/runtime_runner.go:buildTaskDAG`]
- [x] [Review][Patch] Add `ctx.Err()` check before producer re-invocation in `consumesRetryMiddleware`. [`internal/bench/experiment/runtime_runner.go`]
- [x] [Review][Patch] Restrict produces-validation audit denial recording to template-bearing slots only. [`internal/bench/experiment/runtime_runner.go`]
- [x] [Review][Patch] Pin audit-trail Reason wording with regression tests — already present in gate_test.go (`TestQualityGate_ErrorWording_StableForTelemetry`). [pre-applied]
- [x] [Review][Patch] Make `stepKeyedProvider.Complete` fail loudly on unknown stepKey. [`internal/bench/experiment/output_contract_test.go`]
- [x] [Review][Patch] Replace magic 3-element `responses` slice with 1-element (fallback handles remainder). [`internal/bench/experiment/output_contract_test.go`]
- [x] [Review][Patch] Delete `var _ = readSlot` sentinel and unused `readSlot` helper. [`internal/bench/experiment/output_contract_test.go`]
- [x] [Review][Patch] Document REQUIRED-marker grammar limitations — already present in `extractRequiredMarkers` docstring. [pre-applied]
- [x] [Review][Patch] Reject empty REQUIRED payload explicitly — already implemented with stderr warning. [pre-applied]
- [x] [Review][Patch] Rename `indexRequiredOpen` — already renamed to `indexRequiredCommentOpen` with full docstring. [pre-applied]

**Defer (5)**

- [x] [Review][Defer] Multi-key recovery test coverage — partial recovery with multiple consumes slots is untested [`internal/bench/experiment/output_contract_test.go`] — deferred, edge case beyond current scope.
- [x] [Review][Defer] Partial recovery state corruption when one of multiple producers fails mid-recovery [`internal/bench/experiment/runtime_runner.go:consumesRetryMiddleware`] — deferred, multi-consumes scenario; current 10-4 tasks all use single-consumes.
- [x] [Review][Defer] Multiple-produces-with-different-templates only first enforced [`internal/bench/experiment/runtime_runner.go` template-resolution loop] — deferred, pre-existing behavior not introduced by 10-4.
- [x] [Review][Defer] Sentinel-error refactor for `isConsumesValidationError` (replace string-match with `errors.Is/As`) [`internal/bench/experiment/runtime_runner.go:isConsumesValidationError`] — deferred, larger qualitygate refactor.
- [x] [Review][Defer] `sandbox.Delete` error swallowed in `consumesRetryMiddleware` [`internal/bench/experiment/runtime_runner.go:consumesRetryMiddleware`] — deferred, bounded impact (producer rewrite covers stale slot).

## Dev Notes

### Architecture

**Existing infrastructure (do not duplicate):**

- `pkg/qualitygate/gate.go` already has `QualityGateMiddleware` that runs pre-check on consumes
  template markers and post-check on produces template markers. The middleware is wired in
  `runtime_runner.go` around line 487 (`middlewares = append(middlewares, qualitygate.QualityGateMiddleware(policy))`).
- `extractMarkers` in `gate.go:86` recognises `## headings`, `- **` bold list items, and
  `- ` plain list items. This story ADDS a new marker style (REQUIRED HTML comments)
  alongside, not replacing.
- `SlotSpec.ResolveTemplate` (story 10-0) handles `template_file` resolution with the
  TOCTOU-safe symlink check. No changes needed here.
- `sandbox.AccessEntry` (in `sandbox.go:22`) is the audit trail data model used by
  story 10-3. Reuse the same `record(...)` mechanism so all node-boundary violations
  flow through one channel for story 10-6.

**The slot-write-before-validation bug:**

Today the produces validation lives in the middleware POST-check
(`pkg/qualitygate/gate.go:62`). The middleware wraps `runFn`. But `runFn` writes to the
sandbox AT LINE 412 of `runtime_runner.go` (`stepSandbox.Write(stepCopy.ID, "slots/{key}.md", ...)`)
BEFORE `runFn` returns. So by the time the middleware post-check fires, the slot file is
already written. AC #1 explicitly forbids this: "returns a contract violation BEFORE writing
to slots/*.md".

**Recommended fix (T2.1 Option B):** validate inline inside `runFn` before the slot-write loop.
The middleware post-check stays as a safety net (it never fires in the common path because
the inline check already errored). The inline + middleware redundancy is intentional —
defense in depth, and the middleware path is reused by any future caller that bypasses
`runFn`.

### Why HTML-comment markers (`<!-- REQUIRED: name -->`)

The existing heuristic markers (headings, bold lists) are inferred from the template's
structure. They work for "the output should look like a verdict report". They do NOT work
for "the output MUST mention the verdict field" — that's a stronger contract.

HTML comments are invisible to LLMs reading the rendered template (they don't generate the
comment back in their output) but visible to code that parses the template source. This
gives template authors a way to mark "this field is required" without polluting the template
the LLM sees.

The marker name itself (`verdict`, `severity`, etc.) is a literal string that must appear
case-insensitive in the LLM output. The validation is intentionally simple — it does not
parse JSON or check field types. That's deliberate: rich schema validation is a future
story (10-4 follow-up or 10-5 territory).

### Backward compatibility (NFR40)

- Templates with no `<!-- REQUIRED: ... -->` markers behave exactly as before.
- Templates with both heuristic markers (`## Verdict`, `- **decision**`) and REQUIRED
  markers run BOTH validations — fail if either fails.
- Tasks without templates (no `template` and no `template_file`) skip validation entirely.
- The middleware's existing `passthrough` early-return (when both marker sets are empty)
  is preserved — REQUIRED markers count as "non-empty", forcing the middleware to run.

### Telemetry hook (story 10-6)

This story produces audit trail entries for:

1. Successful slot writes (`Allowed: true`, `Action: "slot_write"`)
2. Failed slot writes due to REQUIRED marker missing (`Allowed: false`, `Reason:
   "required marker missing: <name>"`)
3. Failed `ConsumesTemplate` validation that triggers upstream slot clearing
   (`Allowed: false`, `Reason: "consumes template missing: <marker>"` + a separate
   "slot_clear" entry for the upstream that was reset)

Story 10-6 will tally these by step ID. Keep `Reason` strings stable and parseable.

### Files likely modified

- `pkg/qualitygate/gate.go` — REQUIRED marker extraction + new error message
- `pkg/qualitygate/gate_test.go` — new tests for REQUIRED markers
- `internal/bench/experiment/runtime_runner.go` — inline validation, ConsumesTemplate
  retry slot clearing
- `internal/bench/experiment/runtime_runner_test.go` (or `step_guardrails_test.go`) —
  integration tests
- Possibly: `internal/cmd/doctor.go` or `internal/builder/` — recommendation-builder
  named extraction (T3, find the actual file via grep first)

### What NOT to do

- **Don't replace heuristic markers with REQUIRED markers.** Both styles coexist.
  Templates that worked before continue to work. New templates can opt into REQUIRED
  markers for stricter contracts.
- **Don't add JSON schema validation.** Out of scope. The contract is "this name appears
  in the text", not "this is a valid JSON object with these fields".
- **Don't move ALL middleware logic inline.** Only the produces post-check needs to move.
  Consumes pre-check stays in the middleware (it runs before `runFn`, which is the right
  place).
- **Don't introduce a new audit log channel.** Reuse `sandbox.record(...)` so story 10-6
  has one place to look.
- **Don't pre-compile regex for marker extraction.** The template parsing happens once per
  step at DAG build time; per-line `strings.Contains` is fine and matches the existing
  heuristic-marker style.

### Previous story intelligence (from 10-3 in flight + 10-2 + 10-0)

- **10-3 introduced `step.FileAccess`** — orthogonal to output contracts. The slot write
  in `runtime_runner.go` is a sandbox `Write` call that may now be subject to
  `FileAccess: read-only` denial. If 10-3 is implemented before this story, the inline
  validation in T2.1 must run AFTER `FileAccess` permission check (since a read-only step
  cannot write a slot anyway, so validation is moot). If 10-3 is in parallel, document the
  precedence in T2.1's comment.
- **10-2 fixed `template_file` to resolve via `task.TaskDir`** — this story leverages it
  directly. No re-implementation needed; just confirm `SlotSpec.ResolveTemplate` is called
  with `task.TaskDir` (it already is, per `runtime_runner.go:449,462`).
- **10-0 added the post-read symlink check + empty-baseDir rejection** — REQUIRED markers
  inside a `template_file` benefit from these guards transparently. No additional work.
- **10-0 fixed `evaluateRuntimeOutputs` to look up StepResults by step ID** — this story's
  T3 (recommendation-builder named extraction) extends the same principle to the
  recommendation pipeline. Use the SAME defensive pattern (named key lookup with explicit
  error on miss) for consistency.
- **10-0 deferred Task 5.2 + 5.4 to this story** — explicitly absorbed in T3 + T4.

### Project context reference

- **Sprint status:** `_bmad-output/implementation-artifacts/sprint-status.yaml` —
  `10-4-output-contract-validation: ready-for-dev` after this file lands.
- **Epic 10 plan:** `_bmad-output/planning-artifacts/epics.md` lines 343-369 (Story 7-4
  in the original numbering).
- **Recent commits:**
  - `a3296d9 feat(10-0): epic 9 debt cleanup` — landed sandbox TOCTOU fix, defensive
    guardrail copy, Validate StepID enforcement, taskContextContent step-level rendering,
    Compare TaskID alignment with three-way intersection in `CompareThree`
  - `46d041a feat(epic9): close epic 9 (DX) + retro + deferred-work triage and quick wins`
- **Test infrastructure:** 155 tests passing in `internal/bench/experiment/` after 10-0;
  327 race-clean across the affected packages. New tests should keep both green.

## Dev Agent Record

### Agent Model Used

claude-opus-4-7 (Claude Code, learning + explanatory output style)

### Debug Log References

- T2/T5.6 surfaced a latent gap: `runRuntimeTask` did NOT populate `result.ActionLog` on
  the failure path. Story 10-6 telemetry needs the audit trail even when the task did not
  complete, so this was fixed as part of T2 ("T2.X" subtask).
- T4 first attempt put the recovery wrapper around `runFn` directly. That fails because
  middleware fires BEFORE `runFn`, so the wrapper never sees qualitygate's pre-check error.
  Refactored to a `dag.Middleware` placed FIRST in the slice (outermost) so it wraps the
  qualitygate middleware and observes its error. `pkg/dag/middleware.go:buildChain` builds
  the chain innermost-first, so `middlewares[0]` is outermost — confirmed via grep before
  refactor.
- T3 turned out to be a verify-only task: `internal/doctor/orchestrator.go:102` already
  did the named lookup with explicit error on miss; no `len(steps)-1` pattern exists
  anywhere. Tests T5.8 + T5.9 added to PIN this contract for future refactors.

### Completion Notes List

- ✅ AC #1: Inline `qualitygate.ValidateProduces` in `runFn` runs BEFORE the slot write loop.
  Validation failure returns the error from `runFn` AND records a sandbox audit entry
  (`Action: "slot_write"`, `Allowed: false`, stable `Reason` prefix).
- ✅ AC #2: When all REQUIRED markers present, slot is written normally and audit entry
  is `Allowed: true`.
- ✅ AC #3: NFR40 backward compat preserved — empty templates skip validation, existing
  heuristic-marker templates continue to work, the test suite for heuristic markers stays green.
- ✅ AC #4: `template_file` resolution path is reused unchanged (relies on 10-2's TaskDir +
  10-0's TOCTOU/symlink fix).
- ✅ AC #5: `consumesRetryMiddleware` clears stale upstream slots from the sandbox AND
  re-invokes the producer's runFn so the consumer retries with a fresh value.
- ✅ AC #6: Recommendation builder uses named-key extraction with explicit error on miss
  (was already correct; T5.8+T5.9 pin the contract).
- ✅ AC #7: Marker name match is case-insensitive (`<!-- required: VERDICT -->` and output
  `verdict` / `VERDICT` / `Verdict` all match).
- ✅ AC #8: 338 race-clean tests across `internal/bench/experiment/...`, `pkg/qualitygate/...`,
  `pkg/dag/...`. Two unrelated pre-existing failures in `internal/doctor/{context,static}/`
  verified on baseline via `git stash`.

### File List

**Modified:**

- `pkg/qualitygate/gate.go` — added `extractRequiredMarkers`, REQUIRED-marker validation
  in `QualityGateMiddleware`, and the public `ValidateProduces`/`ValidateConsumes` helpers
  for inline use; expanded package docstring.
- `pkg/qualitygate/gate_test.go` — added 8 REQUIRED-marker tests (extract + middleware
  pre/post check + coexistence with heuristic markers).
- `internal/bench/experiment/runtime_runner.go` — pre-resolve produces/consumes templates
  up-front so they can be captured by the runFn closure; inline produces validation +
  audit-trail recording before the slot-write loop; new `consumesRetryMiddleware` +
  `producerByKey` registry; `classifyValidationReason` helper for stable telemetry wording;
  fail-path now populates ActionLog/ActionMetrics on failure.
- `internal/doctor/orchestrator_test.go` — added imports + `TestRunDAG_NamedExtractionFromOutputMap`
  + `TestRunDAG_MissingNamedKey_ReturnsError` to pin the named-extraction contract.
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `10-4-output-contract-validation`
  status flipped `ready-for-dev → in-progress → review`; `last_updated`, `last_story_modified`
  refreshed.
- `_bmad-output/implementation-artifacts/10-4-output-contract-validation.md` — this file.

**Created:**

- `internal/bench/experiment/output_contract_test.go` — T5.6/T5.7 (no slot write on
  failure + audit entry), T5.10 (consumes-retry slot clear + producer rerun), T5.11
  (YAML round-trip with REQUIRED marker), plus a positive-path companion test.

## Change Log

| Date | Change | Notes |
|---|---|---|
| 2026-04-28 | Story 10-4 created from Epic 10 plan (Story 7-4 in epics.md) | Absorbs 10-0 deferred Task 5.2 (recommendation-builder named extract) and Task 5.4 (ConsumesTemplate retry slot clear). Builds on 10-2 (`template_file` resolution) and 10-0 (TOCTOU symlink fix). |
| 2026-04-28 | Story 10-4 implemented and marked for review | All 7 tasks complete (T3 was verify-only — code already correct, tests added to pin contract). 8 acceptance criteria green. 338 race-clean tests on affected packages; two pre-existing failures elsewhere confirmed on baseline. Inline produces validation + consumes-retry middleware + named-extraction tests. |
