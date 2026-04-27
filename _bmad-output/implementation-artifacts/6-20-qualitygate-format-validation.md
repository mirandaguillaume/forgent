# Story 6-20: QualityGate Rewrite — Format Validation via SlotSpec Templates

**Status:** done
**Epic:** 6 — Runtime Harness
**Sprint:** 7

## Goal

Rewrite `QualityGateMiddleware` to validate output format against the `SlotSpec` template (from Story 6-19) rather than checking arbitrary keywords. Add symmetric input validation on `consumes` before step execution.

## Background

The current implementation checks `expected_keywords` — if any keyword is missing, the step fails and retries. This caused 0% runtime completion in the harness experiment because Haiku doesn't reproduce exact keyword strings. The right check is structural: does the output look like the declared format?

## Validation Strategy

**Structural validation** (not parsing/schema): check that the key sections/markers of the template are present in the output. Lightweight, no external dependencies.

Example: template declares `## Bugs\n- **location**:` → validator checks that output contains `## Bugs` and at least one `- **` line.

Markers are extracted from the template by scanning for:
- Markdown headings (`##`, `###`)
- List item prefixes (`- `, `* `)
- Named fields (`**field**:`)
- JSON/YAML keys if template is structured

## Changes

### 1. `pkg/qualitygate/gate.go`

Replace `QualityGatePolicy{Keywords []string}` with:

```go
type QualityGatePolicy struct {
    // Format template to validate against (from SlotSpec.ResolveTemplate())
    // Empty = no format constraint (pass-through)
    ProducesTemplate string
    ConsumesTemplate string // validated before step execution (pre-check)
}
```

Add `extractMarkers(template string) []string` — scans template for structural markers.

Validation logic:
- `ValidateOutput(output, template)`: checks markers present in output (case-insensitive)
- `ValidateInput(input, template)`: same, applied to consumed input before execution

### 2. `buildTaskDAG` in `runtime_runner.go`

For each LLM step, build `QualityGatePolicy` from `SlotSpec`:

```go
for _, slot := range step.Produces {
    if tmpl := slot.ResolveTemplate(); tmpl != "" {
        // attach QualityGateMiddleware with ProducesTemplate
    }
}
for _, slot := range step.Consumes {
    if tmpl := slot.ResolveTemplate(); tmpl != "" {
        // attach pre-execution input validator
    }
}
```

### 3. Remove `expected_keywords` from quality gate path

`expected_keywords` stays in `StepDefinition` for evaluation metrics (keyword hit rate in reports) but is no longer used by the QualityGate middleware for enforcement.

### 4. Update task YAMLs

Add `template` or `template_file` to produces/consumes on relevant steps. Start with `code-review` and `test-generation` as validation cases.

## Acceptance Criteria

**Given** a step with `produces: [{key: bugs, template: "## Bugs\n- **location**:"}]`
**When** the LLM output contains `## Bugs` and a `- **` list item
**Then** the QualityGate passes

**Given** the same step
**When** the LLM output is free-form prose with no `## Bugs` heading
**Then** the QualityGate fails and retries

**Given** a step with no template on produces/consumes
**When** executed
**Then** QualityGate is a no-op pass-through (no regression on steps without templates)

**Given** running the full 16-task experiment after this fix
**When** tasks have no templates defined yet
**Then** runtime completion > 0% (no more blanket 0% from keyword failures)

**Given** `expected_keywords` still present on a step
**When** evaluated in experiment report
**Then** keyword hit rate still appears in metrics (evaluation only, not enforcement)

## Dev Agent Record

### Completion Notes

- Replaced `QualityGatePolicy{Keywords []string}` with `{ProducesTemplate string, ConsumesTemplate string}`
- Added `extractMarkers(tmpl string) []string` — scans headings (`## ...`), bold list items (`- **`), plain list items (`- `)
- Pre-check (ConsumesTemplate): validated against inputs BEFORE handler executes
- Post-check (ProducesTemplate): validated against outputs AFTER handler executes
- Markers lowercased and deduplicated at construction time; validation is case-insensitive substring match
- `buildTaskDAG`: keyword enforcement removed, template-based policy wired from SlotSpec.ResolveTemplate("")
- All 16 tasks now pass QualityGate by default (no template = passthrough)
- `code-review.yaml` and `test-generation.yaml` updated with inline templates as validation cases
- All 34 packages pass. 32 new/updated gate tests.

### File List

- `pkg/qualitygate/gate.go` — rewritten
- `pkg/qualitygate/gate_test.go` — rewritten (32 tests)
- `internal/bench/experiment/runtime_runner.go` — `buildTaskDAG` harness section updated
- `internal/bench/experiment/tasks/code-review.yaml` — produces slots with inline templates
- `internal/bench/experiment/tasks/test-generation.yaml` — produces slots with inline templates

### Review Findings

#### Patch (must fix)

- [x] [Review][Patch] `"* **"` bold list items in LLM output don't match extracted marker `"- **"` — LLM using asterisk-style bullets fails gate even when structurally correct [gate.go:109-113] **MED**

#### Defer

- [x] [Review][Defer] First-slot-wins silently ignores subsequent slot templates — multi-slot produces/consumes enforce only first template [runtime_runner.go:358-370] — deferred; `QualityGatePolicy` is single-template by design; multi-slot enforcement is a future story
- [x] [Review][Defer] `template_file` broken in runtime path — `slot.ResolveTemplate("")` with empty baseDir resolves relative to CWD, bypasses traversal guard [runtime_runner.go:363,368] — deferred; pre-existing from 6-19 (TaskDefinition doesn't carry source path)
- [x] [Review][Defer] `"- "` marker trivially satisfied — dash-space matches any prose with a hyphen-space [gate.go:115] — deferred; spec defines markers as `- ` prefix; tightening requires spec change
- [x] [Review][Defer] Budget middleware 60s overrides node-level 120s — both timeouts active, shorter wins silently [runtime_runner.go:385-389] — deferred; pre-existing harness behavior from 6-18
- [x] [Review][Defer] `MaxRetries = 2` set unconditionally including pass-through nodes — template-less steps now get 2 DAG retries [runtime_runner.go:384] — deferred; retries on network errors are reasonable; not harmful
- [x] [Review][Defer] No warning when template has no extractable markers — passthrough is silent [gate.go:41-43] — deferred; intentional design; observability improvement for future
- [x] [Review][Defer] ConsumesTemplate failure is invariant across retries — upstream step already committed bad output, retries cannot fix it [gate.go:47-53] — deferred; architectural DAG limitation
- [x] [Review][Defer] YAML single-quoted scalar `\n` authoring trap — `\n` in single-quoted strings is literal, not a newline [code-review.yaml:29] — deferred; authoring concern, current YAMLs use correct double-quotes

### Change Log

- 2026-04-17: Story 6-20 implemented — QualityGate rewrite, structural format validation via SlotSpec templates
