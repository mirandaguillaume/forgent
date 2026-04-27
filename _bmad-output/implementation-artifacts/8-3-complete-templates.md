# Story 8.3: Complete Templates + 100% Scaffold Score

Status: done

## Story

As a **developer**,
I want the scaffold templates to cover all 15 section categories,
so that a freshly scaffolded project starts at 100% structural score.

## Acceptance Criteria

1. Templates cover the 4 missing categories: examples, goals, memory_management, workflow_triggers
2. `identity.md` template includes goals section
3. `testing.md` template includes examples section
4. New `memory.md` or section in identity.md covers memory_management
5. New `triggers.md` or section in identity.md covers workflow_triggers
6. After `--fix` scaffold + `--quick` aggregate scoring = 15/15 (100%)
7. Codebase context injected where available (scanner)
8. All existing tests pass + updated tests

## Tasks / Subtasks

- [ ] Task 1: Add missing sections to templates (AC: #1-#5)
  - [ ] 1.1: Add `## Goals` to identity.md template
  - [ ] 1.2: Add `## Examples` to testing.md template
  - [ ] 1.3: Add `## Memory Strategy` to identity.md template (or new file)
  - [ ] 1.4: Add `## When to Invoke` to identity.md template (workflow triggers)
  - [ ] 1.5: Update DefaultFiles categories mapping

- [ ] Task 2: Verify 100% score (AC: #6)
  - [ ] 2.1: Scaffold a test file, run index-aware scoring (Story 8.2), verify 15/15
  - [ ] 2.2: Tests: end-to-end scaffold → score = 100%

## Dev Notes

- Depends on Story 8.2 for index-aware scoring
- Keep templates concise — TODO markers for customization, not bloat
- Estimated effort: ~2 hours
