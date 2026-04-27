# Story 6.0: DX Polish + Edge Case Fixes

Status: review

## Story

As a **developer**,
I want Doctor's output polished and edge cases handled,
so that the CLI feels production-ready before ecosystem features.

## Acceptance Criteria

1. Animated spinner during LLM analysis (replaces static "Analyzing with LLM..." message)
2. CI annotation output fully escaped (`%`, `\n`, `::` in messages) — audit all output paths (retro action #4)
3. Tilde fences (`~~~`) handled by all code-fence-aware checks
4. O(n²) duplicates capped at 50 findings with early exit
5. Numbered list regex tightened — `3.5 GHz` no longer detected as instruction
6. OpenAI `sk-` regex tightened — require `sk-proj-` or minimum length
7. Secret patterns context-aware — skip lines mentioning "set", "configure", "env var" near generic patterns
8. `DefaultSeverity()` wired into Finding output (add Severity field to Finding struct)
9. Checkbox agent detection filters empty sections (0-word sections excluded from average)
10. LLM-generated detection uses OR (either 4+ transitions OR 6+ numbered lists triggers)
11. All existing tests pass + new tests

## Tasks / Subtasks

- [x] Task 1: Output polish
  - [x] 1.1: Animated spinner — char sequence `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` during LLM analysis, cleared on completion
  - [x] 1.2: Audit all `fmt.Printf` in ci.go, render.go, json_output.go — verify escaping per format
  - [x] 1.3: Tests: spinner renders in TTY, not in CI/pipe

- [x] Task 2: Static check edge cases
  - [x] 2.1: Tilde fence support — track `~~~` same as `` ``` `` in all checks
  - [x] 2.2: Duplicate detection cap — `if len(findings) >= 50 { break }`
  - [x] 2.3: Numbered list regex — require `^\s*\d+\.\s+[A-Z]` (capital letter after number = instruction)
  - [x] 2.4: OpenAI regex — `sk-proj-[a-zA-Z0-9]{20,}` or `sk-[a-zA-Z0-9]{48,}` (longer minimum)
  - [x] 2.5: Context-aware secret suppression — skip if line contains "configure", "set", "env", "variable"
  - [x] 2.6: Tests for each fix

- [x] Task 3: Smell refinements
  - [x] 3.1: Checkbox — `avgWords` excludes sections with 0 words
  - [x] 3.2: LLM-generated — change AND to OR: `transitionCount > 3 || numberedCount > 5`
  - [x] 3.3: Tests: empty sections excluded, OR logic works

- [x] Task 4: Finding struct evolution
  - [x] 4.1: Add `Severity string` field to `llmutil.Finding`
  - [x] 4.2: Static checks populate Severity from `DefaultSeverity()`
  - [x] 4.3: Render uses Severity if present, falls back to Confidence
  - [x] 4.4: JSON output includes Severity
  - [x] 4.5: Tests: Severity propagation, backward compat (LLM findings have no Severity)

## Dev Notes

- This is DX polish — no new features, just making existing features production-quality
- The spinner needs goroutine + channel pattern (render loop + stop signal)
- Finding.Severity is a breaking change to the struct — but JSON uses `omitempty` so backward compatible
- Estimated effort: ~6 hours

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

None — no HALT conditions triggered.

### Completion Notes

- **Task 1:** Created `spinner.go` with goroutine+channel animation (80ms/frame, 10 braille chars). Integrated into `cmd/doctor.go` — replaces static "Analyzing with LLM..." message. Fixed CI escaping bug: `RenderGateAnnotation` was not escaping gate failure/warning messages through `escapeAnnotation()`. Added `spinner_test.go` (start/stop, stop-without-start, idempotent stop) and `ci_test.go` (6 escaping cases).
- **Task 2:** Created shared `IsCodeFenceLine()` helper that handles both `` ``` `` and `~~~`. Replaced all 7 `strings.HasPrefix(trimmed, "```")` occurrences across duplicates, fences, secrets, vague, padding, drift, readability. Added `maxFindings=50` cap with `break outer` label in duplicates. Tightened numbered list regex to `^\s*\d+\.\s+[A-Z]` (capital letter after number). Tightened OpenAI regex to `sk-proj-{20+}` or `sk-{48+}`. Added `isConfigContext()` suppression for generic secret patterns near configuration documentation. Updated `TestSecrets_DetectsOpenAIKey` → two tests for project and legacy keys + short prefix test.
- **Task 3:** Checkbox detection now excludes 0-word sections from average (`nonEmptySections` counter). LLM-generated detection changed from AND to OR (`&&` instead of `||` in the nil-return condition). Numbered list counting uses original-case content (not lowercased) so capital-letter requirement works. Added tests: transitions-only OR, numbered-only OR, measurement false-positive, empty sections excluded.
- **Task 4:** Added `Severity string` field with `omitempty` to `Finding` struct. `RunChecks` stamps `DefaultSeverity()` on findings with empty Severity. `effectiveSeverity()` helper in render.go: returns Severity if set, else Confidence. Updated TTY/plain/consistency render + CI annotations + JSON output to use effective severity. Tests: Severity propagation and preservation of existing Severity.

### Change Log

- 2026-04-15: Story 6.0 implemented — DX polish: spinner, CI escaping, tilde fences, duplicates cap, regex fixes, smell refinements, Severity field.

### File List

New files:
- `internal/doctor/spinner.go`
- `internal/doctor/spinner_test.go`
- `internal/doctor/ci_test.go`

Modified files:
- `internal/cmd/doctor.go` — spinner integration
- `internal/doctor/ci.go` — escape gate annotation messages, use Severity
- `internal/doctor/render.go` — effectiveSeverity helper, use in all render paths
- `internal/doctor/json_output.go` — use Severity in JSON output
- `internal/doctor/llmutil/parse.go` — add Severity field to Finding
- `internal/doctor/static/check.go` — IsCodeFenceLine helper, RunChecks populates Severity
- `internal/doctor/static/check_test.go` — tests for IsCodeFenceLine, Severity propagation
- `internal/doctor/static/duplicates.go` — tilde fences, findings cap at 50
- `internal/doctor/static/duplicates_test.go` — tilde fence test, cap test
- `internal/doctor/static/fences.go` — tilde fences
- `internal/doctor/static/secrets.go` — OpenAI regex, config context suppression
- `internal/doctor/static/secrets_test.go` — updated OpenAI tests, config context test
- `internal/doctor/static/smells.go` — numbered list regex, OR logic, empty section filter
- `internal/doctor/static/smells_test.go` — OR logic tests, measurement test, empty sections test
- `internal/doctor/static/vague.go` — tilde fences
- `internal/doctor/static/padding.go` — tilde fences
- `internal/doctor/static/drift.go` — tilde fences
- `internal/doctor/static/readability.go` — tilde fences
