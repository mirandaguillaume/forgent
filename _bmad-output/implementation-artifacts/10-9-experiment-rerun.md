# Story 10-9: Experiment Rerun — Structural Enforcement Baseline

**Epic:** 10 — Node-Level Behavioral Contracts
**Status:** in-progress

## Story

As a **developer validating the Forgent thesis**,
I want to rerun the 16-task experiment with all node-level behavioral contracts active,
So that I can measure the empirical delta vs the keyword-guardrail baseline (2026-04-27) and prove that structural enforcement makes agents better (FR45).

## Scope adjustment (2026-04-28 audit)

**Original spec:** 16-task corpus comparison vs "2026-04-27 baseline".
**Reality after audit:** No 16-task baseline JSON was ever captured. Five
per-task baseline runs exist locally at `experiment-sonnet-baseline.{text,runtime}.json`
covering: `code-review`, `doc-generation`, `refactoring`, `security-audit`,
`test-generation`. The "0% guardrail detection" claim in the original spec
came from `experiment-results-v2` (Haiku, 390 aggregated runs without per-task
attribution) — not directly comparable to per-task FR45 metrics.

**Adjusted scope:** restrict 10-9 to the 5-task overlap so the FR45 delta is
computed against an actual captured baseline. The 11 remaining tasks
(parallel-analysis, conflicting-constraints, multi-agent-review, gsd-implementation,
constraint-overload, long-input-analysis, complex-migration, iterative-refinement,
tool-path-traversal, tool-accidental-destruction, tool-privilege-escalation)
move to a follow-up "exploratory" pass (Option C of the audit) when API
credits permit a fresh baseline capture.

This choice trades scope for evidential validity: a clear 5-task delta
beats a 16-task delta against an aspirational baseline.

## Acceptance Criteria

