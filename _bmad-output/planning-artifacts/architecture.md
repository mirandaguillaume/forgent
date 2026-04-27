---
stepsCompleted:
  - step-01-init
  - step-02-context
  - step-03-starter-skipped
  - step-04-decisions
  - step-05-patterns
  - step-06-structure
  - step-07-validation
  - step-08-complete
status: 'complete'
completedAt: '2026-03-27'
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - docs/index.md
  - docs/architecture.md
  - docs/project-overview.md
  - docs/development-guide.md
  - docs/source-tree-analysis.md
  - docs/ROADMAP.md
  - docs/plans/2026-03-11-research-agent-context-effectiveness.md
  - content/whitepaper/_index.md
  - docs/plans/ (all 31 design documents)
workflowType: 'architecture'
project_name: 'Forgent'
user_name: 'Guillaume'
date: '2026-03-27'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

### Requirements Overview

**Functional Requirements (58 FRs across 8 areas):**

The FRs decompose into four architectural domains:

1. **Doctor domain (FR1-FR25, FR50):** Agent file analysis, recommendations, rewriting. NEW work — format parsers, LLM-powered analysis, format-preserving rewrite, research registry. Doctor IS a Forgent runtime agent (FR50 keystone).

2. **Doctor Skill Design:** Definition of ~7 doctor skills with their consumes/produces contracts, LLM prompts for each node, and DAG topology. New types: `agent_file`, `detected_format`, `structure_analysis`, `guardrail_recommendations`, `security_recommendations`, `ordering_recommendations`, `recommendations`, `improved_file`. This is the design layer between "what doctor does" and "how it executes."

3. **Compiler domain (FR26-FR43):** Skill spec, composition, multi-target compilation. EXISTING — largely built, needs typed contracts (FR33, Growth) and format version pinning (FR43, Growth).

4. **Runtime domain (FR44-FR49):** DAG execution, contract enforcement, provider adapters. EXISTING — `pkg/dag/` and `internal/generator/forgent/` are operational. Needs dedicated Ollama adapter.

5. **Supporting domains (FR51-FR58):** Benchmarking, ecosystem, extensibility. Mix of new (InstructionBench) and existing (plugin interface).

**Non-Functional Requirements (38 NFRs) — Architecture-Shaping:**

- **NFR18 (Privacy):** Local-first by default — all LLM calls go to Ollama unless user explicitly opts into API.
- **NFR8 (Rewrite safety):** Zero broken files. Requires transactional file operations with quality gate.
- **NFR1-2 (Performance):** 30-60s analysis with 7B model. Constrains prompt design.
- **NFR15 (Concurrency):** File locking for concurrent doctor runs.
- **NFR9 (Determinism):** Identical input → identical output for compiler.

**Scale & Complexity:**

- Primary domain: CLI tool with LLM integration and code generation
- Complexity level: High
- Estimated architectural components: ~12 major subsystems

### New Package Layout (Extends, Doesn't Replace)

| New Subsystem | Location | Existing Code Touched |
|---|---|---|
| Format parsers | `internal/doctor/parser/` | None — shares patterns with importer |
| Recommendation engine | `internal/doctor/recommend/` | None — uses linter rules as patterns |
| Research registry | `internal/doctor/registry/` | None — fully new |
| Format-preserving rewriter | `internal/doctor/rewrite/` | None — inverse of generator |
| Rewrite quality gate | `internal/doctor/validate/` | None — code-based validation |
| Doctor skill YAMLs | `skills/doctor/` | None — follows existing patterns |
| Doctor agent YAML | `agents/doctor.agent.yaml` | None — follows existing patterns |
| Ollama adapter | `internal/llm/ollama.go` | Extends `internal/llm/` (one new file) |
| InstructionBench | `internal/bench/instruction.go` | Extends `internal/bench/` (one new file) |
| Defensive file ops | `internal/fileops/` | None — new shared utility |
| Doctor CLI command | `internal/cmd/doctor.go` | Replaces existing doctor command |

All new code lives in new packages. No existing package needs significant modification.

### Technical Constraints & Dependencies

- **Go 1.24** — language locked, cross-compilation to all platforms
- **Ollama** — required for doctor MVP, runs as external service (not embedded)
- **No CGO** — `gotreesitter` is pure Go, binary must cross-compile cleanly
- **Existing codebase** — architecture must extend, not replace. `pkg/` is public API, `internal/` is private
- **LLM non-determinism** — doctor's analysis varies between runs. Architecture must handle confidence scoring

### Cross-Cutting Concerns

