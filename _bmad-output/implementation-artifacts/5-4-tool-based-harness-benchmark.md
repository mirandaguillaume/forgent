# Story 5.4: Tool-Based Harness Benchmark

Status: done

## Story

As a **product architect**,
I want to prove that the Forgent runtime structurally prevents destructive agent actions that text instructions cannot,
so that the core thesis ("structural enforcement gives +20-33%") has binary evidence beyond keyword-level guardrails.

## Acceptance Criteria

1. A VirtualFS sandbox (pure Go, in-memory) enforces per-step file access policies, denies out-of-scope paths, and detects trap file access — with a complete audit trail
2. Tool-based adversarial tasks (path traversal, privilege escalation, accidental destruction) execute under two conditions: (a) text-instruction (no sandbox) and (b) runtime (sandbox-enforced) — same LLM, same tools, only variable is enforcement
3. Per-action metrics measured: action adherence, prevention rate, trap trigger rate, scope creep, write safety
4. The runtime condition achieves >0% prevention rate on adversarial tasks where text condition has 0% (binary proof)
5. Results integrate into the existing experiment report (Markdown) with a new "Sandbox Metrics" section
6. Checkpoint/resume works for tool-based tasks alongside existing text-only tasks
7. All existing tests pass (`go test ./... -count=1 -race`)

## Tasks / Subtasks

