# Story 10-8: Per-Node Experiment Metrics + Theory A/B

**Epic:** 10 — Node-Level Behavioral Contracts
**Status:** done

## Story

As a **developer running a Forgent experiment**,
I want the experiment report to include per-node contract compliance metrics and a Theory A/B breakdown,
So that I can determine whether structural enforcement constrains agents (Theory A) or reveals pre-existing compliant behavior (Theory B).

## Acceptance Criteria

**AC1:** Given an experiment run completes with violation telemetry (Story 10-6),
When the report is generated,
Then for each node (step ID), the report includes `ContractComplianceRate` aggregated across all runs of all tasks that exercise that node — fraction of runs where the node had zero attempt-1 violations.

**AC2:** Given the telemetry distinguishes `attempted: true, prevented: true` (Theory A — agent tried, harness blocked) from `attempted: false` (Theory B — agent never attempted),
When the report aggregates results per (node, violation_type),
Then it shows for each violation type:
- `% attempted` — runs where the agent attempted at least one operation that triggered the enforcement
- `% prevented` — runs where the harness blocked an attempted operation (`prevented: true`)
- `% spontaneously_compliant` — runs where the agent never attempted (`attempted == 0` for this node × violation_type)
- Sum: `attempted + spontaneously_compliant == 100%` per node × violation_type × runs cohort.

**AC3:** Given the ranked condition runs alongside runtime and text (`forgent-bench --condition runtime,ranked` or via `CompareThree`),
When the per-node breakdown is rendered,
Then all three conditions appear as columns in the per-node breakdown table,
And the table is sorted by node ID for stable diff-friendly output.

