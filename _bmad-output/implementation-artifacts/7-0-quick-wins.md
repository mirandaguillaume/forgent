# Story 7.0: Quick Wins — CoT Prompt + LLM Cache

Status: done

## Story

As a **developer**,
I want Doctor's LLM analysis to be more accurate and faster on repeat runs,
so that I get better recommendations and don't wait for unchanged files.

## Acceptance Criteria

1. LLM prompt requires `reasoning` field BEFORE `confidence` in the YAML example template (CoT-before-score pattern)
2. LLM responses are cached by content hash at `.forgent/cache/doctor/`
3. Cache key includes: `sha256(fileContent + promptTemplateHash + modelName + registryVersion)`
4. Cache hit returns findings instantly (< 100ms) without calling LLM
5. Cache is invalidated when: file content changes, model changes, registry version changes
6. `--fix` purges cache for the modified file
7. `--no-cache` flag bypasses cache without purging
8. Cache entries expire after 7 days (TTL)
9. Existing `llmutil/parse.go` continues to work (backward compatible — ignores unknown YAML fields)
10. All existing tests pass (`go test ./... -count=1 -short`)

## Research Backing

- G-Eval (arXiv:2303.16634): CoT evaluation improves human alignment by 10-15%
- ESLint cache pattern: content-hash + config-hash invalidation
- Promptfoo: cache LLM responses, always re-run cheap assertions

## Tasks / Subtasks

- [x] Task 1: CoT-before-score prompt (AC: #1, #9)
  - [x] 1.1: Modify `internal/doctor/analyzer/analyzer.go` → `buildPrompt()` — add `reasoning` field before `confidence` in the YAML example
  - [x] 1.2: Update example output in prompt to show reasoning → issue → score → confidence → evidence → suggestion
  - [x] 1.3: Verify `llmutil/parse.go` ignores `reasoning` field (YAML flexible parsing)
  - [x] 1.4: Update `TestBuildPrompt` to check for `reasoning` in example

- [x] Task 2: Cache implementation (AC: #2-#8)
  - [x] 2.1: Create `internal/doctor/cache.go` — `Cache` struct with `Get(key)` / `Put(key, findings)` / `Purge(fileHash)`
  - [x] 2.2: Cache key: `sha256(fileContent + promptTemplate + modelName + registryVersion)` truncated to 16 hex chars
  - [x] 2.3: Storage: `.forgent/cache/doctor/<key>.json` with `CacheEntry{ContentHash, ConfigHash, Model, RegistryVersion, Timestamp, Findings}`
  - [x] 2.4: `Get()` returns nil if: file missing, TTL expired (7 days), or JSON parse error
  - [x] 2.5: Integrate into `internal/cmd/doctor.go` — check cache before LLM call, store after
  - [x] 2.6: `--no-cache` flag on doctor command (bypasses Get, still stores)
  - [x] 2.7: `--fix` calls `Purge()` after successful rewrite
  - [x] 2.8: Tests: cache hit, cache miss (content changed), cache miss (model changed), cache miss (TTL expired), purge after fix, --no-cache bypass

## Dev Agent Record

### Implementation Plan
- Task 1: Modified prompt template in `buildPrompt()` to include `reasoning` field before `confidence` (CoT-before-score pattern). Added test assertion verifying reasoning appears before confidence.
- Task 2: Created `internal/doctor/cache.go` with Cache struct, CacheKey function, Get/Put/Purge methods. Integrated into `internal/cmd/doctor.go` with `--no-cache` flag. Added `PromptHash()` to registry for cache invalidation. Cache purge on `--fix`.

### File List
- `internal/doctor/analyzer/analyzer.go` — modified prompt template (CoT reasoning)
- `internal/doctor/analyzer/analyzer_test.go` — added reasoning assertion + strings import
- `internal/doctor/cache.go` — NEW: cache implementation
- `internal/doctor/cache_test.go` — NEW: 10 cache tests
- `internal/doctor/registry/registry.go` — added PromptHash() method + crypto/sha256 import
- `internal/cmd/doctor.go` — integrated cache, added --no-cache flag, purge on --fix

### Review Findings

- [x] [Review][Patch] F1: `PromptHash()` only hashes detection prompts — fixed: added promptVersion constant + hash ID, Citation, Finding fields
- [x] [Review][Patch] F2: `os.Getwd()` error discarded — fixed: fallback to filepath.Dir(filePath)
- [x] [Review][Patch] F4: Provider name includes "(auto-detected)" suffix — fixed: canonicalProvider() strips suffix
- [x] [Review][Patch] F6: Cache files written with 0644 — fixed: 0600
- [x] [Review][Patch] F7: `CacheKey` fields concatenated without delimiter — fixed: null-byte separators
- [x] [Review][Patch] F11: Test assertion fragile — fixed: require.NotEqual(-1) guards before assert.Less
- [x] [Review][Defer] F3: Hash truncation to 64-bit — birthday collision risk at ~4B entries. Acceptable for local CLI. — deferred, acceptable for MVP
- [x] [Review][Defer] F5: `Purge()` does O(n) scan — acceptable for local CLI, no eviction beyond TTL — deferred, optimize later
- [x] [Review][Defer] F8: No test for empty findings as valid cache hit — deferred, integration test
- [x] [Review][Defer] F9: `reasoning` field intentionally discarded by parse.go — deferred, document when reasoning is displayed
- [x] [Review][Defer] F10: Timestamp set by caller not inside Put() — deferred, minor robustness
- [x] [Review][Defer] F12: No integration test for --no-cache flag — deferred, integration test

### Change Log
- 2026-03-31: Story 7.0 implemented — CoT-before-score prompt + LLM response cache with content-hash keying

## Dev Notes

- The prompt change is backward compatible — `llmutil/parse.go` uses `yaml.Unmarshal` into a struct that only has known fields; extra fields are silently ignored
- Cache directory created on first write (same pattern as `.forgent/config.yaml`)
- No new dependencies required
- Estimated effort: ~3 hours total
