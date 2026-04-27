# Story 8.1: Scaffold Fix + Index Resolution

Status: done

## Story

As a **developer**,
I want `forgent doctor --fix` to generate an AGENTS.md index + specialized files instead of rewriting a monolithic file,
and I want Doctor to follow index references when analyzing,
so that my agent configuration is decomposed, each concern is in its own file, and the full structure is scored.

## Acceptance Criteria

1. `--fix` generates scaffold: `AGENTS.md` index + `.agents/*.md` specialized files
2. Scaffold is **deterministic** — no LLM needed. Templates populated with codebase context from scanner (language, test commands, directory structure)
3. Content from the original file is **migrated** into the appropriate specialized files (not lost)
4. Missing sections get **template content** with TODO markers for the developer to customize
5. `AGENTS.md` index contains: quick reference (language, test/build/lint commands), table of specialized files with descriptions
6. Specialized files map to Doctor's section categories:
   - `identity.md` — persona, goals, decision authority
   - `security.md` — filesystem, network, secrets
   - `testing.md` — test commands, conventions, coverage
   - `architecture.md` — layers, patterns, data flow
   - `guardrails.md` — timeouts, output limits, constraints
   - `error-handling.md` — fallbacks, recovery, edge cases
7. When Doctor analyzes an AGENTS.md that contains markdown links (`[Title](path)`), it **resolves references** and analyzes each linked file
8. Each linked file is scored individually
9. Aggregate score is the weighted average of all linked files
10. Missing linked files produce a HIGH finding
11. Cross-file consistency runs on all resolved files (existing Story 7.4 feature)
12. `--fix` on an index file scaffolds only the missing linked files (doesn't overwrite existing ones)
13. All existing tests pass + new tests

## Research Backing

- SkillsBench (arXiv:2602.12670): 2-3 focused skills > monolithic (+18.6pp)
- Vasilopoulos (arXiv:2602.20478): multi-tier context (hot/cold) reduces errors
- Chatlatanagulchai et al. (arXiv:2511.12884): 14.5% address security, 62.3% have build commands
- AGENTS.md spec (Linux Foundation): standard file location and loading

## Tasks / Subtasks

- [ ] Task 1: Scaffold generator (AC: #1-#6)
  - [ ] 1.1: Create `internal/doctor/scaffold/scaffold.go` — `Scaffold(analysis, codebaseCtx) (*ScaffoldResult, error)`
  - [ ] 1.2: `ScaffoldResult` struct: IndexContent (AGENTS.md), Files (map[string][]byte), MigratedSections (which original sections went where)
  - [ ] 1.3: Section-to-file mapping: parse original file sections, route each to the appropriate specialized file
  - [ ] 1.4: Template system: each specialized file has a base template with TODO markers
  - [ ] 1.5: Codebase enrichment: scanner provides language, test command, build command, directory structure → injected into templates
  - [ ] 1.6: AGENTS.md index generation: quick reference table + links to specialized files
  - [ ] 1.7: Tests: scaffold from minimal file, scaffold from rich file, content migration verification

- [ ] Task 2: Index resolution in Doctor (AC: #7-#10)
  - [ ] 2.1: Create `internal/doctor/index/resolve.go` — `ResolveIndex(indexContent []byte, basePath string) ([]ResolvedFile, error)`
  - [ ] 2.2: Parse markdown links: regex `\[.*?\]\((\.agents/.*?\.md)\)` or similar
  - [ ] 2.3: For each link: check file exists (os.Stat), read + parse with format detection
  - [ ] 2.4: Missing files → HIGH finding with path
  - [ ] 2.5: Return list of `ResolvedFile{Path, Analysis, Error}`
  - [ ] 2.6: Tests: resolve valid index, missing file, no links, relative paths

- [ ] Task 3: Aggregate scoring (AC: #8-#9)
  - [ ] 3.1: Score each resolved file individually via `static.RunChecks`
  - [ ] 3.2: Aggregate = sum of all files' passed / sum of all files' total
  - [ ] 3.3: Render: per-file scores + aggregate score
  - [ ] 3.4: Tests: aggregate of 3 files, one missing file

- [ ] Task 4: Wire into Doctor CLI (AC: #1, #7, #11, #12)
  - [ ] 4.1: Detect index file: if AGENTS.md contains `[...](...agents/...)` links → index mode
  - [ ] 4.2: `--fix` on non-index file → scaffold (generate AGENTS.md + .agents/)
  - [ ] 4.3: `--fix` on index file → scaffold only missing linked files
  - [ ] 4.4: Regular analysis on index file → resolve + analyze all linked files + cross-file
  - [ ] 4.5: SafeWrite for each generated file (backup originals)
  - [ ] 4.6: Tests: CLI integration for scaffold and index modes

## Dev Notes

- Scaffold is deterministic — NO LLM call needed. This is the key insight: `--fix` becomes instant.
- The scanner (`internal/scanner/`) already provides CodebaseContext with languages, structure, deps. Reuse it.
- Section migration heuristic: match original section headers against the 6 specialized file categories using the same keywords as `presence.go` sectionMapping.
- Template files could be embedded via `go:embed` or generated in code. Start with code generation (simpler), move to embedded templates later.
- Index resolution is a new concept — Doctor currently treats each file independently. This adds the ability to follow references.
- Cross-file consistency (Story 7.4) already handles multiple files with LLM. The index resolver feeds into the same pipeline.
- The `--fix` behavior changes completely: old LLM rewrite is removed. `--fix` = deterministic scaffold only. No `--rewrite` fallback. The rewrite module (`internal/doctor/rewrite/`) becomes dead code — remove or repurpose for future use.
- Estimated effort: ~8-10 hours
