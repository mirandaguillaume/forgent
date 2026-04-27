# Story 3.3: Interactive Rewrite Experience

Status: ready-for-dev

## Story

As a **developer**,
I want to preview diffs and choose which recommendations to apply,
so that I stay in control of every change to my agent files.

## Acceptance Criteria

1. `forgent doctor agent.md` (no `--fix`) displays recommendations read-only — no file modifications (FR19)
2. `forgent doctor agent.md --fix` shows each recommendation's diff and prompts the developer to select which to apply (FR20, FR21)
3. Only selected recommendations are merged into the rewrite. Quality gate validates the merged result before writing (FR24, FR25)
4. `SafeWrite()` creates backup and writes atomically (FR22)
5. If merged rewrite fails quality gate, doctor reports which recommendations conflict and does NOT write the file
6. Rewritten files retain all original frontmatter fields and preserve custom sections
7. UTF-8 non-ASCII characters and emoji are preserved without corruption (NFR33)
8. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Rewriter skill (AC: #3, #6)
  - [ ] 1.1: Create `internal/doctor/rewrite/rewrite.go` — `Rewrite(original []byte, recommendations []llmutil.Finding, format string, provider llm.Provider) ([]byte, error)`
  - [ ] 1.2: Build rewrite prompt: include original file, selected recommendations, format-specific instructions (preserve frontmatter, preserve sections)
  - [ ] 1.3: Parse LLM response with `llmutil.ExtractYAML()` or raw content extraction (rewriter returns full file, not YAML)
  - [ ] 1.4: Handle LLM returning markdown-fenced content — strip fences to get raw file
  - [ ] 1.5: Write tests with mock provider returning known rewrites

- [ ] Task 2: Diff generation and display (AC: #2)
  - [ ] 2.1: Create `internal/doctor/rewrite/diff.go` — `GenerateDiff(original, rewritten []byte) string` producing unified diff format
  - [ ] 2.2: Use simple line-by-line diff with context (3 lines before/after each change)
  - [ ] 2.3: TTY mode: colorize diff (green for additions, red for removals) using fatih/color
  - [ ] 2.4: Plain mode: standard unified diff format
  - [ ] 2.5: Write tests with known input/output pairs

- [ ] Task 3: Interactive selection (AC: #2)
  - [ ] 3.1: Create `internal/doctor/rewrite/select.go` — `SelectRecommendations(findings []llmutil.Finding, tty bool) ([]llmutil.Finding, error)`
  - [ ] 3.2: TTY mode: display numbered list of recommendations, prompt for comma-separated selection (e.g., "1,3,5")
  - [ ] 3.3: Non-TTY mode (pipe): apply all recommendations (no interactive prompt possible)
  - [ ] 3.4: Read selection from stdin using `bufio.Scanner`
  - [ ] 3.5: Validate input: reject out-of-range numbers, empty selection
  - [ ] 3.6: Write tests with simulated stdin

- [ ] Task 4: Wiring `--fix` into CLI (AC: #1, #2, #3, #4, #5)
  - [ ] 4.1: In `internal/cmd/doctor.go`, add `--fix` flag (already partially exists)
  - [ ] 4.2: Without `--fix`: display findings as before (read-only), exit
  - [ ] 4.3: With `--fix`: after analysis, show selection prompt → rewrite selected → quality gate → SafeWrite
  - [ ] 4.4: If quality gate fails: show failure reasons, suggest applying recommendations individually, do NOT write
  - [ ] 4.5: If quality gate passes: call `fileops.SafeWrite()`, show success message with backup path
  - [ ] 4.6: Add `--force` flag to bypass diff-size sanity check
  - [ ] 4.7: Call `fileops.CleanupBackups()` at end of successful run
  - [ ] 4.8: Write integration test for full `--fix` flow with mock provider

- [ ] Task 5: UTF-8 and content preservation (AC: #7)
  - [ ] 5.1: Ensure rewrite prompt instructs LLM to preserve exact content encoding
  - [ ] 5.2: Test with files containing emoji, CJK characters, accented characters
  - [ ] 5.3: Test with files containing code blocks with special characters

## Dev Notes

### Architecture References

- **ADR-2 (Quality Gate):** Quality gate sits between rewriter output and SafeWrite. Fail-closed.
- **ADR-4 (Doctor DAG):** Rewriter is Layer 3, conditional on `--fix`. For this story, wired directly in CLI. Epic 4 moves it to DAG.
- **Architecture data flow:** `recommendations → rewriter → validate.Gate → fileops.SafeWrite`

### Dependencies on Previous Stories

- **Story 3.1 (Defensive File Operations):** SafeWrite, Restore, CleanupBackups — must be complete
- **Story 3.2 (Quality Gate):** Validate() — must be complete
- **Epic 2 (Registry):** Findings include citation_id from registry — rewriter receives enriched findings

### Previous Story Intelligence (from Epic 1 retro)

- **Action #1:** "Resolve frontmatter parsing" — rewriter must handle the same frontmatter edge cases
- **Action #6:** "Size guard on prompts" — rewrite prompt includes full file content + recommendations. Truncate RawContent > 8K tokens for the prompt.
- **Action #4:** "Always review before done."

### Rewrite Prompt Design

```
You are rewriting an agent definition file. Apply ONLY the selected improvements below.

RULES:
1. Preserve ALL frontmatter fields exactly — do not add, remove, or rename any
2. Preserve ALL existing sections — you may add content but never remove sections
3. Preserve the file's original formatting style (indentation, spacing)
4. Output the COMPLETE rewritten file — not a diff, not a partial

ORIGINAL FILE:
---
[original content]
---

IMPROVEMENTS TO APPLY:
[selected recommendations with descriptions]

OUTPUT THE COMPLETE REWRITTEN FILE:
```

### What NOT to Build

- Do NOT implement batch rewrite for directory mode — `--fix` only works on single files
- Do NOT implement auto-apply (applying all without prompt) — always show selection in TTY
- Do NOT implement undo history beyond single `.bak` — one backup per file is sufficient
- Do NOT implement a full diff algorithm (Myers, patience) — simple line diff is sufficient for display

### Project Structure

New files:
- `internal/doctor/rewrite/rewrite.go` — Rewrite() with LLM call
- `internal/doctor/rewrite/rewrite_test.go`
- `internal/doctor/rewrite/diff.go` — GenerateDiff(), colorized output
- `internal/doctor/rewrite/diff_test.go`
- `internal/doctor/rewrite/select.go` — SelectRecommendations() interactive
- `internal/doctor/rewrite/select_test.go`

Modified files:
- `internal/cmd/doctor.go` — `--fix` and `--force` flags, rewrite flow integration

## Testing Strategy

- **Rewriter tests:** Mock provider returns known rewrite. Verify output matches expectations.
- **Diff tests:** Known original + rewritten → expected unified diff. Both TTY and plain modes.
- **Selection tests:** Simulated stdin with "1,3,5" → correct subset returned. Edge cases: empty input, out of range, single selection.
- **Quality gate integration:** Rewrite that passes gate → SafeWrite called. Rewrite that fails gate → error, no write.
- **UTF-8 tests:** Files with emoji, CJK, accented characters → round-trip preservation.
- **Content preservation:** Frontmatter fields, custom sections, code blocks all preserved after rewrite.
- **Integration test:** Full flow: findings → select → rewrite → validate → SafeWrite, with mock provider.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-2] — Quality gate fail-closed
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-4] — Rewriter as Layer 3, conditional
- [Source: _bmad-output/planning-artifacts/prd.md#FR19-FR25] — Read-only default, diff preview, granular apply, backup, restore, format preservation
- [Source: _bmad-output/planning-artifacts/prd.md#NFR33] — UTF-8 handling
- [Source: _bmad-output/implementation-artifacts/epic-1-retro-2026-03-30.md] — Size guard on prompts

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
