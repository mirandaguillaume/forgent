# Story 10-10: Typed Node Output Schema

**Epic:** 10 — Node-Level Behavioral Contracts
**Status:** done

## Story

As a **developer composing Forgent agents**,
I want to declare a typed schema on a step's `produces` slots when that slot feeds another node,
So that inter-node communication is deterministic and typed — not free-text prose —
while steps that produce human-facing documents continue to use Markdown templates.

## Background

DAG nodes already have a typed signature in Go:
```go
RunFn: func(ctx context.Context, inputs map[string]any) (map[string]any, error)
```

**The fundamental asymmetry:** free text in input is fine — a LLM reads and understands
prose naturally. But free text as output between nodes is detrimental: a downstream node
doesn't "understand" the upstream node's prose, it extracts data from it. Unstructured text
at the inter-node boundary introduces non-determinism (format variance) and fragility
(extraction logic) that undermines safe agent composition.

This gives a clear rule:

```
consumes (input)  → free text OK      LLM reads and reasons over prose
produces → human  → template_file     Markdown document for a human reader
produces → node   → schema            Typed map, deterministic, no extraction needed
```

When a `produces` slot declares a `schema`, the runtime:
1. Uses constrained decoding (Anthropic Structured Outputs API) when available — the model
   cannot produce a token that violates the schema; format non-determinism is eliminated
2. Falls back to prompt injection + `extractJSON` for providers without constrained decoding
   (Ollama, OpenRouter) — accepts some format variance but rescues common wrapping patterns
3. Validates the parsed map against the declared `FieldSchema` (`ValidateTyped`)
4. Writes `slots/<key>.json` with the structured value
5. Injects the parsed map directly into the consuming step's inputs — no re-parsing

Steps without a `schema` continue to use the existing Markdown/template behavior (NFR40).

## Data Model

```go
// SlotSpec — extended with optional Schema field
type SlotSpec struct {
    Key          string      `yaml:"key"`
    TemplateFile string      `yaml:"template_file,omitempty"`
    Template     string      `yaml:"template,omitempty"`
    Schema       FieldSchema `yaml:"schema,omitempty"` // NEW — nil = Markdown behavior
}

// FieldSchema — maps field name → spec
type FieldSchema map[string]FieldSpec

// FieldSpec — type contract for a single output field
type FieldSpec struct {
    Type        string   `yaml:"type"`                  // "string" | "number" | "boolean"
    Enum        []string `yaml:"enum,omitempty"`         // if set, value must be one of these
    MinLength   int      `yaml:"min_length,omitempty"`   // for strings: minimum length
    Description string   `yaml:"description,omitempty"`  // shown to LLM in format instruction
}
```

### Example YAML

```yaml
steps:
  - id: review
    name: Review
    prompt: "Analyse this code and return a structured decision."
    produces:
      - key: verdict
        schema:
          ruling:
            type: string
            enum: [approved, rejected, needs-changes]
          severity:
            type: string
            enum: [low, medium, high, critical]
            description: "Impact level of the identified issues"
          rationale:
            type: string
            min_length: 20
            description: "Explanation of the ruling, citing specific code evidence"
```

## Acceptance Criteria

**AC1:** Given a step declares a `schema` on a `produces` slot,
When `buildTaskDAG` constructs the node's `runFn`,
Then if the provider implements `llm.StructuredProvider`, constrained decoding is used
(schema compiled to JSON Schema via `pkg/schema.ToJSONSchema`),
Otherwise a JSON format instruction is appended to the prompt (fallback via
`pkg/schema.ToPromptInstruction`), naming each field, its type/enum, and its description.

**AC2:** Given the LLM response contains a valid JSON object (optionally wrapped in a
` ```json ``` ` code fence),
When `extractJSON` processes the response,
Then the parsed `map[string]any` is returned with all declared fields present.

**AC3:** Given the parsed map satisfies the schema (correct types, enum values, min_length),
When `ValidateTyped` runs,
Then nil is returned and the slot is written as `slots/<key>.json`.

**AC4:** Given the parsed map fails validation (missing field, wrong enum value, string too short),
When `ValidateTyped` returns an error,
Then the node's `runFn` returns the error (triggering DAG retry via `runWithRetry`),
And `slots/<key>.json` is NOT written to the sandbox.

**AC5:** Given the LLM response cannot be parsed as JSON (malformed or no JSON object found),
When `extractJSON` fails,
Then the node's `runFn` returns an error with prefix `"json extraction: "` (triggering retry),
And the audit trail records an entry with `Action: "slot_write"`, `Allowed: false`,
and `Reason` containing the parse failure summary.

