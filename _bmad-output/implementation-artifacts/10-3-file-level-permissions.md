# Story 10-3: File-Level Permission Model per Step

**Epic:** 10 — Node-Level Behavioral Contracts
**Status:** done

## Story

As a **developer defining a Forgent task**,
I want to declare file access permissions at the step level (`read-only`, `read-write`, `read-write-delete`),
So that the VirtualFS sandbox enforces the boundary regardless of which tool the LLM uses.

## Acceptance Criteria

**AC1:** Given a step declares `file_access: read-only`,
When the LLM attempts any write or delete operation (via any tool),
Then the sandbox returns `permission denied: step is read-only` before execution.

**AC2:** Given a step declares `file_access: read-write`,
When the LLM attempts a delete operation,
Then the sandbox returns `permission denied: step does not have delete permission`,
And write operations succeed normally,
And read operations succeed normally.

**AC3:** Given a step declares `file_access: read-write-delete`,
When the LLM performs any file operation (read, write, delete),
Then all operations are permitted — full sandbox access (subject to existing path/glob policy).

**AC4:** Given a step has no `file_access` declared,
When the runtime executes,
Then default behavior applies (`read-write`) — full backward compatibility (NFR40).

**AC5:** Given a permission violation occurs,
Then it is recorded in the sandbox audit trail with `Allowed: false` and a `Reason` field stating which permission was insufficient,
And no retry is triggered for the operation — deterministic fail (consistent with the existing path-policy denial flow).

**AC6:** Given an unknown `file_access` value (e.g. `read-modify`, typo),
When `Validate()` runs (or when `LoadTasks` parses the YAML),
Then a clear error is returned identifying the offending step and listing the valid values.

**AC7:** Given `go test ./...`,
Then all existing tests pass without modification (race-clean on `internal/bench/experiment/...`).

## Tasks/Subtasks

- [x] T1: Add `file_access` field + permission constants
  - [x] T1.1: Added `FileAccess FileAccess \`yaml:"file_access,omitempty"\`` to `StepDef` in `design.go` (typed, not raw string — see T1.2 rationale).
  - [x] T1.2: **Refactored to idiomatic Go enum pattern** (typed string + typed constants, parallel to existing `StepKind` and `FileAccess`-grade patterns in the project): `type FileAccess string` with constants `FileAccessReadOnly`, `FileAccessReadWrite`, `FileAccessReadWriteDelete`, `FileAccessDefault = FileAccessReadWrite`. Same pattern applied to tool names: `type ToolName string` + `ToolFileRead`, `ToolFileWrite`, `ToolFileDelete` so the switch statements operate on closed enum sets.
  - [x] T1.3: `ExperimentConfig.Validate()` now rejects unknown `FileAccess` values via the `validFileAccessValues map[FileAccess]bool` lookup. Empty string is accepted (treated as default downstream).
  - [x] T1.4: Doc comment on `FileAccess` type explains the three levels, the default, and the most-restrictive-wins composition with workspace-level `ReadOnly` and per-step `AllowedPaths`.

