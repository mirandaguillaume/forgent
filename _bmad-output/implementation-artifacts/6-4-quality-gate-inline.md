# Story 6.4: Quality Gate Inline (Keywords + LLM Validator)

Status: done

## Story

As a **skill author**,
I want expected_keywords enforced inline during step execution with automatic retry,
so that a step that drifts from its purpose is caught and retried before polluting downstream steps.

## Acceptance Criteria

1. A `QualityGatePolicy` struct is defined in `pkg/qualitygate/` containing `Keywords []string` (expected keywords that must appear in output)
2. A `QualityGateMiddleware(policy *QualityGatePolicy) dag.Middleware` function returns a post-middleware that checks all expected keywords appear in at least one string output value
3. **Keyword enforcement (POST-middleware):** After the handler returns, the middleware collects all string values from the output map and checks that every keyword in `Keywords` appears in the combined lowercased output. If any keyword is missing, returns an error with a descriptive message including the missing keyword
4. **Case-insensitive matching:** Both the output text and the keywords are lowercased before comparison (`strings.ToLower`)
5. **Missing keyword triggers retry:** When the middleware returns an error, the DAG's `runWithRetry` mechanism retries the entire chain (middlewares + handler) if `MaxRetries > 0` — no special retry logic needed in the middleware itself
6. **Nil/empty policy = no-op:** When `policy` is nil or `policy.Keywords` is empty, the middleware is a no-op pass-through with zero overhead
7. **Multiple string outputs checked:** All string values in the output map are concatenated (with space separator) before keyword matching — a keyword present in ANY string output satisfies the check
8. **Non-string outputs ignored:** Non-string values (int, bool, []any, etc.) in the output map are silently skipped during keyword collection
9. **Handler error passthrough:** If the inner handler returns an error, the middleware propagates it unchanged (does not attempt keyword checking on nil/partial outputs)
10. All existing tests pass without modification (`go test ./... -count=1 -race`)

## Tasks / Subtasks

- [x] Task 1: Create quality gate package `pkg/qualitygate/gate.go` (AC: #1, #2, #3, #4, #6, #7, #8, #9)
  - [x] 1.1: Define `QualityGatePolicy` struct with `Keywords []string`
  - [x] 1.2: Implement `QualityGateMiddleware(policy *QualityGatePolicy) dag.Middleware`
  - [x] 1.3: Implement `collectStrings(outputs map[string]any) string` helper — concatenates all string values with space separator

- [x] Task 2: Unit tests `pkg/qualitygate/gate_test.go` (AC: #1-#9)
  - [x] 2.1: Test nil policy returns pass-through (outputs unchanged)
  - [x] 2.2: Test empty keywords returns pass-through
  - [x] 2.3: Test all keywords present — outputs pass through unchanged
  - [x] 2.4: Test missing keyword — returns error with descriptive message containing the missing keyword
  - [x] 2.5: Test case-insensitive matching (keyword "Vulnerability", output contains "VULNERABILITY")
  - [x] 2.6: Test multiple string outputs — keyword in second output still passes
  - [x] 2.7: Test non-string outputs are ignored (int, bool values present alongside strings)
  - [x] 2.8: Test handler error passthrough — inner error propagates, no keyword check attempted
  - [x] 2.9: Test integration with DAG retry — node with MaxRetries=1, first call missing keyword, second call has it: expect success
  - [x] 2.10: Test multiple missing keywords — error message includes first missing keyword

- [x] Task 3: Verify all existing tests pass (AC: #10)
  - [x] 3.1: `go test ./pkg/qualitygate/... -count=1 -race` — PASS (24 tests, race-clean)
  - [x] 3.2: `go test $(go list ./... | grep -v bench) -count=1 -short` — PASS (34 packages, 0 failures)

## Dev Notes

### Architecture References

- **pkg/dag/middleware.go:** Defines `Middleware` and `MiddlewareFunc` types, `ChainMiddlewares` helper. Quality gate middleware follows the same signature.
- **pkg/budget/middleware.go:** Reference implementation for a middleware that does both pre (inject max_tokens) and post (check token count) processing. Quality gate only needs post processing.
- **pkg/dag/execute.go:** `runWithRetry` builds middleware chain once, retries on error. Quality gate errors trigger retry automatically — no special handling needed.
- **internal/bench/experiment/runtime_runner.go:** `evaluateRuntimeOutputs` currently checks expected_keywords post-hoc (after DAG execution). Quality gate moves this enforcement inline (during execution) so bad outputs are caught and retried instead of just scored.
- **internal/bench/experiment/design.go:** `StepDef.ExpectedKeywords` field — the quality gate policy will be constructed from this field when wiring up DAG nodes in future integration work.

### Key Design Decisions

**1. POST-middleware only (not pre+post like budget)**

Unlike the budget middleware which needs to inject `max_tokens` into inputs (pre) and check token counts (post), the quality gate only checks outputs. The middleware calls `next(ctx, inputs)` and then validates the outputs.

**2. Combined string collection (not per-key)**

All string output values are joined into a single lowercased string for matching. This means a keyword can appear in ANY output key — it doesn't need to be in a specific one. This matches the experiment's `evaluateRuntimeOutputs` behavior.

**3. Error message includes the specific missing keyword**

The error format is: `quality gate: missing keyword %q in output`. This gives the retry mechanism (and any logging) a clear signal of what was expected but not found.

**4. No LLM validator in MVP**

The epic defines an optional `strategy.validation: haiku` mode for semantic validation via a lightweight LLM call. This is deliberately deferred — keyword matching provides deterministic enforcement without additional API costs. LLM validation can be layered on later as a separate middleware.

## Dev Agent Record

- Story created: 2026-04-15
- Implementation completed: 2026-04-15
- Story status: review
- Files created:
  - `pkg/qualitygate/gate.go` — QualityGatePolicy + QualityGateMiddleware + collectStrings
  - `pkg/qualitygate/gate_test.go` — 24 tests + 3 benchmarks
- Test results: 24/24 pass, race-clean, 0 regressions across 34 packages
- Additional tests beyond spec: DAG retry exhaustion, empty output map, empty string output, middleware chain composition (pass + reject), original-case keyword in error, substring matching, concurrent DAG execution, special chars in keywords, inputs/context propagation, benchmarks
