# Sprint Review — Forgent Doctor MVP

**Date:** 2026-03-30
**Reviewer:** Bob (Scrum Master)
**Sprint:** Sprints 1-2 (Epics 1, 2, 3)

## Executive Summary

3 of 6 epics complete. 14 stories shipped (plus 1 tech-debt story). The `forgent doctor` command works end-to-end: analyzes Claude, Copilot, and Forgent agent files with local LLM (Ollama) or cloud API (OpenRouter), produces research-backed recommendations with citations, enriches with codebase context, supports directory analysis with aggregate reports, and can rewrite files interactively with `--fix` backed by quality gates and atomic writes.

## Epic Status

| Epic | Status | Stories | Retro |
|---|---|---|---|
| 1: Core Doctor Analysis | done | 6/6 | done |
| 2: Enriched Research-Backed Analysis | done | 5/5 | done |
| 3: Safe Agent Rewrite | done | 3/3 | pending |
| 4: Doctor as Forgent Runtime Agent | backlog | 0/3 | - |
| 5: Benchmark & Evidence | backlog | 0/3 | - |
| 6: Ecosystem & DX Polish | backlog | 0/3 | - |

**Total stories completed:** 14 (6 + 5 + 3), plus Story 2.0 tech-debt = 15 story files.
**All story files for Epics 4-6 are `ready-for-dev`.**

## Key Deliverables

### What Works (The Demo)

1. **`forgent doctor agent.md`** — Analyzes a single agent file:
   - Auto-detects format (Claude, Copilot, Forgent YAML)
   - Sends to Ollama (local) or OpenRouter (cloud) for LLM analysis
   - Returns 5-12 research-backed recommendations with confidence levels
   - Each recommendation cites a specific paper (12 entries in research registry)
   - Enriches with codebase context (test coverage gaps, structure awareness, stack detection)
   - TTY-aware output (colored in terminal, plain in CI/pipe)

2. **`forgent doctor .claude/agents/`** — Directory analysis:
   - Discovers all agent files recursively via parser registry
   - Analyzes each file sequentially
   - Produces aggregate summary (total findings, by-category, files without guardrails/security)
   - Text-instruction-limit diagnosis (ordering reliability warning)

3. **`forgent doctor agent.md --fix`** — Interactive rewrite:
   - Shows each recommendation's diff
   - Developer selects which to apply (1,3,5 format)
   - Quality gate validates merged rewrite (4 checks + diff-size sanity)
   - Atomic write via SafeWrite (backup, temp file, rename)
   - Backup/restore via `--restore`
   - `--force` bypasses diff-size check

4. **`forgent doctor --update-registry`** — Registry management:
   - Version notification when newer registry available
   - Downloads with checksum verification
   - URL displayed before download (NFR22)

5. **Research Registry** — 12 entries with full citations:
   - Embedded via `//go:embed`, local override at `.forgent/research-registry.yaml`
   - Analyzer prompt generated from registry (not hardcoded)
   - Version tracking in `--debug` mode

### Existing Capabilities (Pre-Doctor)

- `forgent init`, `forgent skill create`, `forgent lint`, `forgent score`
- `forgent build --target claude|copilot|forgent` (multi-target compilation)
- `forgent build --watch` (file watching)
- `forgent import` (agent .md to skill YAML)
- `forgent doctor` (old diagnostics: lint + dependency + loop analysis)
- DAG engine (`pkg/dag/`) with auto-wiring, layers, router nodes, retry
- Formal property analysis (`pkg/analysis/`)

## Test Suite

- **Total tests:** 1,284
- **Packages tested:** 33 (28 with tests, 5 without)
- **All passing:** `go test ./... -count=1 -short` green
- **Binary compiles:** `go build ./cmd/forgent` clean

### Test growth across epics:

| Milestone | Tests |
|---|---|
| Pre-doctor (existing) | ~200 (estimated from pkg/ + existing internal/) |
| After Epic 1 | ~260 (+60 doctor tests) |
| After Epic 2 | ~480 (+220 doctor/registry/context/directory tests) |
| After Epic 3 | 1,284 (+800 fileops/validate/rewrite tests) |

## Functional Requirements Coverage

