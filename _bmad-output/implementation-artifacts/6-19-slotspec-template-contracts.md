# Story 6-19: SlotSpec — Template Contracts for Produces/Consumes

**Status:** done
**Epic:** 6 — Runtime Harness
**Sprint:** 7

## Goal

Replace `produces: []string` and `consumes: []string` in `StepDefinition` with `[]SlotSpec`, where each slot carries an optional format template. This makes the DAG's data-flow contracts formally typed: a step declares not just *what* it produces/consumes but *what shape* that data must have.

## Background

The current QualityGate checks `expected_keywords`, which is too brittle (keyword mismatch = total failure). The right contract is a **format template**: the step that produces `bugs` promises to output data in a specific structure, and the step that consumes `bugs` expects that structure as input.

This story adds the data model only. Story 6-20 rewrites the QualityGate to use it.

## Changes

### 1. `pkg/model/skill.go` or `internal/bench/experiment/design.go`

Add `SlotSpec` type (shared by both produces and consumes):

```go
type SlotSpec struct {
    Key          string `yaml:"key"`
    Template     string `yaml:"template,omitempty"`      // inline format template
    TemplateFile string `yaml:"template_file,omitempty"` // path to external template file
}
```

### 2. `internal/bench/experiment/design.go`

Change `StepDefinition`:

```go
// Before
Produces []string `yaml:"produces,omitempty"`
Consumes []string `yaml:"consumes,omitempty"`

// After
Produces []SlotSpec `yaml:"produces,omitempty"`
Consumes []SlotSpec `yaml:"consumes,omitempty"`
```

Implement YAML unmarshalling that supports both old shorthand (`produces: [key]`) and new full form (`produces: [{key: bugs, template: ...}]`) for backward compatibility.

### 3. `SlotSpec.ResolveTemplate()`

Add a method that returns the resolved template string:
- If `TemplateFile` is set: read file relative to task definition location
- Else: return inline `Template`
- If neither: return `""` (no format constraint)

### 4. Update all callers of `step.Produces` / `step.Consumes`

Any code that iterates `step.Produces` as `[]string` needs updating to use `.Key` field. Key callers:
- `buildTaskDAG` in `runtime_runner.go`
- `evaluateRuntimeOutputs` in `runtime_runner.go`
- Task YAML loader

## Acceptance Criteria

**Given** a task YAML with shorthand `produces: [bugs]`
**When** loaded
**Then** it parses as `SlotSpec{Key: "bugs"}` with no template

**Given** a task YAML with full form `produces: [{key: bugs, template: "## Bugs\n- ..."}]`
**When** loaded
**Then** `SlotSpec.Template` contains the inline template

**Given** a task YAML with `produces: [{key: bugs, template_file: templates/bugs.md}]`
**When** `ResolveTemplate()` is called
**Then** returns the contents of `templates/bugs.md`

**Given** any existing task YAML with `produces: [key1, key2]`
**When** loaded after this change
**Then** parses without error (backward compatibility)

## Review Findings

### Patch (must fix)

- [x] [Review][Patch] Empty Key not validated in UnmarshalYAML — scalar path sets `s.Key = value.Value` without checking for `""`, silently emits blank key into DAG [slotspec.go:29] **HIGH**
- [x] [Review][Patch] ResolveTemplate path traversal — `filepath.Join(baseDir, s.TemplateFile)` does not contain result within baseDir; `../../etc/passwd` resolves cleanly [slotspec.go:49] **HIGH**
- [x] [Review][Patch] Duplicate default-produces logic — `buildTaskDAG` and `evaluateRuntimeOutputs` independently compute the same default, divergence risk if one changes [runtime_runner.go:152, 440] **MED**
- [x] [Review][Patch] No test for UnmarshalYAML default branch (unexpected YAML node kind) — coverage gap for error path [slotspec_test.go] **LOW**

### Defer

- [x] [Review][Defer] Template + TemplateFile both set silently drops Template — TemplateFile wins, undocumented precedence [slotspec.go:44] — deferred, document in story 6-20 when QualityGate wires this
- [x] [Review][Defer] ResolveTemplate never called in runtime path — Template/TemplateFile are dead until story 6-20 [runtime_runner.go:356–374] — deferred, by design (6-19 = model only)
- [x] [Review][Defer] Empty consumes[] on first node skips task_input injection — pre-existing DAG logic, not introduced here [runtime_runner.go:147] — deferred, pre-existing
- [x] [Review][Defer] LoadTasks doesn't bind TaskDefinition to source file for TemplateFile resolution — ambiguous baseDir for multi-directory loading [design.go] — deferred, story 6-20 caller responsibility
- [x] [Review][Defer] Unknown YAML fields silently ignored by gopkg.in/yaml.v3 — typo in template_file undetectable — deferred, library behavior
