# Story 2.3: Directory Analysis and Text-Instruction Diagnosis

Status: ready-for-dev

## Story

As a **team lead**,
I want to run doctor on a directory and see aggregate results across all agent files,
so that I can identify team-wide patterns.

As a **developer**,
I want doctor to warn me about text instruction limits,
so that I can make informed decisions about my agent architecture.

## Acceptance Criteria

1. `forgent doctor .claude/agents/` analyzes each agent file in the directory and produces an aggregate summary: total files, total findings, findings by category, files with no guardrails, files with no security
2. Directory analysis discovers agent files recursively using the parser registry's `Detect()` — only files recognized by a registered parser are analyzed
3. Each file's individual findings are displayed, followed by the aggregate summary at the end
4. When doctor detects an agent relies entirely on text instructions for execution ordering, it includes a finding: "Text instructions are ignored X% of the time on ordering tasks. Structural enforcement available via Forgent runtime." (FR16 — MVP scope is ordering-only; broader text-instruction-limit diagnosis deferred to Growth)
5. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Directory analysis support (AC: #1, #2, #3)
  - [ ] 1.1: In `internal/cmd/doctor.go`, detect whether the argument is a file or directory via `os.Stat()`
  - [ ] 1.2: Create `internal/doctor/directory.go` — `DiscoverAgentFiles(root string) ([]string, error)` that walks the directory recursively and returns paths where any registered parser's `Detect()` returns true
  - [ ] 1.3: Skip hidden directories (`.git`, etc.) during walk using `scanner.SkipDirs`
  - [ ] 1.4: For directory input, analyze each discovered file sequentially (reuse existing single-file flow)
  - [ ] 1.5: Create `internal/doctor/aggregate.go` — `AggregateReport` struct and `Aggregate(results []FileResult) *AggregateReport` function
  - [ ] 1.6: `AggregateReport` contains: total files, total findings, findings by category (map[string]int), files with zero guardrails findings, files with zero security findings, per-file finding counts
  - [ ] 1.7: Create `RenderAggregate(report *AggregateReport, tty bool)` in render.go — display summary after individual results
  - [ ] 1.8: Write tests for: directory with mixed files (agent + non-agent), empty directory, directory with single file, nested directories

- [ ] Task 2: Text instruction limit diagnosis (AC: #4)
  - [ ] 2.1: In `internal/doctor/context/enricher.go` (from Story 2.2), add an ordering-diagnosis check
  - [ ] 2.2: Detect if agent file uses text-based ordering instructions (e.g., "first do X, then do Y", "step 1:", numbered workflow instructions) without structural enforcement
  - [ ] 2.3: If detected, produce a finding with category `"ordering"`, confidence `"high"`, citing the ordering registry entry: "Text-based ordering instructions are unreliable. Research shows LLMs ignore instruction ordering X% of the time. Consider Forgent runtime for structural enforcement."
  - [ ] 2.4: Write tests with agent files that have text ordering vs no ordering instructions

- [ ] Task 3: CLI integration for directory mode (AC: #1, #3)
  - [ ] 3.1: Modify `doctorCmd` Args validation: accept 1 arg that is either file or directory
  - [ ] 3.2: If directory: discover → analyze each → aggregate → render each + summary
  - [ ] 3.3: Exit code for directory mode: 0 (no findings in any file), 1 (error), 2 (findings in any file)
  - [ ] 3.4: Update `--help` long description to show directory usage example

## Dev Notes

### Architecture References

- **ADR-4 (Doctor DAG Topology):** Directory analysis runs the full single-file pipeline per file. No parallel file analysis in this story — sequential is safe and simple.
- **FR8:** Directory analysis with aggregate report.
- **FR16:** Diagnose text instruction limits (ordering-only for MVP).

### Existing Code to Reuse

- **`internal/doctor/parser.DetectFormat(path, content)`** — used to check if a file is an agent file during directory walk
- **`internal/doctor/parser.RegisteredFormats()`** — list all registered parser names
- **`internal/scanner.SkipDirs`** — map of directories to skip during walk (`.git`, `vendor`, `node_modules`, etc.)
- **`internal/doctor.RenderFindings()`** — renders individual file findings (reuse as-is)
- **`internal/doctor.IsTTY()`** — TTY detection for output formatting
- **`internal/doctor/registry.Registry`** (from Story 2.1) — Load(), Get(), version field
- **`internal/doctor/llmutil.Finding`** — finding struct with category, confidence, etc.

### Dependencies on Previous Stories

- **Story 2.1 (Research Registry):** Must be complete. Registry is used for citation resolution during per-file analysis.
- **Story 2.2 (Context Enrichment):** Must be complete. The text-instruction-limit diagnosis (Task 2) adds a check to the context enricher. Directory analysis also calls the context enricher per file.

### Previous Story Intelligence (from Epic 1 retro)

- **Action #1:** "Resolve frontmatter parsing" — should be done before this story. Directory analysis will hit many real files and frontmatter edge cases.
- **Action #2:** "Fix OpenRouter timeout" — relevant if cloud provider is used during directory analysis of many files.
- **Action #4:** "Always review before done" — 3-layer adversarial code review is non-negotiable.
- **Action #5:** "No undocumented scope creep" — the aggregate report fields are defined here (total files, total findings, by-category, zero-guardrails, zero-security). Do not add more.
- **Key risk from retro:** "Directory analysis aggregating results from multiple files — concurrency + reporting." This story uses SEQUENTIAL analysis to avoid concurrency issues. Parallel directory analysis is a Growth optimization.

### Aggregate Report Design

```go
type FileResult struct {
    Path     string
    Format   string
    Findings []llmutil.Finding
    Error    error  // non-nil if analysis failed for this file
}

type AggregateReport struct {
    TotalFiles       int
    AnalyzedFiles    int
    FailedFiles      int
    TotalFindings    int
    ByCategory       map[string]int  // category → count
    FilesNoGuardrails []string       // files with zero guardrails findings
    FilesNoSecurity   []string       // files with zero security findings
}
```

### Text Instruction Detection Heuristics

Patterns that indicate text-based ordering (check `RawContent`):
- Numbered steps: `1.`, `2.`, `3.` in sequence
- Explicit ordering: "first", "then", "next", "after that", "finally"
- Step headers: `### Step 1`, `## Phase 1`
- Workflow markers: "workflow:", "process:", "pipeline:"

Only flag if ≥ 3 ordering indicators are found — avoid false positives on files that mention "first" once.

### What NOT to Build

- Do NOT implement parallel directory analysis — sequential is correct for this story. Parallelism is a Growth optimization.
- Do NOT implement `--fix` for directory mode — that's Epic 3 scope.
- Do NOT implement SARIF or JSON output format — plain text and TTY for now (deferred per architecture).
- Do NOT implement registry notification or `--update-registry` — that's Story 2.4.
- Do NOT add a progress bar for directory analysis — simple "Analyzing file N of M..." is sufficient.
- Do NOT cache analysis results between files in directory mode — each file is independent.

### Project Structure

New files:
- `internal/doctor/directory.go` — DiscoverAgentFiles() function
- `internal/doctor/directory_test.go` — discovery tests with temp dir fixtures
- `internal/doctor/aggregate.go` — AggregateReport struct and Aggregate() function
- `internal/doctor/aggregate_test.go` — aggregation tests

Modified files:
- `internal/cmd/doctor.go` — directory mode, args validation
- `internal/doctor/render.go` — RenderAggregate() function
- `internal/doctor/context/enricher.go` (from Story 2.2) — add text-instruction-limit check

## Testing Strategy

- **Directory discovery tests:** Temp dir with mixed content (agent files + non-agent files + nested dirs). Verify only agent files are discovered. Test empty directory, single file, hidden directories skipped.
- **Aggregate tests:** Table-driven with various FileResult combinations. Verify category counts, zero-guardrails detection, zero-security detection.
- **Text instruction detection tests:** Agent files with explicit ordering vs no ordering. Verify threshold (≥ 3 indicators) prevents false positives.
- **Integration test:** Full directory analysis on `internal/doctor/parser/testdata/` — verify all files are discovered, analyzed, and aggregated.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-4] — Doctor DAG topology, context-enricher as CODE skill
- [Source: _bmad-output/planning-artifacts/prd.md#FR8] — Directory analysis with aggregate report
- [Source: _bmad-output/planning-artifacts/prd.md#FR16] — Diagnose text instruction limits
- [Source: _bmad-output/implementation-artifacts/epic-1-retro-2026-03-30.md] — Key risk: directory analysis concurrency + reporting
- [Source: internal/scanner/scan.go#SkipDirs] — Directories to skip during walk

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
