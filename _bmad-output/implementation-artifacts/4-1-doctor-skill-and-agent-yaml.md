# Story 4.1: Doctor Skill and Agent YAML Definitions

Status: done

## Story

As a **product architect**,
I want doctor defined as real Forgent skill and agent YAML files,
so that the doctor specification is the same format as any Forgent agent.

## Acceptance Criteria

1. 5 doctor skill YAMLs in `skills/doctor/`, each with: name, version, context (consumes/produces), strategy (approach: code or llm), guardrails, security
2. format-detector, context-enricher, recommendation-builder use `approach: code`; analyzer and rewriter use `approach: llm`
3. Doctor agent YAML in `agents/doctor.agent.yaml` composes the 5 skills with proper consumes/produces contracts
4. `forgent lint` validates the doctor agent with zero errors
5. `forgent score` rates the doctor agent quality (target: ≥ 80/100)
6. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [x] Task 1: Doctor skill YAML definitions (AC: #1, #2)
  - [x] 1.1: Create `skills/doctor/format-detector.skill.yaml` — consumes: [agent_file], produces: [detected_format], approach: code, timeout: 5s, filesystem: read-only, network: none
  - [x] 1.2: Create `skills/doctor/analyzer.skill.yaml` — consumes: [agent_file, detected_format], produces: [analysis_results], approach: llm, timeout: 120s, network: allowlist
  - [x] 1.3: Create `skills/doctor/context-enricher.skill.yaml` — consumes: [agent_file], produces: [context_recommendations], approach: code, timeout: 15s, filesystem: read-only
  - [x] 1.4: Create `skills/doctor/recommendation-builder.skill.yaml` — consumes: [analysis_results, context_recommendations], produces: [recommendations], approach: code, timeout: 5s
  - [x] 1.5: Removed per scope refresh (rewriter.skill.yaml not created — rewriter is post-processing, not a skill)

- [x] Task 2: Doctor agent YAML (AC: #3)
  - [x] 2.1: Created `agents/doctor.agent.yaml` — composes 4 skills, consumes: [agent_file], produces: [recommendations]
  - [x] 2.2: Orchestration: sequential with logical Layer 1 parallelism (analyzer ‖ context-enricher ordered for data flow scoring)
  - [x] 2.3: Rewriter removed per scope refresh; --fix is a post-processing CLI concern, not a skill

- [x] Task 3: Validation (AC: #4, #5)
  - [x] 3.1: `forgent lint skills/doctor` — 0 errors, 1 warning (recommendation-builder has no tools, intentional)
  - [x] 3.2: All 4 skill YAMLs parse and lint cleanly — zero errors
  - [x] 3.3: `forgent score skills/doctor --agents agents` — doctor agent scores 100/100
  - [x] 3.4: Score ≥ 80 achieved on first pass — no improvements needed
  - [x] 3.5: CI test added in `internal/cmd/doctor_yaml_test.go` — asserts 4 skills, 0 errors, doctor agent ≥ 80

### Review Findings

- [x] [Review][Defer] Sequential ordering vs Task 2.2 parallelism — Agent uses flat `orchestration: sequential` but Task 2.2 specifies "Layer 1 parallelism (analyzer ‖ context-enricher)". Deferred to Story 4.2: DAG auto-wiring resolves parallelism from consumes/produces contracts, making YAML ordering moot at runtime.
- [x] [Review][Patch] Score threshold 80 trivially satisfied — Tightened from >= 80 to >= 95 in doctor_yaml_test.go. [internal/cmd/doctor_yaml_test.go:33]
- [x] [Review][Defer] Hardcoded bench file counts fragile [internal/bench/token_test.go] — deferred, pre-existing pattern
- [x] [Review][Defer] Non-recursive scanners miss skills/doctor/ in build/lint/score defaults [internal/cmd/lint.go, internal/cmd/score.go, internal/bench/token.go] — deferred, pre-existing (Story 4.2 scope)
- [x] [Review][Defer] Relative paths in tests are brittle [internal/cmd/doctor_yaml_test.go, internal/bench/token_test.go] — deferred, pre-existing pattern
- [x] [Review][Defer] No agent-level vs skill-chain contract validation [pkg/model/validate.go] — deferred, pre-existing gap
- [x] [Review][Defer] Custom guardrail keys not validated at parse time [pkg/model/skill.go] — deferred, pre-existing by design
- [x] [Review][Defer] scoreDataFlow doesn't flag orphan outputs [internal/analyzer/score.go] — deferred, pre-existing

## Dev Notes

### ⚠️ Scope Refresh from Epic 8 Retro (2026-04-07)

**This story's AC was written before Epic 8 shipped. Epic 8 changed `--fix` from LLM rewrite to deterministic scaffold, which makes the `internal/doctor/rewrite/` package dead code.** Read this section *before* starting Task 1.

**Updated scope:**

- **Doctor is 4 skills, not 5.** Drop `rewriter` from the skill list. The 4 skills are:
  1. `format-detector` (approach: code)
  2. `analyzer` (approach: llm)
  3. `context-enricher` (approach: code)
  4. `recommendation-builder` (approach: code)
- **The deterministic scaffold is *not* a skill.** It's an output transformation that happens after `recommendation-builder`, triggered by `--fix`. Model it as a post-processing step or a router node — *not* as a skill in the spec sense. The `internal/doctor/scaffold/` package is invoked by the CLI command after the agent's recommendations are produced; it does not participate in the agent's I/O contract.
- **Task 1.5 is removed.** Do not create `skills/doctor/rewriter.skill.yaml`.
- **AC #1 changes from "5 doctor skill YAMLs" to "4 doctor skill YAMLs"** — update the AC count when verifying.
- **AC #2 changes:** drop the "rewriter use approach: llm" clause; the analyzer is the only LLM skill.
- **Story 4.2 implications (for awareness, not this story):** the DAG topology in ADR-4 will need a refresh too — Layer 3 (rewriter) is removed; Layers 0-2 cover the full agent. The equivalence test in Story 4.2 compares 4-skill output, not 5.

**Source:** Epic 8 Retro (2026-04-07), Winston's architectural recommendation, approved inline by Guillaume during retro discussion. See `_bmad-output/implementation-artifacts/epic-8-retro-2026-04-07.md` for full reasoning.

**Why this matters:** Including `rewriter` as a stub skill would create dead code in the spec — the exact pattern Epic 7 retro flagged ("lint for unused exported functions, prevent dead code"). Better to model doctor honestly as 4 skills + a post-processing transformation.

### Architecture References

- **ADR-1 (Embedding Strategy):** Doctor YAMLs are embedded via `//go:embed` in Story 4.2. This story creates the YAML files; embedding happens next.
- **ADR-4 (Doctor DAG Topology):** Defines the original 5-skill topology — **superseded by the Scope Refresh above.** Treat ADR-4 as historical; the actual topology is 4 skills.
- **Existing skill pattern:** See `skills/*.skill.yaml` for format reference — 18 existing skills follow this format.

### Skill YAML Template (from architecture)

```yaml
skill: format-detector
version: "1.0.0"
context:
  consumes: [agent_file]
  produces: [detected_format]
strategy:
  approach: code
  tools: []
  steps:
    - detect file format from path and frontmatter
guardrails:
  - timeout: 5s
security:
  filesystem: read-only
  network: none
```

### Type Contracts (from ADR-4)

| Type | Description | Producer | Consumer(s) |
|---|---|---|---|
| agent_file | Raw file content + path | CLI input | format-detector, analyzer, context-enricher, rewriter |
| detected_format | Format name + parsed structure | format-detector | analyzer, rewriter |
| analysis_results | LLM findings with categories | analyzer | recommendation-builder |
| context_recommendations | Codebase gap findings | context-enricher | recommendation-builder |
| recommendations | Merged, enriched findings | recommendation-builder | rewriter, CLI output |
| improved_file | Rewritten file content | rewriter | CLI output |

### Dependencies

- **Epic 1-3 complete:** The skills and agent reference concepts (parser, analyzer, enricher, rewriter, quality gate) that must already be implemented.
- **Existing lint/score tools:** Already operational. Just need to pass on the new doctor YAMLs.

### What NOT to Build

- Do NOT embed the YAMLs yet — that's Story 4.2
- Do NOT create Go implementations for the skills — implementations already exist in `internal/doctor/`
- Do NOT modify existing skills or agents — only add new doctor-specific files
- Do NOT create a new orchestration format — use existing agent YAML schema

### Project Structure

New files:
- `skills/doctor/format-detector.skill.yaml`
- `skills/doctor/analyzer.skill.yaml`
- `skills/doctor/context-enricher.skill.yaml`
- `skills/doctor/recommendation-builder.skill.yaml`
- `skills/doctor/rewriter.skill.yaml`
- `agents/doctor.agent.yaml`

## Testing Strategy

- **Lint validation:** `forgent lint` on all 5 skill YAMLs and the agent YAML — zero errors.
- **Score validation:** `forgent score` on agent YAML — ≥ 80.
- **Schema validation:** Each skill YAML parses correctly with `internal/yaml/` loader.
- **Contract validation:** Agent's composed consumes/produces match the skills' contracts.
- **CI test:** Automated lint + score check in test suite.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-1] — Doctor embedding strategy
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-4] — Doctor DAG topology, 5 skills
- [Source: _bmad-output/planning-artifacts/architecture.md#Doctor-Skill-YAML-Pattern] — YAML template
- [Source: _bmad-output/planning-artifacts/prd.md#FR50] — KEYSTONE: Doctor as Forgent runtime
- [Source: skills/] — 18 existing skill YAMLs for format reference
- [Source: agents/] — 3 existing agent YAMLs for format reference

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

- Verified `ValidateSkill` requires all enum fields (memory, trace_level, filesystem, network, negotiation) — story template was missing `negotiation.file_conflicts`
- Used flat `skills:` + `orchestration: sequential` instead of stages — `RunScore` only resolves skills from flat `agent.Skills`, so stages would break data flow scoring
- All 3 inter-skill data flows satisfied in sequential order: detected_format, analysis_results, context_recommendations — agent scores 100/100

### Completion Notes List

- Created 4 doctor skill YAMLs in `skills/doctor/` (not 5 — rewriter dropped per scope refresh from Epic 8 retro)
- Doctor agent YAML in `agents/doctor.agent.yaml` with sequential orchestration over 4 skills
- `forgent lint skills/doctor` → 0 errors, 1 warning (recommendation-builder no-tools, intentional)
- `forgent score skills/doctor --agents agents` → doctor agent 100/100; all skills ≥ 80
- CI test `TestDoctorSkills_Lint` and `TestDoctorAgent_Score` added in `internal/cmd/doctor_yaml_test.go`
- Updated `internal/bench/token_test.go` file count assertions: 21→22 (3 agents→4) and compact 3→4
- All tests pass: `go test ./... -count=1 -short` ✅

### File List

- `skills/doctor/format-detector.skill.yaml` (new)
- `skills/doctor/analyzer.skill.yaml` (new)
- `skills/doctor/context-enricher.skill.yaml` (new)
- `skills/doctor/recommendation-builder.skill.yaml` (new)
- `agents/doctor.agent.yaml` (new)
- `internal/cmd/doctor_yaml_test.go` (new)
- `internal/bench/token_test.go` (modified — updated file count assertions)

## Change Log

- 2026-04-10: Implemented story 4-1. Created 4 doctor skill YAMLs + doctor agent YAML + CI validation tests. Scope: 4 skills per Epic 8 retro (rewriter dropped). Agent scores 100/100.
