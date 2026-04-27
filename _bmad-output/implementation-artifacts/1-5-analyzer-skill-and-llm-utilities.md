# Story 1.5: Analyzer Skill and LLM Utilities

Status: done

## Story

As a **developer**,
I want doctor to analyze my agent file and identify specific issues,
so that I know what to improve with research-backed evidence.

## Acceptance Criteria

1. Analyzer skill takes an `AgentAnalysis` from any format parser and produces structured findings
2. Analyzer sends a structured prompt to the LLM `Provider` interface and receives YAML-formatted output
3. `llmutil.ExtractYAML()` strips markdown code fences and returns clean YAML (handles: ```yaml, ```yml, ```, no fences)
4. `llmutil.ParseFindings()` parses YAML into typed `Finding` structs with: category, issue, confidence, citation_id, current_state, suggested_improvement
5. Each finding includes a confidence level (high/moderate/low)
6. At least 5 recommendation categories supported: guardrails, security, ordering, decomposition, context
7. Analyzer works with any `Provider` (Ollama, OpenRouter, Anthropic)
8. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Create llmutil package — shared LLM output utilities (AC: #3, #4)
  - [ ] 1.1: Create `internal/doctor/llmutil/extract.go` — move `ExtractYAML()` from spike, production-quality version
  - [ ] 1.2: Create `internal/doctor/llmutil/parse.go` — move `ParseFindings()` from spike, production-quality version
  - [ ] 1.3: Define `Finding` struct in llmutil (same schema as ADR-4 analyzer output)
  - [ ] 1.4: Handle edge case: LLM returns valid YAML with wrong key (e.g., `results:` instead of `findings:`) — return specific error, not nil
  - [ ] 1.5: Handle edge case: multiple code fences — extract all and concatenate
  - [ ] 1.6: Handle edge case: empty LLM response — return specific error
  - [ ] 1.7: Write comprehensive tests — port spike tests + add edge cases
- [ ] Task 2: Build analyzer skill (AC: #1, #2, #5, #6)
  - [ ] 2.1: Create `internal/doctor/analyzer/analyzer.go` with `Analyze(analysis *parser.AgentAnalysis, provider llm.Provider) ([]Finding, error)`
  - [ ] 2.2: Build the analysis prompt from `AgentAnalysis` — include format, frontmatter summary, section headers, tool list
  - [ ] 2.3: Prompt must request YAML output with the 5 categories: guardrails, security, ordering, decomposition, context
  - [ ] 2.4: Call `provider.Complete(prompt)` then `llmutil.ExtractYAML()` then `llmutil.ParseFindings()`
  - [ ] 2.5: Validate each finding has a valid category and confidence level — reject invalid findings gracefully
  - [ ] 2.6: Write unit tests with mock provider (returns known YAML output)
  - [ ] 2.7: Write integration test (skipped in short mode) that calls real provider on testdata files
- [ ] Task 3: Verify integration (AC: #7, #8)
  - [ ] 3.1: Test analyzer works with any Provider — use mock that returns valid findings YAML
  - [ ] 3.2: Verify all existing tests pass

## Dev Notes

### Extract from Spike — Production Quality

The spike (`internal/doctor/spike/spike.go`) has working versions of:
- `ExtractYAML()` — regex-based code fence stripping
- `ParseFindings()` — YAML unmarshaling to Finding struct
- `AnalysisPrompt()` — prompt template
- `Finding` struct — the output schema

These are **proven to work** on real data (22 files, 4 models). Move them to production packages with improvements:

**From spike to production:**
| Spike location | Production location | Improvements needed |
|---|---|---|
| `spike.ExtractYAML` | `llmutil.ExtractYAML` | Handle multiple fences, empty input error |
| `spike.ParseFindings` | `llmutil.ParseFindings` | Distinguish "no findings" from "wrong schema" |
| `spike.Finding` | `llmutil.Finding` | Add `citation_id` field (links to research registry) |
| `spike.AnalysisPrompt` | `analyzer.buildPrompt` | Take `AgentAnalysis` as input instead of raw string |

### Finding Struct — ADR-4 Schema

From the architecture document (ADR-4):

```go
type Finding struct {
    Category            string `yaml:"category"`             // guardrails|security|ordering|decomposition|context
    Issue               string `yaml:"issue"`
    Confidence          string `yaml:"confidence"`            // high|moderate|low
    CitationID          string `yaml:"citation_id"`           // references research registry entry
    CurrentState        string `yaml:"current_state"`
    SuggestedImprovement string `yaml:"suggested_improvement"`
}
```

The `CitationID` field links to the research registry (Story 2.1). For now, the LLM may or may not populate it — the analyzer should accept findings with or without it.

### Prompt Design

The analyzer prompt must:
1. Describe the file format and content
2. List the 5 analysis categories with descriptions
3. Request YAML output in the exact `findings:` schema
4. Include `temperature: 0.1` instruction (though temperature is set at the provider level)

Build the prompt from `AgentAnalysis` fields:

```go
func buildPrompt(analysis *parser.AgentAnalysis) string {
    var b strings.Builder
    b.WriteString("Analyze this agent definition file for quality issues.\n\n")
    b.WriteString(fmt.Sprintf("File format: %s\n", analysis.Format))
    b.WriteString(fmt.Sprintf("Frontmatter fields: %v\n", frontmatterSummary(analysis.Frontmatter)))
    b.WriteString(fmt.Sprintf("Tools declared: %v\n", analysis.Tools))
    b.WriteString(fmt.Sprintf("Sections: %v\n", sectionSummary(analysis.Sections)))
    b.WriteString("\nFull content:\n---\n")
    b.Write(analysis.RawContent)
    b.WriteString("\n---\n\n")
    b.WriteString(categoryPrompt) // The 5 categories + YAML schema
    return b.String()
}
```

### Mock Provider for Tests

```go
type mockProvider struct {
    response string
    err      error
}

func (m *mockProvider) Complete(prompt string) (string, error) {
    return m.response, m.err
}
```

Use this in unit tests — no real API calls needed.

### Edge Cases from Review

The code review of Story 1.1 identified these issues that must be fixed in the production llmutil:

1. **Empty LLM response → specific error** (not `Parseable: true, FindingsN: 0`)
2. **Wrong YAML schema → specific error** (e.g., `results:` instead of `findings:`)
3. **Multiple code fences → extract all** (not just first)
4. **Partial findings (missing fields) → still accept** with warning, don't reject

### What NOT to Build

- Do NOT build the recommendation-builder (merging findings + citations) — that's part of this story's analyzer but the registry is Story 2.1
- Do NOT build the CLI command — that's Story 1.6
- Do NOT build the context-enricher — that's Story 2.2
- DO build a self-contained analyzer that takes AgentAnalysis + Provider → returns []Finding

### Project Structure

New files:
- `internal/doctor/llmutil/extract.go` — ExtractYAML production version
- `internal/doctor/llmutil/extract_test.go` — comprehensive tests
- `internal/doctor/llmutil/parse.go` — ParseFindings + Finding struct
- `internal/doctor/llmutil/parse_test.go` — comprehensive tests
- `internal/doctor/analyzer/analyzer.go` — Analyze() function
- `internal/doctor/analyzer/analyzer_test.go` — tests with mock provider

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-4] — Analyzer output schema, 5-skill topology
- [Source: _bmad-output/planning-artifacts/architecture.md#Implementation Patterns] — LLM prompt pattern, llmutil.ExtractYAML usage
- [Source: _bmad-output/planning-artifacts/prd.md#FR12-FR14] — Research-backed recommendations with confidence
- [Source: internal/doctor/spike/spike.go] — Working ExtractYAML, ParseFindings, Finding, AnalysisPrompt
- [Source: internal/llm/provider.go] — Provider interface (Complete method)
- [Source: _bmad-output/implementation-artifacts/1-1-engineering-spike.md#Completion Notes] — Spike patterns are reusable

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