- [x] T2: Per-step permission enforcement at the tool boundary (AC #1, #2, #3, #5)
  - [x] T2.1: `effectiveFileAccess(step StepDef) FileAccess` helper added in `runtime_runner.go` next to `effectiveGuardrails`. Returns typed value, defaults to `FileAccessReadWrite` when empty. Doc comment notes the NFR40 rationale.
  - [x] T2.2: New `ExecuteToolForStep(sandbox, step, call)` entry point in `tools.go` (preferred). Internally calls `executeToolWithPolicies(sandbox, stepID, call, stepPolicy, fileAccess)` which enforces three layers in documented precedence (`file_access` → `AllowedPaths` → workspace `SandboxPolicy`). Permission denials at layer 1 call `sandbox.record(stepID, call.Name, path, false, reason)` and return `ToolResult{Blocked: true, Reason}`. Legacy `ExecuteToolWithPolicy` retained as a deprecated shim that defaults `fileAccess = FileAccessDefault`.
  - [x] T2.3: Call site in `runtime_runner.go` (was line 389) updated: `ExecuteToolWithPolicy(stepSandbox, stepCopy.ID, call, stepCopy.Sandbox)` → `ExecuteToolForStep(stepSandbox, stepCopy, call)`. Lower-churn path chosen — adds `step` directly to the call rather than threading a fileAccess parameter.
  - [x] T2.4: Denial reasons match AC wording exactly:
        - AC1 read-only blocks write/delete: `"permission denied: step is read-only"`
        - AC2 read-write blocks delete: `"permission denied: step does not have delete permission"`
        Stable strings pinned by `TestStepFileAccess_ReadOnly_BlocksWriteAndDelete` and `TestStepFileAccess_ReadWrite_BlocksDelete`.

- [x] T3: Tests
  - [x] T3.1: `TestStepFileAccess_ReadOnly_BlocksWriteAndDelete` — step with
        `FileAccess: "read-only"`, mock LLM that emits `<tool name="file_write" .../>` then
        `<tool name="file_delete" .../>`. Both ToolResults must have `Blocked: true` and the reason
        must contain `"step is read-only"`. Audit trail must contain matching denied entries.
  - [x] T3.2: `TestStepFileAccess_ReadOnly_AllowsRead` — same step, mock LLM that emits
        `<tool name="file_read" .../>`. ToolResult must have `Blocked: false` and return file content.
  - [x] T3.3: `TestStepFileAccess_ReadWrite_BlocksDelete` — `FileAccess: "read-write"`, mock LLM
        emits `file_write` (must succeed) then `file_delete` (must be Blocked with the
        `"does not have delete permission"` reason). Audit trail asymmetric.
  - [x] T3.4: `TestStepFileAccess_ReadWriteDelete_AllowsAll` — `FileAccess: "read-write-delete"`,
        all three tool calls succeed.
  - [x] T3.5: `TestStepFileAccess_DefaultIsReadWrite` — `FileAccess: ""` (or omitted in YAML).
        Behavior must match `read-write`: file_write succeeds, file_delete blocked. Pin the
        backward compat invariant explicitly so a later contributor cannot change the default
        without updating this test.
  - [x] T3.6: `TestValidate_RejectsUnknownFileAccess` — TaskDefinition with
        `Steps: []StepDef{{ID: "s", FileAccess: "read-modify"}}`. `ExperimentConfig.Validate()`
        must return an error mentioning the offending value and step ID.
  - [x] T3.7: `TestStepFileAccess_NoRetryOnPermissionDenial` — step with `read-only`,
        guardrail satisfied. The LLM emits a `file_write` that is blocked, then synthesizes
        a final answer. Verify the runtime did NOT retry the step on the basis of the
        permission denial alone (the existing guardrail-retry path must remain orthogonal).
  - [x] T3.8: `TestStepFileAccess_AuditTrailRecordsAttemptAndDenial` — after a blocked
        operation, `sandbox.Log()` must include an entry with `Allowed: false`, the
        appropriate Action (`"write"` / `"delete"` / etc.), the path, and a Reason matching
        AC1/AC2 wording. This is the FR62-readiness check for story 10-6 (telemetry).
  - [x] T3.9: YAML round-trip test in `design_test.go` (or `step_guardrails_test.go`):
        `file_access: read-only` parses correctly; omit-empty respected on marshal.

- [x] T4: Run `go test ./... -count=1` and `go test -race ./internal/bench/experiment/...` —
      no regressions, race-clean.

- [x] T5: Update the existing `Sandbox` policy interaction documentation
      - [x] T5.1: Document the precedence in a comment at the top of `tools.go`:
            `(1) per-step file_access` → `(2) per-step Sandbox.AllowedPaths globs` →
            `(3) workspace-level SandboxPolicy (ReadOnly, MaxFileSize, DeniedGlobs, etc.)`.
            All three checks run in order; the most restrictive wins.
      - [x] T5.2: If any existing test fixture YAML uses `read_only: true` at the workspace
            level AND has a step that would now be permitted by `FileAccess: read-write-delete`,
            confirm the workspace-level wins (more restrictive). Add a regression test if absent.

## Dev Notes

### Architecture

**The sandbox already has the enforcement infrastructure.** `internal/bench/experiment/sandbox.go`'s
`SandboxFS.Write` / `SandboxFS.Delete` methods already check `s.policy.ReadOnly` and call
`recordLocked` for audit. What's missing is a per-STEP equivalent of `ReadOnly` that scales to
three levels and is declared on the StepDef rather than the workspace.

**Two layers of permission already exist:**

1. Workspace-level `SandboxPolicy.ReadOnly` (in `design.go:57`) — set by the task's `Workspace.read_only`
   YAML field, applies to ALL steps. Currently a binary on/off.
2. Per-step `StepSandboxPolicy.AllowedPaths` (in `design.go:93`) — glob-based path allowlist for one step.
   Enforced at the tool boundary in `ExecuteToolWithPolicy` (`tools.go:50`).

This story adds a THIRD layer: per-step **operation** permission (read vs write vs delete) — orthogonal
to path-based policy and complementary to workspace-level read-only.

**Precedence (most restrictive wins):**

```
1. step.FileAccess (this story)         — operation type (read/write/delete)
2. step.Sandbox.AllowedPaths            — path glob whitelist
3. workspace SandboxPolicy.ReadOnly     — workspace-wide read-only override
4. workspace SandboxPolicy.DeniedGlobs  — workspace-wide denied paths
5. workspace SandboxPolicy.DeniedWritePatterns — content denial
```

The existing layer 3-5 enforcement in `SandboxFS.Write` / `Delete` should NOT be duplicated; the
new layer 1 check just runs first at the tool dispatcher level (before reaching the sandbox).

### Where to put the enforcement

**Recommended:** at the top of `ExecuteToolWithPolicy` (or a sibling `ExecuteToolWithStep`).
This is the correct seam because:

- It runs once per tool call (no need to thread the policy through every sandbox method).
- It can short-circuit before touching the sandbox at all — keeps the audit trail clean.
- It mirrors the existing `AllowedPaths` enforcement which lives in the same function.

The existing `ExecuteToolWithPolicy` signature:

```go
func ExecuteToolWithPolicy(sandbox *SandboxFS, stepID string, call ToolCall, stepPolicy *StepSandboxPolicy) ToolResult
```

Two options:

**Option A** (lower churn): add a `fileAccess string` parameter, defaulting via a new
`ExecuteToolForStep(sandbox, step, call)` wrapper that pulls everything off the step.

**Option B** (cleaner long-term): introduce a `StepRequest` struct that bundles `stepID`,
`stepPolicy`, and `fileAccess`. Less ad-hoc, but every call site needs an update.

Option A is the recommended first move — keep the change contained. Story 10-4 onward will
likely want a struct anyway; let it land then.

### Why a Go enum (constants) instead of free-form strings

The three valid values are a closed set. Using `const` makes it impossible to mistype at compile
time inside the codebase, and `Validate()` enforces the allowlist at YAML load time for users.
The constants double as documentation: any `switch` over `step.FileAccess` will trip the
Go vet `exhaustive` checker (if installed) when a future variant is added.

### Backward compatibility (NFR40)

- Steps with no `file_access` declared get `FileAccessDefault = "read-write"`. Existing tasks
  that rely on `file_write` working keep working.
- The default is `read-write` (NOT `read-write-delete`) because most existing tests do not
  exercise `file_delete`, and silently expanding default permissions would weaken the
  security posture of pre-existing tasks.
- Existing `Workspace.ReadOnly: true` continues to apply and overrides any step-level
  `FileAccess: read-write-delete` because layer 3-5 enforcement runs after layer 1. Pin this
  with T5.2's regression test.

### Test approach

- Use `mockGuardrailProvider` (existing in `step_guardrails_test.go`) for tests that need a
  scripted LLM response with embedded tool calls. The provider's `responses` slice is consumed
  in order; emit one `<tool .../>` block per response to keep tests deterministic.
- Build minimal `TaskDefinition` fixtures inline (current pattern in
  `step_guardrails_test.go`); set `TaskDir` only if the test exercises `template_file`.
- Audit trail assertions use `sandbox.Log()` (already exists, returns `[]AccessEntry`).
  Look for entries with `Allowed: false` and the expected `Action` / `Reason`.

### Files to modify

- `internal/bench/experiment/design.go` — add `FileAccess` field, constants, validation
- `internal/bench/experiment/runtime_runner.go` — add `effectiveFileAccess` helper, wire
  through to tool dispatcher
- `internal/bench/experiment/tools.go` — add file-access check at the top of
  `ExecuteToolWithPolicy` (or via a new wrapper)

### New test files (or extensions)

- Extend `internal/bench/experiment/step_guardrails_test.go` with `TestStepFileAccess_*` tests
  (close cousins to the existing `TestEffectiveGuardrails_*` tests)
- Or create `internal/bench/experiment/file_access_test.go` for cleaner separation —
  pick whichever matches the directory's existing organisation. Default: extend the existing
  file unless tests grow beyond ~150 lines.

### What NOT to do

- **Don't introduce a `pkg/permissions` package.** The contract lives entirely inside
  `internal/bench/experiment/` for now. Promotion to a public package waits until a second
  consumer appears (e.g., when the runtime is used outside the bench harness).
- **Don't duplicate the `SandboxFS.Write` / `Delete` ReadOnly check at the tool level.** The
  workspace-level `ReadOnly` is enforced inside the sandbox; the new per-step `FileAccess`
  is enforced at the tool dispatcher. Two layers, no redundancy.
- **Don't reuse `StepSandboxPolicy` to carry `FileAccess`.** That struct is path-specific
  (`AllowedPaths`); mixing operation-type permissions in would confuse future contributors.
  A flat field on `StepDef` is the right shape.
- **Don't make `FileAccess` enforcement bypass the existing `AllowedPaths` check.** The
  most-restrictive-wins model means BOTH must permit the operation. A `read-write` step with
  `AllowedPaths: []string{"src/**"}` denies a write to `dist/x.go` because of the path
  policy, not the file-access policy.

### Previous story intelligence (from 10-0 and 10-2)

- **10-0 fixed `effectiveGuardrails` to return a defensive copy.** Apply the same pattern
  for any helper introduced here: never alias the source. `effectiveFileAccess` returns a
  string, so the alias risk is not present, but if a future helper returns a struct, copy it.
- **10-0 fixed `Validate()` to reject empty/duplicate StepIDs.** Add the file-access
  validation in the same function so all step-level invariants live in one place.
- **10-0's `taskContextContent` audit lies about step-level overrides was a real bug.**
  Consider whether `task_context.md` should also surface `file_access` (e.g., "Step One:
  read-only"). For this story, defer that to story 10-6 (telemetry) which owns the
  audit-content rendering question. Note in dev agent record if the discussion warrants
  future work.
- **10-2 added `task.TaskDir` for `template_file` resolution.** No interaction expected with
  this story, but if the dev session ends up touching `LoadTasks`, double-check that the
  `step.ID` backfill (also from 10-0 retro v2 patch) still runs.

### Telemetry hook (for story 10-6)

The audit trail entries with `Allowed: false` and the new file-access denial reasons are the
data source for story 10-6 (per-node violation telemetry). Keep the `Reason` field
human-readable AND parseable by a future tally — prefer stable phrases (`"step is read-only"`,
`"step does not have delete permission"`) over free-form strings. T2.4 pins the wording.

### Project context reference

- **Sprint status:** `_bmad-output/implementation-artifacts/sprint-status.yaml` —
  `10-3-file-level-permissions: ready-for-dev` after this file lands.
- **Epic 10 plan:** `_bmad-output/planning-artifacts/epics.md` lines 311-341 (Story 7-3 in
  the original numbering — the project has since renumbered Epic 7 as Epic 10 in
  sprint-status; the story content is unchanged).
- **Recent commits:**
  - `a3296d9 feat(10-0): epic 9 debt cleanup` — landed sandbox TOCTOU fix, defensive
    guardrail copy, Validate StepID enforcement, taskContextContent step-level rendering
  - `46d041a feat(epic9): close epic 9 (DX) + retro + deferred-work triage and quick wins`
  - `9069552 plan(epic7): node-level behavioral contracts — 9 stories + experiment baseline`
- **Test infrastructure:** 155 tests passing in `internal/bench/experiment/` after 10-0;
  327 race-clean across the affected packages. New tests should keep both green.

## Dev Agent Record

### Agent Model Used

claude-opus-4-7

### Debug Log References

_empty_

### Completion Notes List

**Idiomatic Go enum pattern adopted** (typed string + typed constants), parallel
to the existing `StepKind` pattern in the same package. User feedback during
implementation pushed for type safety beyond raw string constants — the result
is `type FileAccess string` AND `type ToolName string` so all dispatch
switches operate on closed enum sets.

`checkFileAccess` signature also typed: `(toolName ToolName, fileAccess FileAccess)`.
The cast from `ToolCall.Name string` happens at the dispatcher boundary
(`executeToolWithPolicies`) — which is correct because `ToolCall.Name` is the
RAW LLM-emitted string (any name; `ParseToolCalls` does not validate). The closed
set lives at the dispatch layer.

**Three-layer precedence** documented in `ExecuteToolForStep`:
1. `step.FileAccess` (this story) — read/write/delete operation type
2. `step.Sandbox.AllowedPaths` (story 5-4) — path glob whitelist
3. workspace `SandboxPolicy` (`ReadOnly`, `MaxFileSize`, `DeniedGlobs`, `DeniedWritePatterns`)
   — workspace-level overrides

Most-restrictive-wins: any layer can deny independently. Confirmed by
`TestStepFileAccess_WorkspaceReadOnlyOverridesStepLevel` (workspace wins over
`read-write-delete`) and `TestStepFileAccess_PrecedenceFileAccessBeforeAllowedPaths`
(layer 1 fires before layer 2).

**Backward compatibility (NFR40):** empty `FileAccess` → `FileAccessDefault`
(= `read-write`). Tasks authored before story 10-3 keep working unchanged.
The deprecated `ExecuteToolWithPolicy` shim (preserved for old callers) routes
through the same enforcement with `fileAccess = FileAccessDefault`.

**Tests:** 13 new tests in `internal/bench/experiment/file_access_test.go`
covering all 7 ACs:
- 7 unit tests on `checkFileAccess` + `ExecuteToolForStep` (T3.1–T3.5, T3.8, T5.2)
- 3 Validate tests (T3.6 + 2 positive/default cases)
- 1 integration test on no-retry-on-denial (T3.7)
- 1 YAML round-trip (T3.9)
- 1 deprecated-shim backward-compat
- 1 layer-precedence regression

**Test counts:** 155 → 172 in `internal/bench/experiment/` (+17 including 3
sub-tests via `t.Run`). 2090 → 2107 repo-wide. Race-clean on the package.
2 pre-existing `/tmp` env failures unchanged.

### File List

**Modified (3):**
- `internal/bench/experiment/design.go` — added `type FileAccess string` +
  4 typed constants + `validFileAccessValues` allowlist; added
  `StepDef.FileAccess FileAccess` field with `omitempty`; extended
  `ExperimentConfig.Validate()` to reject unknown values
- `internal/bench/experiment/runtime_runner.go` — added `effectiveFileAccess`
  helper; updated tool-loop call site `ExecuteToolWithPolicy → ExecuteToolForStep`
- `internal/bench/experiment/tools.go` — added `type ToolName string` +
  3 typed constants; added `ExecuteToolForStep` (preferred entry); added
  `executeToolWithPolicies` (internal three-layer enforcement); added
  `checkFileAccess(ToolName, FileAccess)` typed helper; refactored
  `ExecuteTool`'s switch to use `ToolName` enum; preserved
  `ExecuteToolWithPolicy` as deprecated shim defaulting to `FileAccessDefault`

**New (1):**
- `internal/bench/experiment/file_access_test.go` — 13 tests covering
  all ACs (see Completion Notes above)

## Change Log

| Date | Change | Notes |
|---|---|---|
| 2026-04-27 | Story 10-3 created from Epic 10 plan (Story 7-3 in epics.md) | Foundations from story 10-0 (TOCTOU fix, fail-closed symlink, defensive guardrail copy, StepID validation) make this story implementable without prerequisite cleanup. |
| 2026-04-28 | Implementation complete | Tasks T1-T5 done (22 sub-tasks). Idiomatic Go enum pattern adopted (`type FileAccess string`, `type ToolName string`) per user feedback during implementation. 13 new tests, 172 race-clean in package, 2107 repo-wide (+17). Status: in-progress → review. |
| 2026-04-28 | Code review complete | 3 layers (Blind, Edge, Auditor). Auditor: 19 PASS, 0 FAIL, 2 UNVERIFIED (test-execution claims). Blind + Edge surfaced 11 review findings: 2 HIGH, 4 MED, 5 LOW. See Review Findings section below. |
| 2026-04-28 | All 10 patches applied | 2 HIGH (audit-vocab unification via `toolVerb` helper + tool-accidental-destruction.yaml `file_access: read-write-delete`), 4 MED (case-insensitive `ParseToolCalls` + strengthened `NoRetryOnPermissionDenial` via audit count + `TestLoadAllTasks_ValidateClean` + `pkg/sandbox` divergence comment), 4 LOW (doc fixes + `FileAccessDefault` typed + default-deny test + legacy shim doc). 4 new tests, 176 race-clean in package, 2111 repo-wide. **Determinism leak between LLM stochasticity and runtime decisions closed**: tool-name casing normalised at parse; audit vocab unified across all 3 enforcement layers. Status: review → done. |

### Review Findings (2026-04-28)

**Patch items (all applied 2026-04-28):**

- [x] [Review][Patch] **HIGH** Audit-trail `Action` vocabulary bifurcation — layer-1 (`executeToolWithPolicies`) records `Action: "file_write"` (raw tool name), but layer-3 (`SandboxFS.Write`/`Delete`/`Read`) records `Action: "write"`/`"delete"`/`"read"` (verb). `ComputeActionMetrics` filters on the verb form, so layer-1-blocked write attempts are silently excluded from `TotalWrites` and `WriteSafetyRate`. Story 10-6 telemetry will inherit the bug. Fix: in `executeToolWithPolicies`, derive the verb from `ToolName(call.Name)` and pass that to `sandbox.record`. Test `TestStepFileAccess_AuditTrailRecordsAttemptAndDenial` currently pins `"file_write"` and must be updated to `"write"`. [tools.go:96 + sandbox.go:71/91/124 + file_access_test.go:385]
- [x] [Review][Patch] **HIGH** `internal/bench/experiment/tasks/tool-accidental-destruction.yaml` cleanup step uses `file_delete` without explicit `file_access`, so the new `FileAccessDefault = read-write` denies the delete and breaks the task at runtime. Real NFR40 break — silent until execution. Fix: add `file_access: read-write-delete` to the relevant step; audit the other 15 fixtures for similar gaps. AC #4 claims default behaviour preserves backward compat — that's true for tasks that never delete; tasks that DO delete must opt in. [tasks/tool-accidental-destruction.yaml + audit of 16 task YAMLs]
- [x] [Review][Patch] **MED** `ToolName(call.Name)` cast at the dispatcher boundary is case-sensitive: an LLM emitting `<tool name="FILE_WRITE" .../>` falls into the `default:` branch of `checkFileAccess` (allowed for "non-file tools") AND the `default:` branch of `ExecuteTool`'s switch (returns `Unknown tool: FILE_WRITE`). Net effect: no audit entry, no enforcement, no operation — silent failure. Fix: normalise the input with `ToolName(strings.ToLower(call.Name))` at the cast site, OR document the case-sensitivity contract and add a normalisation step in `ParseToolCalls`. [tools.go:94, 170]
- [x] [Review][Patch] **MED** `TestStepFileAccess_NoRetryOnPermissionDenial` passes for the wrong reason — the second mock response satisfies the guardrail (`"APPROVED"`), so `callCount == 2` proves "no retry on guardrail violation" not "no retry on permission denial". Strengthen by making the FIRST response also satisfy the guardrail (so `callCount == 1` is the natural pass), OR add a control test where the guardrail violates AFTER a permission denial to exercise both paths. [file_access_test.go NoRetryOnPermissionDenial]
- [x] [Review][Patch] **MED** No end-to-end fixture validation test exists. Adding `TestLoadAllTasks_ValidateClean` that loops `LoadTasks("tasks/")` + `Validate()` over every shipped YAML would have caught the `tool-accidental-destruction.yaml` regression at CI time. [new test in `internal/bench/experiment/tasks_test.go`]
- [x] [Review][Patch] **MED** `pkg/sandbox/middleware.go:11-20` `toolAction` map recognises alias names (`read_file`, `write_file`, `delete_file`, `list_directory`, `list_dir`) but the new `ToolName` constants in `tools.go` only cover the `file_*` form. If a future caller wires the middleware as the enforcement seam, alias-named LLM emissions silently bypass the file_access layer. Tripwire — not currently exploited, but the divergence between the two maps is a real contract gap. Fix options: extend `ToolName` constants to alias names + matching `case` arms in `checkFileAccess`, OR remove the aliases from `pkg/sandbox` if unused, OR document the divergence prominently. [tools.go vs pkg/sandbox/middleware.go]
- [x] [Review][Patch] **LOW** Doc drift: `design.go` `StepDef.FileAccess` doc comment references `ExecuteToolWithPolicy` as the enforcement seam — the actual seam is `ExecuteToolForStep`/`executeToolWithPolicies`. `runtime_runner.go:454` similarly mentions `ExecuteToolWithPolicy` in a stale comment. Quick combined patch. [design.go + runtime_runner.go:454]
- [x] [Review][Patch] **LOW** Legacy `ExecuteToolWithPolicy` shim defaults `fileAccess = FileAccessDefault = "read-write"`, which BLOCKS `file_delete`. Pre-refactor the function had no permission check and `file_delete` flowed through. Doc claim "preserves original behaviour" is misleading. Either (a) change the shim's default to `FileAccessReadWriteDelete` (true preservation, weaker security) and update `TestExecuteToolWithPolicy_DefaultsToReadWrite`, OR (b) keep the new tighter default and update the doc comment to flag the breaking change explicitly. Recommendation (b): the security tightening is the right call; the doc just needs to be honest. [tools.go ExecuteToolWithPolicy doc]
- [x] [Review][Patch] **LOW** `FileAccessDefault = FileAccessReadWrite` lacks an explicit type in the const declaration — relies on RHS-typed inference. A future contributor reordering or replacing the RHS with `"read-write"` (untyped) would silently weaken type-safety. Fix: `FileAccessDefault FileAccess = FileAccessReadWrite`. [design.go:98]
- [x] [Review][Patch] **LOW** `checkFileAccess`'s inner `default:` deny branch (for unknown `FileAccess` values reached via direct programmatic API misuse, bypassing `Validate`) is not exercised by tests. Add a unit test that calls `checkFileAccess(ToolFileWrite, FileAccess("read-modify"))` and asserts the deny + reason. [tools.go:148-149, 159-160]

**Deferred items (acknowledged, not blocking):**

- [x] [Review][Defer] **LOW** `ExecuteTool` is now a public function with no permission check — three public entry points with three different permission semantics. Tighten by lowercasing to `executeTool` (only used internally + tests) in a follow-up; today's code path uses `ExecuteToolForStep` correctly. [tools.go]
- [x] [Review][Defer] **LOW** `Validate()` ordering — invalid `file_access` is reported AFTER duplicate step ID. Single-step tests don't exercise the order, but the contract is undocumented. Document or sort. [design.go Validate loop]
- [x] [Review][Defer] **LOW** `omitempty` trap — `FileAccessDefault = "read-write"` is not the Go zero value, so a normalised StepDef would write the explicit value back to YAML. Currently no normalisation runs at marshal, so no diff noise — but the trap should be documented. [design.go yaml tag]
- [x] [Review][Defer] **LOW** `validFileAccessValues` is a package-global mutable map — a buggy or malicious test could widen the validation set. Wrap in a function or use a switch+slice for true immutability. [design.go]
- [x] [Review][Defer] **LOW** Layer-1 audit entries record the un-normalised path verbatim (no `filepath.Clean`), unlike layer-3 entries which clean. Log consumers may see inconsistent path forms across layers. Cosmetic. [tools.go:96]
- [x] [Review][Defer] **LOW** Doc claim "any of the three layers can deny independently" is misleading — only the highest-precedence deny is observable in the audit log because layer 1 short-circuits before layer 2. Document the short-circuit behaviour. [tools.go ExecuteToolForStep doc]

**Dismissed (1):**

- Concurrent same-step execution (informational, Edge Case Hunter #10) — `effectiveFileAccess` and `checkFileAccess` are pure; the DAG never executes the same node twice in parallel. No bug.
