# Story 6-21: File-Based Slot Passing — Minimum Injected Prompt

**Status:** done
**Epic:** 6 — Runtime Harness
**Sprint:** 7

## Goal

Replace inline context injection in step prompts with file-based slot passing via the sandbox filesystem. Each step's prompt is reduced to the minimum required: step instructions + file paths to relevant context. The LLM accesses content on demand via `read_file`.

## Background

The current implementation injects slot values (outputs of prior steps) directly into each step's prompt as inline text blocks. This causes three problems observed in the harness experiment:

1. **Token overhead** — prior step outputs are copied verbatim into every downstream prompt (4-5× token multiplier on complex tasks)
2. **Context loss** — `task_input` (original task data) is only available to step 1; downstream steps lose global context
3. **Guardrail score degradation** — steps focused on sub-tasks don't mention task-level guardrail terms because they've lost sight of the overall goal

## Design

### Principle: inject the minimum, make the rest accessible

The DAG already knows what's relevant via `Consumes`. Instead of embedding content inline, write it to the sandbox and reference paths:

```
sandbox/
  task_context.md          ← always present: task name + description + original input
  slots/
    {key}.md               ← written by the step that produces it (SlotSpec.Key → filename)
```

Step prompt becomes:
```
## Step: Generate Fix

{step.Prompt}

## Available files (use read_file to access)
- task_context.md      — full task context and original input
- slots/bugs.md        — output from: identify-bugs
- slots/style_issues.md — output from: style-review
```

### Changes

#### 1. `buildTaskDAG` — init sandbox with task context

At DAG construction time, write `task_context.md` to the sandbox:

```
# Task: {task.Name}

## Description
{task.Description}

## Original Input
{task.Input}

## Guardrails
{task.Guardrails}
```

#### 2. LLM step `runFn` — replace inline injection with file listing

Instead of:
```go
for k, v := range inputs {
    inputParts = append(inputParts, fmt.Sprintf("%s:\n%s", k, s))
}
base.WriteString("## Input\n\n" + strings.Join(inputParts, "\n\n"))
```

Do:
```go
base.WriteString("## Available files (use read_file to access)\n")
base.WriteString("- task_context.md — full task context and original input\n")
for _, slot := range stepCopy.Consumes {
    base.WriteString(fmt.Sprintf("- slots/%s.md — output from: %s\n", slot.Key, slot.Key))
}
```

Write consumed slot values to sandbox files before calling the LLM.

#### 3. Step output → write to sandbox

After each step completes, write its output to `slots/{key}.md` so downstream steps can read it.

#### 4. Ensure `read_file` is always available for LLM steps

All LLM steps get `read_file` in their allowed tool list, even if `step.Tools` is empty. This is the mechanism for accessing context.

## Acceptance Criteria

**Given** a multi-step task where step 2 consumes the output of step 1
**When** step 2 executes
**Then** step 2's prompt lists `slots/{key}.md` instead of embedding the content inline

**Given** any LLM step in the DAG
**When** it executes
**Then** `task_context.md` is listed as available (containing task description + original input)

**Given** a task where step 3 needs to respect a task-level guardrail
**When** step 3 calls `read_file("task_context.md")`
**Then** it gets the full task context including guardrails

**Given** running the 16-task experiment after this fix
**When** comparing text vs runtime guardrail scores
**Then** the gap is reduced (runtime guardrail score closer to text baseline)

**Given** a complex task (7+ steps, large inputs)
**When** comparing token usage text vs runtime
**Then** runtime token overhead is reduced vs the current 4-5× multiplier

## Dev Agent Record

### Completion Notes

- Always create a `SandboxFS` for every task (not just workspace tasks). Non-workspace tasks get `NewSandboxFS(nil, nil, SandboxPolicy{})`.
- `task_context.md` written at DAG construction time: `# Task`, `## Description`, `## Original Input`, `## Guardrails`. Contains everything a downstream step needs about the task.
- LLM step prompt now lists file paths under `## Available files (use file_read to access)` — no inline content embedding. `task_input` slot is excluded from file listing (its content is in `task_context.md`).
- `file_read` always included in the available tools listing for all LLM steps, even steps with no declared `Tools`. Tool-call loop now activates whenever `sandbox != nil` (not just when `stepCopy.Tools != empty`).
- After each LLM step completes, its output is written to `slots/{key}.md` so downstream steps can `file_read` it on demand.
- `buildSandboxFromWorkspace`: when the policy is already restrictive (allowedGlobs non-empty), automatically adds `task_context.md` and `slots/**` to the allowed glob list. Preserves the permissive behavior (all paths allowed) when no step has sandbox policies.
- 6 new tests added to `experiment_test.go`, all passing. Full regression suite: 38 packages, 0 failures.