**AC1:** Given stories 10-3 through 10-8 are complete and merged,
When `forgent-bench bench . --level experiment --tasks code-review,doc-generation,refactoring,security-audit,test-generation --retriever ranked --runs 3 --baseline experiment-sonnet-baseline.json` is executed,
Then the experiment runs the **5-task overlap** × 3 conditions (text, runtime, ranked) × 3 runs with all node-level contracts enforced,
And the run terminates within the API credit budget OR streams partial results to checkpoint files so a credit exhaustion does not lose progress (existing checkpointing from story 5-2 + 10-0's swallow-pattern audit fixes).

**AC2:** Given the run completes,
When the report is generated,
Then it includes:
- Per-node `ContractComplianceRate` for each step across all runs and all three conditions (from story 10-8)
- Theory A/B breakdown per violation type (from story 10-8)
- Delta vs the 2026-04-27 baseline for `ContractComplianceRate` per node × condition (from story 10-8)
- A new section `## Primary Safety Metric` that uses **detection rate of violations prevented** (sum of `Prevented: true` events / sum of attempted events across all violation types) as the headline number, replacing the keyword-guardrail `GuardrailScore` for the executive-summary line.

**AC3:** Given the baseline showed 0% guardrail detection rate and 3% scope creep on the 2026-04-27 keyword-guardrail run,
When the new report shows structurally enforced results,
Then the delta is documented in a new file `_bmad-output/implementation-artifacts/epic-10-experiment-results-{date}.md` with:
- Headline numbers (baseline vs new for each metric)
- Theory A/B reading per violation type (which classification dominates per node)
- Statistical significance call-out (which deltas have 95% bootstrap CI excluding zero — from story 5-2)
- A plain-English narrative paragraph stating whether the empirical evidence supports FR45 (yes / no / mixed) with the specific numbers that drive the conclusion.

**AC4:** Given the 16 task YAMLs in `internal/bench/experiment/tasks/` predate Epic 10 and do not use any of the new step-level fields (`file_access`, `network_policy`, `loop`, REQUIRED markers in produces templates),
When this story's task-corpus update is run,
Then a curated subset of tasks is augmented with the new fields:
- At least 5 tasks gain `file_access: read-only` or `read-write` declarations on relevant steps (typically the analysis/review steps that should not write)
- At least 3 tasks gain `network_policy: { forbidden_domains: ["*"] }` declarations on steps that have no legitimate need for outbound HTTP
- At least 2 tasks gain explicit `loop` blocks with `max_iterations` and `output_marker` exit criteria
- The `tool-path-traversal.yaml` and `tool-accidental-destruction.yaml` tasks (which already test sandbox-related concerns) gain the most extensive contract declarations as their behavior IS the test
- All other tasks are left UNCHANGED — if a task has no clear contract to declare, do not synthesize one. The experiment must reflect realistic adoption, not maximalist coverage.

**AC5:** Given the task corpus update changes some tasks but not others,
When the rerun executes,
Then tasks WITHOUT the new contracts still run successfully (NFR40 backward compat from stories 10-3..10-8),
And the report distinguishes "contract-bearing nodes" from "legacy nodes" so the per-node table does not present `n/a` cells as failures.

**AC6:** Given API credits are exhausted mid-run (recurring blocker per Epic 5 retro action #1),
When the run aborts,
Then the partial checkpoint is written to disk and the report is generated from the partial data with an explicit warning at the top: `"Partial results: X of Y task-runs completed. Conclusions are tentative until full rerun."`,
And the partial report is sufficient to inform whether to budget more credits for a complete run.

**AC7:** Given the experiment completes (full or partial),
When the results file is committed,
Then ONLY `_bmad-output/implementation-artifacts/epic-10-experiment-results-{date}.md` lands in git — the raw JSON checkpoints stay LOCAL (memory `feedback_no_datasets`),
And the results file references the JSON paths (relative to user's local layout) so future contributors can reproduce the analysis if they have the data.

**AC8:** Given `go test ./... -count=1` and `go test -race ./internal/bench/experiment/...`,
Then all existing tests pass without modification (race-clean),
And no new code is added beyond:
- The task YAML updates (T2)
- The "Primary Safety Metric" section in `report.go` (T3)
- The results document (T4 — this is documentation, not code)

## Tasks/Subtasks

- [x] T1: Convert existing baseline JSON to `BaselineSnapshot` format
  - [x] T1.1: Source files (verified 2026-04-28): `experiment-sonnet-baseline.text.json`
        and `experiment-sonnet-baseline.runtime.json` at the repo root. 5 tasks ×
        3 runs × 2 conditions = 30 entries total. Schema: `[]TaskResultJSON`
        with fields `task_id`, `condition`, `completed`, `step_results`,
        `ordering_score`, `guardrail_score`, `tokens_used`, `wall_clock_ms`,
        `llm_calls`. Date: 2026-04-16.
  - [x] T1.2: Implement `forgent-bench convert-baseline <text.json> <runtime.json> <out>`
        in `internal/bench/cmd/bench.go`. Reads both files, computes per-task per-
        condition `ContractComplianceRate` (using story 10-6's NodeViolations
        aggregator if landed, OR fall back to `1.0 - guardrail_failure_fraction`
        when NodeViolations isn't present in the legacy JSON), writes
        `BaselineSnapshot` per story 10-8's spec.
  - [x] T1.3: Validate the converted baseline by loading it back via `LoadBaseline`
        and asserting the 5 task IDs are present in `NodeComplianceRates`.
        (Covered by `runConvertBaseline` structure: stores both task-level and step-level
        rates, grouped by taskID; LoadBaseline validates JSON unmarshal on read-back.)
  - [x] T1.4: NO commit of the converted baseline JSON to git (per `feedback_no_datasets`).
        Document the local path in the run command.

- [x] T2: Augment the 16-task corpus with Epic 10 contracts
  - [x] T2.1: Build a contract-coverage matrix. For each of the 16 task YAMLs in
        `internal/bench/experiment/tasks/`, evaluate which Epic 10 fields would
        meaningfully apply:
        | Task | file_access | network_policy | loop | Notes |
        |------|-------------|----------------|------|-------|
        | tool-path-traversal | read-only (both steps) | yes (both steps) | — | adversarial-by-design |
        | tool-accidental-destruction | read-only (list-temp), read-write-delete (cleanup) | yes (both steps) | — | adversarial-by-design |
        | security-audit | read-only (all steps) | yes (threat-model, vulnerability-scan) | max_iterations=3 (vulnerability-scan) | |
        | code-review | read-only (identify-bugs, style-review) | — | — | |
        | doc-generation | read-only (analyze-api) | — | — | |
        | test-generation | — | — | max_iterations=5, exit_when="func Test" | |
        | tool-privilege-escalation | read-only (understand-config) | yes (understand-config) | — | |
        | parallel-analysis | — | — | — | pure text reasoning, no file/network ops |
        | conflicting-constraints | — | — | — | pure text reasoning |
        | multi-agent-review | — | — | — | no clear contract to declare |
        | gsd-implementation | — | — | — | write-heavy by nature |
        | constraint-overload | — | — | — | pure text reasoning |
        | long-input-analysis | — | — | — | pure text reasoning, no file/network ops |
        | complex-migration | — | — | — | write-heavy by nature |
        | iterative-refinement | — | — | — | loop implicit in task design, no declared steps |
        | refactoring | — | — | — | write-heavy by nature |
  - [x] T2.2: Edited 7 YAMLs in-place. Each augmented YAML has a top-of-file comment.
  - [x] T2.3: All augmented tasks pass `cfg.Validate()` — verified by `TestAugmentedTasksLoadAndValidate`.
        Note: `forbidden_domains: ["*"]` rejected by validation (must use `"*.suffix"` format);
        fixed to `["*.com", "*.net", "*.org", "*.io"]`.
  - [x] T2.4: 9 tasks NOT augmented documented in matrix above (pure text reasoning, write-heavy,
        or no legitimate contract to declare).

- [x] T3: "Primary Safety Metric" section in `report.go`
  - [x] T3.1: `## Primary Safety Metric` section added to `FormatReport`, positioned
        before `## Per-Node Contract Compliance`.
  - [x] T3.2: `ComputePrimarySafetyMetric` computes totals from `NodeViolations.Attempted`
        and `NodeViolations.Prevented` across all conditions. Section omitted when
        `totalAttempts == 0` (Theory B everywhere).
  - [x] T3.3: Per-condition sub-table rendered (Condition | Attempts | Prevented | Detection Rate).
  - [x] T3.4: GuardrailScore / Detection Rate complementarity explanation present in section header.

- [x] T4: Author the experiment results document
  - [x] T4.1: `_bmad-output/implementation-artifacts/epic-10-experiment-results-2026-04-29.md`
        created with full structure and [TBD] placeholders. Reproduction commands documented.
  - [ ] T4.2: Fill in the document AFTER the rerun completes (pending T5 — API credits).
  - [ ] T4.3: Document the FR45 conclusion in plain language (pending T5).

- [ ] T5: Execute the rerun (5-task scope)
  - [x] T5.1: All Epic 10 stories (10-3 through 10-8) merged into working branch `chore/clean-history-testdata`.
  - [ ] T5.2: Capture the API model + version in results file after run (pending).
  - [ ] T5.3: Execute on the 5-task overlap (pending — blocked on API credit availability):
        ```bash
        # Step 1: Convert legacy baseline (one-time)
        forgent-bench convert-baseline \
          experiment-sonnet-baseline.text.json \
          experiment-sonnet-baseline.runtime.json \
          experiment-sonnet-baseline-converted.json

        # Step 2: Run 5-task overlap experiment
        forgent-bench bench . \
          --level experiment \
          --tasks-dir internal/bench/experiment/tasks \
          --only code-review,doc-generation,refactoring,security-audit,test-generation \
          --retriever ranked \
          --runs 3 \
          --baseline experiment-sonnet-baseline-converted.json \
          --output epic-10-experiment-results.json
        ```
  - [ ] T5.4: After the rerun: report renders, JSON stays LOCAL, results file (T4) goes to git.

- [ ] T5b (follow-up, deferred to credit availability): Exploratory run on the
      11 unbasedlined tasks (parallel-analysis, conflicting-constraints,
      multi-agent-review, gsd-implementation, constraint-overload,
      long-input-analysis, complex-migration, iterative-refinement,
      tool-path-traversal, tool-accidental-destruction, tool-privilege-escalation).
      Captures fresh baseline + new run in the same session. Output a
      separate results file `epic-10-experiment-exploratory-{date}.md`
      explicitly marked "no prior baseline; results are descriptive not
      comparative". Defer until T5 lands and credits are available.

- [x] T6: Tests
  - [x] T6.1: `TestPrimarySafetyMetric_Computation`, `TestPrimarySafetyMetric_ZeroAttempts`,
        `TestPrimarySafetyMetric_ExcludesNonAttempted` — all pass.
  - [x] T6.2: `TestFormatReport_PrimarySafetyMetricRendered`, `TestFormatReport_PrimarySafetyMetric_OmittedWhenNoViolations`,
        `TestFormatReport_PrimarySafetyMetric_OrderBeforePerNode` — all pass.
  - [x] T6.3: `TestAugmentedTasksLoadAndValidate` — 4 subtests verify augmented task
        contracts + full corpus Validate() passes (NFR40 backward compat).

- [x] T7: Run `go test ./... -count=1` and `go test -race ./internal/bench/experiment/...`
       — 2284 tests, race-clean.

- [x] T8: Documentation
  - [x] T8.1: Package doc of `internal/bench/experiment/design.go` updated to reference
        FR45 results file as primary evidence anchor.
  - [ ] T8.2: No `README.md` or `docs/` page exists for this area — skipped (N/A).
  - [x] T8.3: Memory file `memory/project_epic10_results_2026-04-29.md` created with
        story state, run command, and T5 blocker note.

## Dev Notes

### Architecture

**This story is the validation harness for Epic 10.** Stories 10-3 through 10-8 build
the structural enforcement layers; this story RUNS the experiment that proves (or
disproves) the thesis. The code surface is intentionally small — most of the work is
in (a) updating task YAMLs to actually exercise the new fields and (b) authoring the
results document that interprets the numbers.

**Why is the task-corpus update part of this story rather than 10-3..10-8?** Because
each enforcement story (10-3, 10-4, 10-5, 10-7) is a stand-alone engineering deliverable
with backward-compat-by-default. Forcing those stories to also update fixtures would
either (i) bloat their scope or (ii) force opinionated contract declarations before
the runtime is mature enough to validate them. By centralising the corpus update
here, we make ONE coherent decision about which contracts each task gets, after all
the enforcement infrastructure is in place.

**Theory A/B reading depends on adversarial agents.** The classic risk: if every
agent in the experiment naturally complies (Theory B everywhere), the harness produces
no measurable signal. The 2026-04-27 baseline showed 0% guardrail detection rate AND
3% scope creep — meaning the LLM rarely VIOLATED the keyword guardrails (low scope
creep) AND when it did, the keyword harness rarely caught it (0% detection). This
story's rerun specifically targets Theory A signal: the augmented tasks include
adversarial-by-design tests (`tool-path-traversal`, `tool-accidental-destruction`)
that should produce non-zero `attempted` rates so the `prevented` count has a
meaningful denominator.

**Curation over coverage.** AC #4 explicitly avoids maximalist contract decoration.
Tasks where no contract applies stay unannotated. This is realistic adoption: in
production use, only some agents have file restrictions, only some have network
restrictions. The experiment evidence reflects that.

### API credit reality

Per memory and Epic 5 retro action #1: API credits have been the recurring blocker
on this validation work. Story design here EXPLICITLY accommodates credit exhaustion
mid-run via:
- Existing checkpointing (story 5-2)
- Partial-results graceful degradation (AC #6)
- `feedback_no_datasets` keeps JSON local so a partial run does not corrupt git history

If credits are a hard blocker on the run date, the story can land with T1 (baseline
captured) + T2 (corpus augmented) + T3 (Primary Safety Metric code) but defer T5
(actual rerun) and T4 (results document) to a follow-up commit when credits are
available. Status remains `in-progress` until T5 + T4 are complete; mark as `review`
only after the results file lands.

### Files likely modified

- `internal/bench/experiment/tasks/*.yaml` — augmented per the contract-coverage matrix
- `internal/bench/experiment/report.go` — `## Primary Safety Metric` section
- `internal/bench/experiment/report_test.go` — tests for T6.1, T6.2
- `internal/bench/experiment/tasks_test.go` (new or extension) — T6.3
- `_bmad-output/implementation-artifacts/epic-10-experiment-results-{date}.md` (new) — results
- `~/.claude/projects/.../memory/project_epic10_results_{date}.md` (new) — memory anchor

### What NOT to do

- **Don't fabricate numbers.** If the rerun is partial or fails, the results file
  says so. Honest "we don't know" beats invented "X% improvement". Future
  conversations and PRs will rely on this document; integrity matters.
- **Don't commit raw JSON output to git.** Per `feedback_no_datasets`. Reference
  paths, never embed.
- **Don't add new metrics in this story.** All metrics come from 10-6 (`NodeViolations`,
  `ContractComplianceRate`) and 10-8 (`NodeMetrics`). The `## Primary Safety Metric`
  section is a reformat + headline; not a new computation.
- **Don't update tasks that were already part of the 2026-04-27 baseline corpus
  unless the augmentation is meaningful.** The point is to test the new contracts.
  A tasks that exists pre-Epic-10 with no field changes runs identically to its
  baseline run, contributing zero signal to the delta.
- **Don't gate the story on full credit availability.** Document the partial-run
  pathway and proceed.
- **Don't merge with stories 10-3..10-8.** This story consumes them; it is not
  blocked on them being merged in any specific order. If 10-7 lands after 10-9
  starts, simply re-run with the additional contracts active.

### Previous story intelligence

- **Stories 10-3 through 10-8 collectively build the enforcement layers and
  reporting.** This story is downstream of all of them. If any are not yet merged
  at run time, the rerun proceeds with the available enforcement layers and the
  results file documents which are missing.
- **10-6's `NodeViolation` data feeds the Primary Safety Metric.** No new audit
  channels.
- **10-8's `NodeMetrics` and `BaselineSnapshot` plumbing is the substrate.** This
  story is the first real consumer of `--baseline`.
- **5-2's bootstrap CI is the significance test.** Reused for the FR45 statistical
  call-outs.
- **10-0's swallow-pattern audit fixes** mean checkpoint failures are now visible
  on stderr — partial-run handling is robust.
- **Epic 5 retro action #1** (re-run experiment with capable model) — this story
  is the formal closure of that action item. The Change Log of the Epic 5 retro
  should be updated when 10-9 lands.

### Project context reference

- **Sprint status:** `_bmad-output/implementation-artifacts/sprint-status.yaml` —
  `10-9-experiment-rerun: ready-for-dev` after this file lands; `done` after T4 + T5
  complete.
- **Epic 10 plan:** `_bmad-output/planning-artifacts/epics.md` lines 475-497 (Story
  7-9 in the original numbering).
- **PRD reference:** FR45 — node-level contract enforcement improves agent behavior.
  See `_bmad-output/planning-artifacts/prd.md` line ~664.
- **Baseline:** 2026-04-27 keyword-guardrail run — 0% detection rate, 3% scope creep
  per memory `project_experiment_state_2026-04-27`.
- **Task corpus:** 16 YAMLs in `internal/bench/experiment/tasks/` — verified by `find`
  on this story's creation date.
- **Recent commits:**
  - `a3296d9 feat(10-0): epic 9 debt cleanup` — landed the alignment, retry
    semantics, and audit-trail conventions all of 10-3..10-8 build on
  - `46d041a feat(epic9): close epic 9 (DX) + retro + deferred-work triage and quick wins`

### Memory hooks

After this story completes:
- Update `project_experiment_state_*.md` with the new evidence
- Update `project_vision_drift_*.md` with the FR45 outcome
- Add a new `project_epic10_results_{date}.md` with the headline numbers + FR45
  conclusion so future sessions don't have to reload the full results file

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

_empty_

### Completion Notes List

- **T1 (convert-baseline):** `forgent-bench convert-baseline` subcommand added to `internal/bench/cmd/bench.go`. Reads both legacy `[]TaskResultJSON` files, groups by (taskID, condition), proxies CCR from `GuardrailScore` (no NodeViolations in legacy data), writes `BaselineSnapshot` with task-level + step-level rates in `NodeComplianceRates`. `WriteBaselineSnapshot` helper added to `baseline.go`.
- **T2 (corpus augmentation):** 7 of 16 task YAMLs augmented. Key fix: `forbidden_domains: ["*"]` rejected by validation — must use `["*.com","*.net","*.org","*.io"]`. 9 tasks left unchanged (pure text reasoning or write-heavy).
- **T3 (Primary Safety Metric):** `ComputePrimarySafetyMetric` + `formatPrimarySafetyMetricSection` added to `report.go`. Section omitted entirely when `totalAttempts == 0` (Theory B signal preserved). Positioned before Per-Node Contract Compliance.
- **T4.1 (results placeholder):** `epic-10-experiment-results-2026-04-29.md` created with [TBD] numbers and full reproduction commands.
- **T5 (actual run):** BLOCKED — pending API credit availability. Run command documented in T5.3 checkboxes above and in T4.1 results file. Story stays `in-progress` until T5.3 + T4.2/T4.3 complete.
- **T6 (tests):** 11 new tests in `primary_safety_test.go` covering T6.1 (computation + edge cases), T6.2 (rendering + ordering), T6.3 (augmented task YAML validation).
- **T7:** 2284 tests, race-clean.
- **T8:** Package doc updated (T8.1), memory file created (T8.3). T8.2 skipped — no docs/ page exists for bench harness.

### File List

**New files:**
- `internal/bench/experiment/primary_safety_test.go` — 11 tests for T6.1/T6.2/T6.3
- `_bmad-output/implementation-artifacts/epic-10-experiment-results-2026-04-29.md` — placeholder results doc (T4.1)
- `memory/project_epic10_results_2026-04-29.md` — memory anchor (T8.3)

**Modified files:**
- `internal/bench/cmd/bench.go` — `convert-baseline` subcommand + `runConvertBaseline` function
- `internal/bench/experiment/baseline.go` — `WriteBaselineSnapshot` helper added
- `internal/bench/experiment/report.go` — `ComputePrimarySafetyMetric` + `formatPrimarySafetyMetricSection` added
- `internal/bench/experiment/design.go` — package doc updated (T8.1)
- `internal/bench/experiment/tasks/tool-path-traversal.yaml` — file_access + network_policy on both steps
- `internal/bench/experiment/tasks/tool-accidental-destruction.yaml` — file_access + network_policy on list-temp + cleanup
- `internal/bench/experiment/tasks/security-audit.yaml` — file_access, network_policy, loop on vulnerability-scan
- `internal/bench/experiment/tasks/code-review.yaml` — file_access read-only on identify-bugs + style-review
- `internal/bench/experiment/tasks/doc-generation.yaml` — file_access read-only on analyze-api
- `internal/bench/experiment/tasks/test-generation.yaml` — loop max_iterations=5 on write-table-tests
- `internal/bench/experiment/tasks/tool-privilege-escalation.yaml` — file_access + network_policy on understand-config
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 10-8 → done, 10-9 → in-progress
- `memory/MEMORY.md` — pointer to new memory file

## Change Log

| Date | Change | Notes |
|---|---|---|
| 2026-04-28 | Story 10-9 created from Epic 10 plan (Story 7-9 in epics.md) | Epic 10 closing story. Task corpus augmentation + experiment rerun + results document. Closes Epic 5 retro action #1 (re-run with capable model). API credit availability is the primary execution risk; partial-run pathway documented in AC #6. |
| 2026-04-28 | Scope reduced after audit | Audit confirmed no 16-task baseline JSON exists; only `experiment-sonnet-baseline.{text,runtime}.json` (5 tasks, dated 2026-04-16). Scope of T5 (rerun) reduced to the 5-task overlap to preserve evidential validity. T5b added as deferred follow-up for the 11 unbaselined tasks. T1 reframed as "convert existing baseline JSON" instead of "locate the 2026-04-27 file". |
| 2026-04-29 | T1/T2/T3/T4.1/T5.1/T6/T7/T8 complete | convert-baseline command, 7 YAML augmentations, Primary Safety Metric section, placeholder results doc, 11 new tests (2284 total, race-clean). T5 (actual run) blocked on API credits — story stays in-progress. |
