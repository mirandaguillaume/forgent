# Experiment Checkpoint & Resume

## Problem

`runExperimentBench` accumulates all `TaskResult` values in memory and writes the report only at the end (780 LLM calls for 13 tasks x 30 runs x 2 conditions). A crash at any point loses everything.

## Solution

Two complementary persistence mechanisms:

1. **JSONL checkpoint** — each `TaskResult` is appended to a file immediately after completion.
2. **Flush per condition** — a JSON snapshot is written after each condition (text / runtime) finishes.

A `--resume` flag reloads the checkpoint and skips already-completed runs.

## Design

### Callback in ExperimentConfig

Add an `OnResult` callback to `ExperimentConfig`. Runners call it after each run. The bench CLI wires it to the checkpoint writer.

```go
// experiment/design.go
type ExperimentConfig struct {
    // ... existing fields ...
    OnResult func(result TaskResult, run int) // called after each run completes
}
```

This mirrors the existing `outputCapture` pattern in `runtime_runner.go`.

### CheckpointEntry

```go
// experiment/checkpoint.go
type CheckpointEntry struct {
    TaskID    string     `json:"task_id"`
    Condition string     `json:"condition"` // "text" or "runtime"
    Run       int        `json:"run"`
    Result    TaskResultJSON `json:"result"`
}

// TaskResultJSON is a JSON-safe projection of TaskResult (error -> string).
type TaskResultJSON struct {
    TaskID         string        `json:"task_id"`
    Condition      string        `json:"condition"`
    Completed      bool          `json:"completed"`
    StepResults    []StepResult  `json:"step_results"`
    OrderingScore  float64       `json:"ordering_score"`
    GuardrailScore float64       `json:"guardrail_score"`
    TokensUsed     int           `json:"tokens_used"`
    WallClockMs    int64         `json:"wall_clock_ms"`
    LLMCalls       int           `json:"llm_calls"`
    Error          string        `json:"error,omitempty"`
}
```

### CheckpointWriter

```go
// experiment/checkpoint.go
type CheckpointWriter struct {
    f  *os.File
    mu sync.Mutex
}

func NewCheckpointWriter(path string) (*CheckpointWriter, error)
func (w *CheckpointWriter) Append(entry CheckpointEntry) error  // JSON-encode + newline + fsync
func (w *CheckpointWriter) Close() error
```

### LoadCheckpoint

```go
func LoadCheckpoint(path string) ([]CheckpointEntry, error)
```

Reads JSONL, returns all entries. The caller groups by `(task_id, condition)` to determine which runs to skip.

### Flush per condition

After `RunTextExperiment` completes, write `<output>.text.json`.
After `RunRuntimeExperiment` completes, write `<output>.runtime.json`.

### Resume logic

`--resume` (no value): look for `<output>.checkpoint.jsonl`.
`--resume <path>`: use the explicit path.

On resume:
1. Load checkpoint entries.
2. Reconstruct `[]TaskResult` for each condition.
3. Determine completed run counts per `(task_id, condition)`.
4. Pass skip counts to runners — runners skip runs that are already done.

### Runner changes

Both `RunTextExperiment` and `RunRuntimeExperiment` need:
- Accept a `skipMap map[string]int` (task_id -> runs already completed) for resume.
- Call `config.OnResult(result, runIndex)` after each run.

### bench.go orchestration

```
1. Parse --resume, load checkpoint if present
2. Create CheckpointWriter
3. Wire OnResult callback -> checkpoint.Append
4. Run text condition (skipping completed runs)
5. Flush text results to <output>.text.json
6. Run runtime condition (skipping completed runs)
7. Flush runtime results to <output>.runtime.json
8. Compare + write final report
```

## Files changed

| File | Change |
|------|--------|
| `experiment/design.go` | Add `OnResult` to `ExperimentConfig` |
| `experiment/checkpoint.go` | **New** — `CheckpointEntry`, `CheckpointWriter`, `LoadCheckpoint`, `TaskResultJSON` |
| `experiment/text_runner.go` | Call `OnResult`, accept skip map |
| `experiment/runtime_runner.go` | Call `OnResult`, accept skip map |
| `bench/cmd/bench.go` | `--resume` flag, checkpoint wiring, flush per condition |
| `experiment/experiment_test.go` | Tests for checkpoint round-trip, resume logic |