1. **LLM provider abstraction** — used by doctor (local analysis), runtime (skill execution), import (decomposition). Single interface, multiple implementations.
2. **Format parsing** — used by doctor (analyze foreign files), importer (convert to YAML), generators (output to format). Each format is undocumented — parsers are empirical.
3. **Defensive file operations** — backup, restore, locking, transactional write. Used by doctor rewrite. New `internal/fileops/` package.
4. **Rewrite quality gate** — code-based validation between LLM output and file write. Fail-closed by default.
5. **Research registry** — new subsystem consumed by doctor's recommendation engine. Versioned, checksummed, updatable.
6. **Configuration management** — project config, doctor config, provider config, CLI flags. Multiple config sources need consistent resolution.

### Architectural Decisions (Resolved)

**ADR-1: Doctor Embedding Strategy**

**Decision: Embedded YAML with runtime DAG construction (Option C).**

Doctor skill YAMLs and agent YAML are embedded in the binary via `//go:embed`. When `forgent doctor` runs, the Cobra command loads the embedded specs, constructs the DAG in-memory using `pkg/dag`, and executes with the configured provider.

*Rationale:* Single binary distribution (no separate doctor binary). True self-hosting — doctor IS a real Forgent agent defined in YAML, executed by the real DAG engine. Uses existing `pkg/dag`, `internal/yaml`, `internal/llm`. Minimal new code. No compilation step at install time.

*Rejected alternatives:*
- Separate binary (Option A): Requires shipping two binaries or compile-at-install. Friction for Trojan Horse.
- Hardcoded Go DAG (Option B): Doctor "pretends" to be a runtime agent. Loses self-hosting proof.

**ADR-2: Rewrite Quality Gate**

**Decision: Four-check fail-closed validation gate between LLM output and file write.**

Checks:
1. **Frontmatter validity** — rewritten file has parseable YAML frontmatter
2. **Field preservation** — all original frontmatter fields still present
3. **Markdown structure** — headers, code blocks, lists are well-formed
4. **Section preservation** — all user's custom sections still present

Plus: **diff-size sanity check** — if rewrite changes >50% of file, flag as suspicious and require explicit confirmation.

Fail-closed: any validation failure → reject rewrite, keep original, show error. User can force with `--force`.

*Rationale:* 7B local models are less reliable at format-preserving rewrites than cloud models. Code-based validation is deterministic and fast. Satisfies both NFR8 (zero broken files) and NFR18 (local-first).

**ADR-3: Research Registry Hosting**

**Decision: Embedded in binary with local override path.**

Default registry embedded via `//go:embed registry/research-registry.yaml`. Local override at `.forgent/research-registry.yaml` — if present, takes precedence. Growth phase adds `forgent doctor --update-registry` to download newer versions.

Registry format: YAML with `version`, `recommendations[]` containing `id`, `title`, `citation`, `paper`, `url`, `finding`, `confidence`, `detection_prompt`, `suggestion_prompt`.

*Rationale:* Zero network dependency for MVP (consistent with local-first NFR18). Registry has ~7 entries, changes infrequently (when new research publishes). Coupled to release cycle is acceptable for MVP. Local override provides Growth path without architecture changes.

## Core Architectural Decisions

### Decision Priority Analysis

**Already Decided (existing codebase):**
- Go 1.24 + Cobra + yaml.v3 + testify — locked
- Plugin architecture for targets (`pkg/spec.TargetGenerator`, `init()` registration)
- Pipeline: parse → lint → analyze → enrich → generate
- DAG engine: auto-wiring, 12 node kinds, layer execution
- CI/CD: GitHub Actions + release-please + goreleaser

**Critical Decisions (resolved in this step):**

**ADR-4: Doctor DAG Topology — 5 skills, fan-out with conditional rewrite**

```
Layer 0: format-detector (CODE, ~0s)
Layer 1: analyzer (LLM, ~10-15s) ‖ context-enricher (CODE, ~5s)
Layer 2: recommendation-builder (CODE, ~0s)
Layer 3: rewriter (LLM, ~10-15s, conditional on --fix)
```

Skills:

| Skill | Type | Consumes | Produces |
|---|---|---|---|
| format-detector | CODE | agent_file | detected_format |
| analyzer | LLM | agent_file, detected_format | analysis_results |
| context-enricher | CODE | agent_file | context_recommendations |
| recommendation-builder | CODE | analysis_results, context_recommendations | recommendations |
| rewriter | LLM (conditional) | agent_file, recommendations, detected_format | improved_file |

- Read-only mode: 1 LLM call, ~15s total
- Fix mode: 2 LLM calls, ~25-30s total
- Showcases real parallelism (LLM ‖ code in Layer 1) and conditional execution (Layer 3)
- Open to refinement based on LLM quality spike results

