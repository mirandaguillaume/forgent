# Story 7.3: Pipeline Orchestration + Quality Gate + CI Mode

Status: done

## Story

As a **developer**,
I want Doctor to produce a pass/fail quality gate I can use in CI,
so that my team enforces agent file quality on every PR.

## Acceptance Criteria

1. New `internal/doctor/pipeline.go` orchestrates: resolve checks → run static → render structural → run LLM (cached) → merge → post-process → quality gate → exit code
2. Quality gate with AND-logic conditions (SonarQube pattern):
   - Default conditions: `secrets_found == 0` (blocking), `structural_score >= 0.5` (blocking), `high_findings <= 5` (warning)
   - Conditions configurable in `.forgent/config.yaml` under `doctor.gate.conditions`
3. `--ci` flag:
   - Auto-detects: if no LLM provider available, falls back to `--quick` (static only)
   - Plain text output (no colors, no spinner)
   - Quality gate evaluated, exit code reflects result
   - GitHub annotations via `::error file=X,line=Y::message` and `::warning` format
4. `--format json` outputs structured JSON report with: structural_score, indicators, findings, gate result
5. Exit codes: 0 = gate pass, 1 = internal error, 2 = gate fail (findings)
6. Post-processing pipeline: deduplicate (same category + same line), sort (HIGH → MOD → LOW), limit (max 20 findings per file)
7. `--thorough` flag activates all check tags (default + thorough + security + git-aware)
8. `--security` flag activates only security-tagged checks
9. All existing tests pass + new tests for pipeline, gate, CI mode, JSON output

## Research Backing

- SonarQube: AND-logic quality gate, no weighted scores
- golangci-lint: tag-based filtering, processor pipeline, exit codes
- Promptfoo: threshold-based pass/fail overrides individual failures

## Tasks / Subtasks

- [x] Task 1: Pipeline orchestration (AC: #1)
  - [x] 1.1-1.5: `pipeline.go` with RunPipeline(), Report struct, integrates static+LLM+postprocess+gate. 4 tests.

- [x] Task 2: Quality gate (AC: #2)
  - [x] 2.1-2.6: `gate.go` with GateCondition, QualityGate, Evaluate(). AND-logic, blocking/warning. DefaultGate(). 5 tests.

- [x] Task 3: Post-processing pipeline (AC: #6)
  - [x] 3.1-3.4: `postprocess.go` with dedup, sort (HIGH→MOD→LOW), limit. 4 tests.

- [x] Task 4: CI mode (AC: #3)
  - [x] 4.1-4.6: `--ci` flag, `ci.go` with RenderGitHubAnnotations + RenderGateAnnotation, auto-fallback to quick, color.NoColor

- [x] Task 5: JSON output (AC: #4)
  - [x] 5.1-5.4: `--format json` flag, `json_output.go` with ToJSON(), JSONReport schema

- [x] Task 6: Mode flags (AC: #7, #8)
  - [x] 6.1-6.4: `--thorough`, `--security` flags, mode resolution, quick wins if combined

- [x] Task 7: Refactor doctor command (AC: #1)
  - [x] 7.1-7.3: runPipelineMode() wraps pipeline, existing runFileMode/runQuickMode preserved, all tests pass

## Dev Agent Record

### File List
- `internal/doctor/pipeline.go` — NEW: RunPipeline orchestrator
- `internal/doctor/pipeline_test.go` — NEW: 4 pipeline tests
- `internal/doctor/gate.go` — NEW: QualityGate with AND-logic conditions
- `internal/doctor/gate_test.go` — NEW: 5 gate tests
- `internal/doctor/postprocess.go` — NEW: dedup, sort, limit
- `internal/doctor/postprocess_test.go` — NEW: 4 postprocess tests
- `internal/doctor/ci.go` — NEW: GitHub annotations renderer
- `internal/doctor/json_output.go` — NEW: JSON report output
- `internal/cmd/doctor.go` — MODIFIED: added --ci, --thorough, --security, --format flags, runPipelineMode

### Change Log
- 2026-03-31: Story 7.3 implemented — pipeline orchestration, quality gate, CI mode, JSON output, mode flags

## Dev Notes

- This story depends on Story 7.0 (cache) and Story 7.1 (static checks + progressive display)
- The quality gate replaces `internal/doctor/validate/gate.go` (Epic 3's 5-check gate) with a more general condition-based system
- GitHub annotations format is standard for GitHub Actions — no special integration needed
- The pipeline.go is the new central entry point; cmd/doctor.go becomes a thin CLI wrapper
- Estimated effort: ~10 hours total