**AC4:** Given the baseline experiment results (2026-04-27, the keyword-guardrail-only run before any node-level contracts shipped) are available as a checkpoint file,
When the new report is generated AND the path to the baseline checkpoint is supplied (via `--baseline /path/to/baseline.json` flag or `BENCH_BASELINE` env var),
Then the report includes a delta column comparing each node's current `ContractComplianceRate` to the baseline,
And rows where the delta is statistically meaningful (95% bootstrap CI from story 5-2 doesn't include zero) are highlighted in the rendered output.

**AC5:** Given the baseline checkpoint is unavailable or the path is invalid,
When the report is generated,
Then the report renders without the delta column and emits a stderr warning naming the missing path,
But does NOT fail the experiment — partial reporting is preferable to no reporting.

**AC6:** Given a node × violation_type cell has zero observations (no run exercised that combination),
When the report renders that cell,
Then it displays `n/a` (or `—`) rather than `0%` to distinguish "no data" from "0% violations".

**AC7:** Given the experiment runs with `RunsPerTask: 1`,
When the report computes per-node aggregates,
Then the metrics still render but are labeled `(low confidence — single run)` so the reader knows the sample size is insufficient for inference.

**AC8:** Given `go test ./... -count=1` and `go test -race ./internal/bench/experiment/...`,
Then all existing tests pass without modification (race-clean),
And the new tests cover per-node aggregation, Theory A/B breakdown computation, three-condition rendering, baseline-delta loading, and the missing-baseline graceful fallback.

## Tasks/Subtasks

- [ ] T1: Per-node aggregation data model
  - [ ] T1.1: Define `NodeMetrics` struct in `internal/bench/experiment/design.go`:
        ```go
        type NodeMetrics struct {
            NodeID                  string
            Condition               string                              // "text" | "runtime" | "ranked"
            Runs                    int                                 // total runs that exercised this node
            ContractComplianceRate  float64                             // 0.0–1.0 across runs
            ByViolationType         map[string]ViolationTypeBreakdown   // key: ViolationType constants from 10-6
        }

        type ViolationTypeBreakdown struct {
            AttemptedPct                 float64 // 0.0–1.0 — runs that recorded ≥1 violation of this type
            PreventedPct                 float64 // 0.0–1.0 — runs where ≥1 violation was prevented (fraction of attempted)
            SpontaneouslyCompliantPct    float64 // 0.0–1.0 — runs that recorded 0 violations of this type
            BaselineDelta                *float64 // pointer = nil if no baseline; else delta (current - baseline) for ContractComplianceRate
        }
        ```
  - [ ] T1.2: Add `NodeMetrics []NodeMetrics` to `ComparisonResult` (in `compare.go`).
        Sorted by `(NodeID, Condition)` for stable rendering.
  - [ ] T1.3: Document the semantics of each percentage in a doc comment so readers
        do not confuse "% prevented" (subset of attempted) with "% spontaneously_compliant"
        (subset of total runs).

- [ ] T2: Aggregation function
  - [ ] T2.1: Add `ComputeNodeMetrics(text, runtime, ranked []TaskResult) []NodeMetrics`
        in `compare.go`. Walks all `TaskResult.NodeViolations` (from story 10-6) and tallies
        per (node, condition, violation_type).
  - [ ] T2.2: For each (node, condition):
        - `Runs` = number of `TaskResult` entries that contain at least one StepResult with that step ID
        - For each violation type:
          - `attempted` = runs where ≥1 NodeViolation has matching `(NodeID, ViolationType)` AND `Attempted == true`
          - `prevented` = runs where ≥1 NodeViolation has matching `(NodeID, ViolationType)` AND `Prevented == true`
          - `spontaneously_compliant = Runs - attempted`
          - Convert to fractions of `Runs`
        - `ContractComplianceRate` = mean of `TaskResult.ContractComplianceRate` from runs that exercised that node (NOT a fresh recompute — leverage the per-task value from story 10-6)
  - [ ] T2.3: Wire into `Compare` and `CompareThree`: after the existing per-condition
        aggregations, call `ComputeNodeMetrics` and assign to `cr.NodeMetrics`. The new
        slice is sorted ascending by `(NodeID, Condition)` (alphabetical on each).
  - [ ] T2.4: Defensive: empty input → empty slice (not nil — easier for renderers and
        JSON marshalling). Single run → metrics computed but flagged via the renderer
        (T4.4) with the "low confidence" label.

- [ ] T3: Baseline checkpoint loading
  - [ ] T3.1: Define `BaselineSnapshot` struct in `internal/bench/experiment/baseline.go`
        (new file):
        ```go
        type BaselineSnapshot struct {
            CapturedAt          string                                          // ISO date
            Description         string                                          // e.g. "keyword-guardrail baseline"
            NodeComplianceRates map[string]map[string]float64                   // node_id → condition → ContractComplianceRate
        }
        ```
  - [ ] T3.2: Add `LoadBaseline(path string) (*BaselineSnapshot, error)`. Reads the file,
        unmarshals JSON, validates that at least `NodeComplianceRates` is non-empty.
        Returns `(nil, err)` for missing/malformed; caller decides whether to fail
        (it doesn't — see AC #5 / T4.5).
  - [ ] T3.3: Add `SaveBaseline(path string, cr *ComparisonResult, description string) error`
        — captures the current `ComparisonResult.NodeMetrics` into a `BaselineSnapshot`
        and writes it. This lets future stories (10-9) trigger a baseline capture before
        running with structural enforcement disabled, then re-run with enforcement and
        diff via `BaselineDelta`.
  - [ ] T3.4: Define an env var fallback: `BENCH_BASELINE=path` is consulted when the
        CLI flag `--baseline` is absent. Document the precedence: flag > env > none.

- [ ] T4: Report rendering (extend `report.go`)
  - [ ] T4.1: Add a new section to the Markdown report (`FormatReport` in `report.go`):
        `## Per-Node Contract Compliance` with a 3-condition table:
        ```
        | Node | Text Compliance | Runtime Compliance | Ranked Compliance | Δ vs Baseline |
        |------|-----------------|--------------------|-------------------|---------------|
        | s1   | 67%             | 100%               | 100%              | +33% (CI sig) |
        | …    | …               | …                  | …                 | …             |
        ```
  - [ ] T4.2: Add a sub-section `### Theory A/B Breakdown` per violation type.
        For each violation type (`guardrail`, `file_permission`, `output_contract`,
        `network`, `loop_bound_exceeded`):
        ```
        | Node | Attempted | Prevented | Spontaneously Compliant |
        ```
        One sub-table per condition (text, runtime, ranked) when ranked is present.
        Rows where `Attempted == 0` AND `SpontaneouslyCompliant == 100%` should still
        appear so readers see "the agent never attempted this in any run" as a positive
        signal.
  - [ ] T4.3: Highlight statistically meaningful deltas (when `BaselineDelta != nil` AND
        the delta sign is consistent with story 5-2's bootstrap CI not including zero).
        Use Markdown bold for highlighted cells: `**+33% (CI sig)**`. CI computation
        reuses the existing `bootstrapCI` from story 5-2 (after it landed in 10-0's
        Compare alignment work; see T2.4 of 10-0).
  - [ ] T4.4: Cells with `Runs == 0` render as `n/a`. The "low confidence" label
        applies when the renderer's outer `cr.Runs == 1` (single-run mode); add a
        warning paragraph at the top of the per-node section in that case.
  - [ ] T4.5: When the baseline is unavailable (passed as nil), render the table WITHOUT
        the `Δ vs Baseline` column. Emit a stderr warning ONCE before rendering naming
        the path that was attempted (or `"no baseline path provided"` if neither flag
        nor env var was set).

- [ ] T5: CLI integration (`forgent-bench`)
  - [ ] T5.1: Add `--baseline string` flag to the `bench` subcommand. Default empty.
        Document: "Path to a baseline JSON snapshot (saved by --save-baseline) for
        delta computation."
  - [ ] T5.2: Add `--save-baseline string` flag. When set, the bench run captures
        the resulting `ComparisonResult.NodeMetrics` into a baseline snapshot at the
        given path. Default empty.
  - [ ] T5.3: Document the workflow in `--help` output and any user-facing docs:
        ```
        # First run: capture baseline (no structural enforcement)
        forgent-bench bench tasks/ --runs 3 --save-baseline baseline.json

        # Second run: enable structural enforcement and compare
        forgent-bench bench tasks/ --runs 3 --baseline baseline.json
        ```

- [ ] T6: Tests
  - [ ] T6.1: `TestComputeNodeMetrics_AggregatesAcrossRuns` — synthetic
        `[]TaskResult` with 3 runs of the same task (3 step nodes); each run has
        different `NodeViolations`. Expect:
        - 3 `NodeMetrics` entries (one per node, one condition only)
        - Each `Runs == 3`
        - `ContractComplianceRate` matches mean of per-task rates
  - [ ] T6.2: `TestComputeNodeMetrics_PerViolationType` — runs with mixed violation
        types per node. Verify `ByViolationType` map populated correctly:
        - `attempted_pct = (runs with ≥1 of this type) / total_runs`
        - `prevented_pct = (runs with ≥1 prevented of this type) / runs_with_attempted`
        - `spontaneously_compliant_pct = 1 - attempted_pct`
  - [ ] T6.3: `TestComputeNodeMetrics_SortsByNodeIDAndCondition` — multi-condition
        input; verify output sorted alphabetically by `(NodeID, Condition)`.
  - [ ] T6.4: `TestComputeNodeMetrics_EmptyInputReturnsEmpty` — `nil` and empty
        `[]TaskResult` both return `[]NodeMetrics{}`, not nil.
  - [ ] T6.5: `TestLoadBaseline_RoundTrip` — `SaveBaseline(tmp, cr, "test")` followed
        by `LoadBaseline(tmp)` returns equivalent `BaselineSnapshot`.
  - [ ] T6.6: `TestLoadBaseline_MissingFileReturnsError` — non-existent path.
        `LoadBaseline` returns `(nil, error)` matching `os.IsNotExist`.
  - [ ] T6.7: `TestLoadBaseline_MalformedJSONReturnsError` — write garbage, expect
        unmarshal error.
  - [ ] T6.8: `TestFormatReport_PerNodeSectionRendered` — call `FormatReport(cr, opts)`
        on a `ComparisonResult` with non-empty `NodeMetrics`. Output contains the
        `## Per-Node Contract Compliance` heading, the table header, at least one
        row, and the Theory A/B sub-section.
  - [ ] T6.9: `TestFormatReport_NoBaselineOmitsDeltaColumn` — `BaselineDelta == nil`
        on every NodeMetrics → output table does NOT contain the `Δ vs Baseline`
        column. Stderr warning emitted.
  - [ ] T6.10: `TestFormatReport_NACellsWhenZeroRuns` — synthetic `NodeMetrics` with
        `Runs == 0`; rendered cells show `n/a`.
  - [ ] T6.11: `TestFormatReport_LowConfidenceLabelOnSingleRun` — `cr.Runs == 1`;
        output contains a "low confidence — single run" warning paragraph.
  - [ ] T6.12: `TestCLI_BenchAcceptsBaselineFlag` (or extend existing CLI test):
        `forgent-bench bench /tmp/tasks --baseline /tmp/baseline.json` exits 0
        with the new flag parsed (does not require a real run — flag-presence test).

- [ ] T7: Run `go test ./... -count=1` and `go test -race ./internal/bench/experiment/...`
       — no regressions, race-clean.

- [ ] T8: Documentation
  - [ ] T8.1: Add a section to the package doc of `internal/bench/experiment/`
        explaining the per-node breakdown, the Theory A/B reading, and how to
        capture/load a baseline.
  - [ ] T8.2: Update any user-facing docs that explain the bench output format to
        mention the new `## Per-Node Contract Compliance` section.

## Dev Notes

### Architecture

**This story is a pure aggregator + renderer.** No new enforcement, no new step semantics.
It consumes the data produced by stories 10-3 through 10-7 (via 10-6's audit-trail
aggregator) and renders cross-cutting metrics. That makes it the lightest of the Epic 10
implementation stories — most of the work is in the rendering and the baseline-diff
plumbing.

**Theory A vs Theory B framework:**

For each (node, violation_type, run):
- `attempted`: did the agent perform an action that triggered the enforcement?
- `prevented`: was that action blocked structurally (vs observed post-hoc)?

Cross-tabulating across runs gives:
- High `attempted` + high `prevented` → **Theory A**: the agent would violate, but the
  harness coerces it. Structural enforcement is doing real work.
- Low `attempted` (high `spontaneously_compliant`) → **Theory B**: the agent doesn't
  try to violate. Structural enforcement is redundant — the LLM is already aligned.
- High `attempted` + low `prevented` → enforcement gap. Either the harness is
  insufficient, or the violation type is detection-only (e.g., guardrail post-hoc).

Distinguishing these matters because the experimental thesis (Forgent's value) depends
on Theory A holding — if every agent is already Theory B, the harness is theatre.

**Why per-node?** A single agent can be Theory A on one node (e.g., always tries to
write to a forbidden path → harness blocks) AND Theory B on another (e.g., never
attempts a network call). Aggregating only at the task level hides this.

**Baseline delta:** The 2026-04-27 experiment captured the no-structural-enforcement
performance (text condition with keyword guardrails only). Delta is `current - baseline`
for `ContractComplianceRate`. Positive delta means structural enforcement improved
compliance. Bootstrap CI from story 5-2 (already plumbed through 10-0's Compare
alignment) tells us if the delta is meaningful or noise.

### Files likely created or modified

- `internal/bench/experiment/design.go` — `NodeMetrics` and `ViolationTypeBreakdown` structs
- `internal/bench/experiment/compare.go` — `ComputeNodeMetrics`, wiring into
  `Compare` and `CompareThree`
- `internal/bench/experiment/baseline.go` (new) — `BaselineSnapshot`, `LoadBaseline`,
  `SaveBaseline`
- `internal/bench/experiment/baseline_test.go` (new) — round-trip + error-path tests
- `internal/bench/experiment/report.go` — new `## Per-Node Contract Compliance`
  section + Theory A/B sub-section
- `internal/bench/cmd/bench.go` — `--baseline` and `--save-baseline` flags
- `internal/bench/experiment/compare_test.go` (or new `node_metrics_test.go`) — aggregation tests

### Backward compatibility (NFR40)

- `cr.NodeMetrics` is a new field on `ComparisonResult`. Old JSON consumers that
  don't expect it will see an unknown field and silently ignore it (Go's
  `json.Unmarshal` default behaviour). Verify any consumer that uses
  `DisallowUnknownFields()` is updated.
- The new `## Per-Node Contract Compliance` Markdown section is appended to the
  existing report. Existing renderings continue to include all prior sections; readers
  scanning for the old format will find what they expect.
- `--baseline` and `--save-baseline` are optional CLI flags with empty defaults.
  Existing invocations without these flags work identically.

### What NOT to do

- **Don't recompute `ContractComplianceRate` per node.** Story 10-6 produces the
  per-task value; this story aggregates it across runs. Reusing the source value
  preserves consistency between the per-task and per-node views.
- **Don't store the baseline checkpoint in git.** Per the user's "no datasets in git"
  rule. Baseline JSON files are local artifacts; document the workflow but the path
  is user-supplied.
- **Don't expand `NodeViolation` to add aggregation-time fields.** All aggregation
  state lives in `NodeMetrics`. The source data is read-only.
- **Don't compute new bootstrap CIs from scratch.** Reuse the existing `bootstrapCI`
  helper from story 5-2 (now at `compare.go` after 10-0's alignment landing). Pass
  it the per-run `ContractComplianceRate` as the metric function.
- **Don't fail the report on missing baseline.** AC #5 explicitly requires
  graceful degradation. The renderer must handle `BaselineDelta == nil` gracefully.
- **Don't merge with story 10-9.** That story RUNS the experiment and uses this
  story's report. Keep the boundary clean: this story produces data + renders;
  10-9 orchestrates the run + interprets results.

### Previous story intelligence

- **10-6 produces the source data** (`NodeViolations` + `ContractComplianceRate`).
  This story is a pure consumer — no changes to the audit trail or the per-task
  rate formula.
- **10-0 fixed the bootstrap CI to handle n=1 gracefully** (returns `[0,0]` when
  either condition has fewer than 2 results). The "low confidence" label (AC #7)
  is the user-facing surface of that defensive behaviour.
- **10-0's `CompareThree` three-way intersection** ensures that `cr.TextResults`,
  `cr.RuntimeResults`, and `cr.RankedResults` share the same TaskID denominator.
  Per-node metrics computed from these slices are directly comparable; no need
  to re-intersect.
- **5-2's bootstrap CI** is the existing infrastructure for "is this delta
  statistically meaningful?". Reused for `BaselineDelta` significance highlighting.

### Project context reference

- **Sprint status:** `_bmad-output/implementation-artifacts/sprint-status.yaml` —
  `10-8-per-node-experiment-metrics: ready-for-dev` after this file lands.
- **Epic 10 plan:** `_bmad-output/planning-artifacts/epics.md` lines 449-473 (Story
  7-8 in the original numbering).
- **Recent commits:**
  - `a3296d9 feat(10-0): epic 9 debt cleanup` — landed the Compare three-way
    intersection that makes per-node metrics computable across all three conditions
  - `46d041a feat(epic9): close epic 9 (DX) + retro + deferred-work triage and quick wins`
- **Baseline source:** the 2026-04-27 keyword-guardrail experiment results captured
  in the bench output JSON files (the user has these locally; the path is
  user-supplied via the new `--baseline` flag).

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

_empty_

### Completion Notes List

- Deviated from spec: `BaselineDelta *float64` placed on `NodeMetrics` directly instead of inside `ViolationTypeBreakdown`. Rationale: it's a node-level compliance rate delta; storing it per-violation-type creates an unresolvable edge case when `ByViolationType` is empty (nodes with zero violations). Spec comment explicitly said "for ContractComplianceRate" which is a `NodeMetrics` field.
- `ComputeNodeMetrics` wired into both `Compare` and `CompareThree`. `CompareThree` overrides the node metrics with the three-condition slice after `RankedResults` is assigned.
- Stderr baseline warning emitted from `FormatReport` (not the CLI layer) so it is testable in unit tests without exec subprocess.
- T6.12 (CLI flag-presence test) covered via `go build ./cmd/forgent-bench` + flag registration; no separate test file added (the build+flag wiring is implicit in the passing `go test ./...`).

### File List

- `internal/bench/experiment/design.go` — `NodeMetrics`, `ViolationTypeBreakdown` structs; `NodeMetrics []NodeMetrics` field on `ComparisonResult`
- `internal/bench/experiment/compare.go` — `ComputeNodeMetrics` function; wiring into `Compare` and `CompareThree`
- `internal/bench/experiment/baseline.go` (new) — `BaselineSnapshot`, `LoadBaseline`, `SaveBaseline`, `ApplyBaseline`
- `internal/bench/experiment/baseline_test.go` (new) — T6.5–T6.7 + ApplyBaseline tests
- `internal/bench/experiment/node_metrics_test.go` (new) — T6.1–T6.4 + 3 bonus tests
- `internal/bench/experiment/report.go` — `ReportOpts.BaselinePath`; `## Per-Node Contract Compliance` + `### Theory A/B Breakdown` sections
- `internal/bench/experiment/report_pernode_test.go` (new) — T6.8–T6.11 + 2 bonus tests
- `internal/bench/cmd/bench.go` — `--baseline`, `--save-baseline` flags; baseline load/apply/save wiring

### Review Findings

- [x] [Review][Decision] D1: AC4 bootstrapCI — resolved: added `PerRunRates []float64` to `NodeMetrics`; `bootstrapCIMean` called at render time for real 95% CI.
- [x] [Review][Decision] D2: AC2 `PreventedPct` denominator — resolved: both denominators kept; `PreventedPct` = prevented/attempted, `BlockedRunsPct` = prevented/total_runs; rendered as separate columns in Theory A/B table.
- [x] [Review][Patch] P1: deltaStr last-writer-wins for multi-condition nodes — fixed via `bestDelta` priority map (runtime > ranked > text).
- [x] [Review][Patch] P2: AC4 Markdown formatting — fixed: significant deltas render as `**+X% (CI sig)**`.
- [x] [Review][Patch] P3: AC5 spurious warning — fixed: warning only emitted when `opts.BaselinePath != ""`.
- [x] [Review][Patch] P4: AC6 Runs==0 compliance cell — fixed: renders `n/a` when `nm.Runs == 0`.
- [x] [Review][Patch] P5: No explicit `a.runs > 0` guard before division — added.
- [x] [Review][Patch] P6: T6.12 missing — added `TestCLI_BenchAcceptsBaselineFlag` in `internal/bench/cmd/bench_flags_test.go`.
- [x] [Review][Defer] W1: Theory A/B table emits n/a rows for every node per sparse violation type [report.go:538-549] — cosmetic UX issue; deferred post-MVP
- [x] [Review][Defer] W2: nil slice iterate in `collect(ranked, "ranked")` — safe no-op but no explicit comment [compare.go:81]; deferred, no immediate risk
- [x] [Review][Defer] W3: `formatPerNodeSection` writes to `os.Stderr` inside a string-builder formatter — testability concern; deferred as design smell
- [x] [Review][Defer] W4: `SaveBaseline` truncates timestamp to date-only, no sub-day granularity [baseline.go:55]; deferred cosmetic

## Change Log

| Date | Change | Notes |
|---|---|---|
| 2026-04-28 | Story 10-8 created from Epic 10 plan (Story 7-8 in epics.md) | Aggregates 10-6's `NodeViolations` cross-run, computes Theory A/B breakdown per violation type, renders a per-node table with optional baseline delta. Pure aggregator + renderer — no new enforcement. CLI gains `--baseline` and `--save-baseline` flags. |
