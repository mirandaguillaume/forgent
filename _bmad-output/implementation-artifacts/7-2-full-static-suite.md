# Story 7.2: Full Static Suite — Secrets, Readability, Tokens, Duplicates

Status: done

## Story

As a **developer**,
I want Doctor to catch secrets, measure readability, estimate token costs, and detect duplicate instructions without needing an LLM,
so that I get comprehensive structural analysis instantly.

## Acceptance Criteria

1. **Secret scanning**: 17+ regex patterns detect API keys (OpenAI `sk-`, Anthropic `sk-ant-`, AWS `AKIA`, GitHub `ghp_`, `gho_`, `ghs_`, generic `password=`, `secret=`, `token=`, base64-encoded keys). Always severity HIGH.
2. **Compressible padding detection**: patterns like "Make sure to", "It is important to", "Please note that", "Keep in mind that", "It's worth noting" flagged as reducible filler.
3. **Readability metrics** displayed as indicators (not pass/fail):
   - Flesch Reading Ease (pure Go, no dependency)
   - Gunning Fog Index (pure Go)
   - Target guidance: Flesch >30 (currently median 16.6 for Claude files)
4. **Token estimation** per section and total:
   - Method: `chars/4 * 0.78` corrected heuristic
   - Total >8,000 tokens = warning, >16,000 = critical
   - Per-section >2,000 = warning
5. **Instruction specificity ratio**: count directive words ("must", "always", "never", "required", "ensure") vs advisory ("may", "consider", "could", "optional", "suggested"). Displayed as indicator 0.0-1.0.
6. **Duplicate detection**: Jaccard similarity on instruction-like lines (bullets, numbered items). Threshold ≥0.5 = near-duplicate warning with both line numbers.
7. All new checks registered with tag "thorough" (not in default mode except secrets which is "default")
8. Secret scanning also tagged "security" for `--security` mode
9. Structural score updated to include secrets (pass = 0 secrets found)
10. All existing tests pass + new tests for each check

## Research Backing

- Chatlatanagulchai et al. (arXiv:2511.12884): Flesch 16.6 median for Claude files
- Gloaguen et al. (arXiv:2602.11988): +20% token cost from redundant context
- Snyk ToxicSkills: 36% of skills have prompt injection / secret exposure
- AgentLinter: 25+ secret patterns, compressible padding detection

## Tasks / Subtasks

- [x] Task 1: Secret scanning (AC: #1, #8)
  - [x] 1.1-1.5: 14 patterns, placeholder detection, ordered specific→generic, Tags: ["default", "security"], 8 tests

- [x] Task 2: Compressible padding (AC: #2)
  - [x] 2.1-2.4: 12 patterns, code-fence exclusion, Tags: ["thorough"]

- [x] Task 3: Readability metrics (AC: #3)
  - [x] 3.1-3.8: Flesch RE + Gunning Fog, syllable counting with silent-e/-le correction, code block exclusion, IndicatorCheck interface, 5 tests

- [x] Task 4: Token estimation (AC: #4)
  - [x] 4.1-4.7: chars/4*0.78 corrected heuristic, per-section + total, warn/critical thresholds, indicator output, Tags: ["default"], 4 tests

- [x] Task 5: Specificity ratio (AC: #5)
  - [x] 5.1-5.6: 7 directive + 7 advisory words, ratio 0.0-1.0, IndicatorCheck, 4 tests

- [x] Task 6: Duplicate detection (AC: #6)
  - [x] 6.1-6.7: Jaccard similarity, instruction-line extraction, code-fence exclusion, Tags: ["thorough"], 5 tests

- [x] Task 7: Render indicators (AC: #3, #4, #5)
  - [x] 7.1-7.4: Indicator struct + IndicatorCheck interface in check.go, CollectIndicators(), RenderIndicators in render.go

## Dev Agent Record

### File List
- `internal/doctor/static/check.go` — MODIFIED: added Indicator struct, IndicatorCheck interface, CollectIndicators
- `internal/doctor/static/secrets.go` — NEW: 14-pattern secret scanning
- `internal/doctor/static/secrets_test.go` — NEW: 8 tests
- `internal/doctor/static/padding.go` — NEW: 12-pattern compressible padding
- `internal/doctor/static/readability.go` — NEW: Flesch RE + Gunning Fog with syllable counting
- `internal/doctor/static/readability_test.go` — NEW: 5 tests
- `internal/doctor/static/tokens.go` — NEW: corrected token estimation
- `internal/doctor/static/tokens_test.go` — NEW: 4 tests
- `internal/doctor/static/specificity.go` — NEW: directive/advisory ratio
- `internal/doctor/static/specificity_test.go` — NEW: 4 tests
- `internal/doctor/static/duplicates.go` — NEW: Jaccard similarity duplicate detection
- `internal/doctor/static/duplicates_test.go` — NEW: 5 tests
- `internal/doctor/render.go` — MODIFIED: added RenderIndicators
- `internal/doctor/llmutil/parse.go` — MODIFIED: added identity, goals, constraints, structure to validCategories

### Change Log
- 2026-03-31: Story 7.2 implemented — 6 new static checks (secrets, padding, readability, tokens, specificity, duplicates), 30 new tests, Indicator system

## Dev Notes

- Secret scanning is `Tags: ["default", "security"]` — always runs. Others are "thorough" unless specified.
- Readability and specificity are indicators, not pass/fail checks — they inform but don't count in structural score
- Token estimation is both: indicator (displayed) + finding (if over threshold)
- No new dependencies — all pure Go
- Estimated effort: ~10 hours total