**Code-only skills:** Skills with `strategy.approach: code` make no LLM call. The runtime DAG engine executes their RunFunc directly. Only skills with `strategy.approach: llm` make provider calls. Doctor proves that not all skills need an LLM — some are pure computation.

**Context-enricher note:** Reuses `internal/scanner/` for codebase data collection (TF-IDF, entropy, AST symbols) but has its own recommendation logic that transforms `CodebaseContext` into actionable recommendations (e.g., "your project has 47 test files not referenced by the agent").

**Analyzer output schema** (contract between analyzer and recommendation-builder):

```yaml
# analyzer produces: analysis_results
findings:
  - category: guardrails | security | ordering | decomposition | context
    issue: string
    confidence: high | moderate | low
    citation_id: string  # references research registry entry
    current_state: string  # what exists in the file now
    suggested_improvement: string
```

**ADR-5: Ollama Integration — Direct HTTP to Ollama REST API**

New `internal/llm/ollama.go` implementing the existing `Provider` interface. Direct HTTP calls to Ollama's local API (`localhost:11434`). No intermediary.

**Error handling with actionable messages:**
- Connection refused → "Ollama not running. Start with `ollama serve` or use `--provider anthropic`."
- Model not found → "Model not available. Run `ollama pull <model>` or use `--model <available_model>`."
- Malformed output → "Ollama returned invalid response. Try a different model or use `--provider anthropic`."
- Timeout (NFR35, 120s default) → "Analysis timed out. Try a smaller model or use `--provider anthropic` for cloud processing."

**ADR-6: Format Parser Architecture — Interface + Registry**

`FormatParser` interface with `init()` registration. Same plugin pattern as `TargetGenerator`. Each format is a module in `internal/doctor/parser/`.

```go
type FormatParser interface {
    Format() string                    // "claude", "copilot", etc.
    Detect(path string, content []byte) bool  // Can this parser handle this file?
    Parse(content []byte) (*AgentAnalysis, error)  // Extract structure for analysis
    Validate(original, rewritten []byte) error  // Quality gate validation
    Rewrite(original []byte, recommendations []Recommendation) ([]byte, error)  // Format-preserving rewrite
}
```

Reuses patterns from `internal/importer/` for frontmatter extraction and framework detection. Each format parser handles both reading (for analysis) and writing (for rewrite) in the same module — keeping format knowledge co-located.

**ADR-7: Configuration Resolution — Standard Precedence**

CLI flags > environment variables > config file (`.forgent/config.yaml`) > defaults. Standard pattern. Cobra handles flag binding; Viper or manual resolution for env/config.

**ADR-8: Local LLM Quality Validated — Proceed with Local-First Architecture**

**Decision: Local-first architecture confirmed. Recommended default model: Llama 4 Scout (or equivalent 8B+ model via Ollama).**

Engineering spike (Story 1.1, 2026-03-30) tested 3 local-class models and 1 cloud baseline on 5 real agent files via OpenRouter:

| Model | Parse Rate | Avg Findings | Avg Latency | Notes |
|---|---|---|---|---|
| **Llama 4 Scout** | **100%** | 5.2 | **498ms** | Best local — fast, reliable, recommended default |
| Qwen 3 8B | 100% | 4.2 | 1438ms | Reliable fallback, slower |
| Gemma 3 12B | 60% | 5.3 | 629ms | Fast but YAML output unreliable (40% parse failure) |
| **Claude Sonnet 4** (baseline) | 100% | 6.6 | 1071ms | Cloud baseline |

Key metrics:
- **Local parse rate: 100%** (Llama 4 Scout, Qwen 3 8B) — matches cloud baseline
- **Local finding rate: 79%** of cloud (5.2 vs 6.6 findings/file) — exceeds 50% threshold
- **Local latency: 2x faster** than cloud (498ms vs 1071ms)
- **Decision gate: PASSED** — local accuracy ≥ 50% of cloud baseline

*Rationale:* Local models reliably produce structured YAML output from the analysis prompt. Parse rate matches cloud. Finding count is ~80% of cloud — acceptable for MVP, can improve with prompt engineering. Latency is better than cloud. The local-first promise (NFR18) is architecturally viable.

*Implications for ADR-5 (Ollama):* Direct Ollama adapter proceeds as planned. Recommended default model for `forgent doctor`: any 8B+ instruction-tuned model (Llama 3/4, Qwen 3, Mistral). Gemma is not recommended — unreliable YAML output.

*Evidence:* Full results at `docs/plans/2026-03-spike-llm-quality.md`. Test corpus: 22 real agent files in `internal/doctor/parser/testdata/`.

**ADR-9: Ollama Provider — Context-Only Timeout, errors.As, URL Validation (from code review)**

**Decision: Single timeout via `context.WithTimeout`, use `errors.As` from stdlib, validate apiKey-as-URL.**

