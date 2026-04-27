# Story 5.2: Runtime-vs-Text Controlled Experiment

Status: done

## Story

As a **product architect**,
I want to prove that Forgent runtime agents outperform text-instruction agents on complex tasks,
so that the core thesis has evidence.

## Acceptance Criteria

1. A set of complex multi-step tasks (code review, documentation, analysis) executed with: (a) text-instruction agent and (b) Forgent runtime agent — same LLM model, same tools, ONLY variable is orchestration method
2. Metrics: task completion rate, step ordering accuracy, guardrail adherence, token usage. Delta reported with confidence interval.
3. Methodology is documented — a third party can reproduce the experiment and verify results
4. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [x] Task 1: Experiment design (AC: #1, #3)
  - [x] 1.1: `design.go` — ExperimentConfig, TaskDefinition, StepDef, TaskResult, StepResult, ComparisonResult, LoadTasks, Validate
  - [x] 1.2: 5 tasks defined: code-review (3 steps), doc-generation (4 steps), security-audit (3 steps), refactoring (4 steps), test-generation (3 steps)
  - [x] 1.3: Each task: ordered steps with consumes/produces, expected keywords, guardrails (positive + negative `!` constraints)
  - [x] 1.4: `tasks/` directory with 5 YAML files
  - [x] 1.5: `METHODOLOGY.md` with design, metrics, statistical analysis, reproducibility instructions

- [x] Task 2: Text-instruction baseline (AC: #1)
  - [x] 2.1: `text_runner.go` — `RunTextExperiment`, `buildTextPrompt`, `evaluateTextResponse`
  - [x] 2.2: Single monolithic prompt with all steps, constraints, output format
  - [x] 2.3: Collects: completion, step ordering score, guardrail score, token estimate
  - [x] 2.4: TestRunTextExperiment with mock provider

- [x] Task 3: Runtime agent execution (AC: #1)
  - [x] 3.1: `runtime_runner.go` — `RunRuntimeExperiment`, `buildTaskDAG`, `evaluateRuntimeOutputs`
  - [x] 3.2: DAG auto-wired from consumes/produces; outputCapture callback captures all node outputs (not just terminal)
  - [x] 3.3: Same metrics as text runner for comparison
  - [x] 3.4: TestRunRuntimeExperiment with mock provider

- [x] Task 4: Comparison and reporting (AC: #2, #3)
  - [x] 4.1: `compare.go` — `Compare(text, runtime) *ComparisonResult`
  - [x] 4.2: Deltas: completion, ordering, guardrail, token (runtime - text; positive = runtime better)
  - [x] 4.3: Bootstrap CI (1000 resamples, seed=42, 95% level)
  - [x] 4.4: `report.go` — Markdown with Experiment Parameters, Methodology, Aggregate Results, Per-Task Results
  - [x] 4.5: All required sections present
  - [x] 4.6: TestCompare_WithResults, TestCompare_Empty, TestCompare_Reproducible, TestFormatReport_AllSections

- [x] Task 5: CLI integration (AC: #1)
  - [x] 5.1: `experiment` level added to `forgent-bench --level experiment`
  - [x] 5.2: Flags: `--tasks-dir`, `--provider`, `--model`, `--runs`, `--output`
  - [x] 5.3: TestEndToEnd_TextVsRuntime serves as integration smoke test

## Dev Notes

### Architecture References

- **FR52:** "Developer can run a controlled runtime-vs-text experiment with reproducible results."
- **PRD insight:** "Text instructions give +4% at best — structural enforcement gives +20-33%." This experiment produces the evidence for that claim.
- **Existing bench infrastructure:** `internal/bench/` provides pipeline, eval, report, h2h (head-to-head) comparison.

### Existing Head-to-Head Framework

`internal/bench/h2h.go` and `internal/bench/h2h_multipass.go` already implement head-to-head comparison patterns. Reuse the comparison methodology and reporting format.

### Metric Definitions

| Metric | Definition | How Measured |
|---|---|---|
| Task completion rate | % of tasks where all expected steps completed | Check each expected output is present |
| Step ordering accuracy | % of steps executed in correct order | Compare step sequence against expected |
| Guardrail adherence | % of tasks where constraints were respected | Check output against defined constraints |
| Token usage | Total tokens consumed | Provider token counting |

### Dependencies

- **Epic 4 complete:** Doctor as runtime agent proves the DAG execution path works. This story uses the same DAG infrastructure for the experiment tasks.
- **Existing bench framework:** h2h, pipeline, eval, report.

### What NOT to Build

- Do NOT implement a full statistical test suite (t-test, ANOVA) — bootstrap confidence intervals are sufficient
- Do NOT implement automated task generation — tasks are manually designed
- Do NOT implement a web dashboard — Markdown report only
- Do NOT run the experiment in CI — it requires an LLM and takes minutes. CI only tests the framework with mocks.

### Project Structure

New files:
- `internal/bench/experiment/design.go` — ExperimentConfig
- `internal/bench/experiment/text_runner.go` — text-instruction baseline
- `internal/bench/experiment/runtime_runner.go` — Forgent DAG execution
- `internal/bench/experiment/compare.go` — comparison logic
- `internal/bench/experiment/report.go` — Markdown report
- `internal/bench/experiment/tasks/` — task YAML definitions
- `internal/bench/experiment/METHODOLOGY.md` — reproducibility documentation
- `internal/bench/experiment/*_test.go` — tests
- `internal/bench/cmd/experiment.go` — CLI subcommand

## Testing Strategy

- **Design tests:** ExperimentConfig validation, task definition parsing.
- **Runner tests:** Mock provider, verify metrics collected correctly for both text and runtime runners.
- **Comparison tests:** Known result sets → expected deltas and confidence intervals.
- **Report tests:** Verify Markdown output format.
- **Reproducibility test:** Same seed + same mock → identical results.
- **CI gate:** `go test ./... -count=1 -race` (mock provider only)

## References

- [Source: _bmad-output/planning-artifacts/prd.md#FR52] — Runtime-vs-text experiment
- [Source: _bmad-output/planning-artifacts/prd.md] — "Text instructions give +4% at best"
- [Source: internal/bench/h2h.go] — Existing head-to-head comparison
- [Source: internal/bench/h2h_multipass.go] — Multi-pass comparison
- [Source: internal/bench/report.go] — Report generation patterns

## Dev Agent Record

### Agent Model Used

claude-opus-4-6

### Completion Notes List

- Experiment package with 5 files: design, text_runner, runtime_runner, compare, report
- 5 multi-step task definitions (code-review, security-audit, doc-generation, refactoring, test-generation)
- Bootstrap CI with fixed seed for reproducibility
- DAG runtime captures all node outputs via outputCapture callback (fixes terminal-only outputs issue)
- 14 tests (5 validation, 2 runner, 3 comparison, 1 report, 1 task loading, 1 e2e, 1 reproducibility)
- CLI integration: `forgent-bench --level experiment --tasks-dir <dir>`

### File List

- `internal/bench/experiment/design.go` (new)
- `internal/bench/experiment/text_runner.go` (new)
- `internal/bench/experiment/runtime_runner.go` (new)
- `internal/bench/experiment/compare.go` (new)
- `internal/bench/experiment/report.go` (new)
- `internal/bench/experiment/experiment_test.go` (new — 14 tests)
- `internal/bench/experiment/METHODOLOGY.md` (new)
- `internal/bench/experiment/tasks/code-review.yaml` (new)
- `internal/bench/experiment/tasks/security-audit.yaml` (new)
- `internal/bench/experiment/tasks/doc-generation.yaml` (new)
- `internal/bench/experiment/tasks/refactoring.yaml` (new)
- `internal/bench/experiment/tasks/test-generation.yaml` (new)
- `internal/bench/cmd/bench.go` (modified — experiment level + flags)

### Review Findings (AI) — 2026-04-15

- [x] [Review][Defer] `FormatReport` panic si `ComparisonResult` est nil — ajouter nil guard [report.go]
- [x] [Review][Defer] `SkipMap` + `Compare` indexation désalignée si un runner est repris et l'autre non [compare.go]
- [x] [Review][Defer] Bootstrap CI avec n=1 produit intervalle dégénéré — documenter ou warning [compare.go]
- [x] [Review][Defer] `consumes` sans producteur silencieusement ignoré dans le DAG — limitation pré-existante
- [x] [Review][Defer] `evaluateTextResponse` divise tokens par `len(Steps)` — garanti par Validate en amont

## Change Log

- 2026-04-10: Implemented story 5-2. Runtime-vs-text experiment framework with text runner, DAG runtime runner, bootstrap comparison, Markdown report, 5 task definitions, CLI integration. 14 tests pass.
