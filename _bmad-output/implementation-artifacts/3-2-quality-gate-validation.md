# Story 3.2: Quality Gate Validation

Status: ready-for-dev

## Story

As a **developer**,
I want doctor to verify every rewrite before applying it,
so that my agent files are never broken by a bad recommendation.

## Acceptance Criteria

1. Quality gate checks: (a) frontmatter is valid YAML, (b) all original frontmatter fields preserved, (c) markdown structure is well-formed, (d) all user custom sections preserved
2. Diff-size sanity check — if rewrite changes >50% of file content, flag as suspicious and require `--force` to apply
3. Any quality gate check failure → rewrite REJECTED, original file untouched, error with specific failure reason
4. Test corpus of ≥ 30 real Claude files and ≥ 30 real Copilot files — 100% of rewrites pass all quality gate checks
5. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Quality gate core (AC: #1, #3)
  - [ ] 1.1: Create `internal/doctor/validate/gate.go` — `Validate(original, rewritten []byte) *GateResult`
  - [ ] 1.2: `GateResult` struct: `Passed bool`, `Failures []GateFailure`, where `GateFailure` has `Check string`, `Detail string`
  - [ ] 1.3: Check 1 — frontmatter validity: extract YAML frontmatter from rewritten content, parse with yaml.v3
  - [ ] 1.4: Check 2 — field preservation: compare original frontmatter field names against rewritten, report any missing fields
  - [ ] 1.5: Check 3 — markdown structure: verify headers are well-formed (# prefix), code blocks are balanced (``` pairs), lists have proper indentation
  - [ ] 1.6: Check 4 — section preservation: extract section headers from original and rewritten, report any missing sections
  - [ ] 1.7: Each check runs independently — all failures are collected and reported together
  - [ ] 1.8: Write tests for each check with both passing and failing inputs

- [ ] Task 2: Diff-size sanity check (AC: #2)
  - [ ] 2.1: Add `DiffRatio(original, rewritten []byte) float64` — computes the proportion of changed lines (using simple line-by-line diff)
  - [ ] 2.2: If DiffRatio > 0.5, add a `GateFailure` with check `"diff_size"` and detail showing the percentage
  - [ ] 2.3: `Validate()` accepts an `opts` parameter with `Force bool` — if true, diff_size check is a warning, not a failure
  - [ ] 2.4: Write tests for small diff (pass), large diff (fail), large diff with force (pass)

- [ ] Task 3: Format-specific validation via parsers (AC: #1)
  - [ ] 3.1: Extend `FormatParser` interface with `Validate(original, rewritten []byte) error` (already defined in ADR-6)
  - [ ] 3.2: Claude parser `Validate()`: verify frontmatter fields match Claude format expectations
  - [ ] 3.3: Copilot parser `Validate()`: verify description ≤ 1024 chars, frontmatter matches Copilot format
  - [ ] 3.4: Forgent parser `Validate()`: verify YAML parses as valid skill/agent spec
  - [ ] 3.5: Quality gate calls the format-specific `Validate()` as an additional check
  - [ ] 3.6: Write tests for each parser's Validate method

- [ ] Task 4: Test corpus validation (AC: #4)
  - [ ] 4.1: Ensure ≥ 30 Claude files in `internal/doctor/parser/testdata/claude/` (add more from public repos if needed)
  - [ ] 4.2: Ensure ≥ 30 Copilot files in `internal/doctor/parser/testdata/copilot/`
  - [ ] 4.3: Create `internal/doctor/validate/gate_corpus_test.go` — runs quality gate on all testdata files (identity rewrite: original == rewritten should always pass)
  - [ ] 4.4: Create synthetic "bad rewrite" test cases for each failure mode

## Dev Notes

### Architecture References

- **ADR-2 (Quality Gate):** Four-check fail-closed validation. This story implements the exact four checks plus diff-size sanity.
- **ADR-6 (Format Parser):** `Validate(original, rewritten []byte) error` method on FormatParser interface.
- **NFR8:** "100% of rewritten files retain valid YAML frontmatter and valid Markdown structure."
- **Architecture enforcement rule:** Quality gate is code-based validation — deterministic, no LLM calls.

### Frontmatter Extraction

Reuse the frontmatter extraction logic from `internal/doctor/parser/claude.go` (or extract to a shared utility in `internal/doctor/parser/`). Frontmatter is delimited by `---` markers.

### Diff Ratio Algorithm

Simple line-based comparison. No need for a full diff algorithm — count lines that differ:

```go
func DiffRatio(original, rewritten []byte) float64 {
    origLines := strings.Split(string(original), "\n")
    rewriteLines := strings.Split(string(rewritten), "\n")
    maxLen := max(len(origLines), len(rewriteLines))
    if maxLen == 0 { return 0 }
    changed := 0
    for i := 0; i < maxLen; i++ {
        if i >= len(origLines) || i >= len(rewriteLines) || origLines[i] != rewriteLines[i] {
            changed++
        }
    }
    return float64(changed) / float64(maxLen)
}
```

### Dependencies on Previous Stories

- **Story 3.1 (Defensive File Operations):** SafeWrite is used by Story 3.3, not this story. Quality gate is validation only — it decides whether to proceed, but does not write files.
- **Epic 1 parsers:** Claude and Copilot parsers already exist. This story adds the `Validate()` method to them.

### Previous Story Intelligence (from Epic 1 retro)

- **Action #1:** "Resolve frontmatter parsing" — the quality gate's frontmatter check must handle the same edge cases that the parser had trouble with (unquoted colons in descriptions).
- **Action #4:** "Always review before done."
- **Recurring finding:** "CRLF not handled" — diff ratio must normalize line endings before comparison.

### What NOT to Build

- Do NOT implement the rewriter LLM skill — that's Story 3.3
- Do NOT implement the interactive `--fix` flow — that's Story 3.3
- Do NOT implement SafeWrite integration — the gate only returns pass/fail, Story 3.3 wires it
- Do NOT implement JSON/SARIF output for quality gate results — plain structs for now

### Project Structure

New files:
- `internal/doctor/validate/gate.go` — Validate(), GateResult, GateFailure, DiffRatio
- `internal/doctor/validate/gate_test.go` — unit tests
- `internal/doctor/validate/gate_corpus_test.go` — corpus validation tests

Modified files:
- `internal/doctor/parser/claude.go` — add Validate() method
- `internal/doctor/parser/copilot.go` — add Validate() method
- `internal/doctor/parser/forgent.go` — add Validate() method
- `internal/doctor/parser/registry.go` — verify FormatParser interface includes Validate

## Testing Strategy

- **Check-by-check unit tests:** Each of the 4 checks tested independently with passing and failing inputs.
- **Diff ratio tests:** Small diff (< 50%), large diff (> 50%), identical files (0%), completely different files (100%).
- **Force mode tests:** Large diff with `Force: true` passes, without force fails.
- **Corpus tests:** Identity rewrite (original == rewritten) on all 60+ testdata files — must all pass.
- **Synthetic failure tests:** Missing frontmatter field, broken markdown, missing section, invalid YAML frontmatter.
- **CRLF tests:** Files with mixed line endings pass validation.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-2] — Four-check quality gate
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-6] — FormatParser.Validate() method
- [Source: _bmad-output/planning-artifacts/prd.md#NFR8] — Zero broken rewrites
- [Source: _bmad-output/planning-artifacts/prd.md#FR24] — Format-preserving Claude rewrite
- [Source: _bmad-output/planning-artifacts/prd.md#FR25] — Format-preserving Copilot rewrite
- [Source: _bmad-output/implementation-artifacts/epic-1-retro-2026-03-30.md] — Frontmatter parsing issues

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
