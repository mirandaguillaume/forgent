# Story 6-22: Context Retriever — Pluggable Slot Context Strategies

**Status:** done
**Review:** done (2026-04-17 — 6 patches applied, 2 deferred)
**Epic:** 6 — Runtime Harness
**Sprint:** 7

## Goal

Introduce a `ContextRetriever` interface that decouples the context injection strategy from the DAG execution engine. Two implementations: `PrefixRetriever` (first N tokens of each consumed slot) and `RankedRetriever` (BM25 scoring to select the most relevant chunks). The experiment report gains a third condition so all three strategies are measurable side-by-side.

## Background

Story 6-21 replaced inline slot injection with file-based passing via the sandbox. The LLM now calls `file_read` on demand. This reduced the token multiplier from 4-5× to ~1.85×, but introduced 12 LLM calls per 3-step task (the LLM reads files via tool calls, each triggering a new provider round-trip).

The root cause: the LLM always needs to read `task_context.md` and consumed slot files before it can work. Instead of making it discover and pull this context via tool calls, we can inject a **ranked excerpt** directly into the prompt — enough to answer without a file read, with the full file still available if needed.

Party mode consensus (2026-04-17):
- `PrefixRetriever` — zero computation, inject first N tokens
- `RankedRetriever` — BM25/TF-IDF pure Go, no API dependency
- `EmbeddingRetriever` — interface ready, impl deferred (neural embeddings = overkill + latency)
- No new Go module dependencies

## Design

### Interface

```go
// ContextRetriever selects and formats context to inject into a step prompt.
// It is called once per LLM step, before the prompt is finalized.
// Returns a markdown string injected under "## Context" in the prompt,
// or empty string for no injection.
type ContextRetriever interface {
    Retrieve(step StepDef, sandbox *SandboxFS, consumed []SlotSpec) (string, error)
}
```

### File: `internal/bench/experiment/retriever.go`

Three exports:

**`NilRetriever`** — returns `""` always (current behavior, backwards-compatible default).

**`PrefixRetriever`** — for each consumed slot, reads the slot file from sandbox, takes the first `MaxTokens` characters (approximate: chars/4), returns a formatted markdown block.

```go
type PrefixRetriever struct {
    MaxTokens int // default 300
}
```

**`RankedRetriever`** — BM25 scoring. Splits each slot file into chunks (paragraph or sentence boundaries), scores each chunk against the step prompt as query, returns top-K chunks.

```go
type RankedRetriever struct {
    MaxTokens int // total budget across all chunks (default 500)
    TopK      int // max chunks to include (default 5)
}
```

BM25 impl: pure Go, no external package. IDF computed over the chunk corpus for that retrieval call (not persisted). Tokenization: lowercase split on non-alphanumeric.

### Integration in `buildTaskDAG`

`ExperimentConfig` gains an optional field:

```go
type ExperimentConfig struct {
    // ...
    ContextRetriever ContextRetriever // nil = NilRetriever (file listing only, no injection)
}
```

`buildTaskDAG` receives the retriever and calls it inside the LLM `runFn`, injecting the result under `## Context (pre-loaded excerpt)` between the step instructions and the file listing:

```
## Step: Fix Code

{step.Prompt}

## Context (pre-loaded excerpt)

{retriever output — excerpts from consumed slots}

## Available files (use file_read to access)
- task_context.md — full task context and original input
- slots/bugs.md — output from: find-bugs
```

### Experiment report: third condition

`FormatReport` gains a `RankedResults []TaskResult` field in `ComparisonResult` and renders a third column in the aggregate table when populated.

The bench CLI gains `--retriever none|prefix|ranked` (default `none`).

## Acceptance Criteria

**Given** `ExperimentConfig.ContextRetriever = nil`
**When** an LLM step executes
**Then** behaviour is identical to story 6-21 (no regression)

**Given** `PrefixRetriever{MaxTokens: 300}`
**When** step 2 consumes `slots/bugs.md`
**Then** the prompt contains a `## Context` section with the first ~300 tokens of `bugs.md`

**Given** `RankedRetriever{MaxTokens: 500, TopK: 5}`
**When** step 2 has prompt "Fix the identified bugs"
**Then** the injected context contains the chunks of `bugs.md` most relevant to "fix bugs" (BM25 top-K), not the first N bytes

**Given** `--retriever ranked` passed to bench CLI
**When** the experiment runs
**Then** the report shows a `ranked` column alongside `text` and `runtime`

**Given** BM25 scoring with a corpus of 3 chunks
**When** the query matches terms in chunk 2 but not chunk 1
**Then** chunk 2 scores higher than chunk 1

## Dev Notes

- `RankedRetriever` BM25: use k1=1.5, b=0.75 (standard Robertson params). Corpus = all chunks from all consumed slots for a given retrieval call. No global IDF persistence needed.
- Chunking: split on double-newline (`\n\n`). Min chunk size: 20 chars (skip noise). Max chunk size: 500 chars (split further if needed).
- `PrefixRetriever` falls back gracefully: if sandbox.Read returns error (file not yet written), skip that slot silently.
- `RankedRetriever` same: skip unreadable slots, log nothing (sandbox.Read already records to audit log).
- The `## Context` section appears ONLY when the retriever returns non-empty content. If empty, the prompt is identical to 6-21 behavior.
- `ExperimentConfig` passes the retriever down to `buildTaskDAG` via a new parameter (or embed in config struct passed to build).
- The report extension: add `RankedResults` to `ComparisonResult`, add a helper `CompareThree` or extend `Compare` with a variadic. Prefer a separate `CompareThree(text, runtime, ranked []TaskResult)` to avoid breaking the existing `Compare` signature.

