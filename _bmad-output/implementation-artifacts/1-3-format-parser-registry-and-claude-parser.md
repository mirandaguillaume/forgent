# Story 1.3: Format Parser Registry and Claude Parser

Status: done

## Story

As a **developer**,
I want Forgent to automatically detect and parse my Claude Code agent files,
so that doctor can analyze them without me specifying the format.

## Acceptance Criteria

1. `FormatParser` interface with `Detect()`, `Parse()`, `Validate()` methods exists in `internal/doctor/parser/`
2. Parser registry uses `init()` + `Register()` pattern (same as `pkg/spec.TargetGenerator`)
3. Claude parser detects files at `.claude/agents/*.md`, `.claude/skills/*/SKILL.md`, `CLAUDE.md` based on path pattern + frontmatter
4. Claude parser extracts: frontmatter fields, section headers, section content, tool references, overall structure
5. `Parse()` returns an `AgentAnalysis` struct usable by the analyzer skill
6. All 12 real Claude files in `internal/doctor/parser/testdata/claude/` are detected and parsed correctly
7. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [x] Task 1: Define FormatParser interface and registry (AC: #1, #2)
  - [x] 1.1: Created `internal/doctor/parser/registry.go` with `FormatParser` interface
  - [x] 1.2: Interface: Format(), Detect(), Parse(), Validate()
  - [x] 1.3: Register() + Get() + RegisteredFormats()
  - [x] 1.4: DetectFormat() iterates registered parsers
  - [x] 1.5: 5 registry tests: register, get, get-unknown, detect, detect-no-match, registered-formats
- [x] Task 2: Define AgentAnalysis struct (AC: #5)
  - [x] 2.1: Created `internal/doctor/parser/types.go`
  - [x] 2.2: Fields: Format, Frontmatter, Sections, Tools, RawContent
  - [x] 2.3: Section struct: Header, Content, Level
  - [x] 2.4: Tested via Parse tests
- [x] Task 3: Implement Claude parser (AC: #3, #4, #6)
  - [x] 3.1: Created `internal/doctor/parser/claude.go`
  - [x] 3.2: Registered via init() + Register("claude", ...)
  - [x] 3.3: Detect(): path patterns (.claude/agents/, .claude/skills/, CLAUDE.md) + frontmatter (name + tools/model/description/color)
  - [x] 3.4: Parse(): YAML frontmatter extraction, markdown section splitting, tool name extraction (string + list formats)
  - [x] 3.5: Validate(): frontmatter field preservation, section preservation, broken frontmatter detection
  - [x] 3.6: 24 tests total — 12 testdata files, detect tests, parse tests, validate tests, edge cases
  - [x] 3.7: All existing tests pass

## Dev Notes

### FormatParser Interface Design

From ADR-6 in the architecture document:

```go
type FormatParser interface {
    Format() string                              // "claude", "copilot", etc.
    Detect(path string, content []byte) bool     // Can this parser handle this file?
    Parse(content []byte) (*AgentAnalysis, error) // Extract structure for analysis
    Validate(original, rewritten []byte) error   // Quality gate validation (used by Story 3.2)
}
```

The `Validate()` method is defined now but fully used later (Story 3.2 — quality gate). Implement it for Claude now so it's ready.

### Registry Pattern — Follow Existing Code

The exact same pattern exists in two places:

**`pkg/spec/generator.go`:**
```go
var generators = make(map[string]func() Generator)

func Register(target string, factory func() Generator) {
    generators[target] = factory
}
```

**`internal/linter/registry.go`:**
```go
var rules []Rule

func Register(r Rule) {
    rules = append(rules, r)
}
```

The parser registry should follow the `pkg/spec` map-based pattern (not the linter's slice pattern) because parsers are looked up by name.

### Claude File Detection Patterns

Claude Code agent files can be at:
- `.claude/agents/<name>.md` — agent definitions
- `.claude/skills/<name>/SKILL.md` — skill definitions
- `CLAUDE.md` — project-level instructions

Detection logic:
1. Path contains `.claude/` → likely Claude format
2. File has YAML frontmatter with `---` delimiters
3. Frontmatter contains `name`, `description`, or `tools` fields

From the testdata corpus (12 files), examine a few to understand the real format:

### YAML Frontmatter Extraction

The importer already has frontmatter extraction at `internal/importer/frontmatter.go`. **Reuse the pattern but don't import the package** — parser should be self-contained. The pattern is simple:

```go
func extractFrontmatter(content []byte) (map[string]interface{}, []byte, error) {
    // Split on "---" delimiters
    // Parse YAML between first two "---"
    // Return frontmatter map + remaining body
}
```

### Markdown Section Parsing

Split the body (after frontmatter) into sections by markdown headers:

```go
type Section struct {
    Header  string // "## Rules", "### Tools", etc.
    Content string // Everything between this header and the next
    Level   int    // 1 for #, 2 for ##, 3 for ###, etc.
}
```

Use simple line-by-line scanning — no need for a markdown AST parser.

### Previous Story Intelligence

From Story 1.1 (spike):
- `ExtractYAML()` and `ParseFindings()` in spike package are reusable patterns for YAML handling
- Test corpus in `testdata/claude/` is ready — 12 real files with source attribution
- Spike proved that Claude agent files have varied structures — some have frontmatter, some don't

From Story 1.2 (Ollama adapter, if completed):
- The `Provider` interface pattern informs the `FormatParser` interface design
- Same `init()` + `Register()` pattern

### What NOT to Build

- Do NOT implement Copilot or Forgent YAML parsers — that's Story 1.4
- Do NOT implement the Rewrite() method on FormatParser — the interface has Validate() but actual rewriting is Story 3.3
- Do NOT build the analyzer skill — that's Story 1.5
- DO implement Validate() even though it's fully used in Story 3.2 — the interface should be complete

### Project Structure

New files:
- `internal/doctor/parser/registry.go` — interface + registry
- `internal/doctor/parser/registry_test.go` — registry tests
- `internal/doctor/parser/types.go` — AgentAnalysis, Section structs
- `internal/doctor/parser/claude.go` — Claude parser implementation
- `internal/doctor/parser/claude_test.go` — Claude parser tests (uses testdata/claude/)

### Testing Strategy

Table-driven tests using the 12 real testdata files:

```go
func TestClaudeParser_Parse(t *testing.T) {
    files, _ := filepath.Glob("testdata/claude/*.md")
    for _, f := range files {
        t.Run(filepath.Base(f), func(t *testing.T) {
            content, _ := os.ReadFile(f)
            p := &claudeParser{}
            analysis, err := p.Parse(content)
            require.NoError(t, err)
            assert.NotEmpty(t, analysis.Sections)
            // Each file should have at least 1 section
        })
    }
}
```

Test edge cases:
- File with no frontmatter (some agent files skip it)
- File with empty frontmatter (`---\n---`)
- File with only a body, no headers
- Very short file (< 10 lines)

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-6] — FormatParser interface + registry pattern
- [Source: _bmad-output/planning-artifacts/architecture.md#Project Structure] — `internal/doctor/parser/` location
- [Source: _bmad-output/planning-artifacts/architecture.md#Implementation Patterns] — init() + Register(), naming conventions
- [Source: _bmad-output/planning-artifacts/prd.md#FR1] — Doctor analyzes Claude Code agent files
- [Source: _bmad-output/planning-artifacts/prd.md#FR4] — Auto-detect format
- [Source: internal/doctor/parser/testdata/README.md] — Test corpus source attribution
- [Source: pkg/spec/generator.go] — Reference implementation for registry pattern
- [Source: internal/importer/frontmatter.go] — YAML frontmatter extraction pattern

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
