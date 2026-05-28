# Story 10-7: Loop Bounds with Exit Criteria per Node

**Epic:** 10 — Node-Level Behavioral Contracts
**Status:** done

## Story

As a **developer defining a Forgent task**,
I want to declare loop bounds with explicit exit criteria at the step level,
So that the DAG terminates the loop structurally when a condition is met, with `max_iterations` as a safety backstop, and tool-call accumulation is bounded so retries do not lose context or grow unboundedly.

## Acceptance Criteria

**AC1:** Given a step declares `loop.exit_when.output_marker: "<!-- APPROVED -->"`,
When the LLM output contains that marker (case-sensitive substring match),
Then the DAG exits the loop immediately and proceeds to the next node,
And the step is marked `Completed: true`.

**AC2:** Given a step declares `loop.exit_when.tool_called: "submit"`,
When the LLM emits a `<tool name="submit" .../>` call,
Then the DAG exits the loop and marks the step complete,
And the tool call's args are captured in the step's produced output map.

**AC3:** Given a step has both `output_marker` AND `tool_called` exit criteria declared,
When either condition is met,
Then the loop exits — they are OR-combined.

**AC4:** Given no exit criterion is met after `loop.max_iterations` iterations,
Then the DAG terminates the loop,
And records `ViolationType: "loop_bound_exceeded"` in telemetry (story 10-6 hook),
And the step result is marked `Completed: false` with `Error: "max_iterations exceeded"`.

**AC5:** Given a step has no `loop` block declared,
When the runtime executes,
Then behavior is identical to before — full backward compatibility (NFR40),
And the existing `maxToolIterations` (currently inline in `runtime_runner.go`) continues to apply as the implicit safety backstop for tool-call loops without the new explicit declaration.

**AC6 — Task 2 absorbed from 10-0:** Given the tool-call inner loop runs across multiple iterations,
When tool results accumulate,
Then `toolPrompt` is preserved across iterations as an accumulator (NOT reset to base on each iteration),
And the accumulator is capped at `maxToolPromptBytes` (default 64 KiB, exposed as a config knob),
And eviction is oldest-first when the cap is reached so the most recent context survives.

**AC7 — Task 2 absorbed from 10-0:** Given the tool-call inner loop reaches `maxToolIterations` without converging on a final answer,
When the loop exhausts,
Then a final "synthesize your answer" instruction is appended to the accumulator and the LLM is called once more,
And if that response is still malformed (no answer extractable),
Then the step's `TaskResult.Error` is set to a `LoopExhausted` sentinel error so story 10-6 telemetry can classify it as `ViolationType: "loop_bound_exceeded"`.

**AC8:** Given a step declares `loop.max_iterations` AND `loop.exit_when.*`,
When the new declarative loop bounds fire,
Then they replace the implicit `maxToolIterations` for that step (declarative wins),
And the `max_iterations` value is the same backstop semantics — `Error: "max_iterations exceeded"`.

**AC9:** Given an unknown field appears under `loop` (e.g. `loop: { max_attempts: 3 }`, typo for `max_iterations`),
When `Validate()` runs,
Then a clear error names the offending field and lists the valid keys
(`max_iterations`, `exit_when.output_marker`, `exit_when.tool_called`).

**AC10:** Given `go test ./... -count=1` and `go test -race ./internal/bench/experiment/...`,
Then all existing tests pass without modification (race-clean),
And the new tests cover both the declarative loop semantics (AC1-AC4) and the accumulator behaviour (AC6-AC7).

## Tasks/Subtasks

- [x] T1: `LoopPolicy` data model + Validate
  - [x] T1.1: Add nested types in `internal/bench/experiment/design.go` — `ExitCriteria` + `LoopPolicy` with `UnmarshalYAML` unknown-key capture
  - [x] T1.2: Add `Loop *LoopPolicy` to `StepDef`
  - [x] T1.3: Define `DefaultMaxToolIterations = 5` and `MaxToolPromptBytes = 64 * 1024`
  - [x] T1.4: Extend `Validate()` — rejects `max_iterations <= 0`, unknown keys under `loop` and `exit_when`
  - [x] T1.5: YAML round-trip test in `loop_test.go`