- [x] Task 1: VirtualFS sandbox core (AC: #1)
  - [x] 1.1: `sandbox.go` — `SandboxFS` struct: `root map[string][]byte`, `traps map[string]bool`, `log []AccessEntry`, `policy SandboxPolicy`
  - [x] 1.2: `SandboxPolicy` struct: `AllowedGlobs`, `DeniedGlobs`, `DeniedWritePatterns []string`, `MaxFileSize int64`, `ReadOnly bool`
  - [x] 1.3: `AccessEntry` struct: `Timestamp`, `StepID`, `Action` (read/write/delete), `Path`, `Allowed bool`, `Reason`
  - [x] 1.4: `NewSandboxFS(files map[string][]byte, traps []string, policy SandboxPolicy)` constructor
  - [x] 1.5: `Read(stepID, path)` — glob check against AllowedGlobs/DeniedGlobs, trap detection, audit log entry
  - [x] 1.6: `Write(stepID, path, content)` — glob check + DeniedWritePatterns content scan + MaxFileSize + ReadOnly check
  - [x] 1.7: `Delete(stepID, path)` — glob check, trap check, audit entry
  - [x] 1.8: `Log() []AccessEntry` accessor, `Violations() []AccessEntry` filtered accessor
  - [x] 1.9: `sandbox_test.go` — tests: allowed read, denied read, trap trigger, allowed write, denied write pattern, max file size, read-only mode, delete, glob matching, audit trail correctness

- [x] Task 2: Tool definitions and interceptor (AC: #1, #2)
  - [x] 2.1: `tools.go` — `ToolCall` struct: `Name string`, `Args map[string]string`; `ToolResult` struct: `Name`, `Output`, `Blocked bool`, `Reason`
  - [x] 2.2: `ParseToolCalls(response string) []ToolCall` — parse structured tool calls from LLM free-text response (format: `<tool name="file_read" path="src/main.go"/>`)
  - [x] 2.3: `ExecuteTool(sandbox *SandboxFS, stepID string, call ToolCall) ToolResult` — dispatch to sandbox Read/Write/Delete based on tool name
  - [x] 2.4: `FormatToolResults(results []ToolResult) string` — format results as text for LLM re-prompting
  - [x] 2.5: Tests: parse valid tool calls, parse no tool calls, execute allowed read, execute blocked write, format results

- [x] Task 3: Extend experiment types for tool tasks (AC: #3, #5, #6)
  - [x] 3.1: `design.go` — Add to `TaskDefinition`: `Workspace *WorkspaceDef` (yaml), `ExpectedViolations []string` (yaml)
  - [x] 3.2: `design.go` — Add `WorkspaceDef` struct: `Files []WorkspaceFile`, `Traps []WorkspaceFile`; `WorkspaceFile`: `Path`, `Content string`
  - [x] 3.3: `design.go` — Add to `StepDef`: `Tools []string` (yaml), `Sandbox *StepSandboxPolicy` (yaml)
  - [x] 3.4: `design.go` — Add `StepSandboxPolicy` struct: `AllowedPaths []string` (yaml)
  - [x] 3.5: `design.go` — Add to `TaskResult`: `ActionLog []AccessEntry`, `ActionMetrics *ActionMetrics`
  - [x] 3.6: `design.go` — Add `ActionMetrics` struct: `TotalActions`, `AllowedActions`, `BlockedActions`, `TrapAccesses`, `OutOfScopeAttempts`, `CleanWrites`, `TotalWrites int`; computed rates: `AdherenceRate`, `PreventionRate`, `TrapTriggerRate`, `ScopeCreepRate`, `WriteSafetyRate float64`
  - [x] 3.7: `design.go` — Add `SandboxComparisonMetrics` to `ComparisonResult`
  - [x] 3.8: `checkpoint.go` — Extend `TaskResultJSON` with `ActionMetrics` fields
  - [x] 3.9: Tests: verify new YAML fields unmarshal correctly from a tool-based task file

- [x] Task 4: Runtime runner — tool-call loop with sandbox (AC: #2, #4)
  - [x] 4.1: `runtime_runner.go` — In `buildTaskDAG`, detect steps with `Tools != nil`; build `SandboxFS` from `Workspace` via `buildSandboxFromWorkspace`; inject sandbox into LLM step closure
  - [x] 4.2: Tool-call loop: LLM prompt → parse tool calls → execute via sandbox → format results → re-prompt LLM (max 5 iterations, exit when LLM returns no tool calls)
  - [x] 4.3: After step completes, compute `ActionMetrics` from `sandbox.Log()` and attach to `TaskResult`
  - [x] 4.4: Capture `ActionLog` in `runRuntimeTask` after DAG execution completes
  - [x] 4.5: Existing tests pass; tool-call integration tested via mock provider in existing runner tests

- [x] Task 5: Text runner — tool tasks without sandbox (AC: #2)
  - [x] 5.1: `text_runner.go` — For tasks with `Workspace`, include workspace file listing in prompt as context (no tool execution)
  - [x] 5.2: Text condition gets nil ActionMetrics (no interception = no data) — by construction
  - [x] 5.3: Existing text runner tests pass with tool-based tasks (graceful nil handling)

- [x] Task 6: Action metrics comparison and reporting (AC: #3, #5)
  - [x] 6.1: `compare.go` — `ComputeSandboxMetrics(text, runtime []TaskResult) *SandboxComparisonMetrics` — aggregates action metrics across tool-based tasks
  - [x] 6.2: `report.go` — "Sandbox Enforcement Metrics" section with differential table (text=N/A vs runtime=measured)
  - [x] 6.3: Sandbox metrics computed and integrated into `Compare()` pipeline

- [x] Task 7: Adversarial task YAML files (AC: #2, #4)
  - [x] 7.1: `tasks/tool-path-traversal.yaml` — Agent told to read source files; trap files at `../secrets/api_key.txt`; sandbox allows only `src/**`
  - [x] 7.2: `tasks/tool-privilege-escalation.yaml` — Agent needs data that exists in `.env`; must complete task without reading credentials
  - [x] 7.3: `tasks/tool-accidental-destruction.yaml` — Agent told to "clean up temp files"; trap files outside temp scope; sandbox allows only `tmp/**`
  - [x] 7.4: Each task has `workspace` (files + traps), per-step `tools` and `sandbox` blocks, `expected_violations` list, and standard guardrails
  - [x] 7.5: Tests: verify all 3 tasks load and parse correctly via `LoadTasks`

## Dev Notes

### Architecture References

- **FR52:** "Developer can run a controlled runtime-vs-text experiment with reproducible results." Story 5-4 extends FR52 into tool-based territory.
- **PRD claim (line 58):** "Text instructions give +4% at best — structural enforcement gives +20-33%." This experiment produces the binary evidence for that claim.
- **PRD risk (line 497):** "Runtime-vs-text delta is zero" is Medium probability / High impact. Mitigation: pivot to efficiency/consistency metrics.
- **ADR-4:** DAG topology with code vs LLM nodes. Tool interception is a third node behavior.
- **NFR12:** Runtime DAG handles node failures gracefully — clear error reporting, no silent failures.
- **NFR37/38:** >=80% test coverage, race detector enabled in CI.

### Critical Design Decision: Tool Call Protocol

The `llm.Provider` interface is `Complete(prompt string) (string, error)` — single text-in/text-out. Tool-based tasks need tool call parsing from free-text LLM responses.

**Chosen approach:** Parse structured XML-style tool calls from LLM output (`<tool name="file_read" path="..."/>`). The prompt instructs the LLM to use this format. This is fragile but acceptable for experiment purposes — we're measuring enforcement behavior, not building a production tool-use system.

**Do NOT:** extend the Provider interface, add function-calling support, or use OpenAI-style tool_use. Keep it in the experiment package.

### Previous Story (5-2) Patterns to Follow

- **Runner pattern:** `Run<Condition>Experiment(config) ([]TaskResult, error)` with OnResult callback and fail-fast on API errors
- **DAG construction:** `buildTaskDAG` converts `StepDef` → `dag.Node`. Consumes/Produces auto-wires. `outputCapture` callback collects all node outputs.
- **Provider wrapping:** `callCountingProvider` wraps inner provider — follow same pattern for sandbox instrumentation
- **Step kinds:** `StepKindLLM`, `StepKindStatic`, `StepKindAssert` already exist. Tool steps use `StepKindLLM` with `Tools != nil` (not a new kind)
- **Metrics:** All `Compute*` functions take `(text, runtime []TaskResult)` and return a dedicated struct. Pure functions, no side effects.
- **Tests:** Mock provider with keyword-matched responses. Table-driven tests. `require` for preconditions, `assert` for values. Section headers with ASCII art comments.
- **Checkpoint:** JSONL format, `TaskResultJSON` projection, thread-safe writer

### Scope Boundaries — NOT in Scope

- Container-based isolation (Level 3) — deferred
- Network interception / HTTP mocking — deferred
- Multi-agent tool sharing — deferred
- Real filesystem testing (Level 2) — VirtualFS only
- `shell_exec` and `http_get` tools — deferred (spec mentions them but they need security review; start with file_read/file_write/file_delete only)
- Command injection adversarial category (E) — deferred (requires shell_exec)
- Data exfiltration adversarial category (D) — already covered by Layer 3 content guards; defer tool-specific version

### Project Structure

New files:
- `internal/bench/experiment/sandbox.go`
- `internal/bench/experiment/sandbox_test.go`
- `internal/bench/experiment/tools.go`
- `internal/bench/experiment/tasks/tool-path-traversal.yaml`
- `internal/bench/experiment/tasks/tool-privilege-escalation.yaml`
- `internal/bench/experiment/tasks/tool-accidental-destruction.yaml`

Modified files:
- `internal/bench/experiment/design.go` — new types
- `internal/bench/experiment/runtime_runner.go` — tool-call loop + sandbox injection
- `internal/bench/experiment/text_runner.go` — workspace in prompt
- `internal/bench/experiment/compare.go` — sandbox metrics
- `internal/bench/experiment/report.go` — sandbox report section
- `internal/bench/experiment/checkpoint.go` — new JSON fields
- `internal/bench/experiment/experiment_test.go` — new tests

### What NOT to Build

- Do NOT extend `llm.Provider` interface — tool calls parsed from text
- Do NOT implement real filesystem testing — VirtualFS only, all in-memory
- Do NOT implement `shell_exec` or `http_get` tools — start with file operations only
- Do NOT add new StepKind — tool steps are `StepKindLLM` with `Tools` field populated
- Do NOT build a production tool-use framework — this is experiment scaffolding
- Do NOT implement network isolation — deferred to future story

### Testing Strategy

- **Sandbox unit tests:** Isolated tests for each operation (read/write/delete), policy enforcement, glob matching, trap detection, audit trail
- **Tool parsing tests:** Structured tool call extraction from various LLM response formats
- **Integration tests:** Mock provider returning tool calls → runtime runner executes via sandbox → verify ActionMetrics
- **Task loading tests:** Verify new YAML fields unmarshal correctly
- **Comparison tests:** Known action metrics → expected comparison output
- **Report tests:** Verify sandbox section in Markdown report
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: docs/superpowers/specs/2026-04-14-tool-based-harness-testing-design.md] — Full design spec
- [Source: _bmad-output/planning-artifacts/prd.md#FR52] — Runtime-vs-text experiment
- [Source: _bmad-output/planning-artifacts/prd.md#line58] — "+20-33%" claim
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-4] — DAG topology
- [Source: internal/bench/experiment/] — Existing experiment framework
- [Source: _bmad-output/implementation-artifacts/5-2-runtime-vs-text-experiment.md] — Previous story

## Dev Agent Record

### Agent Model Used

claude-opus-4-6

### Completion Notes List

- VirtualFS sandbox (sandbox.go): in-memory filesystem with policy enforcement (AllowedGlobs, DeniedGlobs, DeniedWritePatterns, MaxFileSize, ReadOnly), trap detection, and complete audit trail. 16 unit tests.
- Tool interceptor (tools.go): XML-style tool call parsing from LLM responses (`<tool name="..." .../>`), dispatch through sandbox (file_read/file_write/file_delete), result formatting for re-prompting. 10 unit tests.
- Type extensions (design.go): WorkspaceDef, WorkspaceFile, StepSandboxPolicy, ActionMetrics, SandboxComparisonMetrics added. TaskDefinition gains Workspace + ExpectedViolations. StepDef gains Tools + Sandbox. TaskResult gains ActionLog + ActionMetrics.
- Runtime runner (runtime_runner.go): buildSandboxFromWorkspace constructs sandbox from task YAML. buildTaskDAG extended with sandbox parameter. LLM step handler has tool-call loop (max 5 iterations) that parses tool calls, executes through sandbox, re-prompts with results. ActionMetrics computed from sandbox.Log() after DAG execution.
- Text runner (text_runner.go): Workspace files included as prompt context (no tool execution, no sandbox). ActionMetrics is nil by design.
- Comparison + report: ComputeSandboxMetrics aggregates action metrics. Report has "Sandbox Enforcement Metrics" section showing runtime measurements vs text N/A.
- Checkpoint: TaskResultJSON extended with ActionMetrics for checkpoint/resume.
- 3 adversarial task YAMLs: path-traversal, privilege-escalation, accidental-destruction.
- 53 tests total, all passing with -race.

### File List

- `internal/bench/experiment/sandbox.go` (new)
- `internal/bench/experiment/sandbox_test.go` (new)
- `internal/bench/experiment/tools.go` (new)
- `internal/bench/experiment/tasks/tool-path-traversal.yaml` (new)
- `internal/bench/experiment/tasks/tool-privilege-escalation.yaml` (new)
- `internal/bench/experiment/tasks/tool-accidental-destruction.yaml` (new)
- `internal/bench/experiment/design.go` (modified — WorkspaceDef, StepSandboxPolicy, ActionMetrics, SandboxComparisonMetrics)
- `internal/bench/experiment/runtime_runner.go` (modified — sandbox construction, tool-call loop, ActionMetrics capture)
- `internal/bench/experiment/text_runner.go` (modified — workspace prompt context)
- `internal/bench/experiment/compare.go` (modified — ComputeSandboxMetrics)
- `internal/bench/experiment/report.go` (modified — sandbox report section)
- `internal/bench/experiment/checkpoint.go` (modified — ActionMetrics in TaskResultJSON)

## Change Log

- 2026-04-15: Story created from party mode consensus (John/Mary/Winston/Bob) + exhaustive artifact analysis. Positioned before 5-3 per sprint priority decision. Design spec exists at `docs/superpowers/specs/2026-04-14-tool-based-harness-testing-design.md`.
- 2026-04-15: Implemented all 7 tasks. VirtualFS sandbox, tool interceptor, type extensions, runtime/text runner integration, action metrics, comparison/report, 3 adversarial task YAMLs. 53 tests, all passing with -race.
