# Story 7.1: Static Check Foundation + Progressive Display

Status: done

## Story

As a **developer**,
I want to see instant structural feedback while the LLM analysis runs,
so that I get immediate value and the tool feels responsive.

## Acceptance Criteria

1. New `internal/doctor/static/` package with `Check` interface and `init()` self-registration
2. `Check` interface: `ID() string`, `Tags() []string`, `Category() string`, `DefaultSeverity() string`, `Check(*parser.AgentAnalysis) []llmutil.Finding`
3. Three core static checks implemented:
   - **Section presence**: 15 sections checked (security, guardrails, testing, examples, error_handling, build_commands, architecture, identity, output_format, decision_authority, workflow_triggers, dependency_declaration, memory_management, goals, constraints)
   - **Vague pattern detection**: 17 regex patterns ("be thorough", "if needed", "when appropriate", "be helpful", "be careful", "as needed", "when possible", "if appropriate", "make sure", "try to", "be sure to", "keep in mind", "it is important", "please note", "do your best", "be mindful", "use your judgment") — excludes content inside code fences
   - **Instruction ordering**: critical sections (identity, guardrails, security) in first 33% of file content
4. Static checks produce `llmutil.Finding` objects identical to LLM output format
5. Progressive display in TTY mode:
   - Structural results appear instantly (< 100ms)
   - Spinner shown during LLM analysis
   - LLM findings appear after analysis completes
6. `--quick` flag runs ONLY static checks (no LLM call), exits with structural score
7. Structural score displayed as `X/Y ✅ (Z%)` with list of missing sections
8. All existing tests pass + new tests for each static check

## Research Backing

- Liu et al. (arXiv:2307.03172): primacy/recency effect — critical content at beginning
- Chatlatanagulchai et al. (arXiv:2511.12884): 14.5% address security, 62.3% have build commands
- AgentLinter: 17 vague conditional patterns as competitive baseline
- Party mode decision: 70% of checks are static, LLM needed for 30%

## Tasks / Subtasks

- [x] Task 1: Check interface + registry (AC: #1, #2)
  - [x] 1.1: Create `internal/doctor/static/check.go` — `Check` interface, `checks []Check` slice, `RegisterCheck()`, `RunChecks()`
  - [x] 1.2: Tags support: "default", "thorough", "security", "git-aware"
  - [x] 1.3: `ResolveChecks(mode string) []Check` — filters by tag based on mode
  - [x] 1.4: Tests: registration, resolution by tag, empty registry

- [x] Task 2: Section presence check (AC: #3a)
  - [x] 2.1: Create `internal/doctor/static/presence.go` — checks 15 sections
  - [x] 2.2: Migrate logic from `context/enricher.go` gap detection (5 categories) + extend to 15
  - [x] 2.3: Each missing section = one Finding with category matching registry ID
  - [x] 2.4: Section detection: heading text matching (case-insensitive, partial match)
  - [x] 2.5: Tests against existing testdata (smoke test on real files)

- [x] Task 3: Vague pattern detection (AC: #3b)
  - [x] 3.1: Create `internal/doctor/static/vague.go` — 17 regex patterns
  - [x] 3.2: Skip lines inside code fences (use existing `inCodeFence` logic from parser)
  - [x] 3.3: Each match = one Finding with category "specificity", line number, matched text
  - [x] 3.4: Tests: matches in prose, no matches in code blocks, multiple matches per file

- [x] Task 4: Instruction ordering check (AC: #3c)
  - [x] 4.1: Create `internal/doctor/static/ordering.go`
  - [x] 4.2: Checks critical sections (identity, guardrails, security) in first 33% of sections
  - [x] 4.3: Tests: correct ordering, incorrect ordering, missing critical sections (no false positive)

- [x] Task 5: Progressive display (AC: #5, #7)
  - [x] 5.1: Modify `internal/doctor/render.go` — new `RenderStructural()` + `ComputeStructural()`
  - [x] 5.2: Structural score: count passed/total checks, format as `X/Y passed (Z%)`
  - [x] 5.3: List missing sections with x prefix
  - [x] 5.4: "Analyzing with LLM..." message during LLM phase
  - [x] 5.5: Call `RenderStructural()` before LLM call in `internal/cmd/doctor.go`

- [x] Task 6: --quick flag (AC: #6)
  - [x] 6.1: Add `--quick` flag to doctor command
  - [x] 6.2: When set: run static checks only, skip LLM, render structural score, exit
  - [x] 6.3: Exit code 0 if all structural checks pass, exit code 2 if findings
  - [x] 6.4: Smoke tested on real testdata files + directory mode

## Dev Agent Record

### Implementation Plan
- Task 1: Created `internal/doctor/static/` package with Check interface, tag-based registration (init() pattern from linter/), and ResolveChecks for mode filtering.
- Task 2: Section presence check — 15 sections mapped with keyword matching on headings + body content. Categories match registry IDs.
- Task 3: Vague pattern detection — 17 patterns with code-fence exclusion and one-finding-per-line limit.
- Task 4: Instruction ordering — checks 3 critical sections (identity, guardrails, security) are in the first third of sections. Skips files with <3 sections.
- Task 5: Progressive display — ComputeStructural counts presence-check pass/fail, RenderStructural shows X/Y + missing list. Called before LLM in runFileMode.
- Task 6: --quick flag — runQuickMode runs only static checks, renders structural + non-presence findings, exits with appropriate code.

### File List
- `internal/doctor/static/check.go` — NEW: Check interface, registry, RunChecks
- `internal/doctor/static/check_test.go` — NEW: 3 tests (register, run, empty)
- `internal/doctor/static/presence.go` — NEW: 15-section presence check
- `internal/doctor/static/presence_test.go` — NEW: 4 tests
- `internal/doctor/static/vague.go` — NEW: 17-pattern vague detection
- `internal/doctor/static/vague_test.go` — NEW: 6 tests
- `internal/doctor/static/ordering.go` — NEW: critical section ordering
- `internal/doctor/static/ordering_test.go` — NEW: 5 tests
- `internal/doctor/render.go` — MODIFIED: added StructuralResult, ComputeStructural, RenderStructural
- `internal/cmd/doctor.go` — MODIFIED: added --quick flag, runQuickMode, progressive display in runFileMode

### Change Log
- 2026-03-31: Story 7.1 implemented — static check package (3 checks, 18 tests), progressive display, --quick flag

## Dev Notes

- The `static/` package follows the same `init()` registration pattern as `internal/linter/`
- `llmutil.Finding` is the shared currency between static and LLM checks — no new types needed
- Gap detection in `context/enricher.go` is migrated, not duplicated — remove old code after migration
- `--quick` + `--ci` = instant structural check in CI without any LLM provider
- Estimated effort: ~10 hours total