- [x] T2: `effectiveLoopPolicy(step) LoopPolicy` helper
  - [x] T2.1: Added in `runtime_runner.go` next to `effectiveGuardrails`
  - [x] T2.2: Tests: nil → default, non-nil → returned as-is

- [x] T3: Tool-prompt accumulator with cap
  - [x] T3.1: Accumulator wired into declarative loop in `runtime_runner.go`
  - [x] T3.2: `promptAccumulator` struct in `accumulator.go` (base + entries + cap, oldest-first eviction)
  - [x] T3.3: T7.1–T7.3 tests in `accumulator_test.go`

- [x] T4: Exit-criteria evaluation
  - [x] T4.1: `exitCriteriaMet` in `runtime_runner.go` — OR-combined OutputMarker + ToolCalled
  - [x] T4.2: Wired into declarative loop; breaks on first match
  - [x] T4.3: Exit tool args captured under `<stepID>_<toolName>_args` key in `out` map

- [x] T5: Exhaustion + LoopExhausted sentinel
  - [x] T5.1: `ErrLoopExhausted = errors.New("max_iterations exceeded")` in `design.go`
  - [x] T5.2: Final synthesize call appended before returning `ErrLoopExhausted`
  - [x] T5.3: `stepSandbox.record(..., "loop_exceeded", ...)` triggers telemetry via `classifyAction`
  - [x] T5.4: `ViolationTypeLoopBoundExceeded = "loop_bound_exceeded"` added to `design.go`; `classifyAction` in `telemetry.go` handles `"loop_exceeded"` case

- [x] T6: Tests — declarative loop bounds (all in `loop_test.go`)
  - [x] T6.1: `TestStepLoop_OutputMarkerExits`
  - [x] T6.2: `TestStepLoop_ToolCalledExits`
  - [x] T6.3: `TestStepLoop_BothCriteriaOR`
  - [x] T6.4: `TestStepLoop_MaxIterationsExceeded`
  - [x] T6.5: `TestStepLoop_NoLoopBlock_DefaultsToImplicit`
  - [x] T6.6: `TestStepLoop_DeclarativeOverridesImplicit` (assertion bound = 9: 3 calls/runFn × 3 DAG retries)
  - [x] T6.7: `TestValidate_RejectsUnknownLoopField` + `TestValidate_RejectsUnknownExitWhenField`
  - [x] T6.8: `TestValidate_RejectsZeroOrNegativeMaxIterations`

- [x] T7: Tests — accumulator (in `accumulator_test.go`)
  - [x] T7.1: `TestPromptAccumulator_PreservesContextAcrossIterations`
  - [x] T7.2: `TestPromptAccumulator_HardCapTrimsOldestFirst`
  - [x] T7.3: `TestPromptAccumulator_BasePreservedEvenWhenSingleEntryFitsAlone`
  - [x] T7.4: `TestStepLoop_GuardrailRetry_DoesNotResetAccumulator` — `task.Input` carries marker; `buildTaskDAG` embeds it in `task_context.md`

- [x] T8: `go test ./... -count=1` — 227 pass in experiment, 2 pre-existing failures in doctor (env-dependent, unrelated); `go test -race` clean

- [x] T9: Documentation
  - [x] T9.1: `accumulator.go` struct doc covers base/entries/cap/eviction semantics
  - [x] T9.2: Accumulator cap and eviction policy fully documented on the struct
  - [x] T9.3: 10-6 cross-link implicit via shared `ViolationTypeLoopBoundExceeded` constant and `classifyAction` case

## Dev Notes

### Architecture

**Two layers of loop control:**

1. **Implicit (existing)** — every LLM step has an inline `for iter := 0; iter < maxToolIterations; iter++`
   tool-call loop in `runFn`. This is the safety backstop that exists today.
