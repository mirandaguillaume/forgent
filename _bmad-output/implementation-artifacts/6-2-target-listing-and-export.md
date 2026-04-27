# Story 6.2: Target Listing and Export

Status: ready-for-dev

## Story

As a **developer**,
I want to see what format targets are available and export doctor's analysis as YAML,
so that I can discover capabilities and bridge into the compiler when ready.

## Acceptance Criteria

1. `forgent targets` lists all registered target generators and format parsers
2. Contributed targets are included if registered via `init()`
3. `forgent doctor agent.md --export-yaml` outputs the internal analysis as Forgent skill YAML files
4. Exported YAML is valid and can be compiled with `forgent build`
5. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Target listing command (AC: #1, #2)
  - [ ] 1.1: Create `internal/cmd/targets.go` — `targetsCmd` Cobra command
  - [ ] 1.2: List registered target generators from `pkg/spec.RegisteredGenerators()` (or equivalent)
  - [ ] 1.3: List registered format parsers from `parser.RegisteredFormats()`
  - [ ] 1.4: TTY mode: formatted table with name, type (generator/parser), description. Plain mode: simple list.
  - [ ] 1.5: Write test verifying at least 3 generators (claude, copilot, forgent) and 3 parsers (claude, copilot, forgent) appear

- [ ] Task 2: Export YAML command (AC: #3, #4)
  - [ ] 2.1: Add `--export-yaml` flag to doctor command
  - [ ] 2.2: Create `internal/doctor/export.go` — `ExportAsYAML(findings []llmutil.Finding, analysis *parser.AgentAnalysis, format string) ([]byte, error)`
  - [ ] 2.3: Generate a Forgent skill YAML from the analysis: map agent file structure to skill facets (context from sections, strategy from tools, guardrails from constraints)
  - [ ] 2.4: Include findings as observability annotations in the YAML
  - [ ] 2.5: Output to stdout (pipe-friendly) or to file if `--output <path>` specified
  - [ ] 2.6: Validate exported YAML with `forgent lint` — must pass
  - [ ] 2.7: Write tests: export from Claude file → valid YAML, export from Copilot file → valid YAML

- [ ] Task 3: Generator registry query (AC: #1, #2)
  - [ ] 3.1: Add `RegisteredGenerators() []string` to `pkg/spec/` if not already present
  - [ ] 3.2: Ensure both generators and parsers follow the same registration pattern
  - [ ] 3.3: Write test that adding a mock generator makes it appear in `forgent targets`

## Dev Notes

### Architecture References

- **FR54:** "Contributor can add a new target generator by implementing TargetGenerator and registering via init()."
- **FR55:** "Developer can list all available target generators via forgent targets."
- **FR56:** "Developer can export doctor's analysis as Forgent skill YAML files."
- **Existing plugin pattern:** `pkg/spec.Register()` for generators, `parser.Register()` for parsers.

### Export YAML Mapping

The export maps agent file analysis to Forgent skill YAML:

```yaml
skill: <derived from filename>
version: "1.0.0"
context:
  consumes: [<inferred from agent's tool declarations>]
  produces: [<inferred from agent's output sections>]
strategy:
  approach: llm
  tools: <from frontmatter tool declarations>
  steps: <from section headers>
guardrails: <from findings with category "guardrails">
security: <from findings with category "security">
observability:
  doctor_findings: <summary of all findings>
```

### Dependencies

- **Epic 1-2:** Parser registry, analyzer, doctor command.
- **Existing compiler:** `forgent lint` and `forgent build` for validating exported YAML.

### What NOT to Build

- Do NOT implement `forgent targets --add` — contributors add targets via code, not CLI
- Do NOT implement format conversion (Claude → Copilot) — export is one-way to Forgent YAML
- Do NOT implement batch export for directories — single file only for MVP
- Do NOT implement import from exported YAML — that's the existing `forgent import` command

### Project Structure

New files:
- `internal/cmd/targets.go` — targetsCmd
- `internal/cmd/targets_test.go`
- `internal/doctor/export.go` — ExportAsYAML
- `internal/doctor/export_test.go`

Modified files:
- `internal/cmd/doctor.go` — `--export-yaml` flag
- `pkg/spec/generator.go` — RegisteredGenerators() if not present

## Testing Strategy

- **Target listing tests:** Verify known generators and parsers appear. Mock additional registrations appear.
- **Export tests:** Export from Claude file → YAML is valid, parseable, passes lint. Export from Copilot file → same.
- **Round-trip test:** Export YAML → `forgent build --target claude` → valid output.
- **TTY/plain tests:** Formatted table in TTY, plain list in pipe.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/prd.md#FR54-FR56] — Target listing, export
- [Source: _bmad-output/planning-artifacts/architecture.md] — Plugin pattern for generators and parsers
- [Source: pkg/spec/generator.go] — Register(), existing generator registry

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
