# Story 7.5: Agent Smell Detection

Status: done

## Story

As a **developer**,
I want Doctor to detect anti-patterns in my agent files,
so that I avoid common mistakes that reduce agent effectiveness.

## Acceptance Criteria

1. **God agent**: file >3000 words AND >8 distinct section headings (H2+) → "This agent has too many responsibilities. Consider decomposing into 2-3 focused skills."
2. **Copy-paste agent**: Jaccard similarity >80% against known default templates (Claude default CLAUDE.md, Copilot default copilot-instructions.md) → "This file is nearly identical to the default template. Customize it for your project."
3. **LLM-generated agent**: heuristic detection of LLM writing patterns (>3 "Additionally"/"Furthermore"/"Moreover" transitions, >5 numbered lists, consistently formatted bullet points with semicolons) → "This file appears LLM-generated. Research shows LLM-generated context files have negative impact (-3%, Gloaguen et al.)."
4. **Checkbox agent**: all 8+ major sections present BUT average section length <20 words → "All sections present but too shallow. Add substance to each section."
5. **Over-constrained agent**: >10 prohibitions ("never", "don't", "must not", "do not", "avoid") with <3 positive permissions ("should", "can", "allowed to") → "This agent is mostly prohibitions. Add positive guidance about what to do, not just what not to do."
6. All smell checks tagged "thorough" (not default mode)
7. Smells rendered as a separate "Anti-patterns" section in output
8. All existing tests pass + tests for each smell detector

## Research Backing

- Gloaguen et al. (arXiv:2602.11988): LLM-generated context files = -3% performance
- SkillsBench (arXiv:2602.12670): 4+ skills = +5.2pp vs 2-3 = +20.0pp (god agent antipattern)
- Party mode innovation: "agent smells" concept — no existing tool does this

## Tasks / Subtasks

- [x] Task 1: God agent detection (AC: #1)
  - [x] 1.1-1.5: detectGodAgent — >3000 words AND >8 H2+ sections. Tags: ["thorough"]. 2 tests.

- [x] Task 2: Copy-paste detection (AC: #2)
  - [x] 2.1-2.5: detectCopyPaste — Jaccard >80% vs embedded template word sets. Tags: ["thorough"].

- [x] Task 3: LLM-generated detection (AC: #3)
  - [x] 3.1-3.5: detectLLMGenerated — >3 transition phrases AND >5 numbered items. Tags: ["thorough"]. 2 tests.

- [x] Task 4: Checkbox agent detection (AC: #4)
  - [x] 4.1-4.5: detectCheckboxAgent — >8 sections AND avg <20 words. Tags: ["thorough"]. 2 tests.

- [x] Task 5: Over-constrained detection (AC: #5)
  - [x] 5.1-5.5: detectOverConstrained — >10 prohibitions AND <3 permissions. Tags: ["thorough"]. 2 tests.

- [x] Task 6: Render anti-patterns section (AC: #7)
  - [x] 6.1-6.4: RenderAntiPatterns in render.go — filters agent_smell category, separate section, icon + suggestion

## Dev Agent Record

### File List
- `internal/doctor/static/smells.go` — NEW: 5 agent smell detectors
- `internal/doctor/static/smells_test.go` — NEW: 11 tests
- `internal/doctor/render.go` — MODIFIED: added RenderAntiPatterns
- `internal/doctor/llmutil/parse.go` — MODIFIED: added agent_smell to validCategories

### Change Log
- 2026-03-31: Story 7.5 implemented — 5 agent smells (God agent, Copy-paste, LLM-generated, Checkbox, Over-constrained), 11 tests, Anti-patterns rendering

## Dev Notes

- All 5 smells are purely static — no LLM needed
- Tagged "thorough" so they don't clutter default output
- The LLM-generated detection is heuristic and will have false positives — that's acceptable for a "thorough" mode warning
- Default templates for copy-paste detection need to be scraped from Claude/Copilot docs and embedded
- Estimated effort: ~6 hours total