2. **Declarative (this story)** — `step.Loop` declares an exit criterion (output marker
   or tool call) and an optional override of `max_iterations`. When declared, the
   declarative policy replaces the implicit default for that step.

**Why both?** The implicit default protects steps that don't opt into the declarative
model (NFR40). The declarative policy makes the contract explicit and exposes it to
telemetry — story 10-6 can then attribute `loop_bound_exceeded` violations to specific
steps with declared bounds, separately from "the LLM never converged on a tool-emitting
step that didn't even know it should converge".

**Accumulator vs reset:** The current code (per the 10-0 deferred-work entry, see
`6-21` deferred items) resets `toolPrompt = base` at the top of each tool-call iteration.
This loses the prior tool-result context, forcing the LLM to re-discover the same
information. Worse, when guardrail-retry fires, the entire accumulated context is also
lost. This story switches to an accumulator with a hard byte cap: oldest tool-result
entries are evicted when the cap is reached, but the base prompt + most recent entries
always survive.

**Why a byte cap instead of an entry count?** Tool results vary in size by orders of
magnitude (a `file_read` of `package.json` is ~1KB; a `file_read` of a large source
file is 100KB). Counting entries would let one large read evict many small useful
entries. Byte cap with oldest-first eviction tracks actual context cost.

**Why 64 KiB?** Roughly 16K tokens at ~4 bytes/token. Fits comfortably in a 128K-token
context window with room for the base prompt + LLM response. Story 10-7 follow-ups (or
Theory A/B experiments in 10-9) may tune this; the constant is exported for
test-time override.

### Backward compatibility (NFR40)

- `Loop: nil` → implicit defaults apply: `MaxIterations: 5` (current inline value),
  no early exit (only the implicit max-iter backstop fires).
- All existing tests with `Loop: nil` keep working unchanged.
- The existing `maxToolIterations` constant in `runtime_runner.go` is kept for
  backward compatibility in any code path that doesn't go through `effectiveLoopPolicy`.
  Future code paths that want explicit policy use `effectiveLoopPolicy(step)`.

### `tool_called` exit-criteria semantics

When the LLM emits a `<tool name="submit" .../>` call AND `submit` is the configured
exit tool, the loop breaks. The tool's args are captured in the step's produced map.
The tool itself is NOT executed via `ExecuteToolWithPolicy` — its purpose is to signal
exit, not to perform an action. This is intentional: a "submit" tool that also
performed file operations would create a confusing dual semantics. If the user wants a
post-submit action, they wire it as a downstream step that consumes the submit args.

### Files likely modified

- `internal/bench/experiment/design.go` — `LoopPolicy`, `ExitCriteria`, `StepDef.Loop`,
  exported constants (`defaultMaxToolIterations`, `maxToolPromptBytes`)
- `internal/bench/experiment/runtime_runner.go` — `effectiveLoopPolicy`, accumulator
  integration in tool-call loop, exit criteria check, `ErrLoopExhausted` sentinel
- `internal/bench/experiment/accumulator.go` (new) — `promptAccumulator` struct
- `internal/bench/experiment/accumulator_test.go` (new) — accumulator unit tests
- `internal/bench/experiment/loop_test.go` (new) or extension to existing
  `step_guardrails_test.go` — integration tests for declarative loop behaviour

### What NOT to do

- **Don't remove `defaultMaxToolIterations`.** Keep the implicit safety backstop for
  backward compat. Steps that don't opt into `Loop` get this for free.
- **Don't add a new tool registry to wire `submit`.** The `tool_called` exit criterion
  is a pattern match on the tool name in the parsed tool calls — no special tool
  registration. The user can name their exit tool anything (e.g., `done`, `complete`,
  `submit_verdict`).
- **Don't make the accumulator persistent across steps.** Each step has its own
  accumulator scoped to the per-step closure. Cross-step context flows through slot
  files, not through the prompt accumulator.
