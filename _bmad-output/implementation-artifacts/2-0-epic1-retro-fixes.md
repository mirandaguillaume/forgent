# Story 2.0: Epic 1 Retro Fixes — Tech Debt

Status: ready-for-dev

## Story

As a **developer**,
I want the frontmatter parser and OpenRouter provider hardened against known failure modes,
so that Epic 2 stories (directory analysis, multi-file processing) don't hit these known issues at scale.

## Context

Epic 1 retrospective (2026-03-30) identified two HIGH-priority action items that are prerequisites for Story 2.2 (Codebase Context Enrichment) and Story 2.3 (Directory Analysis). Neither was scoped into any Epic 2 story. This tech-debt story addresses both before Story 2.2 begins.

## Acceptance Criteria

### Fix 1: Frontmatter Parsing (Retro Action #1 — HIGH)

1. `extractFrontmatter()` in `internal/doctor/parser/claude.go` handles real-world Claude agent files with unquoted colons, special characters, and long descriptions that yaml.v3 rejects
2. Solution: either switch to yaml.v2 for frontmatter parsing, implement a custom first-level key extractor, or add a text-based fallback that reliably extracts key-value pairs
3. All 22 existing testdata files (12 Claude, 10 Copilot) parse without error
4. Copilot parser (`internal/doctor/parser/copilot.go`) also benefits if it uses the same `extractFrontmatter` path
5. No regression in existing tests (`go test ./internal/doctor/parser/... -count=1`)

### Fix 2: OpenRouter Timeout (Retro Action #2 — HIGH)

6. `OpenRouterProvider.Complete()` in `internal/llm/openrouter.go` uses `context.WithTimeout` (matching the Ollama adapter pattern at `internal/llm/ollama.go:72`)
7. Default timeout: 120 seconds (matching Ollama's `ollamaDefaultTimeout`)
8. Timeout is configurable via the provider struct (same pattern as Ollama's `timeout` field)
9. Uses `http.NewRequestWithContext` instead of `http.NewRequest` (line 65)
10. Does NOT use `http.DefaultClient` — creates a dedicated `http.Client` or uses the context-based request pattern
11. Actionable error message on timeout: "OpenRouter request timed out after 120s. Try a smaller file or check your connection."
12. Test coverage: timeout test analogous to `TestOllamaProvider_Complete_Timeout` in `ollama_test.go:66`
13. All existing OpenRouter tests pass (`go test ./internal/llm/... -count=1`)

## Tasks / Subtasks

- [ ] Task 1: Fix frontmatter parsing (AC: #1-#5)
  - [ ] 1.1: Identify which testdata files currently fail with yaml.v3 strict parsing
  - [ ] 1.2: Implement robust frontmatter extraction (custom key extractor or yaml.v2 or text fallback)
  - [ ] 1.3: Update `extractFrontmatter()` in `internal/doctor/parser/claude.go`
  - [ ] 1.4: Verify Copilot parser path if shared
  - [ ] 1.5: Run all parser tests, confirm 22/22 testdata files parse cleanly
- [ ] Task 2: Fix OpenRouter timeout (AC: #6-#13)
  - [ ] 2.1: Add `timeout` field to `OpenRouterProvider` struct
  - [ ] 2.2: Use `context.WithTimeout` + `http.NewRequestWithContext` in `Complete()`
  - [ ] 2.3: Replace `http.DefaultClient.Do(req)` with context-aware request
  - [ ] 2.4: Add actionable error message for timeout
  - [ ] 2.5: Write `TestOpenRouterProvider_Complete_Timeout` (mirror Ollama pattern)
  - [ ] 2.6: Update `registerOpenRouterProvider()` to set default timeout
  - [ ] 2.7: Run all LLM provider tests

## Dev Notes

### Frontmatter Issue Detail

The `extractFrontmatter()` function at `internal/doctor/parser/claude.go:136` does:
1. Splits on `---` delimiters
2. Passes the YAML block to `yaml.v3.Unmarshal`

yaml.v3 is strict about quoting — real Claude agent files often have unquoted colons in description fields (e.g., `description: Review code: find bugs and suggest fixes`), which yaml.v3 rejects. The retro noted a text-based fallback exists in `Detect()` but not in `Parse()`.

### OpenRouter Issue Detail

`internal/llm/openrouter.go:65` uses `http.NewRequest` (no context), and line 73 uses `http.DefaultClient` (no timeout). The Ollama adapter (`internal/llm/ollama.go:67-73`) already implements the correct pattern:

```go
timeout := p.timeout
if timeout == 0 {
    timeout = ollamaDefaultTimeout
}
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()
```

Mirror this pattern for OpenRouter.

### What NOT to Build

- Do NOT add retry on 429 rate limit — deferred to Growth (see deferred-work.md)
- Do NOT add io.LimitReader on responses — deferred to Growth
- Do NOT change the Provider interface — these are implementation fixes only

### References

- [Source: _bmad-output/implementation-artifacts/epic-1-retro-2026-03-30.md] — Action items #1 and #2
- [Source: _bmad-output/implementation-artifacts/deferred-work.md] — Items explicitly deferred to Growth
- [Source: internal/llm/ollama.go:67-73] — Reference implementation for timeout pattern

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
