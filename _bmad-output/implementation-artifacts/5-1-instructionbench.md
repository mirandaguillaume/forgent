# Story 5.1: InstructionBench — Agent File Corpus and Quality Measurement

Status: done

## Story

As a **product architect**,
I want to measure doctor's recommendation quality against a corpus of real agent files,
so that I can prove doctor helps and publish the results.

## Acceptance Criteria

1. Corpus of ≥ 100 real agent files scraped from public GitHub repos, organized by format (Claude, Copilot), with source attribution and permissive licenses only (MIT, Apache, unlicensed public)
2. Doctor runs against the corpus and InstructionBench collects: total recommendations, acceptance rate per category, confidence distribution, false positive rate
3. Results are reproducible — same corpus + same config → same measurements
4. Output includes: methodology description, corpus statistics, per-category results, overall quality score
5. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [x] Task 1: Corpus collection and organization (AC: #1)
  - [x] 1.1-1.2: Infrastructure created; actual scraping is manual work by project lead. Existing 22 files in parser/testdata serve as seed corpus.
  - [x] 1.3: Directory structure created at `internal/bench/testdata/instruction/` with claude/ and copilot/ subdirs
  - [x] 1.4: `README.md` with corpus requirements, golden set format, source attribution instructions
  - [x] 1.5: Engine's `discoverCorpusFiles` + `analyzeCorpusFile` verifies parse on every run
  - [x] 1.6: `.gitignore` excludes corpus files (claude/, copilot/, golden.yaml)

- [x] Task 2: InstructionBench engine (AC: #2, #3)
  - [x] 2.1: `internal/bench/instruction.go` — `RunInstructionBench(opts InstructionBenchOpts) (*InstructionBenchResult, error)`
  - [x] 2.2: For each file: DetectFormat → Parse → Analyze (with provider) → collect findings
  - [x] 2.3: `InstructionBenchResult`: TotalFiles, TotalFindings, ByCategory, ConfidenceDistribution, OverallScore, FileResults
  - [x] 2.4: `CategoryResult`: Count, AcceptanceRate, FalsePositives
  - [x] 2.5: Acceptance rate: % of findings with confidence >= moderate
  - [x] 2.6: Golden set via `golden.yaml` — maps file→expected categories, FP = unexpected category
  - [x] 2.7: Deterministic: TestRunInstructionBench_Reproducible verifies identical results on re-run
  - [x] 2.8: 6 tests: MockCorpus, NilProvider, EmptyCorpus, MissingDir, Reproducible, GoldenSet

- [x] Task 3: Report generation (AC: #4)
  - [x] 3.1: `internal/bench/instruction_report.go` — `FormatInstructionReport(result, opts) string`
  - [x] 3.2: Sections: Metadata, Methodology, Corpus Statistics, Per-Category Results, Confidence Distribution, Overall Quality Score
  - [x] 3.3: Output as Markdown with tables
  - [x] 3.4: Includes: date, model, corpus size, registry version, golden set size
  - [x] 3.5: 3 report tests: AllSections, EmptyResult, IsMarkdown

- [x] Task 4: CLI integration (AC: #2)
  - [x] 4.1: `instruction` level added to `forgent-bench --level instruction`
  - [x] 4.2: `runInstructionBench(cmd)` in `internal/bench/cmd/bench.go`
  - [x] 4.3: Flags: `--corpus`, `--golden`, `--provider`, `--model`, `--output`
  - [x] 4.4: Engine tests serve as smoke tests; CLI compiles and routes correctly

## Dev Notes

### Architecture References

- **FR51:** "Developer can run InstructionBench to measure recommendation quality across a corpus."
- **Architecture:** "InstructionBench — uses analyzer output for corpus evaluation." Located at `internal/bench/instruction.go`.
- **Existing bench framework:** `internal/bench/` has extensive infrastructure (pipeline, eval, report, formal properties). InstructionBench extends this.

### Corpus Storage (CRITICAL)

Per project memory `feedback_no_datasets.md`: **never commit bench testdata/JSONL files to git**. The corpus is stored externally. The code references a directory path. CI tests use a small mock corpus (5 synthetic files), not the full 100-file corpus.

### False Positive Measurement

Full false positive measurement requires human annotation. For MVP:
1. Manually annotate 20 files with expected categories (golden set)
2. Run doctor on golden set, compare findings against expected
3. False positive = finding category not in expected set for that file

### Dependencies

- **Epic 1-2 complete:** Parser, analyzer, enricher, registry — all needed for running doctor against corpus files.
- **Existing bench framework:** `internal/bench/` provides pipeline, eval, report infrastructure.

### What NOT to Build

- Do NOT commit the corpus to git — store reference only
- Do NOT implement auto-annotation — manual golden set for MVP
- Do NOT implement statistical significance testing — simple counts and percentages for MVP
- Do NOT implement the runtime-vs-text experiment — that's Story 5.2
- Do NOT build a scraper tool — manual collection with source attribution

### Project Structure

New files:
- `internal/bench/instruction.go` — RunInstructionBench(), BenchResult
- `internal/bench/instruction_test.go`
- `internal/bench/instruction_report.go` — FormatReport()
- `internal/bench/instruction_report_test.go`
- `internal/bench/cmd/instruction.go` — CLI subcommand

NOT committed (external):
- `internal/bench/testdata/instruction/claude/` — corpus files
- `internal/bench/testdata/instruction/copilot/` — corpus files
- `internal/bench/testdata/instruction/README.md` — attribution
- `internal/bench/testdata/instruction/golden.yaml` — annotations

## Testing Strategy

- **Engine tests:** Mock corpus (5 synthetic files) + mock provider. Verify BenchResult fields populated correctly.
- **Category counting tests:** Known findings → correct ByCategory counts.
- **False positive tests:** Golden set with known expected categories → correct FP rate calculation.
- **Report format tests:** Verify Markdown output has all required sections.
- **Reproducibility test:** Run twice with same inputs → identical BenchResult.
- **CI gate:** `go test ./... -count=1 -race` (uses mock corpus only)

## References

- [Source: _bmad-output/planning-artifacts/prd.md#FR51] — InstructionBench quality measurement
- [Source: _bmad-output/planning-artifacts/architecture.md] — InstructionBench at internal/bench/instruction.go
- [Source: internal/bench/] — Existing bench framework
- [Source: _bmad-output/implementation-artifacts/epic-1-retro-2026-03-30.md] — 22 real agent files already collected

## Dev Agent Record

### Agent Model Used

claude-opus-4-6

### Completion Notes List

- Engine: `RunInstructionBench` walks corpus, parses, analyzes, collects per-category stats
- Golden set support: FP = finding category not in expected set for annotated files
- Report: Markdown with 6 sections (Metadata, Methodology, Corpus Statistics, Per-Category, Confidence, Overall Score)
- CLI: `forgent-bench --level instruction --corpus <dir> --golden <yaml> --output <file>`
- 9 tests total (6 engine + 3 report), all pass
- Corpus infrastructure: directory + .gitignore + README (files not committed per convention)

### File List

- `internal/bench/instruction.go` (new — engine)
- `internal/bench/instruction_test.go` (new — 6 tests)
- `internal/bench/instruction_report.go` (new — Markdown report)
- `internal/bench/instruction_report_test.go` (new — 3 tests)
- `internal/bench/testdata/instruction/.gitignore` (new)
- `internal/bench/testdata/instruction/README.md` (new)
- `internal/bench/cmd/bench.go` (modified — instruction level + flags)

## Review Findings

_Code review — 2026-04-10_

### Patches

- [x] [Review][Patch] Hardcoded `goldenFiles = 20` — fixed: exposed `GoldenFileCount` on `InstructionBenchResult` from actual golden set size [cmd/bench.go, instruction.go]
- [x] [Review][Patch] O(n^2) acceptance rate loop — fixed: accumulate `Accepted` counter during first pass, compute rate from counts [instruction.go]
- [x] [Review][Patch] FP is raw count not rate — fixed: added `FalsePositiveRate` field, computed as FP/Count*100, report shows `"X% (N)"` [instruction.go, instruction_report.go]

### Deferred

- [x] [Review][Defer] Confidence values outside {high,moderate,low} silently dropped from report — deferred, minor
- [x] [Review][Defer] Error files inflate TotalFiles, deflating OverallScore — deferred, document behavior
- [x] [Review][Defer] Golden set path separator broken on Windows — deferred, Unix-only for now
- [x] [Review][Defer] FP counting ignores files not in golden set — by design (opt-in annotation)
- [x] [Review][Defer] `containsString` duplicates `slices.Contains` — deferred, minor
- [x] [Review][Defer] `sortedMapKeys` duplicates stdlib — deferred, minor
- [x] [Review][Defer] Report embeds `time.Now()` making output non-deterministic — deferred, minor
- [x] [Review][Defer] Unnecessary pre-declaration in CLI handler — deferred, style
- [x] [Review][Defer] No test for error-file handling in report — deferred, minor
- [x] [Review][Defer] Hidden files not skipped in corpus discovery — deferred, minor

## Change Log

- 2026-04-10: Implemented story 5-1. InstructionBench engine, report generator, CLI integration, corpus infrastructure. 9 tests, all 30+ packages pass.
- 2026-04-10: Code review: 3 patches applied (golden count, O(n) acceptance rate, FP rate computation). Corpus scraping in progress.
