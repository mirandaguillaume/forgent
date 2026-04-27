# Story 2.2: Codebase Context Enrichment

Status: ready-for-dev

## Story

As a **developer**,
I want doctor to understand my project when analyzing agent files,
so that recommendations include context-specific suggestions like "your project has 47 test files not referenced by the agent."

## Acceptance Criteria

1. When `forgent doctor agent.md` runs inside a project directory, the context-enricher skill scans the codebase using `internal/scanner/ScanCodebase()` and produces context-specific recommendations, implementing at least the 5 gap analysis categories: (a) test file coverage, (b) codebase structure awareness, (c) key symbol references, (d) security context match, (e) stack/language awareness
2. Context recommendations identify concrete gaps per category — e.g., "Project has 47 test files but agent has no testing section", "Agent doesn't reference any of the 12 project directories", "Project is 95% Go but agent doesn't mention Go conventions"
3. When `forgent doctor ~/Downloads/agent.md` runs outside any project, the context-enricher produces empty results gracefully — no crash, no error — and the recommendation-builder proceeds with analysis-only output
4. Context enrichment completes within 15 seconds for projects with ≤ 50,000 LOC (NFR4)
5. Context findings are merged with LLM analysis findings before rendering — the user sees a unified list
6. Each context recommendation includes a confidence level and a descriptive issue/suggestion
7. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Create context-enricher package (AC: #1, #2)
  - [ ] 1.1: Create `internal/doctor/context/enricher.go` — `Enrich(analysis *parser.AgentAnalysis, root string) ([]llmutil.Finding, error)` function
  - [ ] 1.2: Implement project root detection — walk upward from agent file's directory looking for `.git/`, `go.mod`, `package.json`, or `Makefile`
  - [ ] 1.3: Call `scanner.ScanCodebase(root)` to collect `CodebaseContext` (structure, stack, symbols)
  - [ ] 1.4: Implement gap analysis — compare agent file content against codebase context:
    - Test file coverage: count `_test.go`/`*.test.*` files, check if agent mentions testing
    - Codebase structure: check if agent references project directories
    - Key symbols: check if agent references key types/interfaces
    - Security context: check if agent has filesystem/network restrictions matching project structure
    - Stack awareness: check if agent mentions the project's language/framework
  - [ ] 1.5: Return findings as `[]llmutil.Finding` with category `"context"`, confidence levels, and specific suggestions
  - [ ] 1.6: Write unit tests with a temp directory fixture containing a minimal Go project structure

- [ ] Task 2: Graceful degradation (AC: #3)
  - [ ] 2.1: If project root detection finds no markers, return empty `[]llmutil.Finding` and `nil` error
  - [ ] 2.2: If `ScanCodebase()` fails (permission error, etc.), log to stderr in debug mode and return empty findings
  - [ ] 2.3: Write tests for: no project root, permission denied, empty directory

- [ ] Task 3: Wire into doctor CLI (AC: #5, #6)
  - [ ] 3.1: In `internal/cmd/doctor.go`, after LLM analysis, call `context.Enrich()` to get context findings
  - [ ] 3.2: Merge context findings with LLM findings into a single slice before rendering
  - [ ] 3.3: Context findings appear after LLM findings, clearly attributable (category = "context")
  - [ ] 3.4: In debug mode, log: project root path, scan duration, number of context findings
  - [ ] 3.5: Write integration test verifying the merge behavior with mock LLM + real scanner

- [ ] Task 4: Performance validation (AC: #4)
  - [ ] 4.1: Add a benchmark test (`BenchmarkEnrich`) scanning the forgent codebase itself
  - [ ] 4.2: Verify scan + enrichment < 15s on the forgent codebase (~10K LOC)
  - [ ] 4.3: If scanner is too slow, add a file count limit (default 5000 files) with early termination

## Dev Notes

### Architecture References

- **ADR-4 (Doctor DAG Topology):** Context-enricher is a CODE skill in Layer 1, running in parallel with the LLM analyzer. For this story, we wire it directly in the CLI command. Epic 4 (Story 4.2) will move it into the DAG.
- **ADR-4 note on context-enricher:** "Reuses `internal/scanner/` for codebase data collection (TF-IDF, entropy, AST symbols) but has its own recommendation logic that transforms `CodebaseContext` into actionable recommendations."
- **Architecture enforcement rule #9:** Use the `Provider` interface for all LLM calls. Context-enricher is a CODE skill — no LLM calls. It uses `scanner.ScanCodebase()` directly.

### Existing Code to Reuse

- **`internal/scanner/ScanCodebase(root string) (*CodebaseContext, error)`** — collects structure, stack info, symbols. Already tested, race-clean.
- **`internal/scanner/types.go`** — `CodebaseContext`, `SymbolEntry`, `DirEntry`, `StackInfo`, `LangInfo`, `DepInfo`, `CommandInfo`
- **`internal/enricher/`** — existing enricher renders codebase context for compiler output. The doctor context-enricher has DIFFERENT logic: it compares agent content against codebase context to find gaps, rather than rendering context for embedding.
- **`internal/doctor/llmutil.Finding`** — reuse this struct for context findings (same fields: category, issue, confidence, current_state, suggested_improvement)
- **`internal/doctor/parser/types.go`** — `AgentAnalysis` (input to enricher — we compare `RawContent`, `Sections`, `Tools` against codebase)

### Previous Story Intelligence (from Epic 1 retro)

- **Action #4:** "Always review before done — 3-layer review is non-negotiable." Run adversarial code review before marking done.
- **Action #5:** "No undocumented scope creep." The gap analysis categories (test coverage, structure, symbols, security, stack) are defined here — do not add more without an ADR.
- **Action #6:** "Size guard on prompts." Not directly applicable (no LLM call), but scanner may return large `CodebaseContext` — the enricher should handle large projects gracefully.
- **Recurring finding:** "Tests dependent on relative paths" — use `t.TempDir()` for all file-system tests.
- **Recurring finding:** "CRLF not handled" — when scanning agent file `RawContent` for keyword matches, normalize line endings.

### Project Root Detection Logic

```
func detectProjectRoot(startDir string) string {
    markers := []string{".git", "go.mod", "package.json", "Makefile", "Cargo.toml", "pyproject.toml"}
    dir := startDir
    for {
        for _, marker := range markers {
            if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
                return dir
            }
        }
        parent := filepath.Dir(dir)
        if parent == dir {
            return "" // reached filesystem root
        }
        dir = parent
    }
}
```

### Gap Analysis Categories

| Category | What to Check | Confidence | Example Finding |
|---|---|---|---|
| Test coverage | Count test files, check if agent mentions testing | high | "Project has 47 test files but agent has no testing section" |
| Structure awareness | Check if agent references any project directories | moderate | "Agent doesn't reference any of the 12 project directories" |
| Key symbols | Check if agent references key types/interfaces from codebase | low | "Codebase has 8 exported interfaces — agent doesn't reference any" |
| Security match | Check if agent has fs/network restrictions appropriate for stack | moderate | "Project uses network calls but agent has no network restrictions" |
| Stack awareness | Check if agent mentions project's language/framework | high | "Project is 95% Go but agent doesn't mention Go conventions" |

### What NOT to Build

- Do NOT implement the recommendation-builder package (`internal/doctor/recommend/`) — that's an Epic 4 concern when doctor becomes a DAG. For now, merge findings in the CLI command.
- Do NOT call any LLM from the context-enricher — this is a CODE skill (ADR-4).
- Do NOT modify the existing `internal/enricher/` package — it serves the compiler, not doctor.
- Do NOT add directory analysis (walking multiple files) — that's Story 2.3.
- Do NOT implement codebase_index embedding — that's the compiler enricher's job, not doctor's.

### Project Structure

New files:
- `internal/doctor/context/enricher.go` — Enrich() function, project root detection, gap analysis
- `internal/doctor/context/enricher_test.go` — unit tests with temp dir fixtures

Modified files:
- `internal/cmd/doctor.go` — add context enrichment call after LLM analysis, merge findings

## Testing Strategy

- **Unit tests:** Table-driven tests for each gap analysis category. Use `t.TempDir()` to create minimal project fixtures (go.mod, _test.go files, directory structure).
- **Graceful degradation tests:** No project root → empty findings. Scanner error → empty findings + debug log. Empty project → empty findings.
- **Integration test:** Full doctor flow with mock LLM + real scanner on forgent's own codebase. Verify merged output contains both LLM and context findings.
- **Benchmark:** `BenchmarkEnrich` on forgent codebase — must complete under 15s.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-4] — Context-enricher as CODE skill in Layer 1, parallel with analyzer
- [Source: _bmad-output/planning-artifacts/architecture.md#Implementation-Notes] — Context-enricher graceful degradation
- [Source: _bmad-output/planning-artifacts/prd.md#FR9] — Doctor can enrich recommendations with codebase context
- [Source: _bmad-output/planning-artifacts/prd.md#NFR4] — Enrichment pipeline completes within 15 seconds for ≤ 50,000 LOC
- [Source: _bmad-output/implementation-artifacts/epic-1-retro-2026-03-30.md] — Action items for Epic 2
- [Source: internal/scanner/scan.go] — ScanCodebase() function signature and types
- [Source: internal/scanner/types.go] — CodebaseContext, SymbolEntry, DirEntry structs
- [Source: internal/enricher/enricher.go] — Existing enricher (compiler, not doctor — different purpose)

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