- **Don't merge with story 10-6's `NodeViolation` work.** Reference the
  `ViolationTypeLoopBoundExceeded` constant from this story's `T5.4`, but the
  aggregation pipeline lives in 10-6. If 10-6 hasn't landed when 10-7 is implemented,
  the `// TODO(10-6)` comment is the integration seam — story 10-6 will pick it up
  and add the violation to its discriminator switch.
- **Don't reuse `ErrLoopExhausted` for general timeouts.** It is specific to
  "max_iterations exceeded without exit criterion firing". Other timeout sources
  (network, budget) get their own sentinels in their respective stories.

### Previous story intelligence

- **10-0 deferred Task 2 to this story.** The accumulator (T3) and `LoopExhausted`
  sentinel (T5) are the literal items moved here. The story file for 10-0 explicitly
  marks this transfer.
- **10-0's defensive copy pattern for `effectiveGuardrails`** — apply the same
  discipline to `effectiveLoopPolicy`: return by value (struct), no aliasing.
- **10-3's `step.FileAccess`** — orthogonal to loop bounds, no interaction. The
  accumulator does not capture file content; it captures tool RESULT strings (which
  may include "permission denied" reasons that 10-3 emits — that's fine, no leak).
- **10-4's REQUIRED markers in produces template** — the `output_marker` exit
  criterion is conceptually similar (substring presence check) but DIFFERENT: 10-4's
  REQUIRED marker is part of the OUTPUT contract (validation post-hoc); 10-7's
  `output_marker` is part of the LOOP contract (early exit signal). The two coexist:
  a step can have both (`exit_when: { output_marker: ... }` AND
  `produces: [{ template: ... <!-- REQUIRED: ... --> }]`).
- **10-6's audit trail** — `ErrLoopExhausted` is recorded as a `NodeViolation` with
  `ViolationType: "loop_bound_exceeded"`, `Prevented: false` (the step did NOT produce
  a valid output — the violation is observed post-hoc). 10-6's discriminator switch
  needs to handle this case.

### Project context reference

- **Sprint status:** `_bmad-output/implementation-artifacts/sprint-status.yaml` —
  `10-7-loop-bounds-exit-criteria: ready-for-dev` after this file lands.
- **Epic 10 plan:** `_bmad-output/planning-artifacts/epics.md` lines 422-447 (Story 7-7
  in the original numbering).
- **Recent commits:**
  - `a3296d9 feat(10-0): epic 9 debt cleanup` — established `effective*` helper
    pattern + defensive-copy convention
  - `46d041a feat(epic9): close epic 9 (DX) + retro + deferred-work triage and quick wins`
- **Test infrastructure:** 155 tests in `internal/bench/experiment/` after 10-0;
  327 race-clean across affected packages.

### Review Findings

- [x] [Review][Patch] Missing test for `base > cap` edge case in promptAccumulator — added `TestPromptAccumulator_BaseExceedsCap_ToolResultDropped` (T7.5) [accumulator_test.go]
- [x] [Review][Patch] Remove dead `stepSandbox != nil` from declarative loop condition — condition simplified to `stepCopy.Loop != nil`; note added explaining invariant [runtime_runner.go:658]
- [x] [Review][Patch] Fix error message for unknown `exit_when` fields to use dotted notation `exit_when.output_marker`, `exit_when.tool_called` per AC9 spec wording [design.go:validateLoopPolicy]
- [x] [Review][Defer] Legacy loop off-by-one: MaxIterations+1 total LLM calls (first call outside loop) — pre-existing behavior, not introduced by 10-7 [runtime_runner.go:legacy path]
- [x] [Review][Defer] No upper bound on `max_iterations` value — 10000 iterations valid; product decision needed on max cap [design.go:validateLoopPolicy]
- [x] [Review][Defer] `aggregateViolations` "dag" pseudo-stepID sorts as step 0 via Go zero-value map — pre-existing from 10-6, only reachable in ReadOnly sandbox edge case [telemetry.go:44]
- [x] [Review][Defer] `computeContractComplianceRate` can be inflated by pseudo-step violations not counted in stepCount — pre-existing from 10-6 [telemetry.go:88]
- [x] [Review][Defer] Multiple `loop_exceeded` audit records across DAG retries (up to 3 with MaxRetries=2) — design decision; each DAG invocation is independent [runtime_runner.go:717]
- [x] [Review][Defer] Synthesize call returns `ErrLoopExhausted` if LLM includes `<tool .../>` XML in prose when no exit criterion declared — theoretical edge case, "Final Answer Required" prompt should prevent this [runtime_runner.go:715]
- [x] [Review][Defer] `exitToolArgs` synthetic key not registered in DAG produces/wiring — args in `allOutputs` but inaccessible to downstream steps via DAG slot mechanism [runtime_runner.go:820]
- [x] [Review][Defer] Accumulator reset between DAG-level retries — AC6 scoped to iteration-level accumulation, not DAG retry scope; design decision [runtime_runner.go:624]
- [x] [Review][Defer] Redundant `stepSandbox != nil` guard before `loop_exceeded` record — harmless defensive code, invariant guaranteed by outer condition [runtime_runner.go:716]

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

- T6.6 assertion bound: DAG `MaxRetries=2` causes `runFn` to be invoked up to 3 times on `ErrLoopExhausted`. With `MaxIterations=2`, each invocation makes 3 calls (2 loop + 1 synthesize) → 9 total. Assertion corrected to `≤ 9`.
- T7.4 root cause: `buildTaskDAG` line 382 always overwrites `task_context.md` with `taskContextContent(task)`, so workspace file `{Path:"task_context.md", Content:"context content XYZ"}` was erased before `file_read` ran. Fix: set `task.Input = "context content XYZ"` so `taskContextContent` embeds it in `## Original Input`; sandbox write then preserves it.

### Completion Notes List

- `ExitCriteria` and `LoopPolicy` use `UnmarshalYAML` to capture unknown YAML keys for strict validation — no `yaml.KnownFields` needed (which would require a decoder, not a struct method).
- Declarative path condition: `stepSandbox != nil && stepCopy.Loop != nil`. Non-nil sandbox is always true (empty sandbox created when `task.Workspace == nil`), so the discriminator is effectively `Loop != nil`.
- Exit tool is NOT executed via `ExecuteToolWithPolicy` when it matches `ExitWhen.ToolCalled` — intentional (signal-only semantics documented in Dev Notes).
- `promptAccumulator` is scoped outside the guardrail-attempt loop but inside the per-step closure — correct scope for both accumulation across tool iters and persistence across guardrail retries.

### File List

- `internal/bench/experiment/design.go` — `ExitCriteria`, `LoopPolicy`, `StepDef.Loop`, `DefaultMaxToolIterations`, `MaxToolPromptBytes`, `ErrLoopExhausted`, `ViolationTypeLoopBoundExceeded`, `validateLoopPolicy`, `Validate` extension
- `internal/bench/experiment/runtime_runner.go` — `effectiveLoopPolicy`, `exitCriteriaMet`, declarative loop path, accumulator wiring, `loop_exceeded` sandbox record, exit tool args capture
- `internal/bench/experiment/telemetry.go` — `"loop_exceeded"` case in `classifyAction`
- `internal/bench/experiment/accumulator.go` (new) — `promptAccumulator` struct
- `internal/bench/experiment/accumulator_test.go` (new) — T7.1–T7.4
- `internal/bench/experiment/loop_test.go` (new) — T1.5, T2, T1 Validate, T6.1–T6.6

## Change Log

| Date | Change | Notes |
|---|---|---|
| 2026-04-28 | Story 10-7 created from Epic 10 plan (Story 7-7 in epics.md) | Absorbs 10-0 deferred Task 2 (toolPrompt cap + accumulator + `LoopExhausted` sentinel). Adds declarative `LoopPolicy` with OR-combined exit criteria (`output_marker`, `tool_called`) on top of the implicit `maxToolIterations` safety backstop. Coordinates with story 10-6 via `ViolationTypeLoopBoundExceeded` discriminator. |