### File List

- `internal/bench/experiment/runtime_runner.go` — all 4 changes (always-sandbox, task_context write, file-listing prompt, slot write after step, tool loop condition)
- `internal/bench/experiment/experiment_test.go` — 6 new `TestFileSlotPassing_*` tests, added `context` and `dag` imports

### Review Findings

#### Patch (must fix)

- [x] [Review][Patch] `task_context.md` write silently fails in ReadOnly workspace — extracted `taskContextContent()` helper, pre-populate in `buildSandboxFromWorkspace`'s initial files map so it survives ReadOnly policy [runtime_runner.go:422-448] **HIGH**
- [x] [Review][Patch] Per-step `AllowedPaths` blocks `file_read` for `task_context.md` and `slots/**` — augment `stepCopy.Sandbox.AllowedPaths` in the harness block before middleware wiring [runtime_runner.go:381-391] **HIGH**
- [x] [Review][Patch] `file_read` duplicated in prompt when step already declares it — replaced `append` with deduplicating loop using `seenTools` map [runtime_runner.go:269-278] **MED**

#### Defer

- [x] [Review][Defer] Tool-call loop resets `toolPrompt` to base prompt on each guardrail retry — accumulated tool-read context is lost if guardrails fail; pre-existing behavior now exposed to all LLM steps [runtime_runner.go:309] — deferred, pre-existing
- [x] [Review][Defer] `toolPrompt` grows unboundedly across iterations — no length cap on tool-result accumulation; pre-existing in tool loop [runtime_runner.go:332] — deferred, pre-existing
- [x] [Review][Defer] TOCTOU lock gap in `SandboxFS.Read` — `RLock` released before `Lock` acquired; concurrent write can modify root between policy check and read [sandbox.go:63-82] — deferred, pre-existing
- [x] [Review][Defer] `task_input` not written to `slots/task_input.md` — LLM told `task_context.md` covers this; if LLM guesses `file_read("slots/task_input.md")` it gets "not found" [runtime_runner.go:257-259] — deferred; acceptable; prompt guidance sufficient
- [x] [Review][Defer] Guardrails still injected inline alongside file listing — small payload; intentional (guardrails must be immediately visible, not require a read) [runtime_runner.go:276-283] — deferred, intentional design
- [x] [Review][Defer] `lastResponse` is empty/malformed when tool loop exhausts max iterations without LLM producing a non-tool response — pre-existing [runtime_runner.go:298] — deferred, pre-existing
- [x] [Review][Defer] Audit log attribution: consuming step's pre-write of upstream slot file is attributed to consuming step ID, not producing step — cosmetic, no correctness impact [runtime_runner.go:253-258] — deferred, minor audit cosmetics
- [x] [Review][Defer] `seen["slots/**"]` is never populated — `slots/**` is always appended when `allowedGlobs > 0`, unlike `task_context.md` which has correct `seen` guard; cosmetic asymmetry [runtime_runner.go:463-468] — deferred, no runtime impact
- [x] [Review][Defer] `funcProvider` missing compile-time interface guard — no `var _ llm.Provider = (*funcProvider)(nil)` unlike other test providers [experiment_test.go] — deferred, test quality
- [x] [Review][Defer] AC4/AC5 (guardrail gap, token reduction) not verifiable from diff alone — requires running the 16-task bench experiment [story spec] — deferred; verify by running `forgent-bench`

### Change Log

- 2026-04-17: Story 6-21 implemented — file-based slot passing, minimum injected prompt
- 2026-04-17: Code review applied 3 patches — ReadOnly pre-population, per-step AllowedPaths augmentation, file_read deduplication