**AC6:** Given a consuming step declares `consumes` on a slot produced as `.json`,
When `buildTaskDAG` wires the DAG edge,
Then the typed map is read from `slots/<key>.json`, unmarshaled, and injected directly
into the step's `inputs` map under the slot key — no re-parsing needed.

**AC7:** Given a step declares NO `schema` on its `produces` slot (only `template_file` or nothing),
When the runtime executes,
Then the behavior is IDENTICAL to the pre-story behavior (Markdown/template validation,
`slots/<key>.md` slot files) — NFR40 backward compat.

**AC10 (human vs node rule):** Given a step produces a slot consumed by a human (final output,
report, document),
When the task YAML is written,
Then `template_file` is the correct declaration — NOT `schema`.
`schema` is reserved for slots that feed another node's `consumes`.
The runtime does NOT enforce this rule (it cannot know the consumer's nature); it is a
linting rule documented in `forgent lint` guidance and enforced by convention.

**AC8:** Given `ExperimentConfig.Validate()` runs on a task with a schema,
When a field spec has an unknown `type` value (not `"string"`, `"number"`, `"boolean"`),
Then `Validate()` returns a clear error naming the step, field, and the unsupported type.

**AC9:** Given a step produces a typed slot and the next step consumes it,
When `go test ./internal/bench/experiment/... -race` runs,
Then all existing tests pass (no regressions), the new typed-schema tests pass, and the
run is race-clean.

## Tasks/Subtasks

- [x] T1: Data model — `FieldSchema` and `FieldSpec` types
  - [x] T1.1: `pkg/schema/field.go` (new) — `FieldSchema`, `FieldSpec`; `SlotSpec.Schema schema.FieldSchema` added to `slotspec.go`.
  - [x] T1.2: `ExperimentConfig.Validate()` extended to reject unknown `FieldSpec.Type` values.
  - [x] T1.3: YAML round-trip tests in `slotspec_test.go` — valid schema unmarshals; unknown type triggers error.

- [x] T2: JSON extraction (`internal/bench/experiment/json_extract.go` — new file)
  - [x] T2.1: `extractJSON` — code-fence strip, bare unmarshal, string-aware brace scan.
  - [x] T2.2: 11 unit tests in `json_extract_test.go`.

- [x] T3: Schema validation (`pkg/qualitygate/typed.go` — new file)
  - [x] T3.1: `ValidateTyped(data map[string]any, s schema.FieldSchema) error`. Import cycle resolved via `pkg/schema` (Option A).
  - [x] T3.2: 9 unit tests in `pkg/qualitygate/typed_test.go`.

- [x] T4: Schema compilation and provider dispatch
  - [x] T4.1: `pkg/schema/compile.go` — `ToJSONSchema`, `ToPromptInstruction` (sorted keys, deterministic).
  - [x] T4.2: `runtime_runner.go` — `getStructuredProvider` helper peels `callCountingProvider`; format instruction appended to base prompt for non-StructuredProvider; `CompleteStructured` called for StructuredProvider.
  - [x] T4.3: `llm.StructuredProvider` interface added to `internal/llm/provider.go`.
  - [x] T4.4: `TestTypedSchema_ProviderDispatch_*` tests in `typed_schema_test.go`.

- [x] T5: Runtime wiring — produce typed slot
  - [x] T5.1–T5.2: Typed produce block in `runtime_runner.go` runFn — extractJSON/CompleteStructured, ValidateTyped, sandbox `.json` write, out map with parsed map.
  - [x] T5.3: `classifyValidationReason` extended: `"json extraction:"` → `"json extraction failed"`, `"typed schema: field"` → `"typed schema violation: <field>"`.
  - [x] T5.4: Pin tests `TestClassifyValidationReason_JSON*` and `TestClassifyValidationReason_Typed*` in `typed_schema_test.go`.

- [x] T6: Runtime wiring — consume typed slot
  - [x] T6.1: Consume-side sandbox writes detect `map[string]any` inputs → write `.json`; file listing uses correct extension per input type.
  - [x] T6.2: Typed inputs skip template validation automatically (`collectStrings` ignores non-string values). `consumesRetryMiddleware` clears both `.md` and `.json` variants.

- [x] T7: Integration tests (`internal/bench/experiment/typed_schema_test.go`)
  - [x] T7.1–T7.5: All 5 integration tests pass.

- [x] T8: Corpus review
  - [x] T8.1: All 16 existing tasks produce human-facing prose — no `template_file` → `schema` migration needed. Added `structured-verdict.yaml` as typed-schema demo task.
  - [x] T8.2: Full suite 2245 tests, 0 regressions.

- [x] T9: `go test ./... -race -count=1` — 2245 tests pass, race-clean.

### Review Findings

- [x] [Review][Patch] Skip text loop for typed-only StructuredProvider steps — call `CompleteStructured` directly from `acc.String()`; guardrails do not run (schema is the enforcement mechanism for typed outputs) [internal/bench/experiment/runtime_runner.go]
- [x] [Review][Patch] Add Validate() guard rejecting step with >1 typed produces slot — only `typedProducesSlots[0].Schema` is used for prompting/extraction but all slots are validated against the same parsed map; guaranteed silent failure when schemas differ [internal/bench/experiment/design.go]
- [x] [Review][Patch] `ToJSONSchema` drops `MinLength` — StructuredProvider path sends weaker contract than text-fallback path; add `"minLength": spec.MinLength` when `MinLength > 0` [pkg/schema/compile.go:19-29]
- [x] [Review][Patch] Add Validate() guard rejecting typed produces + declarative Loop with ExitWhen — exit criteria fire before the typed-produces block is reached; the step exhausts iterations and returns `ErrLoopExhausted` [internal/bench/experiment/design.go:670]
- [x] [Review][Patch] Add Validate() check rejecting `min_length` on non-string `FieldSpec` types — silently accepted then silently ignored by ValidateTyped [internal/bench/experiment/design.go:670]
- [x] [Review][Patch] `json.Marshal(parsed)` error silently ignored before slot write — propagate the error [internal/bench/experiment/runtime_runner.go:899]
- [x] [Review][Patch] Add Validate() guard rejecting step with both typed and Markdown produces slots — typed path returns early at line 926, bypassing Markdown produces validation at line 934 [internal/bench/experiment/runtime_runner.go:858-926]
- [x] [Review][Defer] `getStructuredProvider` only unwraps one wrapper layer — deeper nesting silently falls back to text extraction [internal/bench/experiment/runtime_runner.go:160] — deferred, no other wrapper types today
- [x] [Review][Defer] `callCountingProvider` not incremented for `CompleteStructured` calls — cost telemetry under-counts typed-produce steps using constrained decoding [internal/bench/experiment/runtime_runner.go:162] — deferred, accepted limitation
- [x] [Review][Defer] `ToJSONSchema` lacks `additionalProperties: false` — strict-mode providers may reject schema or allow unconstrained extra fields [pkg/schema/compile.go:34] — deferred, no real StructuredProvider in production yet
- [x] [Review][Defer] Retrievers only read `.md` slot files — typed `.json` slots silently skipped, no context injection for typed lineages [internal/bench/experiment/retriever.go] — deferred, retriever is best-effort context
- [x] [Review][Defer] `StructuredProvider` failure is fatal with no text-extraction fallback — transient API errors cause hard step failure [internal/bench/experiment/runtime_runner.go:871] — deferred, deliberate design (fail hard, preserve guarantee)
- [x] [Review][Defer] No real provider implements `StructuredProvider` — structured path is test-mock-only in production today [internal/llm/] — deferred, implementation tracked in future epic
- [x] [Review][Defer] Lint rule for `schema` = inter-node vs `template_file` = human-facing not yet in `forgent lint` — deferred, planned for lint epic
- [x] [Review][Defer] `ValidateTyped` iterates schema map in non-deterministic order — per-field violation telemetry varies across runs [pkg/qualitygate/typed.go:13] — deferred, validation result is deterministic even if error message varies
- [x] [Review][Defer] `consumesRetryMiddleware` passes empty inputs to typed producers on recovery — re-produced value may diverge semantically from original [internal/bench/experiment/runtime_runner.go:95] — deferred, pre-existing design documented in code comment
- [x] [Review][Defer] `extractJSON` brace scan tries first `{` in string — fails if preamble contains `{..}` before real JSON [internal/bench/experiment/json_extract.go:47] — deferred, real LLM outputs rarely hit this pattern
- [x] [Review][Defer] Retry prefix not included in `CompleteStructured` call — structured retries via DAG `runWithRetry` carry no failure context [internal/bench/experiment/runtime_runner.go:870] — deferred, DAG provides the retry; context enrichment is a future enhancement
- [x] [Review][Defer] `extractJSON` has no input size limit — large LLM responses scanned char-by-char [internal/bench/experiment/json_extract.go:17] — deferred, experiment harness context; real DoS protection belongs at the budget layer

## Dev Notes

### Import cycle resolution

`pkg/qualitygate` must not import `internal/bench/experiment`. If `FieldSchema`/`FieldSpec`
are defined in `internal/bench/experiment/design.go`, `ValidateTyped` cannot import them directly.
Two options:
- **Option A**: Move `FieldSchema`/`FieldSpec` to `pkg/schema` (new zero-dep package). Both
  `pkg/qualitygate` and `internal/bench/experiment` import `pkg/schema`.
- **Option B**: Pass a `map[string]any` representation of the schema to `ValidateTyped` and
  reconstruct the FieldSpec from it. Avoids the new package but loses type safety.

Prefer Option A. Create `pkg/schema/field.go` with `FieldSchema` and `FieldSpec`. Update
`design.go` to use `schema.FieldSchema`.

### JSON extraction strategy

LLMs rarely produce a bare JSON object when prompted in prose. Common wrapping:
```
Here is the structured verdict:
```json
{"ruling": "approved", "severity": "medium", "rationale": "The code is clean."}
```
```

`extractJSON` must handle this. The strategy: try ` ```json ``` ` strip first, then
bare `json.Unmarshal`, then brace-scan. Never fail silently — always return an error
with `"json extraction:"` prefix so `classifyValidationReason` catches it.

### Prompt injection wording

The injected format instruction must be at the END of the prompt (after the task content)
so the LLM's last instruction is the format constraint. Placing it before the task
description causes LLMs to focus on the format and ignore the task content. Empirically,
trailing format instructions have higher compliance rates.

### Slot file extension

Typed slots → `slots/<key>.json`. Markdown slots → `slots/<key>.md`. The consuming step
detects which to read by checking whether `slot.Schema` is non-nil. No ambiguity — the
schema flag in the YAML drives the extension, not filename detection.

### Backward compatibility (NFR40)

- Steps with no `Schema` on any `produces` slot → zero behavior change. `extractJSON` is never
  called. Slot files are `.md`. Template/REQUIRED validation runs as before.
- Mixed task: some steps produce typed slots, others produce Markdown slots → both paths
  coexist within the same task run.

### When to use `schema` vs `template_file`

| Produces destination | Declaration | Slot file | Validation |
|---|---|---|---|
| Another node (inter-node) | `schema` | `slots/<key>.json` | `ValidateTyped` + constrained decoding |
| A human (final output) | `template_file` | `slots/<key>.md` | Heuristic markers + REQUIRED |
| Unstructured pass-through | _(neither)_ | `slots/<key>.md` | None |

**Rule:** if the slot feeds a `consumes` on another step → `schema`. If it is a document,
report, or narrative for a human reader → `template_file`. The runtime does not enforce
this distinction; `forgent lint` should warn when a `template_file` slot is consumed by
another step without type annotation.

### Relationship to stories 10-4 and 10-10 (original)

Story 10-4 introduced `ValidateProduces` for Markdown outputs (REQUIRED markers, heuristic
structural checks). That validation remains valid and unchanged for human-facing outputs
(`template_file` slots). This story adds `ValidateTyped` for inter-node typed outputs
(`schema` slots) — a parallel path, not a replacement.

The original 10-10 (section-aware REQUIRED markers) is superseded: for inter-node slots,
`schema` replaces word-boundary checking entirely. For human-facing slots, `template_file`
+ existing heuristics remain correct.

### Telemetry hook (story 10-6)

New stable audit Reason prefixes introduced by this story:
- `"json extraction failed"` — LLM output had no parseable JSON
- `"typed schema violation: <field>"` — field missing, wrong type, or enum mismatch

Story 10-6's tally must recognize these prefixes. Document in 10-6 story file.

### Files created or modified

- `pkg/schema/field.go` (new) — `FieldSchema`, `FieldSpec`
- `pkg/qualitygate/typed.go` (new) — `ValidateTyped`
- `pkg/qualitygate/typed_test.go` (new)
- `internal/bench/experiment/design.go` — `SlotSpec.Schema field.FieldSchema`
- `internal/bench/experiment/json_extract.go` (new) — `extractJSON`
- `internal/bench/experiment/json_extract_test.go` (new)
- `internal/bench/experiment/runtime_runner.go` — prompt injection, typed produce/consume wiring, `classifyValidationReason` extension
- `internal/bench/experiment/output_contract_test.go` — T5.4 pin tests
- `internal/bench/experiment/typed_schema_test.go` (new)
- Experiment corpus YAML files (T8.1)

### What NOT to do

- **Don't use full JSON Schema (draft-07).** `FieldSchema` is intentionally minimal — type,
  enum, min_length. Full JSON Schema brings `$ref`, `anyOf`, `allOf`, `additionalProperties` —
  complexity that isn't justified for an experiment harness. Extend only when a concrete
  story needs it.
- **Don't make schema mandatory.** Markdown-output steps are first-class citizens. Typed schema
  is an opt-in upgrade for steps that produce structured decisions.
- **Don't validate field ORDER.** JSON objects are unordered. Do not check that fields appear
  in schema declaration order in the output.
- **Don't use `schema` for human-facing outputs.** A step that produces a code review report
  for a developer should use `template_file`, not `schema`. Typed schemas are for inter-node
  data flow — not for documents. Mixing the two means the LLM is instructed to produce JSON
  where a human expects prose.
- **Don't skip the `StructuredProvider` interface check.** Hardcoding the Anthropic-specific
  API call in `runtime_runner.go` would break Ollama and OpenRouter tests. Always dispatch
  via the interface — fallback path must work correctly for the benchmark.

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

_empty_

### Completion Notes List

- `pkg/schema/field.go` (new): `FieldSchema map[string]FieldSpec`, `FieldSpec{Type, Enum, MinLength, Description}`. Zero-dep package, solves import cycle (Option A from Dev Notes).
- `pkg/schema/compile.go` (new): `ToJSONSchema` (JSON Schema object for constrained decoding), `ToPromptInstruction` (natural-language fallback). Both sort keys for determinism.
- `internal/llm/provider.go`: `StructuredProvider` interface added with `CompleteStructured(ctx, prompt, jsonSchema) (map[string]any, error)`.
- `slotspec.go`: `Schema schema.FieldSchema` field added to `SlotSpec`. YAML decoding handled by existing `slotAlias` pattern (map type decoded natively).
- `design.go`: `ExperimentConfig.Validate()` extended — rejects unknown `FieldSpec.Type` per-slot, per-field.
- `json_extract.go` (new): `extractJSON` — code-fence, bare, brace-scan (string-aware depth counter). Always returns `"json extraction:"` prefix errors.
- `pkg/qualitygate/typed.go` (new): `ValidateTyped` — missing field, wrong type, enum violation, min_length. Uses `pkg/schema.FieldSchema`.
- `runtime_runner.go`: `getStructuredProvider` helper peels `callCountingProvider` wrapper. Typed produce block: StructuredProvider path (CompleteStructured once) and fallback path (format instruction + extractJSON). Consume side: `map[string]any` inputs written as `.json`; file listing uses correct extension. `classifyValidationReason` extended for two new prefixes. `evaluateRuntimeOutputs` handles `map[string]any` outputs. `consumesRetryMiddleware` clears both `.md` and `.json`.
- Corpus: 16 existing tasks produce prose (no migration). Added `structured-verdict.yaml` demo task.
- 2245 tests, 0 regressions, race-clean.

### File List

- `pkg/schema/field.go` (new)
- `pkg/schema/compile.go` (new)
- `internal/llm/provider.go` — `StructuredProvider` interface
- `internal/bench/experiment/slotspec.go` — `SlotSpec.Schema` field
- `internal/bench/experiment/design.go` — `Validate()` extended
- `internal/bench/experiment/json_extract.go` (new)
- `internal/bench/experiment/json_extract_test.go` (new)
- `internal/bench/experiment/slotspec_test.go` — T1.3 tests appended
- `internal/bench/experiment/typed_schema_test.go` (new)
- `internal/bench/experiment/runtime_runner.go` — typed produce/consume, classifyValidationReason, evaluateRuntimeOutputs
- `internal/bench/experiment/tasks/structured-verdict.yaml` (new)
- `pkg/qualitygate/typed.go` (new)
- `pkg/qualitygate/typed_test.go` (new)

## Change Log

| Date | Change | Notes |
|---|---|---|
| 2026-04-28 | Story 10-10 rewritten — "Typed Node Output Schema" | Supersedes original "section-aware REQUIRED markers" scope. Motivated by party mode consensus: steps are functions; outputs are typed return values, not Markdown documents. Prerequisite for 10-9 experiment rerun. |
| 2026-04-28 | Amended — human vs node distinction formalized | Free text input OK; free text inter-node output is detrimental (non-determinism). Rule: schema = inter-node, template_file = human-facing. StructuredProvider interface added for constrained decoding dispatch. Description field added to FieldSpec. |