Code review of Story 1.2 identified three issues resolved during implementation:

1. **Single timeout mechanism:** Removed dual `context.WithTimeout` + `http.Client.Timeout` (non-deterministic error messages when both fire). Now uses only `context.WithTimeout` on the request — idiomatic Go approach. `http.DefaultClient` is used for connection pooling.

2. **`errors.As` from stdlib:** Replaced custom `errorAs()` function (manual unwrap loop) with `errors.As` from the `errors` package. The custom version didn't handle Go 1.20+ `Unwrap() []error` (joined errors). No reason to avoid a zero-dependency stdlib import.

3. **URL validation on apiKey override:** The factory repurposes `apiKey` as base URL for remote Ollama. Now validates with `url.Parse()` — only accepts if scheme + host are present. Invalid values (actual API keys, garbage) are silently ignored and the default localhost is used.

*Rationale:* All three issues were caught by the adversarial code review (Blind Hunter + Edge Case Hunter). Fixing during review prevents subtle bugs in production.

### Deferred Decisions (Post-MVP)

| Decision | Reason | When |
|---|---|---|
| Typed contracts schema | Design complexity, not needed for doctor MVP | 6-month milestone |
| Format version pinning | Useful but not MVP-critical | Growth |
| JSON/SARIF output format | Human text only for MVP | Growth |
| InstructionBench experiment design | Needs corpus first | MVP Phase D |
| `--deep` decomposition architecture | Requires cloud API, beyond MVP | Growth |
| Concurrency tuning for Ollama | Optimize after measuring real performance | Post-spike |

### Decision Impact Analysis

**Implementation Sequence (maps to PRD scoping):**

1. **Spike:** Ollama adapter (ADR-5) + test analyzer prompt on 20 files
2. **Phase A:** Format parsers (ADR-6) + analyzer skill + recommendation-builder + research registry (ADR-3)
3. **Phase C:** Doctor as DAG (ADR-1 + ADR-4) — embed YAML, construct DAG, execute
4. **Phase D:** InstructionBench — uses analyzer output for corpus evaluation
5. **Phase B:** Rewriter skill + quality gate (ADR-2) — hardest part, last

**Cross-Component Dependencies:**

- ADR-4 (topology) depends on ADR-1 (embedding) — YAML must be embeddable
- ADR-4 (analyzer) depends on ADR-5 (Ollama) — needs LLM provider
- ADR-4 (context-enricher) depends on existing `internal/scanner/` — reuses it
- ADR-6 (format parser) feeds into ADR-4 (format-detector skill) — parser = skill implementation
- ADR-2 (quality gate) depends on ADR-6 (format parser) — parser validates output
- ADR-3 (registry) feeds into ADR-4 (recommendation-builder) — citations come from registry

## Implementation Patterns & Consistency Rules

### Established Patterns (from existing codebase — ALL new code MUST follow)

**Package Organization:**
- `pkg/` — Public API. Stable interfaces. Importable by external code.
- `internal/` — Private implementation. Can change freely. All new doctor code goes here.
- `cmd/` — Binary entry points. Thin wrappers calling `internal/cmd.Execute()`.
- New packages: one concept per package. `internal/doctor/parser/`, not `internal/doctor/parser-and-validator/`.

**Import Grouping:**
```go
import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"

    "github.com/mirandaguillaume/forgent/internal/doctor/parser"
    "github.com/mirandaguillaume/forgent/pkg/dag"
)
```
Three groups separated by blank lines: stdlib, external, internal. `goimports` enforces this.

**Naming Conventions:**
- Files: `lowercase.go`, `lowercase_test.go`. No hyphens, no camelCase.
- Packages: single lowercase word (`parser`, `registry`, `fileops`). No underscores.
- Exported types: `PascalCase` (`FormatParser`, `ResearchRegistry`, `QualityGate`).
- Unexported: `camelCase` (`parseClaudeFile`, `validateFrontmatter`).
- Test files: `*_test.go` in same package (white-box) or `_test` package (black-box).
- Interfaces: named by capability, not "I" prefix. `Parser`, not `IParser`.

**Plugin Registration Pattern:**
```go
func init() {
    Register("claude", func() FormatParser { return &claudeParser{} })
}
```
Follows existing patterns in `pkg/spec/generator.go` and `internal/linter/registry.go`.

**Testing Patterns:**
- Table-driven tests with `testify` assertions. Every test function uses `t.Run()` subtests.
- Temp directory fixtures for file I/O tests (`t.TempDir()`).
- No mocks for simple interfaces — use real implementations or test doubles.
- Test naming: `TestFunctionName_Scenario` (e.g., `TestParseClaudeFile_MissingFrontmatter`).
- CI gate: `go test ./... -count=1 -race`.