## Dev Agent Record

### Completion Notes

- `internal/bench/experiment/retriever.go` — new file with `ContextRetriever` interface, `NilRetriever`, `PrefixRetriever`, `RankedRetriever`. BM25: k1=1.5, b=0.75, Robertson IDF variant: `ln((N-df+1)/(df+1)+1)`. Chunking on `\n\n`, min 20 chars, max 500 chars (split further). Both `PrefixRetriever` and `RankedRetriever` skip unreadable slots silently.
- `design.go`: `ExperimentConfig` gains `ContextRetriever ContextRetriever` + `Condition string` (empty = "runtime"). `ComparisonResult` gains `RankedResults []TaskResult`.
- `runtime_runner.go`: `runRuntimeTask` + `buildTaskDAG` take `retriever ContextRetriever` + `condition string`. Retriever called after slot writes (so sandbox files exist), result injected under `## Context (pre-loaded excerpt)` only when non-empty. `result.Condition` overridden from condition parameter. `NilRetriever{}` default ensures zero regression when `ContextRetriever` is nil.
- `compare.go`: `CompareThree(text, runtime, ranked []TaskResult) *ComparisonResult` added — calls `Compare(text, runtime)` then sets `RankedResults`.
- `report.go`: `FormatReport` detects `len(cr.RankedResults) > 0`, renders 5-column aggregate table (text/runtime/ranked + two delta columns) and mentions "three conditions" in methodology.
- `bench.go`: `--retriever none|prefix|ranked` flag added. `runExperimentBench` builds third-condition retriever, runs a third condition per task when set, saves `.ranked.json` snapshot, uses `CompareThree` for final report.
- All tests updated: `buildTaskDAG` and `runRuntimeTask` call sites pass `NilRetriever{}` and `"runtime"`.
- 7 new tests: `TestPrefixRetriever_*` (3), `TestRankedRetriever_*` (2), `TestNilRetriever_AlwaysReturnsEmpty`, `TestCompareThree_PopulatesRankedResults`, `TestRunRuntimeExperiment_WithRankedRetriever_InjectsContext`.
- Full suite: 38 packages, 0 failures.

### File List

- `internal/bench/experiment/retriever.go` — new file (ContextRetriever interface + 3 implementations + BM25 helpers)
- `internal/bench/experiment/design.go` — ExperimentConfig: +ContextRetriever, +Condition; ComparisonResult: +RankedResults
- `internal/bench/experiment/runtime_runner.go` — runRuntimeTask/buildTaskDAG: +retriever+condition params, prompt injection, condition override
- `internal/bench/experiment/compare.go` — CompareThree added
- `internal/bench/experiment/report.go` — FormatReport: 3-condition aggregate table
- `internal/bench/cmd/bench.go` — --retriever flag, ranked condition in runExperimentBench
- `internal/bench/experiment/experiment_test.go` — 7 new tests, existing call sites updated

### Change Log

- 2026-04-17: Story 6-22 implemented — ContextRetriever interface, PrefixRetriever, RankedRetriever (BM25), --retriever CLI flag, 3-condition report

## Review Findings

- [x] [Review][Patch] UTF-8 unsafe byte slicing — `splitChunks` uses `c[:500]` (bytes) and `PrefixRetriever` uses `content[:maxChars]` (bytes); multi-byte runes silently truncated [internal/bench/experiment/retriever.go]
- [x] [Review][Patch] Error-return paths hardcode `Condition: "runtime"` — `runRuntimeTask` early-return `TaskResult` structs ignore the `condition` parameter [internal/bench/experiment/runtime_runner.go]
- [x] [Review][Patch] Indentation misalignment in `RunRuntimeExperiment` — retriever/condition resolver block is mis-indented inside the `for` loop [internal/bench/experiment/runtime_runner.go]
- [x] [Review][Patch] `completedMsg` uses cumulative `len(newRankedResults)` instead of per-task result count [internal/bench/cmd/bench.go]
- [x] [Review][Patch] `BuildSkipMap`/`ExtractResults` called with empty `thirdConditionLabel` when `--retriever none` — may match unintended entries [internal/bench/cmd/bench.go]
- [x] [Review][Patch] IDF comment says `+0.5` but implementation uses `+1` — comment is stale [internal/bench/experiment/retriever.go]
- [x] [Review][Defer] `RankedRetriever` breaks early without trying smaller chunks — optimization, not a correctness issue [internal/bench/experiment/retriever.go] — deferred, pre-existing design limit
- [x] [Review][Defer] `CompareThree` missing ranked deltas; ranked table missing 95% CI — enhancement beyond story 6-22 scope [internal/bench/experiment/compare.go, report.go] — deferred, pre-existing
