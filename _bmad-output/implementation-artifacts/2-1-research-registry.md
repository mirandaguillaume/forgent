# Story 2.1: Research Registry — Embedded with Local Override

Status: done

## Story

As a **developer**,
I want every recommendation to cite a specific published paper,
so that I can trust the suggestions are grounded in evidence, not opinions.

## Acceptance Criteria

1. Research registry YAML file embedded via `//go:embed` in `internal/doctor/registry/`
2. Registry provides 12 entries (matching current 12 categories), each with: id, title, citation, paper, url, finding, confidence, detection_prompt, suggestion_prompt
3. Local override file at `.forgent/research-registry.yaml` takes precedence over embedded default
4. Registry version tracking — `--debug` reports which version is active
5. Analyzer prompt is GENERATED from the registry, not hardcoded
6. Each finding's `citation_id` resolves to a registry entry with full citation
7. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Create registry package and data (AC: #1, #2)
  - [ ] 1.1: Create `internal/doctor/registry/registry.go` — Registry struct, Load(), Get(), All() functions
  - [ ] 1.2: Create `internal/doctor/registry/data/research-registry.yaml` — 12 entries with full research citations
  - [ ] 1.3: Embed via `//go:embed data/research-registry.yaml`
  - [ ] 1.4: Define `Recommendation` struct: id, title, citation, paper, url, finding, confidence, detection_prompt, suggestion_prompt
  - [ ] 1.5: Write tests for Load(), Get(), All()
- [ ] Task 2: Local override support (AC: #3, #4)
  - [ ] 2.1: On Load(), check for `.forgent/research-registry.yaml` in current dir and parent dirs
  - [ ] 2.2: If found, parse and use instead of embedded default
  - [ ] 2.3: Version field in registry YAML — report in debug mode
  - [ ] 2.4: Write tests for override precedence
- [ ] Task 3: Wire registry into analyzer (AC: #5, #6)
  - [ ] 3.1: Refactor `buildPrompt()` in analyzer to generate categories from registry entries
  - [ ] 3.2: Include `detection_prompt` from each registry entry in the analysis prompt
  - [ ] 3.3: After analysis, resolve `citation_id` in findings against registry entries
  - [ ] 3.4: Enrich finding output with full citation (paper name, URL)
  - [ ] 3.5: Update doctor render to show citations
  - [ ] 3.6: Write tests with mock registry

## Dev Notes

### Registry YAML Format (from ADR-3)

```yaml
version: "2026.03"
recommendations:
  - id: guardrails
    title: "Add guardrails section"
    citation: "Liu et al., 2024"
    paper: "Lost in the Middle: How Language Models Use Long Contexts"
    url: "https://arxiv.org/abs/2307.03172"
    finding: "Critical instructions at prompt start improve adherence"
    confidence: high
    detection_prompt: "Are there explicit behavioral constraints (timeouts, output limits, prohibitions)?"
    suggestion_prompt: "Add a guardrails section at the top with timeout, output limits, and behavioral boundaries."
```

### The 12 Entries to Create

| id | citation | paper |
|---|---|---|
| guardrails | Liu et al., 2024 | Lost in the Middle |
| security | Agent READMEs, 2025 | Agent READMEs Empirical Study |
| ordering | Liu et al., 2024 | Lost in the Middle (primacy bias) |
| decomposition | SkillsBench, 2026 | SkillsBench: Benchmarking Skills |
| context | CodeScout, 2026 | CodeScout: Pre-exploration via Knowledge Graph |
| redundancy | ETH Zurich, 2026 | Evaluating AGENTS.md |
| tool_usage | Agent READMEs, 2025 | Agent READMEs Empirical Study |
| specificity | GitHub, 2026 | Lessons from 2,500 repos |
| testing | SkillsBench, 2026 | SkillsBench (test-aware agents) |
| examples | Anthropic, 2025 | Demystifying evals for AI agents |
| error_handling | Confucius, 2025 | Confucius Code Agent |
| scope | SkillsBench, 2026 | SkillsBench (2-3 skills optimal) |

### Prompt Generation from Registry

Replace the hardcoded prompt in `analyzer.go` with:

```go
func buildPrompt(analysis *parser.AgentAnalysis, reg *registry.Registry) string {
    // ... file context ...

    entries := reg.All()
    b.WriteString(fmt.Sprintf("TASK: Analyze this agent file for quality issues across %d categories.\n\n", len(entries)))
    b.WriteString("CATEGORIES:\n")
    for i, e := range entries {
        b.WriteString(fmt.Sprintf("%d. %s — %s (%s: %s)\n", i+1, e.ID, e.DetectionPrompt, e.Citation, e.Finding))
    }
    // ... example + instructions ...
}
```

### Previous Story Intelligence (from Epic 1 retro)

- **Action #5:** No undocumented scope creep — the 12 categories are already in the code, the registry formalizes them
- **Action #1:** Frontmatter parsing fix — not this story's scope but should be done alongside or before
- Code reviews found real bugs every time — review this story before marking done

### What NOT to Build

- Do NOT implement `--update-registry` download command — that's Story 2.3 (advanced features)
- Do NOT implement registry checksum verification — defer to Growth
- DO implement the full read path: embed → load → override → generate prompt → resolve citations

### Project Structure

New files:
- `internal/doctor/registry/registry.go` — Registry struct + Load/Get/All
- `internal/doctor/registry/registry_test.go` — tests
- `internal/doctor/registry/data/research-registry.yaml` — the 12 entries

Modified files:
- `internal/doctor/analyzer/analyzer.go` — refactor buildPrompt to use registry
- `internal/doctor/render.go` — add citation display
- `internal/cmd/doctor.go` — load registry at startup, pass to analyzer

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-3] — Research registry embedded with local override
- [Source: _bmad-output/planning-artifacts/prd.md#FR15-FR18] — Versioned research registry
- [Source: _bmad-output/implementation-artifacts/epic-1-retro-2026-03-30.md] — Action items for Epic 2
- [Source: docs/plans/2026-03-11-research-agent-context-effectiveness.md] — Full research review with all citations
- [Source: content/whitepaper/_index.md#References] — Paper URLs and citations

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