**Error Handling:**
- Return `error`, don't panic. Only `main()` calls `os.Exit()`.
- Wrap errors with context: `fmt.Errorf("parsing claude file: %w", err)`.
- Actionable messages for user-facing errors (NFR28).
- Sentinel errors for expected conditions: `var ErrFormatNotDetected = errors.New("format not detected")`.

**Output Convention:**
```go
// User-facing output: fatih/color to stdout
color.Green("✓ 5 recommendations found")

// Debug output: stderr, only with --debug
if debug { fmt.Fprintf(os.Stderr, "[DEBUG] prompt sent: %s\n", prompt) }

// Errors: stderr, always
fmt.Fprintf(os.Stderr, "Error: %s\n", err)
```
No logging library. Color for stdout, plain for stderr.

**CLI Command Pattern:**
```go
var doctorCmd = &cobra.Command{
    Use:   "doctor <file>",
    Short: "Analyze and improve agent files",
    RunE:  runDoctor,
}

func init() {
    rootCmd.AddCommand(doctorCmd)
    doctorCmd.Flags().Bool("fix", false, "Apply selected recommendations")
}
```

### New Patterns (for doctor subsystems)

**Doctor Skill YAML Pattern:**
```yaml
skill: format-detector
version: "1.0.0"
context:
  consumes: [agent_file]
  produces: [detected_format]
strategy:
  approach: code  # "code" = no LLM call, "llm" = calls provider
  tools: []
  steps:
    - detect file format from path and frontmatter
guardrails:
  - timeout: 5s
security:
  filesystem: read-only
  network: none
```

**FormatParser Implementation Pattern:**
```go
// internal/doctor/parser/claude.go
type claudeParser struct{}

func init() {
    parser.Register("claude", func() parser.FormatParser { return &claudeParser{} })
}

func (p *claudeParser) Format() string              { return "claude" }
func (p *claudeParser) Detect(path string, content []byte) bool { /* check path + frontmatter */ }
func (p *claudeParser) Parse(content []byte) (*parser.AgentAnalysis, error) { /* extract structure */ }
func (p *claudeParser) Validate(original, rewritten []byte) error { /* quality gate checks */ }
```

**Format Parser Test Pattern:**
```go
func TestClaudeParser_Detect(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        content string
        want    bool
    }{
        {"claude skill file", ".claude/skills/foo/SKILL.md", "---\nname: foo\n---", true},
        {"random markdown", "docs/readme.md", "# Hello", false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := &claudeParser{}
            assert.Equal(t, tt.want, p.Detect(tt.path, []byte(tt.content)))
        })
    }
}
```
Every parser MUST have Detect, Parse, and Validate tests with ≥5 real files in `internal/doctor/parser/testdata/` per format.

**Research Registry Entry Pattern:**
```yaml
- id: guardrail-placement
  title: "Add guardrails section"
  citation: "Liu et al., 2024"
  paper: "Lost in the Middle: How Language Models Use Long Contexts"
  url: "https://arxiv.org/abs/2307.03172"
  finding: "Critical instructions at prompt start improve adherence"
  confidence: high
  detection_prompt: "Does this agent file have explicit behavioral constraints near the beginning?"
  suggestion_prompt: "Add a guardrails section at the top with timeout, output limits, and behavioral boundaries."
```

**Defensive File Operations Pattern:**
```go
// internal/fileops/ — ALL user file writes go through here
func SafeWrite(path string, content []byte) error {
    backup := path + ".bak"
    if err := copyFile(path, backup); err != nil { return err }
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, content, 0644); err != nil { return err }
    return os.Rename(tmp, path)  // Atomic rename
}

func Restore(path string) error {
    return os.Rename(path+".bak", path)
}
```
Never `os.WriteFile()` directly on user files. SafeWrite crash-safety tests must verify recovery from interruption at each step (after backup, after write, after rename).

**LLM YAML Extraction Utility:**
```go
// internal/doctor/llmutil/extract.go — shared across all doctor skills
func ExtractYAML(llmOutput string) (string, error) {
    // Strip ```yaml, ```yml, ``` wrappers
    // Handle: fenced, unfenced, and malformed variations
    // Return clean YAML string for yaml.v3 parsing
}
```
Every LLM skill that expects structured output uses this utility. Never parse LLM output directly.

**LLM Prompt Pattern:**
```go
const analyzerPrompt = `Analyze this {{.Format}} agent file for issues.

File content:
{{.Content}}

Check for:
1. Guardrail issues (missing constraints, poor placement)
2. Security gaps (missing declarations)
3. Instruction ordering (primacy bias)
4. Decomposition opportunities (monolithic structure)

Respond in YAML:
findings:
  - category: guardrails|security|ordering|decomposition
    issue: <description>
    confidence: high|moderate|low
    citation_id: <registry_id>
    current_state: <what exists>
    suggested_improvement: <what to change>`
