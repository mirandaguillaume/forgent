# Story 7.4: Cross-File Consistency + Codebase Drift Detection

Status: done

## Story

As a **developer**,
I want Doctor to detect contradictions between my agent files and warn me about stale references,
so that my agent configuration stays consistent and up-to-date with my codebase.

## Acceptance Criteria

1. **Cross-file consistency** (directory mode): when `forgent doctor .` finds multiple agent files, LLM receives all files in a single prompt with a cross-file consistency check
2. Cross-file findings detect: contradictory instructions, persona conflicts, permission mismatches, overlapping responsibilities
3. Cross-file findings rendered in a separate "Consistency" section with references to both source files
4. **Codebase drift — file references**: when agent file mentions file paths (e.g., `src/utils/auth.go`), check if file exists via `os.Stat()`. Flag missing files.
5. **Codebase drift — command references**: when agent file mentions commands (e.g., `make gen`, `npm test`, `go test ./...`), validate against Makefile/package.json/go.mod. Flag commands not found.
6. **Git staleness**: when `--thorough` and inside a git repo, check `git log -1 --format=%cr -- <file>`. Warn if agent file not modified in >90 days.
7. Drift checks are static (no LLM) and tagged "default" (file refs) + "git-aware" (staleness)
8. Cross-file is LLM-based and tagged "default" (only in directory mode)
9. All existing tests pass + new tests

## Research Backing

- Chatlatanagulchai et al. (arXiv:2511.12884): 67.4% of agent files modified multiple times
- Packmind: codebase drift detection is their strongest differentiating feature
- AgentLinter: cross-file persona conflict detection

## Tasks / Subtasks

- [x] Task 1: File reference validation (AC: #4)
  - [x] 1.1-1.6: drift.go with file path regex, os.Stat validation, example context skipping, Tags: ["default"], 6 tests

- [x] Task 2: Command validation (AC: #5)
  - [x] 2.1-2.5: make/npm/go-test pattern detection, Makefile target check, package.json script check, go test path validation

- [x] Task 3: Git staleness (AC: #6)
  - [x] 3.1-3.6: staleness.go with git log timestamp check, 90-day threshold, Tags: ["git-aware"]

- [x] Task 4: Cross-file consistency (AC: #1, #2, #3)
  - [x] 4.1-4.7: AnalyzeCrossFile() + buildCrossFilePrompt() in analyzer.go, "consistency" category added to validCategories, max 5 files

## Dev Agent Record

### File List
- `internal/doctor/static/drift.go` — NEW: file path + command validation
- `internal/doctor/static/drift_test.go` — NEW: 6 tests
- `internal/doctor/static/staleness.go` — NEW: git staleness check
- `internal/doctor/analyzer/analyzer.go` — MODIFIED: added AnalyzeCrossFile + buildCrossFilePrompt
- `internal/doctor/llmutil/parse.go` — MODIFIED: added consistency + readability to validCategories

### Change Log
- 2026-03-31: Story 7.4 implemented — file drift detection, command validation, git staleness, cross-file consistency prompt

## Dev Notes

- File/command reference validation is purely static — fast, no LLM needed
- Cross-file consistency requires LLM because detecting contradictions is semantic
- Git staleness uses exec.Command("git", ...) — same pattern as scanner/
- For large projects with many agent files, cross-file prompt may exceed context. Limit to max 5 files (largest first) and warn if truncated.
- Estimated effort: ~8 hours total