| FR Range | Description | Status |
|---|---|---|
| FR1-FR7 | Core doctor (3 formats, auto-detect, Ollama, API, TTY) | done (Epic 1) |
| FR8-FR9 | Directory analysis, codebase context | done (Epic 2) |
| FR10-FR14 | Static fallback, model detection, recommendations, confidence | done (Epic 1) |
| FR15-FR18 | Research registry, version notification, update | done (Epic 2) |
| FR19-FR25 | Safe rewrite (read-only, diff, apply, backup, restore, format preservation) | done (Epic 3) |
| FR26-FR42 | Skill spec, composition, compilation (existing) | done (pre-doctor) |
| FR44-FR50 | Doctor as runtime agent (DAG, contracts, router, retry, KEYSTONE) | backlog (Epic 4) |
| FR51-FR53 | Benchmark, experiment, fidelity tests | backlog (Epic 5) |
| FR54-FR58 | Targets, export, config, shell completion | backlog (Epic 6) |

**FR coverage:** 40 of 58 FRs delivered (69%). Remaining 18 are in Epics 4-6.

## New Packages Created (Epics 1-3)

| Package | Epic | Purpose |
|---|---|---|
| `internal/doctor/parser/` | 1 | Format detection, parsing, registry (Claude, Copilot, Forgent) |
| `internal/doctor/analyzer/` | 1 | LLM analysis skill |
| `internal/doctor/llmutil/` | 1 | YAML extraction, finding parsing |
| `internal/doctor/spike/` | 1 | Engineering spike |
| `internal/llm/` (ollama) | 1 | Ollama provider adapter |
| `internal/doctor/registry/` | 2 | Research registry (embed, load, override, update) |
| `internal/doctor/context/` | 2 | Codebase context enrichment |
| `internal/doctor/` (directory, aggregate) | 2 | Directory analysis, aggregate reporting |
| `internal/fileops/` | 3 | SafeWrite, Restore, Lock, Signal, Cleanup |
| `internal/doctor/validate/` | 3 | Quality gate (4-check + diff-size) |
| `internal/doctor/rewrite/` | 3 | Rewriter, diff generation, interactive selection |

## What's Left

### Epic 4: Doctor as Forgent Runtime Agent (KEYSTONE)

Doctor reimplemented as a Forgent agent — 5 skill YAMLs, 1 agent YAML, embedded via `//go:embed`, DAG constructed at runtime. Every `forgent doctor` run exercises the full Forgent stack.

- 4.1: Doctor skill and agent YAML definitions
- 4.2: Embedded YAML and runtime DAG construction
- 4.3: Runtime error handling and diagnostics

**Risk:** Equivalence testing (direct Go doctor vs DAG-based doctor must produce identical output).

### Epic 5: Benchmark & Evidence

- 5.1: InstructionBench (100-file corpus, quality measurement)
- 5.2: Runtime-vs-text controlled experiment
- 5.3: Format fidelity test suite

**Risk:** Corpus licensing, reproducibility requirements.

### Epic 6: Ecosystem & DX Polish

- 6.1: Doctor configuration (config file, env vars, flag precedence)
- 6.2: Target listing and `--export-yaml`
- 6.3: Shell completion (bash, zsh, fish, powershell)

**Risk:** Low — mostly CLI polish.

## Process Health

### What's Working

- Story scoping with "What NOT to Build" sections — prevented scope creep
- Tech-debt stories as gates — Epic 1 retro fixes done before Epic 2 features
- Code reviews finding real bugs when they run
- Test-first approach validated (frontmatter fix, quality gate)
- 1,284 tests all green, binary compiles clean

### What Needs Improvement

- **Review gate not enforced** — Story 2.4 and some Epic 3 stories shipped without 3-layer review (process violation repeated across epics). The `review` vs `done` status distinction exists but is not structurally enforced.
- **Retros skipped under velocity pressure** — Epic 3 started before Epic 2 retro was complete.
- **MEDIUM items slip** — `.bak` file cleanup and size guard on prompts have been carried for 2 epics without resolution.
- **Dev Agent Records empty** — Story completion metadata not maintained, making retros data-poor.

## Technical Debt Summary

| Debt | Severity | Carried Since |
|---|---|---|
| No size guard on RawContent in prompt | Medium | Epic 1 |
| `parseSections` naive code fence tracking | Medium | Epic 1 |
| `.bak` files not deleted | Low | Epic 1 |
| No `io.LimitReader` on API responses | Low | Epic 1 |
| No retry on 429 rate limit | Low | Epic 1 |
| Prompt injection via agent content | Low | Epic 1 |