```
Prompts are Go templates. Versioned with the research registry.

**Concurrency Pattern:**
```go
opts := []dag.Option{dag.WithTimeout(120 * time.Second)}
if provider.IsLocal() {
    opts = append(opts, dag.WithConcurrency(1))  // Serialize local LLM calls
}
dag.Execute(d, opts...)
```

### Enforcement Guidelines

**All LLM agents implementing Forgent code MUST:**

1. Follow existing package organization (`pkg/` vs `internal/`)
2. Use three-group import ordering (stdlib / external / internal)
3. Use `init()` + `Register()` for all pluggable modules
4. Write table-driven tests with testify for every new function
5. Use `fileops.SafeWrite()` for any user file modification
6. Use `llmutil.ExtractYAML()` for all structured LLM output parsing
7. Wrap errors with context using `fmt.Errorf("context: %w", err)`
8. Return actionable error messages for user-facing errors
9. Use the `Provider` interface for all LLM calls — never call Ollama/Anthropic directly
10. Embed doctor skill YAMLs via `//go:embed` — no external file dependencies
11. Pass `go vet ./...` and `go test ./... -count=1 -race` before considering work complete

**Anti-Patterns (NEVER do these):**

- Don't create `utils/` or `helpers/` packages — put code where it's used
- Don't add `interface{}` or `any` types — use concrete types or specific interfaces
- Don't use global state except for `init()` registration
- Don't write to user files without `fileops.SafeWrite()`
- Don't hardcode Ollama URLs — use configuration
- Don't skip error handling with `_ =`
- Don't parse LLM output directly — always use `llmutil.ExtractYAML()`

## Project Structure & Boundaries

### Complete Project Directory Structure

Existing structure with new doctor additions marked with `← NEW`:

```
forgent/
├── cmd/
│   ├── forgent/
│   │   └── main.go                    # CLI entry point
│   └── forgent-bench/
│       └── main.go                    # Benchmark CLI (internal)
│
├── pkg/                               # Public API (stable)
│   ├── model/                         # SkillBehavior, AgentComposition
│   ├── spec/                          # TargetGenerator interface + registry
│   ├── dag/                           # DAG engine (12 node kinds)
│   └── analysis/                      # Diff analysis pipeline
│
├── internal/                          # Private implementation
│   ├── cmd/
│   │   ├── root.go
│   │   ├── doctor.go                  # ← REWRITE: new doctor command
│   │   ├── build.go
│   │   ├── lint.go
│   │   ├── score.go
│   │   ├── init.go
│   │   ├── skill_create.go
│   │   ├── watch.go
│   │   └── import.go
│   │
│   ├── doctor/                        # ← NEW: doctor subsystem
│   │   ├── doctor.go                  # ← NEW: orchestrator (DAG construction + execution)
│   │   ├── parser/                    # ← NEW: format parsers
│   │   │   ├── registry.go           # ← NEW: FormatParser interface + Register()
│   │   │   ├── claude.go             # ← NEW: Claude Code parser
│   │   │   ├── copilot.go            # ← NEW: GitHub Copilot parser
│   │   │   ├── forgent.go            # ← NEW: Forgent YAML parser
│   │   │   └── testdata/             # ← NEW: ≥5 real files per format
│   │   │       ├── claude/
│   │   │       └── copilot/
│   │   ├── recommend/                 # ← NEW: recommendation engine
│   │   │   ├── engine.go             # ← NEW: merge analysis + context + citations
│   │   │   └── engine_test.go
│   │   ├── registry/                  # ← NEW: research registry
│   │   │   ├── registry.go           # ← NEW: load, query, version check
│   │   │   ├── registry_test.go
│   │   │   └── data/                  # ← NEW: embedded registry YAML
│   │   │       └── research-registry.yaml
│   │   ├── rewrite/                   # ← NEW: format-preserving rewriter
│   │   │   ├── rewrite.go            # ← NEW: LLM-powered rewrite
│   │   │   └── rewrite_test.go
│   │   ├── validate/                  # ← NEW: quality gate
│   │   │   ├── gate.go               # ← NEW: four-check validation
│   │   │   └── gate_test.go
│   │   └── llmutil/                   # ← NEW: shared LLM utilities
│   │       ├── extract.go            # ← NEW: YAML extraction from LLM output
│   │       └── extract_test.go
│   │
│   ├── fileops/                       # ← NEW: defensive file operations
│   │   ├── safe_write.go             # ← NEW: backup + atomic write + restore
│   │   └── safe_write_test.go
│   │
│   ├── llm/                           # LLM providers
│   │   ├── provider.go               # Provider interface
│   │   ├── registry.go               # Provider factory registry
│   │   ├── anthropic.go              # Anthropic Messages API
│   │   ├── openrouter.go             # OpenRouter API
│   │   └── ollama.go                 # ← NEW: Ollama direct HTTP adapter
│   │
│   ├── builder/                       # Build orchestration (existing)
│   ├── linter/                        # Lint rules (existing)
│   ├── analyzer/                      # Scoring + analysis (existing)
│   ├── scanner/                       # Codebase scanning (existing, reused by doctor)
│   ├── enricher/                      # Context injection (existing)
│   ├── importer/                      # Agent .md import (existing)
│   ├── yaml/                          # YAML parsing (existing)
│   ├── templates/                     # Embedded templates (existing)
│   ├── bench/                         # Benchmark framework (existing)
│   │   └── instruction.go            # ← NEW: InstructionBench
│   └── generator/                     # Target generators (existing)
│       ├── claude/
│       ├── copilot/
│       └── forgent/
│
├── skills/                            # Skill YAML definitions
│   ├── doctor/                        # ← NEW: doctor skill definitions
│   │   ├── format-detector.skill.yaml
│   │   ├── analyzer.skill.yaml
│   │   ├── context-enricher.skill.yaml
│   │   ├── recommendation-builder.skill.yaml
│   │   └── rewriter.skill.yaml
│   ├── bug-scanner.skill.yaml         # (18 existing skills)
│   └── ...
│
├── agents/
│   ├── doctor.agent.yaml              # ← NEW: doctor agent composition
│   ├── code-reviewer.agent.yaml       # (3 existing agents)
│   ├── ci-reviewer.agent.yaml
│   └── staged-reviewer.agent.yaml
│
├── contracts/                         # Output format contracts (existing)
├── templates/                         # Scaffold templates (existing)
├── content/                           # Hugo site + whitepaper (existing)
├── docs/                              # Documentation (existing)
├── .github/                           # CI/CD + generated Copilot output (existing)
├── .claude/                           # Generated Claude output (existing)
├── .forgent/                          # Generated Forgent runtime output (existing)
├── go.mod
├── go.sum
├── CLAUDE.md
├── README.md
└── CHANGELOG.md
```

