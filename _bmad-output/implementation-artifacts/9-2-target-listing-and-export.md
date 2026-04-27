# Story 9-2: Target Listing and Export

Status: done

## Story

As a **developer**,
I want to see what format targets are available and export doctor's analysis as YAML,
so that I can discover capabilities and bridge into the compiler when ready.

## Acceptance Criteria

1. `forgent targets` lists all registered build targets (`spec.Available()`) and format parsers (`parser.RegisteredFormats()`)
2. `forgent doctor <file> --export-yaml` outputs the parsed analysis as a valid Forgent skill YAML on stdout
3. The exported YAML has the minimum required fields: `name`, `version`, `context`, `strategy` and is accepted by `forgent build`
4. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [x] Task 1: `forgent targets` command (AC: #1)
  - [x] 1.1: Create `internal/cmd/targets.go` — `targetsCmd` registered with `rootCmd`
  - [x] 1.2: Print registered build targets from `spec.Available()` with header "Build targets:"
  - [x] 1.3: Print registered format parsers from `parser.RegisteredFormats()` with header "Format parsers:"
  - [x] 1.4: Write tests verifying output contains known target names (claude, copilot, forgent) and parser names

- [x] Task 2: `--export-yaml` flag on doctor command (AC: #2, #3)
  - [x] 2.1: Create `internal/doctor/export/export.go` — `ToSkillYAML(analysis *parser.AgentAnalysis, filePath string) ([]byte, error)`
  - [x] 2.2: `ToSkillYAML` builds a `model.SkillBehavior` from the analysis with all required enum defaults
  - [x] 2.3: Serialize using `yaml.v3` and return bytes
  - [x] 2.4: Add `--export-yaml` flag to `doctorCmd` in `internal/cmd/doctor.go`
  - [x] 2.5: When `--export-yaml` set: parse the file, call `ToSkillYAML`, write to `cmd.OutOrStdout()`, return nil
  - [x] 2.6: `--export-yaml` is mutually exclusive with `--fix` and directory targets (error if combined)
  - [x] 2.7: Write tests for `ToSkillYAML` covering: name extraction, fallback to filename, description extraction, YAML validity

## Dev Notes

### Architecture References

- **FR56:** "Developer can list available build targets and format parsers."
- **spec.Available():** `pkg/spec/generator.go:82` — returns sorted list of registered target names.
- **parser.RegisteredFormats():** `internal/doctor/parser/registry.go:54` — returns sorted list of registered parser names.
- **model.SkillBehavior:** `pkg/model/skill.go` — the canonical skill spec struct serialized to YAML.

### `forgent targets` Command

Minimal new command. No flags needed. Output format:

```
Build targets:
  claude
  copilot
  forgent

Format parsers:
  claude
  copilot
  forgent
```

Generators are registered via `init()` in `internal/generator/{claude,copilot,forgent}/`. Parsers are registered in `internal/doctor/parser/{claude,copilot,forgent}.go`. Both are imported by `cmd/forgent/main.go`.

### `ToSkillYAML` — Name Extraction

Priority:
1. `frontmatter["name"]` (string)
2. `frontmatter["description"]` first word(s) sanitized as slug
3. Fallback: filename without extension, normalized (hyphens/underscores to spaces, title-cased)

```go
// Fallback: clean filename → name
name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
name = strings.ReplaceAll(name, "-", " ")
name = strings.ReplaceAll(name, "_", " ")
```

### `ToSkillYAML` — Minimum Valid Output

```yaml
name: my-agent
version: "0.1.0"
context:
  description: "..."
strategy:
  effort: medium
```

This is the minimum `model.SkillBehavior` that satisfies the YAML loader. The `forgent build` command will compile it into target-specific files.

### What NOT to Build

- Do NOT implement full semantic mapping of all AgentAnalysis sections to skill facets — minimum viable extraction only
- Do NOT implement `--export-yaml` for directory mode — single file only
- Do NOT implement custom output path for export — stdout only (pipe-friendly)
- Do NOT implement `--format` for targets command — plain text only

### Project Structure

New files:
- `internal/cmd/targets.go` — targetsCmd
- `internal/cmd/targets_test.go` — targets command output tests
- `internal/doctor/export/export.go` — ToSkillYAML
- `internal/doctor/export/export_test.go` — unit tests for export

Modified files:
- `internal/cmd/doctor.go` — add `--export-yaml` flag + handling block

## Testing Strategy

- **Targets output test:** Capture stdout from `targetsCmd`, assert contains "claude", "copilot", "forgent" for both sections.
- **ToSkillYAML tests:** Test name extraction from frontmatter, fallback to filename, description extraction, YAML round-trip parses without error.
- **Export flag integration:** Call the doctor RunE with `--export-yaml` set on a temp file, assert stdout is valid YAML with expected `name` field.
- **Mutual exclusion test:** Combining `--export-yaml` with `--fix` returns an error.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/prd.md#FR56] — Target listing and export
- [Source: pkg/spec/generator.go] — spec.Available()
- [Source: internal/doctor/parser/registry.go] — parser.RegisteredFormats()
- [Source: pkg/model/skill.go] — model.SkillBehavior struct

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

_empty_

### Completion Notes List

Implemented `forgent targets` command — lists `spec.Available()` (build targets) and `parser.RegisteredFormats()` (format parsers) using `cmd.OutOrStdout()` for testability. Created `internal/doctor/export` package with `ToSkillYAML` that maps `AgentAnalysis` → `model.SkillBehavior` with all required enum defaults (memory, trace_level, filesystem, network, file_conflicts) so the output passes `model.ValidateSkill`. Name extracted from frontmatter `name` key, falling back to sanitized filename. Description/instruction placed in `strategy.steps[0]`, truncated at 500 chars. Added `--export-yaml` flag to doctor command — mutually exclusive with `--fix` and directory targets. 14 new tests (3 targets, 7 export unit, 4 export integration). 2055 total passing.

### File List

- `internal/cmd/targets.go` — new: targetsCmd
- `internal/cmd/targets_test.go` — new: 3 tests
- `internal/doctor/export/export.go` — new: ToSkillYAML, build, nameFromPath
- `internal/doctor/export/export_test.go` — new: 7 unit tests
- `internal/cmd/doctor_export_test.go` — new: 4 integration tests
- `internal/cmd/doctor.go` — modified: export import, exportYAMLFlag, --export-yaml handling block, flag registration

### Review Findings

| ID | Title | Class | Sev | File |
|----|-------|-------|-----|------|
| P1 | `nameFromPath` dotfile (`.env`) strips all → `skill: ""` emitted without error | patch | HIGH | `export.go:77` |
| P2 | `ToSkillYAML(nil, …)` panics — no nil guard in `build()` | patch | MEDIUM | `export.go:20` |
| P3 | `--export-yaml` + `--update-registry` silently ignores registry update | patch | MEDIUM | `doctor.go` |
| P4 | `_, _ = cmd.OutOrStdout().Write(yamlBytes)` — write error silently discarded | patch | LOW | `doctor.go` |
| P5 | `nameFromPath` over-strips multi-dot names (`a.b.c.md` → `a`) | patch | LOW | `export.go:77` |
| P6 | `content[:500]` byte truncation can split a UTF-8 multi-byte rune | patch | LOW | `export.go:59` |
| D1 | Config var mutation across test invocations (pre-existing, story 9-1 scope) | defer | MEDIUM | `cmd/` tests |

Dismissed (12): missing "Format parsers:" header test (already tested), test file naming convention, missing `context.description` in exported YAML, `cmd.OutOrStdout()` testability pattern, unused `fconfig` import risk, Acceptance Auditor F1/F3/F5 false positives (test files present in diff), and 5 minor style findings.

## Change Log

- 2026-04-27: Story 9-2 implemented. `forgent targets` + `forgent doctor --export-yaml`. 14 new tests, 2055 total passing.
- 2026-04-27: Code review complete. 0 decision-needed, 6 patch, 1 defer, 12 dismissed. All 6 patches applied. 5 new tests (P1+P2+P3+P5+P6). 2060 total passing.
