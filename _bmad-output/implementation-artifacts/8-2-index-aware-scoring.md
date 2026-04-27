# Story 8.2: Index-Aware Scoring + Aggregate

Status: done

## Story

As a **developer**,
I want Doctor to score an AGENTS.md index by resolving its links and computing an aggregate,
so that I see the real quality of my decomposed agent configuration.

## Acceptance Criteria

1. When `forgent doctor AGENTS.md` detects an index (2+ markdown links), it resolves all links
2. Each linked file is analyzed individually via static checks
3. Each file is scored on its **relevant categories only** — not all 16. Mapping: identity.md → identity/goals/decision_authority, security.md → security/prompt_injection, etc.
4. Aggregate = unique sections covered across all files / total sections (15)
5. Missing linked files produce HIGH finding
6. Render: per-file score + aggregate score + missing sections
7. Cross-file consistency runs on all resolved files (if LLM available)
8. `--format json` includes per-file scores and aggregate
9. All existing tests pass + new tests

## Tasks / Subtasks

- [ ] Task 1: Category-to-file mapping (AC: #3)
  - [ ] 1.1: Create mapping in `internal/doctor/index/` — each DefaultFile has its relevant categories
  - [ ] 1.2: Filter static check findings per file — only count categories relevant to that file
  - [ ] 1.3: Tests: identity.md scored on 3 categories, security.md on 2, etc.

- [ ] Task 2: Aggregate scoring (AC: #4, #5)
  - [ ] 2.1: Compute unique sections covered = union of all files' passed categories
  - [ ] 2.2: Missing linked files = their categories count as uncovered
  - [ ] 2.3: Tests: 6 files all covered = 15/15, 1 missing = 12/15

- [ ] Task 3: Wire into pipeline (AC: #1, #6, #7)
  - [ ] 3.1: In `runPipelineMode` or `runFileMode`: detect index via `docindex.IsIndex()`
  - [ ] 3.2: Resolve links, analyze each file, compute aggregate
  - [ ] 3.3: Render per-file + aggregate
  - [ ] 3.4: Run cross-file consistency on resolved files (if LLM)
  - [ ] 3.5: Tests: pipeline with index file, pipeline with missing link

- [ ] Task 4: JSON output (AC: #8)
  - [ ] 4.1: Extend JSONReport with `per_file_scores` array
  - [ ] 4.2: Tests: JSON contains per-file and aggregate

## Dev Notes

- The category-to-file mapping reuses `scaffold.DefaultFiles[].Categories`
- `ComputeStructural` needs a variant that filters by relevant categories
- The aggregate is NOT the average of individual scores — it's the union of covered categories
- Estimated effort: ~4 hours