### Architectural Boundaries

**Doctor Boundary:**
- `internal/doctor/` is self-contained. It imports from `pkg/dag`, `pkg/model`, `internal/llm`, `internal/scanner`, `internal/fileops` — but nothing imports FROM `internal/doctor/`.
- The only entry point is `internal/cmd/doctor.go` which creates the doctor DAG and executes it.
- Doctor skills are data (YAML), not code. They live in `skills/doctor/` and are embedded via `//go:embed`.

**LLM Provider Boundary:**
- All LLM calls go through `internal/llm.Provider` interface. Doctor, runtime, and import all use the same interface.
- Provider selection is resolved at startup based on config (ADR-7: flags > env > config > defaults).
- No package except `internal/llm/` makes HTTP calls to LLM APIs.

**File Operations Boundary:**
- All user file writes go through `internal/fileops/`. No other package writes to files the user owns.
- Build output (`.claude/`, `.github/`, `.forgent/`) uses existing generator code, not fileops.
- Doctor rewrite is the only operation that modifies files the user already has.

**Format Parser Boundary:**
- `internal/doctor/parser/` handles all format-specific knowledge.
- Format parsers are pluggable via the registry. Adding Cursor support means adding `internal/doctor/parser/cursor.go` + `init()` registration. No other code changes.

### FR Category to Structure Mapping

| FR Category | Primary Location | Shared Dependencies |
|---|---|---|
| Agent Analysis (FR1-FR11) | `internal/doctor/`, `internal/doctor/parser/` | `internal/llm/`, `internal/scanner/` |
| Recommendations (FR12-FR18) | `internal/doctor/recommend/`, `internal/doctor/registry/` | Research registry data |
| Agent Rewriting (FR19-FR25) | `internal/doctor/rewrite/`, `internal/doctor/validate/`, `internal/fileops/` | `internal/llm/`, `internal/doctor/parser/` |
| Skill Spec (FR26-FR33) | `pkg/model/`, `internal/yaml/`, `internal/linter/` | Existing — no new code |
| Compilation (FR34-FR43) | `internal/generator/`, `internal/builder/` | Existing — no new code for MVP |
| Runtime (FR44-FR50) | `pkg/dag/`, `internal/generator/forgent/` | `internal/llm/` for Ollama adapter |
| Benchmarking (FR51-FR53) | `internal/bench/` | New `instruction.go` |
| Ecosystem (FR54-FR58) | `pkg/spec/`, `internal/cmd/` | Existing plugin interface |

