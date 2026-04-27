# Static & Assert Steps for Skill Specs

## Problem

Today all steps in skill specs and experiment tasks are implicitly LLM calls. This makes Forgent an LLM orchestrator (n8n-like), not a harness compiler. Many steps don't need an LLM — aggregation, template rendering, verification checks are deterministic. Running them as LLM calls wastes tokens and time without improving quality.

## Solution

Add a `kind` field to steps in skill specs and experiment tasks. Three kinds:

- **`llm`** (default) — LLM call. The System Under Test.
- **`static`** — Deterministic Go template. Harness logic (concat, format, transform).
- **`assert`** — Verification check. Harness logic (completeness, guardrails, schema).

Future extension: `exec` (shell command) — deferred to separate spec due to security implications.

## Design

### StepKind type (pkg/model)

```go
// pkg/model/step.go

type StepKind string

const (
    StepKindLLM    StepKind = "llm"
    StepKindStatic StepKind = "static"
    StepKindAssert StepKind = "assert"
)

// SkillStep defines one step within a skill's strategy.
type SkillStep struct {
    ID       string   `yaml:"id"`
    Name     string   `yaml:"name"`
    Kind     StepKind `yaml:"kind,omitempty"`     // default: "llm"
    Prompt   string   `yaml:"prompt,omitempty"`   // for kind: llm
    Template string   `yaml:"template,omitempty"` // for kind: static (Go text/template)
    Check    string   `yaml:"check,omitempty"`    // for kind: assert (predefined check name)
    Consumes []string `yaml:"consumes,omitempty"`
    Produces []string `yaml:"produces,omitempty"`
}
```

Backwards-compatible: `kind` is omitempty. Missing kind = `llm`.

### StrategyFacet evolution

`StrategyFacet.Steps` changes from `[]string` to `[]SkillStep`. Backwards-compatible: the YAML unmarshaller handles both `steps: ["do X", "do Y"]` (strings become SkillStep with Name only) and the full struct form.

### StepDef evolution (experiment)

```go
// internal/bench/experiment/design.go

type StepDef struct {
    ID               string          `yaml:"id"`
    Name             string          `yaml:"name"`
    Kind             model.StepKind  `yaml:"kind,omitempty"`
    Prompt           string          `yaml:"prompt,omitempty"`
    Template         string          `yaml:"template,omitempty"`
    Check            string          `yaml:"check,omitempty"`
    Consumes         []string        `yaml:"consumes"`
    Produces         []string        `yaml:"produces"`
    ExpectedKeywords []string        `yaml:"expected_keywords"`
}
```

### Predefined assert checks

| Check name | What it verifies |
|------------|-----------------|
| `all_inputs_present` | All consumed keys have non-empty values |
| `keywords_present` | Expected keywords appear in aggregated inputs |
| `guardrails_respected` | Guardrail constraints (positive/negative) pass on inputs |
| `min_length` | Combined input exceeds a minimum character count |
| `no_empty_steps` | No consumed input is empty string |

Assert checks are pure Go functions registered in a `CheckRegistry`. New checks can be added without changing the spec format.

### Static template execution

Templates use Go `text/template` with inputs available as `.inputs` (map[string]any). Example:

```yaml
- id: aggregate
  kind: static
  template: |
    # Aggregated Findings
    {{range $key, $val := .inputs}}
    ## {{$key}}
    {{$val}}
    {{end}}
  consumes: [section_a_bugs, section_b_bugs, section_c_bugs]
  produces: [all_bugs]
```

The template receives all consumed inputs as the data context.

## Impact per subsystem

### 1. Experiment runners (internal/bench/experiment/)

**runtime_runner.go:**
- `kind: llm` → calls `provider.Complete()` (unchanged)
- `kind: static` → executes Go template against inputs, no LLM call, no token cost
- `kind: assert` → runs the named check function against inputs, produces pass/fail + details

