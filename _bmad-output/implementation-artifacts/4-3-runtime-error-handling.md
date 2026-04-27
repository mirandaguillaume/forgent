# Story 4.3: Runtime Error Handling and Diagnostics

Status: done

## Story

As a **developer**,
I want clear error messages when a doctor DAG node fails,
so that I can diagnose and report issues.

## Acceptance Criteria

1. When a skill node fails, the runtime reports: which node failed, what input it received, and the failure reason (FR49)
2. When doctor runs without `--fix`, Layer 3 (rewriter) is skipped entirely — DAG terminates after recommendation-builder
3. When a retry-configured node fails after all retries, the error includes: node name, attempt count, all failure reasons
4. In `--debug` mode, each node logs: start time, input data keys, LLM prompt (if applicable), output data keys, duration
5. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [x] Task 1: Rich error diagnostics (AC: #1, #3)
  - [x] 1.1: Create `internal/doctor/diagnostic.go` — `NodeError` struct with: NodeID, Input keys, Err, Attempts (for retry), AllErrors (slice)
  - [x] 1.2: `NodeError.Error()` produces actionable message: "Node 'analyzer' failed: <reason>. Input: [agent_file, detected_format]. Retry: 2/3 attempts."
  - [x] 1.3: Wrap each node's Run function to catch panics and convert to `NodeError`
  - [x] 1.4: For retry nodes (analyzer), collect all attempt errors in AllErrors (rewriter dropped from scope in 4.2)
  - [x] 1.5: Write tests for single failure, retry exhaustion, panic recovery

- [x] Task 2: Conditional rewriter skip (AC: #2)
  - [x] 2.1: Rewriter node was removed from scope in Story 4.2 (Epic 8 retro) — documented in BuildDAG comment with guidance for future re-introduction
  - [x] 2.2: `pkg/dag` already silently skips nil Run nodes — verified via TestNilRunNodeIsSkipped
  - [x] 2.3: DAG terminates after Layer 2 (recommendation-builder) — verified by TestBuildDAG_HasExactlyThreeLayers and TestBuildDAG_RecommendationBuilderIsTerminal
  - [x] 2.4: Tests verify 3-layer topology and terminal node; full --fix comparison deferred with rewriter

- [x] Task 3: Debug mode logging (AC: #4)
  - [x] 3.1: Created `wrapWithDebug(node *dag.Node, w io.Writer)` in diagnostic.go; BuildDAG(debug bool) passes os.Stderr when debug=true
  - [x] 3.2: On node start: `[DEBUG] Node '<id>' starting, input keys: [...]`
  - [x] 3.3: For LLM nodes (consumer of "provider"): extra `[DEBUG] Node '<id>': LLM analysis call, context keys: [...]` line (prompt content not accessible from wrapper layer)
  - [x] 3.4: On node end: `[DEBUG] Node '<id>' completed in <duration>, output keys: [...]`
  - [x] 3.5: On node failure: `[DEBUG] Node '<id>' failed after <duration>: <error>`
  - [x] 3.6: Tests: TestWrapWithDebug_LogsStartAndComplete, TestWrapWithDebug_LogsOnFailure, TestWrapWithDebug_SilentWhenNilWriter, TestWrapWithDebug_LLMNodeLogsExtraLine

- [x] Task 4: Retry configuration (AC: #3)
  - [x] 4.1: Analyzer node: 3 total attempts (2 retries) via wrapWithRetry(analyzerNode(), 3) — backoff not supported by pkg/dag and not added per "Do NOT modify pkg/dag" constraint
  - [x] 4.2: Rewriter node removed from scope — N/A
  - [x] 4.3: Code nodes (format-detector, context-enricher, recommendation-builder) have maxAttempts=1 (no retry)
  - [x] 4.4: Retry implemented via wrapWithRetry wrapper in doctor orchestrator layer; Node.MaxRetries zeroed to prevent pkg/dag double-retry
  - [x] 4.5: TestRunDAG_RetryOnTransientFailure (fails once → succeeds), TestRunDAG_ExhaustsRetries_ReturnsNodeError

## Dev Notes

### Architecture References

- **ADR-4 (Topology):** "Layer 3: rewriter (LLM, ~10-15s, conditional on --fix)"
- **FR49:** "Runtime reports clear error diagnostics on node failure — which node, what input, failure reason."
- **pkg/dag execute.go:** Already has `runWithRetry` supporting S4 (RetryCount) and S5 (RetryDelay). This story uses existing retry infrastructure.
- **NFR28:** "Actionable error messages with suggestions."

### Existing DAG Retry Support

`pkg/dag/execute.go` already implements retry:
- `Node.RetryCount` — max attempts (0 = no retry)
- `Node.RetryDelay` — backoff between attempts
- `runWithRetry` — handles the loop, passes attempts to error

This story configures retry on LLM nodes and adds richer error reporting.

### Dependencies on Previous Stories

- **Story 4.2 (Embedded YAML and Runtime DAG):** Must be complete — provides BuildDAG, RunDAG, node wrappers.
- **`pkg/dag`:** Existing retry support, nil-node skip, layer execution.

### Previous Story Intelligence (from Epic 1 retro)

- **Action #4:** "Always review before done."
- **OpenRouter timeout issue:** "Existing OpenRouter provider has no timeout, blocks indefinitely on large files." The diagnostic wrapper should log timeouts clearly. If Action #2 (fix OpenRouter timeout) hasn't been resolved, diagnostic logging helps identify the issue.

### What NOT to Build

- Do NOT modify `pkg/dag` — adapt diagnostics in the doctor orchestrator layer
- Do NOT implement structured logging (logrus, zap) — use stderr with `[DEBUG]` prefix as per architecture pattern
- Do NOT implement log file output — stderr only for debug mode
- Do NOT implement HITL (human-in-the-loop) pause/resume — that's v2 DAG scope

### Project Structure

New files:
- `internal/doctor/diagnostic.go` — NodeError, wrapWithDebug, panic recovery
- `internal/doctor/diagnostic_test.go`

Modified files:
- `internal/doctor/orchestrator.go` (from Story 4.2) — add debug wrapping, retry config, conditional rewriter

## Testing Strategy

- **Error diagnostic tests:** Node failure → NodeError with correct fields. Retry exhaustion → AllErrors populated.
- **Panic recovery tests:** Node Run panics → caught, converted to NodeError, not propagated.
- **Conditional tests:** --fix=false → Layer 3 skipped. --fix=true → Layer 3 executes.
- **Debug output tests:** Capture stderr, verify debug lines present when debug=true, absent when false.
- **Retry tests:** Mock provider fails once → retry succeeds. Mock provider fails all attempts → all errors collected.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-4] — Conditional rewriter, layer topology
- [Source: _bmad-output/planning-artifacts/prd.md#FR49] — Error diagnostics on node failure
- [Source: _bmad-output/planning-artifacts/prd.md#NFR28] — Actionable error messages
- [Source: _bmad-output/planning-artifacts/prd.md#NFR36] — --debug mode with full diagnostic output
- [Source: pkg/dag/execute.go] — runWithRetry, RetryCount, RetryDelay

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

- `wrapWithRetry` resets `Node.MaxRetries = 0` to prevent pkg/dag from double-retrying the already-retrying Run closure
- `BuildDAG` signature changed to `BuildDAG(debug bool)` — all callers updated (orchestrator_test.go, equivalence_test.go, cmd/doctor.go)
- Task 2 (conditional rewriter skip): rewriter was dropped in Story 4.2, so 2.1 is documented via comment in BuildDAG; nil-skip behavior verified via TestNilRunNodeIsSkipped
- Task 3.3 (prompt logging): prompt is only accessible inside analyzer.Analyze; debug wrapper logs LLM node context keys as a pragmatic substitute
- RetryDelay not implemented: pkg/dag Node struct has no RetryDelay field (constraint: "Do NOT modify pkg/dag"); omitted with comment

### Completion Notes List

- Created `internal/doctor/diagnostic.go` with `NodeError`, `wrapWithPanic`, `wrapWithRetry`, `wrapWithDebug`, and `sortedKeys` helpers
- Updated `BuildDAG(debug bool)` to wire all three wrappers for every node; analyzer gets 3 total attempts (2 retries)
- `wrapWithDebug` uses `io.Writer` parameter for testability; BuildDAG passes os.Stderr when debug=true
- All tests pass: 16 diagnostic tests + 12 orchestrator tests (including retry, debug, topology) + full suite (30 packages)

## Review Findings

_Code review — 2026-04-10_

### Patches

- [x] [Review][Patch] CRITICAL: `wrapWithRetry` panics on `allErrors[len(allErrors)-1]` when context is cancelled before any attempt — fixed: guard appends `ctx.Err()` when allErrors is empty [internal/doctor/diagnostic.go]
- [x] [Review][Patch] Panic-then-retry composition double-wraps `NodeError` and retries deterministic panics — fixed: retry loop detects panic-originated NodeError and breaks immediately [internal/doctor/diagnostic.go]
- [x] [Review][Patch] AC #3 gap: `Error()` only includes the last failure reason — fixed: Error() now lists all per-attempt errors with `[1] ... [2] ... [3] ...` format [internal/doctor/diagnostic.go]

### Deferred

- [x] [Review][Defer] Concurrent `fmt.Fprintf` to shared `os.Stderr` from parallel Layer 1 nodes — cosmetic interleaving only; `os.File.Write` is atomic for short lines on Linux — deferred, pre-existing
- [x] [Review][Defer] No backoff between retry attempts — constrained by "Do NOT modify pkg/dag" and no `RetryDelay` in Node struct — deferred, architectural
- [x] [Review][Defer] Panic wrapper swallows stack trace — `recover()` loses location info; add `debug.Stack()` in future hardening — deferred
- [x] [Review][Defer] LLM node detection via magic string `"provider"` — no shared constant; defer to constants work — deferred, minor
- [x] [Review][Defer] Hardcoded retry counts as magic numbers (1 and 3) — consider named constants — deferred, minor
- [x] [Review][Defer] Debug wrapper only sees final retry error, not per-attempt — observability gap for intermittent LLM failures — deferred, minor
- [x] [Review][Defer] Misleading `Attempts` count when context cancels mid-retry — metadata is technically correct for completed attempts — deferred, edge case

## Change Log

- 2026-04-10: Implemented story 4-3. Created NodeError + diagnostic wrappers (panic recovery, retry with AllErrors, debug logging). Updated BuildDAG(debug bool) to apply all wrappers. Analyzer configured with 3 total attempts (2 retries). All 30 test packages pass.

### File List

- `internal/doctor/diagnostic.go` (new)
- `internal/doctor/diagnostic_test.go` (new)
- `internal/doctor/orchestrator.go` (modified — BuildDAG signature + wrappers)
- `internal/doctor/orchestrator_test.go` (modified — BuildDAG(false) + new tests)
- `internal/doctor/equivalence_test.go` (modified — BuildDAG(false))
- `internal/cmd/doctor.go` (modified — doctor.BuildDAG(debugFlag))
