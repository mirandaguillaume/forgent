# Story 1.4: Copilot and Forgent YAML Parsers

Status: done

## Story

As a **developer**,
I want Forgent to analyze my GitHub Copilot agents and Forgent skill YAML files,
so that doctor works across all formats I use.

## Acceptance Criteria

1. Copilot parser detects files at `.github/agents/*.agent.md`, `.github/copilot-instructions.md`, `.github/skills/*/SKILL.md`
2. Copilot parser extracts frontmatter (including 1024-char description limit), sections, and tool references
3. All 10 real Copilot files in `internal/doctor/parser/testdata/copilot/` are detected and parsed correctly
4. Forgent YAML parser detects `*.skill.yaml` and `*.agent.yaml` files
5. Forgent YAML parser uses existing `internal/yaml/` loader for parsing
6. Forgent YAML parser subsumes old `forgent doctor` diagnostics (lint + dependency + loop analysis results as AgentAnalysis)
7. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Implement Copilot parser (AC: #1, #2, #3)
  - [ ] 1.1: Create `internal/doctor/parser/copilot.go` with `copilotParser` struct
  - [ ] 1.2: Register via `init()` + `Register("copilot", ...)`
  - [ ] 1.3: Implement `Detect()`: match `.github/agents/` and `.github/skills/` path patterns + check frontmatter
  - [ ] 1.4: Implement `Parse()`: extract YAML frontmatter, handle `description` field (Copilot has 1024-char limit), split sections, extract tools from frontmatter
  - [ ] 1.5: Implement `Validate()`: check frontmatter parseable, markdown structure valid, sections preserved
  - [ ] 1.6: Write table-driven tests using all 10 testdata/copilot/*.md files
- [ ] Task 2: Implement Forgent YAML parser (AC: #4, #5, #6)
  - [ ] 2.1: Create `internal/doctor/parser/forgent.go` with `forgentParser` struct
  - [ ] 2.2: Register via `init()` + `Register("forgent", ...)`
  - [ ] 2.3: Implement `Detect()`: match `*.skill.yaml` and `*.agent.yaml` file extensions
  - [ ] 2.4: Implement `Parse()`: use `internal/yaml.ParseSkillYAML()` / `ParseAgentYAML()` to load, convert to `AgentAnalysis` struct
  - [ ] 2.5: Enrich `AgentAnalysis` with lint results from `internal/linter.LintSkill()` and analyzer results from `internal/analyzer/`
  - [ ] 2.6: Write tests using existing skill/agent YAML files from `skills/` and `agents/` directories
- [ ] Task 3: Verify integration (AC: #7)
  - [ ] 3.1: Test `DetectFormat()` with mixed file set (Claude + Copilot + Forgent YAML) — correct parser selected each time
  - [ ] 3.2: Verify all existing tests pass

## Dev Notes

### Copilot Agent File Format

Copilot agents live in `.github/agents/` with `.agent.md` extension. Key differences from Claude:

```yaml
---
name: ci-reviewer
description: "Reviews CI pipeline changes..." # Max 1024 chars
tools:
  - execute
  - search
  - read
---

Instructions for the agent...
```

- `description` field has 1024-char limit (truncated by Copilot if longer)
- Tools use Copilot-specific names (`execute`, `search`, `read` vs Claude's `Bash`, `Grep`, `Read`)
- `.github/copilot-instructions.md` is a master instructions file (no frontmatter sometimes)
- `.github/skills/*/SKILL.md` follows similar pattern to Claude skills

### Forgent YAML Parser — Reuse Existing Code

The Forgent YAML parser is unique: it doesn't parse markdown, it parses the project's own YAML format. Reuse heavily:

```go
import (
    "github.com/mirandaguillaume/forgent/internal/yaml"
    "github.com/mirandaguillaume/forgent/internal/linter"
    "github.com/mirandaguillaume/forgent/internal/analyzer"
    "github.com/mirandaguillaume/forgent/pkg/model"
)
```

Convert `model.SkillBehavior` → `AgentAnalysis`:
- `Frontmatter`: skill name, version, context, strategy fields
- `Sections`: map each facet (context, strategy, guardrails, observability, security) to a Section
- `Tools`: from `strategy.tools`
- Additionally: run linter and analyzer, include findings as metadata

This subsumes the old `forgent doctor` command — running Forgent's own analysis tools on its own format.

### Copilot Detection Patterns

```go
func (p *copilotParser) Detect(path string, content []byte) bool {
    dir := filepath.Dir(path)
    base := filepath.Base(path)

    // Path-based detection
    if strings.Contains(dir, ".github/agents") && strings.HasSuffix(base, ".agent.md") {
        return true
    }
    if strings.Contains(dir, ".github/skills") && base == "SKILL.md" {
        return true
    }
    if base == "copilot-instructions.md" && strings.Contains(dir, ".github") {
        return true
    }
    return false
}
```

### Previous Story Intelligence

From Story 1.3 (parser registry + Claude parser):
- `FormatParser` interface and registry already exist in `internal/doctor/parser/registry.go`
- `AgentAnalysis` and `Section` structs already exist in `internal/doctor/parser/types.go`
- Claude parser at `internal/doctor/parser/claude.go` is the template — follow its exact patterns
- Frontmatter extraction pattern established — reuse same approach for Copilot

### What NOT to Build

- Do NOT modify the FormatParser interface — it's defined in Story 1.3
- Do NOT build the analyzer LLM skill — that's Story 1.5
- Do NOT build the CLI command — that's Story 1.6
- DO import from `internal/yaml`, `internal/linter`, `internal/analyzer` for Forgent parser — these are stable existing packages

### Project Structure

New files:
- `internal/doctor/parser/copilot.go` — Copilot parser
- `internal/doctor/parser/copilot_test.go` — Copilot tests (uses testdata/copilot/)
- `internal/doctor/parser/forgent.go` — Forgent YAML parser
- `internal/doctor/parser/forgent_test.go` — Forgent YAML tests (uses existing skills/ and agents/)

### Testing Strategy

Copilot — table-driven on 10 testdata files:
```go
func TestCopilotParser_Parse(t *testing.T) {
    files, _ := filepath.Glob("testdata/copilot/*.md")
    for _, f := range files {
        t.Run(filepath.Base(f), func(t *testing.T) {
            content, _ := os.ReadFile(f)
            p := &copilotParser{}
            assert.True(t, p.Detect(f, content))
            analysis, err := p.Parse(content)
            require.NoError(t, err)
            assert.NotEmpty(t, analysis.Sections)
        })
    }
}
```

Forgent YAML — use real project skill files:
```go
func TestForgentParser_Parse(t *testing.T) {
    files, _ := filepath.Glob("../../../../skills/*.skill.yaml")
    for _, f := range files {
        t.Run(filepath.Base(f), func(t *testing.T) {
            content, _ := os.ReadFile(f)
            p := &forgentParser{}
            assert.True(t, p.Detect(f, content))
            analysis, err := p.Parse(content)
            require.NoError(t, err)
            assert.NotEmpty(t, analysis.Format)
        })
    }
}
```

Cross-format detection test:
```go
func TestDetectFormat_MixedFiles(t *testing.T) {
    tests := []struct{ path, format string }{
        {".claude/agents/foo.md", "claude"},
        {".github/agents/bar.agent.md", "copilot"},
        {"skills/review.skill.yaml", "forgent"},
    }
    // ...
}
```

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-6] — FormatParser interface
- [Source: _bmad-output/planning-artifacts/prd.md#FR2] — Doctor analyzes Copilot files
- [Source: _bmad-output/planning-artifacts/prd.md#FR3] — Doctor analyzes Forgent YAML files
- [Source: internal/yaml/loader.go] — ParseSkillYAML(), ParseAgentYAML()
- [Source: internal/linter/registry.go] — LintSkill() for Forgent YAML analysis
- [Source: internal/doctor/parser/claude.go] — Template for parser implementation (from Story 1.3)
- [Source: internal/doctor/parser/testdata/copilot/] — 10 real Copilot agent files

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