**text_runner.go:**
- All steps (including static/assert) are included in the monolithic prompt. The text condition must have the same task — the LLM handles everything. Static/assert steps are marked in the report for separate token accounting.

**Metrics impact:**
- Token usage: runtime should decrease (static/assert = 0 tokens)
- LLM calls: runtime should decrease (fewer calls)
- Wall-clock: runtime should decrease (no network round-trip for static)
- Completion/ordering: unchanged (same guarantees)
- Detection rate: should increase (assert steps catch failures deterministically)

### 2. DAG runtime generator (internal/generator/forgent/)

- `kind: llm` → generates LLM call node (unchanged)
- `kind: static` → generates Go template function inline
- `kind: assert` → generates check function call

### 3. Text generators (internal/generator/claude/, copilot/)

- `kind: llm` → generates instruction text (unchanged)
- `kind: static` → inlines the template as "expected output format" in the Markdown. Instead of "combine the results", shows the exact template.
- `kind: assert` → generates a "verification checklist" section: "Before proceeding, verify: [check criteria]"

### 4. Linter (internal/linter/)

New rules:
- `static_without_template`: step has `kind: static` but no `template` field
- `assert_without_check`: step has `kind: assert` but no `check` field
- `llm_without_prompt`: step has `kind: llm` (or no kind) but no `prompt` field
- `unknown_check`: step has `kind: assert` with an unregistered check name

### 5. pkg/dag

No changes. The DAG already supports arbitrary `Run` functions. The `kind` is a spec-level concept that the compiler translates to the appropriate `Run` function for each `dag.Node`.

## Task YAML examples (updated experiment tasks)

### long-input-analysis — aggregate step becomes static

```yaml
- id: aggregate
  kind: static
  template: |
    # Aggregated Bugs ({{len .inputs}} sections)
    {{range $key, $val := .inputs}}
    ## {{$key}}
    {{$val}}
    {{end}}
    Total sections analyzed: {{len .inputs}}
  consumes: [section_a_bugs, section_b_bugs, section_c_bugs, section_d_bugs]
  produces: [all_bugs]
```

### multi-agent-review — add assert step

```yaml
- id: completeness-check
  kind: assert
  check: all_inputs_present
  consumes: [blind_findings, security_findings, design_findings]
  produces: [completeness_verified]

- id: triage
  kind: llm
  prompt: "Triage and deduplicate findings..."
  consumes: [blind_findings, security_findings, design_findings, completeness_verified]
  produces: [triaged_findings]
```

## Files changed

| File | Change |
|------|--------|
| `pkg/model/step.go` | **New** — `StepKind`, `SkillStep`, `CheckRegistry` |
| `pkg/model/step_test.go` | **New** — YAML unmarshal tests, backwards-compat |
| `pkg/model/skill.go` | Modify `StrategyFacet.Steps` from `[]string` to `[]SkillStep` |
| `internal/bench/experiment/design.go` | Add `Kind`, `Template`, `Check` to `StepDef` |
| `internal/bench/experiment/runtime_runner.go` | Handle static/assert kinds in DAG node construction |
| `internal/bench/experiment/text_runner.go` | Track static/assert tokens separately |
| `internal/bench/experiment/static.go` | **New** — template execution + assert check registry |
| `internal/bench/experiment/static_test.go` | **New** — template + assert tests |
| `internal/bench/experiment/tasks/*.yaml` | Update aggregate/verify steps with `kind: static/assert` |
| `internal/generator/claude/claude.go` | Handle static/assert kinds in Markdown output |
| `internal/generator/copilot/copilot.go` | Handle static/assert kinds in Markdown output |
| `internal/generator/forgent/forgent.go` | Handle static/assert kinds in Go codegen |
| `internal/linter/rules.go` | Add kind-specific validation rules |

## Deferred

- `kind: exec` (shell command execution) — separate spec, security review needed
- `kind: tool` (MCP tool call) — potential future kind
- Custom check functions via plugins — post-MVP
