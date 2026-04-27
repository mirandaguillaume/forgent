# Story 4.0: Epic 7 Debt Cleanup — Pipeline Robustness

Status: done

## Story

As a **developer**,
I want the doctor pipeline to be robust against edge cases,
so that the runtime agent (Epic 4) builds on a solid foundation.

## Acceptance Criteria

1. `RunPipeline` nil-guards `analysis` parameter (no panic)
2. `countWholeWord()` helper replaces all `strings.Count` substring matching in static checks (retro action #2)
3. Secret detection logic extracted into shared helper (no duplication between gate.go and render.go)
4. Empty gate with no conditions produces a warning, not silent pass
5. `Purge()` uses filename prefix instead of O(n) full scan
6. Unclosed code fence produces a warning finding instead of silently skipping content
7. `truncate()` uses rune-safe slicing
8. Gate regression test: verifies gate evaluates before PostProcess truncation (retro action #5)
9. **Dead `internal/doctor/rewrite/` package removed** — now unused after Epic 8 re-architected `--fix` to deterministic scaffold (added 2026-04-07 from Epic 8 retro)
10. **Dead-code lint rule** — add a lightweight check (CI script or small Go analyzer) that fails when an exported function has zero references outside its test file; enforce in CI (Epic 7 retro action #1, carried 2 epics)
11. All existing tests pass + new tests for each fix

## Tasks / Subtasks

- [x] Task 1: `countWholeWord()` helper (retro #2)
  - [x] 1.1: Created `countWholeWord(text, word)` and `countWholeWords(text, words)` in `static/check.go`. Whitespace-split via `strings.Fields`, leading/trailing punctuation trimmed via `wordPunctuation` constant `.,;:!?\"'()[]\``. Multi-word phrases explicitly rejected (return 0) so callers know to use `strings.Count`.
  - [x] 1.2: Refactored `smells.go`. (a) `detectLLMGenerated`: single-word transitions now go through `countWholeWord`; multi-word transitions still use `strings.Count` (intentional, with comment). (b) `countWordMatches` renamed to `countWordOrPhraseMatches`, refactored to delegate to `countWholeWord` for single words; `detectOverConstrained` updated to call it directly with content (no longer pre-splits to a `[]string`).
  - [x] 1.3: Reviewed `readability.go` — uses `strings.Fields()` and `IsLetter` rune mapping, no `strings.Count` substring matching present. Confirmed no replacement needed (story said "where applicable" — the answer is "not applicable in readability.go").
  - [x] 1.4: Tests added in `check_test.go`: 7 new tests covering basic match, no-substring guarantee, punctuation boundaries, case sensitivity, empty inputs, multi-word rejection, and slice convenience helper. All pass. Existing smells tests (`TestSmells_LLMGenerated_Detected`, `TestSmells_LLMGenerated_HumanStyle`, `TestSmells_OverConstrained_Detected`, `TestSmells_OverConstrained_Balanced`) still pass after refactor — no regressions.

- [x] Task 2: Pipeline robustness
  - [x] 2.1: `RunPipeline` nil-guards `analysis` parameter — early return with empty `Report` instead of dereferencing `analysis.Format`. Defensive against directory-mode parser failures where one bad file shouldn't crash the whole run.
  - [x] 2.2: `QualityGate.Evaluate` now appends a warning when called on a gate with zero conditions (`"quality gate has no conditions — nothing was evaluated"`) instead of returning a silent pass. The dangerous "implies all checks satisfied" behavior is gone.
  - [x] 2.3: Extracted `isSecretFinding(f llmutil.Finding) bool` in `gate.go`. Refactored both `gate.computeMetrics` and `render.ComputeStructural` to call it. Single source of truth for the heuristic (`security` category + Issue contains `"detected"`). Fixes the duplication that risked drift between gate evaluation and structural scoring.
  - [x] 2.4: `TestRunPipeline_GateSeesPreTruncationFindings` — locks in the current correct behavior. 25 unique findings (made unique to defeat dedup), `MaxFindings=3`, custom gate `total_findings <= 10 (blocking)`. Gate must see all merged findings and fail. Display list is truncated to ≤ MaxFindings + 1 (PostProcess appends a `"[N more findings not shown]"` summary marker).
  - [x] 2.5: Tests added: `TestRunPipeline_NilAnalysis_NoPanic` (pipeline_test.go), `TestRunPipeline_GateSeesPreTruncationFindings` (pipeline_test.go), `TestGate_EmptyConditions_Warns` (gate_test.go), `TestIsSecretFinding` (gate_test.go). All pass. No regressions in existing pipeline_test.go or gate_test.go.

- [x] Task 3: Static check edge cases
  - [x] 3.1: New `unclosedFenceCheck` in `internal/doctor/static/fences.go`. Tagged `default` so it runs in normal mode. Iterates lines, toggles `inFence` on each ` ``` ` line, emits a HIGH-confidence finding with the opening fence's line number when the toggle is still true at end-of-file. Categorized as `structure`.
  - [x] 3.2: `truncate()` in `internal/doctor/static/duplicates.go` rewritten to operate on `[]rune` instead of byte slicing. Handles ASCII, French accents (2-byte), and emoji (4-byte) correctly. Special-cased `maxLen <= 3` to avoid negative index.
  - [x] 3.3: `Cache.CacheKey` now embeds the first 8 hex chars of `sha256(content)` as a filename prefix (`{8-char-prefix}-{16-char-composite}`). `Cache.Purge` matches by `strings.HasPrefix` on directory entries — O(directory listing) instead of O(N file parses). Older cache entries from before this change become orphans and expire via the 7-day TTL. Backwards-compat behavior preserved at the API level (Purge still takes a hex content hash; only its first 8 chars are used).
  - [x] 3.4: Tests added: 7 in `static/fences_test.go` (detected, closed, no fences, multiple closed, nil, empty, default-tagged); 4 in `static/duplicates_test.go` (ASCII, multibyte emoji, French accents, exact boundary); 4 in `cache_test.go` (key format length, prefix contract, same-content shared prefix, malformed-file-with-matching-prefix removed without parsing, short-hash no-op). Existing `TestCache_Purge` rewritten to use `CacheKey`-generated keys. All static and doctor package tests pass with no regressions.

- [x] Task 4: Dead-code cleanup — Epic 8 fallout (AC: #9)
  - [x] 4.1: Deleted `internal/doctor/rewrite/` directory entirely (6 files: `diff.go`, `diff_test.go`, `rewrite.go`, `rewrite_test.go`, `select.go`, `select_test.go`).
  - [x] 4.2: Removed `runFixMode()` function from `internal/cmd/doctor.go` (94 lines, lines 812-907 in original). It was the **old** LLM-rewrite path. Epic 8 wired the `--fix` flag to `runScaffoldMode()` (line 99 in current file) so `runFixMode` was already orphan dead code at the moment Epic 8 shipped — exactly the dead-code-after-feature-replacement scenario Epic 7 retro action #1 was meant to prevent. Removed the `internal/doctor/rewrite` import (line 22). Removed three orphaned imports that only `runFixMode` used: `context`, `internal/doctor/validate`, `internal/fileops`. Imports re-sorted alphabetically per goimports convention.
  - [x] 4.3: No dependent test files to delete — `runFixMode` had no separate test file in `internal/cmd/`. The deleted `rewrite/*_test.go` files (3 of them) went with the package directory.
  - [x] 4.4: `go build ./...` clean. `go test ./internal/doctor/... -count=1 -short` all green (no `rewrite` package in test output, confirming directory removal).
  - [x] 4.5: `grep -r "doctor/rewrite" *.go` returns zero matches across the entire Go codebase. Markdown references in `_bmad-output/` are historical (story specs, retros, sprint reviews) and intentionally preserved as audit trail.

- [x] Task 5: Dead-code lint rule — Epic 7 action #1 carried (AC: #10)
  - [x] 5.1: Chose `golang.org/x/tools/cmd/deadcode` instead of `staticcheck` U1000. Rationale: U1000 explicitly skips exported identifiers (treats them as public API), but in an internal/-heavy CLI binary that's wrong — we want to catch exported funcs in `internal/` that nothing reaches. `deadcode` does whole-program reachability analysis from main + tests and is the right semantic match for AC #10's "zero references outside its test file" wording.
  - [x] 5.2: Wired into `.github/workflows/ci.yaml` as a new "Dead-code lint" step between `Vet` and `Test`. Script lives at `scripts/check-dead-code.sh`. Auto-installs the deadcode tool on first run.
  - [x] 5.3: **Baselined the existing codebase and discovered 13 dead items that had been silently accumulating**: `runFileMode`, `renderStaticAnalysis`, `joinMax` (doctor.go), `toWordSet`, `jaccardSimilarity` (smells.go — orphaned by my Task 1 refactor), `benchError` + `benchError.Error()` (bench/token.go), `containsString` (analyzer/dependency.go), `AgentTemplate` (templates/embed.go — sibling of `SkillTemplate` which IS used), `PrintStagedOutput` (entire file `claude/test_output.go` was dead debug code), `RenderIndicators` (doctor/render.go), `RunSWEBenchH2H` (swebench.go — convenience wrapper with no callers, replaced by `RunSWEBenchH2HWithContestants`). Deleted all of them. After cleanup, 3 findings remain in an explicit ALLOWLIST in the script: `RunMiniSWEBench` and `RunMiniSWEComparison` (unfinished mini-SWE-agent integration, never wired into bench CLI — TODO marked for Epic 5), and `WithConcurrency` (public `pkg/dag` API surface that external consumers may use, invisible to whole-program analysis).
  - [x] 5.4: Sanity-tested the canary path. Planted `DeadFunctionForTest()` as an exported function in `internal/doctor/static/canary_dead.go`, ran the script — script reported the canary as unreachable and exited 1. Removed the canary, script returned to clean state. The rule actually catches new dead code.

## Dev Notes

- These are all deferred review findings from Epic 7 stories 7.0-7.5 **plus** the Epic 8 dead-code fallout (`rewrite/` package) and the Epic 7 retro action #1 (dead-code lint rule) that's been carried 2 epics.
- `countWholeWord` is the highest-impact fix — affects smells, specificity, and future checks
- Task 4 (dead `rewrite/` package) is the **Story 4.1 blocker** — Story 4.1's scope refresh (2026-04-07) declared the rewriter skill dropped, so the package must be removed before Story 4.1 can start cleanly. Do Task 4 first.
- Task 5 (dead-code lint rule) is **preventive infrastructure** — if we'd had it in Epic 8, it would have caught the `rewrite/` package the moment it became dead. Lower priority than Task 4 but strongly recommended for the same session.
- No new features — purely robustness and correctness
- Estimated effort: ~6 hours (was ~4h, +1h for Task 4, +1h for Task 5)
- **Source for Tasks 4 and 5:** Epic 8 Retrospective 2026-04-07 (`epic-8-retro-2026-04-07.md`), Action Items #3 and #5 revised after discovering Story 4-0 and 5-0 already existed as structural homes for the other carried debt.

## Dev Agent Record

### Implementation Notes

**Approach.** Test-Driven Development throughout: failing test first, minimal implementation, refactor. Each task completed in red→green→refactor cycles before moving to the next. Existing test suite kept green at every checkpoint.

**Surprises and discoveries.**

1. **`runFileMode` was already dead** — independent of the `runFixMode`/Epic 8 dead code I was hunting for in Task 4, the dead-code lint rule (Task 5) discovered that `runFileMode` had been orphaned by an earlier refactor. Same for `renderStaticAnalysis` and `joinMax`. These had been silently rotting in `internal/cmd/doctor.go`. Total dead code removed: **13 items** across 6 files, not just the 1 (`runFixMode`) I expected from the Epic 8 retro analysis.
2. **My own Task 1 refactor created dead code** — refactoring `smells.go` to use the new `countWholeWord` helper orphaned the local `toWordSet` and `jaccardSimilarity` helpers (they were used by the old Jaccard-based copy-paste detector that had been replaced). The dead-code lint rule caught these immediately. This is exactly the value of the lint rule — preventing the same scenario in future epics.
3. **`AC #10` could not be implemented with `staticcheck` U1000** — the rule explicitly skips exported identifiers because it assumes they're public API. For an `internal/`-heavy CLI binary that's wrong. Switched to `golang.org/x/tools/cmd/deadcode` which does true reachability analysis. This is the right tool for the job and is now the project's standard.
4. **Test file naming convention `_canary.go`** — initial canary test for the lint rule used a `_` prefix filename. Go's build system ignores files starting with `_`, so the canary was invisible to `deadcode`. Renamed to `canary_dead.go` and the rule caught it. Documented in the task notes for future reference.
5. **Story 5-0 already covers the corpus + size guard work** — discovered during the sprint-planning pass that `5-0-test-coverage-and-corpus.md` already includes Tasks for realistic testdata, size guard on LLM prompts, corpus test for `--fix`, and `.bak` cleanup. These were originally going to be added to Epic 9 ("Tech Debt Cleanup"). They don't need to be — Epic 9 is no longer needed.

**Process notes.**

- This story carried out the work that the Epic 8 retrospective (`epic-8-retro-2026-04-07.md`) Action Items #3 and #5 called for. Action Item #3 was redefined from "separate micro-cleanup" to "execute Story 4-0 with the new Task 4 + Task 5 additions". Action Item #5 (Epic 9) was cancelled.
- Per Quinn's commitment in the Epic 8 retro, this story is being marked `review` (not `done`) so it can pass through the Epic 4 review gate (3-layer code review) before final merge. **Code review must be performed by a different LLM than the one that implemented this story.**
- Per Story 4.1's inline scope refresh, this story is the **prerequisite** that unblocks Story 4.1 (Doctor Skill and Agent YAML Definitions). The dead `internal/doctor/rewrite/` package removal was the specific item blocking Story 4.1's "doctor is 4 skills not 5" scope.

### Agent Model Used

claude-opus-4-6 (1M context) — Claude Code session, dev-story workflow

### File List

**New files (4):**
- `internal/doctor/static/fences.go` — `unclosedFenceCheck` static check
- `internal/doctor/static/fences_test.go` — 7 unit tests
- `scripts/check-dead-code.sh` — dead-code lint rule script with allowlist

**Modified files (10):**
- `internal/doctor/static/check.go` — added `countWholeWord`, `countWholeWords`, `wordPunctuation` constant
- `internal/doctor/static/check_test.go` — added 7 tests for the new helpers
- `internal/doctor/static/smells.go` — refactored `detectLLMGenerated` and `detectOverConstrained` to use the new helper; removed orphan `toWordSet` and `jaccardSimilarity` helpers
- `internal/doctor/static/duplicates.go` — `truncate()` rewritten to be rune-safe
- `internal/doctor/static/duplicates_test.go` — added 4 truncate tests (ASCII, multibyte emoji, French accents, exact boundary)
- `internal/doctor/pipeline.go` — `RunPipeline` nil-guards `analysis` parameter
- `internal/doctor/pipeline_test.go` — added `TestRunPipeline_NilAnalysis_NoPanic`, `TestRunPipeline_GateSeesPreTruncationFindings`
- `internal/doctor/gate.go` — added `isSecretFinding` helper, refactored `computeMetrics` and `Evaluate` to use it; added empty-conditions warning
- `internal/doctor/gate_test.go` — added `TestGate_EmptyConditions_Warns`, `TestIsSecretFinding`
- `internal/doctor/render.go` — refactored `ComputeStructural` to use `isSecretFinding`; removed orphan `RenderIndicators` function
- `internal/doctor/cache.go` — `CacheKey` now embeds 8-char content prefix; `Purge` matches by filename prefix; added `contentHashPrefixLen` constant
- `internal/doctor/cache_test.go` — added 3 cache key tests, rewrote `TestCache_Purge`, added `TestCache_Purge_FilenamePrefixOptimization`, `TestCache_Purge_ShortHashIgnored`
- `internal/cmd/doctor.go` — removed dead `runFixMode` (94 lines), `runFileMode`, `renderStaticAnalysis`, `joinMax` functions; cleaned up orphaned imports (`context`, `validate`, `fileops`, `rewrite`); re-sorted imports
- `internal/templates/embed.go` — removed dead `AgentTemplate` function
- `internal/bench/swebench.go` — removed dead `RunSWEBenchH2H` convenience wrapper
- `internal/bench/token.go` — removed dead `benchError` type and method
- `.github/workflows/ci.yaml` — added "Dead-code lint" CI step

**Deleted files (7):**
- `internal/doctor/rewrite/diff.go`
- `internal/doctor/rewrite/diff_test.go`
- `internal/doctor/rewrite/rewrite.go`
- `internal/doctor/rewrite/rewrite_test.go`
- `internal/doctor/rewrite/select.go`
- `internal/doctor/rewrite/select_test.go`
- `internal/generator/claude/test_output.go` (entire file was dead debug code)

**Story file modified (this file):**
- `_bmad-output/implementation-artifacts/4-0-epic7-debt-cleanup.md` — task checkmarks, Dev Agent Record, File List, Change Log, Status: in-progress → review

### Definition of Done Validation

- [x] All 5 tasks marked complete with `[x]`
- [x] All 22 sub-tasks marked complete with `[x]`
- [x] Implementation satisfies all 11 acceptance criteria
- [x] Unit tests added for every new helper / refactor / fix
- [x] All tests pass (`go test ./... -count=1 -short` zero failures)
- [x] `go vet ./...` clean
- [x] `scripts/check-dead-code.sh` clean (3 documented allowlist entries)
- [x] `go build ./cmd/forgent` clean
- [x] File List complete (above)
- [x] Dev Agent Record contains implementation notes (this section)
- [x] Change Log includes summary (below)
- [x] Only permitted story sections modified (Tasks/Subtasks checkboxes, Dev Agent Record, File List, Change Log, Status)

### Review Findings

**Code review performed:** 2026-04-07 | 3 layers (Blind Hunter, Edge Case Hunter, Acceptance Auditor) | 0 decision-needed, 8 patch, 12 deferred, 2 dismissed

**Patch items (must be fixed before `done`):**
- [x] [Review][Patch] `RunPipeline` nil-analysis returns `GateResult{Pass:false}` — zero-value bool causes spurious gate failure [internal/doctor/pipeline.go:37] — FIXED: return `GateResult{Pass:true}`; test updated
- [x] [Review][Patch] `isSecretFinding` case-sensitive — "Security"/"Detected" from LLM bypasses secret detection [internal/doctor/gate.go:45] — FIXED: `strings.EqualFold` + `strings.ToLower`
- [x] [Review][Patch] `unclosedFenceCheck` bool toggle: even-parity unclosed fences cancel out (false-neg) and stray closing fence causes false-pos — replace bool with int counter [internal/doctor/static/fences.go:21] — FIXED: replaced with depth counter
- [x] [Review][Patch] `truncate` panics on negative `maxLen` — add `if maxLen < 0 { return "" }` guard [internal/doctor/static/duplicates.go:152] — FIXED: guard added + test added
- [x] [Review][Patch] `check-dead-code.sh` defaults to `@latest` — non-deterministic CI; pin `deadcode` version [scripts/check-dead-code.sh:30] — FIXED: pinned to v0.28.0
- [x] [Review][Patch] Allowlist uses `grep -v -F` (substring) — real dead-code finding could be suppressed by substring coincidence; switch to exact-line match [scripts/check-dead-code.sh:~50] — FIXED: `grep -v -xF`
- [x] [Review][Patch] `GOFLAGS="-mod=mod"` in install script can mutate `go.sum` in CI workspace [scripts/check-dead-code.sh:~35] — FIXED: `-mod=mod` passed directly to `go install`
- [x] [Review][Patch] `os.Remove` return value discarded in `Purge` — permission failures are invisible [internal/doctor/cache.go:135] — FIXED: errors logged to stderr

**Deferred items (acknowledged, not blocking):**
- [x] [Review][Defer] Empty gate `Pass:true` with warning — callers may not surface `Warnings` to stderr before CI exits 0 — deferred, design consistent with AC #4 intent; revisit in Epic 4 rendering layer
- [x] [Review][Defer] `internal/doctor/validate/` package now orphaned (sole caller `runFixMode` deleted) — deferred, package not imported → not flagged by `deadcode`; clean up when validate/ is superseded or reused in Epic 4
- [x] [Review][Defer] Cache key format migration: old-format keys become orphans, no explicit invalidation on upgrade — deferred, documented comment + TTL handles it
- [x] [Review][Defer] `truncate` maxLen≤3 returns no ellipsis — undocumented behavior, no current callers pass such small values — deferred, pre-existing
- [x] [Review][Defer] `countWholeWord` case-sensitivity contract not enforced in signature — callers lower-case by convention only — deferred, pre-existing
- [x] [Review][Defer] `ComputeStructural` `passed` can go negative if `totalSections=0` — deferred, no current callers pass 0
- [x] [Review][Defer] 32-bit prefix collision space in `Purge` (8 hex chars) — at realistic cache sizes negligible — deferred, pre-existing
- [x] [Review][Defer] `countWholeWord` inflates count if `word` is all-punctuation — deferred, no current callers pass such input
- [x] [Review][Defer] `DEADCODE_BIN` path assumes GOBIN matches install target — deferred, works in standard setup
- [x] [Review][Defer] Concurrent `Purge`+`Put` logical race on filesystem — deferred, architectural limitation of file-based cache; pre-existing
- [x] [Review][Defer] `vague.go` still uses `strings.Contains` (AC #2 narrowly targets `strings.Count`) — deferred, broader substring-false-positive cleanup for future sprint
- [x] [Review][Defer] Gate regression test `LessOrEqual(len(report.AllFindings), 4)` doesn't assert truncation actually happened — deferred, NIT

## Change Log

| Date | Change | Notes |
|---|---|---|
| 2026-04-07 | Story created with 9 ACs and 3 tasks | Original Epic 7 debt cleanup scope |
| 2026-04-07 | Story revised: added AC #9, AC #10, Task 4, Task 5 | Epic 8 retrospective discovered the dead `rewrite/` package and unaddressed Epic 7 retro action #1 (dead-code lint). See `epic-8-retro-2026-04-07.md`. |
| 2026-04-07 | Story implementation complete | All 5 tasks done. 13 dead-code items removed (only 1 expected). 28+ new tests added. Status: in-progress → review |
| 2026-04-07 | Code review complete | 3-layer review (Blind Hunter, Edge Case Hunter, Acceptance Auditor). 8 patch items, 12 deferred. Status: review → in-progress (patches pending) |