### Data Flow

```
User runs: forgent doctor agent.md

CLI (internal/cmd/doctor.go)
  → Load embedded doctor YAML (//go:embed)
  → Construct DAG (pkg/dag)
  → Execute:
    Layer 0: format-detector
      → parser.Registry.Detect(path, content) → detected_format
    Layer 1 (parallel):
      → analyzer: llm.Provider.Complete(prompt) → analysis_results
      → context-enricher: scanner.ScanCodebase() → context_recommendations
    Layer 2: recommendation-builder
      → recommend.Engine.Build(analysis, context, registry) → recommendations
    Layer 3 (conditional, --fix only):
      → rewriter: llm.Provider.Complete(rewrite_prompt) → improved_content
      → validate.Gate.Check(original, improved) → pass/fail
      → fileops.SafeWrite(path, improved) → done
  → Output recommendations to stdout (color.*)
```

## Architecture Validation Results

### Coherence Validation

**Decision Compatibility:** All 7 ADRs are internally consistent. Technology choices (Go 1.24, Cobra, yaml.v3, Ollama) have no conflicts. Plugin registry pattern used uniformly for generators, parsers, and providers.

**Pattern Consistency:** All implementation patterns are Go-idiomatic and consistent with the existing codebase. Import grouping, naming, testing, error handling follow established conventions.

**Structure Alignment:** New doctor packages cleanly isolated. Dependency graph is acyclic: `internal/doctor/` depends on `pkg/dag`, `internal/llm`, `internal/scanner` — nothing depends on `internal/doctor/`.

### Requirements Coverage

**58 FRs:** All architecturally supported. Each FR category maps to specific packages in the project structure.

**38 NFRs:** All addressed. Performance via topology optimization. Reliability via quality gate + SafeWrite. Privacy via local-first Ollama. Testing via enforcement guidelines.

### Implementation Readiness

**Decision Completeness:** 7 ADRs covering all critical decisions. Each includes rationale and rejected alternatives.

**Structure Completeness:** Full directory tree with every new file mapped. FR-to-structure mapping table provided.

**Pattern Completeness:** 11 mandatory enforcement rules. 7 anti-patterns. Code examples for every pattern.

### Implementation Notes

**Context-enricher graceful degradation:** When `forgent doctor` runs on a file outside a project directory (no codebase to scan), the context-enricher produces empty results. The recommendation-builder merges whatever it gets — if context is empty, recommendations come from analysis only. No crash, no error, just fewer recommendations.

**Testdata sourcing:** Format parser testdata files (`internal/doctor/parser/testdata/`) are real agent files from public GitHub repos, committed to the repository with source attribution in a README. No synthetic files — tests validate against real-world format variations.

### Architecture Completeness Checklist

- [x] Project context analyzed (58 FRs, 38 NFRs, 4 domains)
- [x] Starter template evaluated (brownfield — skipped, stack locked)
- [x] 7 ADRs documented with rationale and rejected alternatives
- [x] Implementation patterns with code examples (11 rules, 7 anti-patterns)
- [x] Complete project structure with new file mapping
- [x] FR category to structure mapping
- [x] Data flow diagram
- [x] Architectural boundaries defined (doctor, LLM provider, file ops, format parser)
- [x] Cross-component dependencies mapped
- [x] Validation passed — no critical gaps

### Architecture Readiness Assessment

**Overall Status:** READY FOR IMPLEMENTATION

**Confidence Level:** High

**Key Strengths:**
- Extends existing codebase cleanly — no refactoring needed
- All new code in new packages — minimal risk to existing functionality
- Doctor DAG topology optimized for local LLM performance (1-2 calls)
- Self-hosting proof is architecturally sound (ADR-1 + ADR-4)
- Plugin pattern enables Growth targets (Cursor, Windsurf) without architecture changes

**Areas for Future Enhancement:**
- Doctor config schema (simple YAML, defer to implementation)
- Concurrency tuning for Ollama (optimize post-spike)
- Format version pinning mechanism (Growth)

### Implementation Handoff

**AI Agent Guidelines:**
- Follow all 7 ADRs exactly as documented
- Use all 11 implementation patterns consistently
- Respect the 4 architectural boundaries
- New code goes in new packages — don't modify existing packages except `internal/cmd/doctor.go` and `internal/llm/ollama.go`

**First Implementation Priority:**
Engineering spike — `internal/llm/ollama.go` + test analyzer prompt on 20 real agent files. Validates ADR-5 and de-risks the highest-risk assumption (local LLM quality) before committing to full MVP.
```
