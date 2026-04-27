# Experiment Checkpoint & Resume Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add JSONL checkpoint persistence and `--resume` flag to the experiment bench so that a crash doesn't lose all completed results.

**Architecture:** Each `TaskResult` is appended to a `.checkpoint.jsonl` file immediately via an `OnResult` callback in `ExperimentConfig`. Condition-level snapshots (`.text.json`, `.runtime.json`) provide a secondary safety net. The `--resume` flag reloads the checkpoint and skips already-completed runs.

**Tech Stack:** Go 1.22, encoding/json, os (file I/O), testify (tests), cobra (CLI flags)

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/bench/experiment/design.go` | Modify | Add `OnResult` callback + `SkipMap` to `ExperimentConfig` |
| `internal/bench/experiment/checkpoint.go` | Create | `CheckpointWriter`, `LoadCheckpoint`, `TaskResultJSON`, `CheckpointEntry` |
| `internal/bench/experiment/checkpoint_test.go` | Create | Round-trip, append, resume skip logic tests |
| `internal/bench/experiment/text_runner.go` | Modify | Call `OnResult`, respect `SkipMap` |
| `internal/bench/experiment/runtime_runner.go` | Modify | Call `OnResult`, respect `SkipMap` |
| `internal/bench/experiment/experiment_test.go` | Modify | Add tests for `OnResult` callback + skip behavior |
| `internal/bench/cmd/bench.go` | Modify | `--resume` flag, checkpoint wiring, flush per condition |

---

### Task 1: Add OnResult callback and SkipMap to ExperimentConfig

**Files:**
- Modify: `internal/bench/experiment/design.go:18-26`
- Test: `internal/bench/experiment/experiment_test.go`

- [ ] **Step 1: Write the failing test for OnResult callback**

In `experiment_test.go`, add after the existing `TestExperimentConfig_Validate` tests:

```go
func TestOnResult_CalledPerRun(t *testing.T) {
	var called []TaskResult
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(),
		Provider:    sampleProvider(),
		RunsPerTask: 2,
		OnResult: func(r TaskResult, run int) {
			called = append(called, r)
		},
	}

	results, err := RunTextExperiment(cfg)
	require.NoError(t, err)
	assert.Len(t, results, 4) // 2 tasks x 2 runs
	assert.Len(t, called, 4, "OnResult should be called for each run")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bench/experiment/ -run TestOnResult_CalledPerRun -v`
Expected: compilation error — `OnResult` field does not exist on `ExperimentConfig`

- [ ] **Step 3: Add OnResult and SkipMap fields to ExperimentConfig**

In `design.go`, modify the `ExperimentConfig` struct:

```go
type ExperimentConfig struct {
	Tasks       []TaskDefinition
	Provider    llm.Provider
	Model       string
	RunsPerTask int
	Seed        int64
	OnResult    func(result TaskResult, run int) // called after each run completes; nil = no-op
	SkipMap     map[string]int                   // taskID → runs already completed (for resume)
}
```

- [ ] **Step 4: Call OnResult in RunTextExperiment**

In `text_runner.go`, after `result := runTextTask(...)`, add the callback call:

```go
func RunTextExperiment(config ExperimentConfig) ([]TaskResult, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	total := len(config.Tasks) * config.RunsPerTask
	done := 0
	var results []TaskResult
	for i, task := range config.Tasks {
		skip := config.SkipMap[task.ID]
		for run := 0; run < config.RunsPerTask; run++ {
			done++
			if run < skip {
				fmt.Fprintf(os.Stderr, "  [text] task %d/%d (%s) run %d/%d (skipped, resumed)\n",
					i+1, len(config.Tasks), task.ID, run+1, config.RunsPerTask)
				continue
			}
			fmt.Fprintf(os.Stderr, "  [text] task %d/%d (%s) run %d/%d (%d/%d total)\n",
				i+1, len(config.Tasks), task.ID, run+1, config.RunsPerTask, done, total)
			result := runTextTask(task, config.Provider)
			results = append(results, result)
			if config.OnResult != nil {
				config.OnResult(result, run)
			}
		}
	}
	return results, nil
}
```

- [ ] **Step 5: Call OnResult in RunRuntimeExperiment**

In `runtime_runner.go`, same pattern after `result := runRuntimeTask(...)`:

```go
func RunRuntimeExperiment(config ExperimentConfig) ([]TaskResult, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	total := len(config.Tasks) * config.RunsPerTask
	done := 0
	var results []TaskResult
	for i, task := range config.Tasks {
		skip := config.SkipMap[task.ID]
		for run := 0; run < config.RunsPerTask; run++ {
			done++
			if run < skip {
				fmt.Fprintf(os.Stderr, "  [runtime] task %d/%d (%s) run %d/%d (skipped, resumed)\n",
					i+1, len(config.Tasks), task.ID, run+1, config.RunsPerTask)
				continue
			}
			fmt.Fprintf(os.Stderr, "  [runtime] task %d/%d (%s, %d steps) run %d/%d (%d/%d total)\n",
				i+1, len(config.Tasks), task.ID, len(task.Steps), run+1, config.RunsPerTask, done, total)
			result := runRuntimeTask(task, config.Provider)
			results = append(results, result)
			if config.OnResult != nil {
				config.OnResult(result, run)
			}
		}
	}
	return results, nil
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/bench/experiment/ -v`
Expected: all tests PASS including `TestOnResult_CalledPerRun`

- [ ] **Step 7: Commit**

```bash
git add internal/bench/experiment/design.go internal/bench/experiment/text_runner.go internal/bench/experiment/runtime_runner.go internal/bench/experiment/experiment_test.go
git commit -m "feat(experiment): add OnResult callback and SkipMap to ExperimentConfig"
```

---

### Task 2: Create checkpoint.go with CheckpointWriter and LoadCheckpoint

**Files:**
- Create: `internal/bench/experiment/checkpoint.go`
- Create: `internal/bench/experiment/checkpoint_test.go`

- [ ] **Step 1: Write failing tests for TaskResultJSON round-trip**

Create `checkpoint_test.go`:

```go
package experiment

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskResultJSON_RoundTrip(t *testing.T) {
	original := TaskResult{
		TaskID:         "code-review",
		Condition:      "text",
		Completed:      true,
		OrderingScore:  0.9,
		GuardrailScore: 1.0,
		TokensUsed:     500,
		WallClockMs:    1200,
		LLMCalls:       3,
		StepResults: []StepResult{
			{StepID: "s1", Completed: true, KeywordsHit: 2, KeywordsTotal: 3, TokensUsed: 100},
		},
	}

	j := ToResultJSON(original)
	back := j.ToTaskResult()

	assert.Equal(t, original.TaskID, back.TaskID)
	assert.Equal(t, original.Completed, back.Completed)
	assert.Equal(t, original.OrderingScore, back.OrderingScore)
	assert.Equal(t, original.TokensUsed, back.TokensUsed)
	assert.Len(t, back.StepResults, 1)
	assert.Nil(t, back.Error)
}

func TestTaskResultJSON_WithError(t *testing.T) {
	original := TaskResult{
		TaskID:    "t1",
		Condition: "runtime",
		Error:     fmt.Errorf("transient LLM error"),
	}

	j := ToResultJSON(original)
	assert.Equal(t, "transient LLM error", j.Error)

	back := j.ToTaskResult()
	assert.NotNil(t, back.Error)
	assert.Equal(t, "transient LLM error", back.Error.Error())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/bench/experiment/ -run TestTaskResultJSON -v`
Expected: compilation error — `ToResultJSON`, `TaskResultJSON` not defined

- [ ] **Step 3: Write TaskResultJSON and conversion functions**

Create `checkpoint.go`:

```go
package experiment

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// TaskResultJSON is a JSON-safe projection of TaskResult (error → string).
type TaskResultJSON struct {
	TaskID         string       `json:"task_id"`
	Condition      string       `json:"condition"`
	Completed      bool         `json:"completed"`
	StepResults    []StepResult `json:"step_results"`
	OrderingScore  float64      `json:"ordering_score"`
	GuardrailScore float64      `json:"guardrail_score"`
	TokensUsed     int          `json:"tokens_used"`
	WallClockMs    int64        `json:"wall_clock_ms"`
	LLMCalls       int          `json:"llm_calls"`
	Error          string       `json:"error,omitempty"`
}

// ToResultJSON converts a TaskResult to its JSON-safe form.
func ToResultJSON(r TaskResult) TaskResultJSON {
	j := TaskResultJSON{
		TaskID:         r.TaskID,
		Condition:      r.Condition,
		Completed:      r.Completed,
		StepResults:    r.StepResults,
		OrderingScore:  r.OrderingScore,
		GuardrailScore: r.GuardrailScore,
		TokensUsed:     r.TokensUsed,
		WallClockMs:    r.WallClockMs,
		LLMCalls:       r.LLMCalls,
	}
	if r.Error != nil {
		j.Error = r.Error.Error()
	}
	return j
}

// ToTaskResult converts back to a TaskResult.
func (j TaskResultJSON) ToTaskResult() TaskResult {
	r := TaskResult{
		TaskID:         j.TaskID,
		Condition:      j.Condition,
		Completed:      j.Completed,
		StepResults:    j.StepResults,
		OrderingScore:  j.OrderingScore,
		GuardrailScore: j.GuardrailScore,
		TokensUsed:     j.TokensUsed,
		WallClockMs:    j.WallClockMs,
		LLMCalls:       j.LLMCalls,
	}
	if j.Error != "" {
		r.Error = fmt.Errorf("%s", j.Error)
	}
	return r
}

// CheckpointEntry is one line in the JSONL checkpoint file.
type CheckpointEntry struct {
	TaskID    string         `json:"task_id"`
	Condition string         `json:"condition"`
	Run       int            `json:"run"`
	Result    TaskResultJSON `json:"result"`
}

// CheckpointWriter appends entries to a JSONL file.
type CheckpointWriter struct {
	f  *os.File
	mu sync.Mutex
}

// NewCheckpointWriter opens (or creates) a JSONL checkpoint file for appending.
func NewCheckpointWriter(path string) (*CheckpointWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open checkpoint: %w", err)
	}
	return &CheckpointWriter{f: f}, nil
}

// Append writes a single checkpoint entry as one JSON line and flushes to disk.
func (w *CheckpointWriter) Append(entry CheckpointEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	data = append(data, '\n')

	if _, err := w.f.Write(data); err != nil {
		return fmt.Errorf("write entry: %w", err)
	}
	return w.f.Sync()
}

// Close closes the underlying file.
func (w *CheckpointWriter) Close() error {
	return w.f.Close()
}

// LoadCheckpoint reads all entries from a JSONL checkpoint file.
func LoadCheckpoint(path string) ([]CheckpointEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open checkpoint: %w", err)
	}
	defer f.Close()

	var entries []CheckpointEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB lines
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry CheckpointEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan checkpoint: %w", err)
	}
	return entries, nil
}

// BuildSkipMap groups checkpoint entries by (condition, taskID) and returns
// a map of taskID → completed run count for the given condition.
func BuildSkipMap(entries []CheckpointEntry, condition string) map[string]int {
	counts := make(map[string]int)
	for _, e := range entries {
		if e.Condition == condition {
			counts[e.TaskID]++
		}
	}
	return counts
}

// ExtractResults converts checkpoint entries for a given condition back to TaskResults.
func ExtractResults(entries []CheckpointEntry, condition string) []TaskResult {
	var results []TaskResult
	for _, e := range entries {
		if e.Condition == condition {
			results = append(results, e.Result.ToTaskResult())
		}
	}
	return results
}
```

- [ ] **Step 4: Run tests to verify round-trip passes**

Run: `go test ./internal/bench/experiment/ -run TestTaskResultJSON -v`
Expected: PASS

- [ ] **Step 5: Write tests for CheckpointWriter and LoadCheckpoint**

Append to `checkpoint_test.go`:

```go
func TestCheckpointWriter_AppendAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.checkpoint.jsonl")

	w, err := NewCheckpointWriter(path)
	require.NoError(t, err)

	entries := []CheckpointEntry{
		{TaskID: "t1", Condition: "text", Run: 0, Result: ToResultJSON(TaskResult{TaskID: "t1", Condition: "text", Completed: true, TokensUsed: 100})},
		{TaskID: "t1", Condition: "text", Run: 1, Result: ToResultJSON(TaskResult{TaskID: "t1", Condition: "text", Completed: true, TokensUsed: 110})},
		{TaskID: "t2", Condition: "text", Run: 0, Result: ToResultJSON(TaskResult{TaskID: "t2", Condition: "text", Completed: false, TokensUsed: 200})},
		{TaskID: "t1", Condition: "runtime", Run: 0, Result: ToResultJSON(TaskResult{TaskID: "t1", Condition: "runtime", Completed: true, TokensUsed: 300})},
	}

	for _, e := range entries {
		require.NoError(t, w.Append(e))
	}
	require.NoError(t, w.Close())

	loaded, err := LoadCheckpoint(path)
	require.NoError(t, err)
	assert.Len(t, loaded, 4)
	assert.Equal(t, "t1", loaded[0].TaskID)
	assert.Equal(t, "runtime", loaded[3].Condition)
}

func TestBuildSkipMap(t *testing.T) {
	entries := []CheckpointEntry{
		{TaskID: "t1", Condition: "text", Run: 0},
		{TaskID: "t1", Condition: "text", Run: 1},
		{TaskID: "t2", Condition: "text", Run: 0},
		{TaskID: "t1", Condition: "runtime", Run: 0},
	}

	textSkip := BuildSkipMap(entries, "text")
	assert.Equal(t, 2, textSkip["t1"])
	assert.Equal(t, 1, textSkip["t2"])
	assert.Equal(t, 0, textSkip["t3"]) // missing = 0

	rtSkip := BuildSkipMap(entries, "runtime")
	assert.Equal(t, 1, rtSkip["t1"])
	assert.Equal(t, 0, rtSkip["t2"])
}

func TestExtractResults(t *testing.T) {
	entries := []CheckpointEntry{
		{TaskID: "t1", Condition: "text", Result: ToResultJSON(TaskResult{TaskID: "t1", Condition: "text", Completed: true})},
		{TaskID: "t1", Condition: "runtime", Result: ToResultJSON(TaskResult{TaskID: "t1", Condition: "runtime", Completed: true})},
		{TaskID: "t2", Condition: "text", Result: ToResultJSON(TaskResult{TaskID: "t2", Condition: "text", Completed: false})},
	}

	textResults := ExtractResults(entries, "text")
	assert.Len(t, textResults, 2)
	assert.Equal(t, "t1", textResults[0].TaskID)
	assert.Equal(t, "t2", textResults[1].TaskID)

	rtResults := ExtractResults(entries, "runtime")
	assert.Len(t, rtResults, 1)
}

func TestLoadCheckpoint_FileNotFound(t *testing.T) {
	_, err := LoadCheckpoint("/nonexistent/path.jsonl")
	require.Error(t, err)
}
```

- [ ] **Step 6: Run all checkpoint tests**

Run: `go test ./internal/bench/experiment/ -run "TestCheckpoint|TestBuildSkipMap|TestExtractResults|TestLoadCheckpoint_FileNotFound" -v`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/bench/experiment/checkpoint.go internal/bench/experiment/checkpoint_test.go
git commit -m "feat(experiment): add JSONL checkpoint writer and loader"
```

---

### Task 3: Add SkipMap support to runners and test resume behavior

**Files:**
- Modify: `internal/bench/experiment/experiment_test.go`

- [ ] **Step 1: Write test for SkipMap in text runner**

```go
func TestRunTextExperiment_WithSkipMap(t *testing.T) {
	var called int
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(),
		Provider:    sampleProvider(),
		RunsPerTask: 3,
		SkipMap:     map[string]int{"code-review": 2}, // skip first 2 runs of code-review
		OnResult: func(r TaskResult, run int) {
			called++
		},
	}

	results, err := RunTextExperiment(cfg)
	require.NoError(t, err)

	// code-review: 3 runs - 2 skipped = 1 new
	// doc-gen: 3 runs - 0 skipped = 3 new
	assert.Len(t, results, 4)
	assert.Equal(t, 4, called)
}

func TestRunRuntimeExperiment_WithSkipMap(t *testing.T) {
	var called int
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(),
		Provider:    sampleProvider(),
		RunsPerTask: 3,
		SkipMap:     map[string]int{"doc-gen": 3}, // skip all runs of doc-gen
		OnResult: func(r TaskResult, run int) {
			called++
		},
	}

	results, err := RunRuntimeExperiment(cfg)
	require.NoError(t, err)

	// code-review: 3 new, doc-gen: 0 new
	assert.Len(t, results, 3)
	assert.Equal(t, 3, called)
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/bench/experiment/ -run "TestRunTextExperiment_WithSkipMap|TestRunRuntimeExperiment_WithSkipMap" -v`
Expected: PASS (the SkipMap logic was added in Task 1 Step 4-5)

- [ ] **Step 3: Run full test suite**

Run: `go test ./internal/bench/experiment/ -v`
Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/bench/experiment/experiment_test.go
git commit -m "test(experiment): add SkipMap and OnResult coverage"
```

---

### Task 4: Wire checkpoint and --resume into bench CLI

**Files:**
- Modify: `internal/bench/cmd/bench.go:1487-1545`

- [ ] **Step 1: Add --resume flag to bench command**

In `bench.go`, in the `init()` function, after the `benchCmd.Flags().Int("runs", ...)` line, add:

```go
benchCmd.Flags().String("resume", "", "Resume from checkpoint file (empty string with --output = auto-detect)")
```

- [ ] **Step 2: Rewrite runExperimentBench with checkpoint wiring**

Replace the entire `runExperimentBench` function:

```go
func runExperimentBench(cmd *cobra.Command) error {
	tasksDir, _ := cmd.Flags().GetString("tasks-dir")
	if tasksDir == "" {
		return fmt.Errorf("--tasks-dir is required for experiment level")
	}

	modelName, _ := cmd.Flags().GetString("model")
	runs, _ := cmd.Flags().GetInt("runs")
	outputPath, _ := cmd.Flags().GetString("output")
	resumePath, _ := cmd.Flags().GetString("resume")

	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return fmt.Errorf("experiment requires an LLM provider: %w", err)
	}

	tasks, err := experiment.LoadTasks(tasksDir)
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	// Resolve checkpoint path
	checkpointPath := resumePath
	if checkpointPath == "" && outputPath != "" {
		checkpointPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".checkpoint.jsonl"
	}

	// Load existing checkpoint for resume
	var existingEntries []experiment.CheckpointEntry
	var priorTextResults, priorRuntimeResults []experiment.TaskResult

	if resumePath != "" || (cmd.Flags().Changed("resume") && outputPath != "") {
		loadPath := resumePath
		if loadPath == "" {
			loadPath = checkpointPath
		}
		if _, err := os.Stat(loadPath); err == nil {
			existingEntries, err = experiment.LoadCheckpoint(loadPath)
			if err != nil {
				return fmt.Errorf("load checkpoint: %w", err)
			}
			priorTextResults = experiment.ExtractResults(existingEntries, "text")
			priorRuntimeResults = experiment.ExtractResults(existingEntries, "runtime")
			fmt.Fprintf(os.Stderr, "Resumed from checkpoint: %d text results, %d runtime results\n",
				len(priorTextResults), len(priorRuntimeResults))
		}
	}

	// Open checkpoint writer
	var cpWriter *experiment.CheckpointWriter
	if checkpointPath != "" {
		cpWriter, err = experiment.NewCheckpointWriter(checkpointPath)
		if err != nil {
			return fmt.Errorf("open checkpoint: %w", err)
		}
		defer cpWriter.Close()
	}

	// Build configs per condition
	textConfig := experiment.ExperimentConfig{
		Tasks:       tasks,
		Provider:    provider,
		Model:       modelName,
		RunsPerTask: runs,
		SkipMap:     experiment.BuildSkipMap(existingEntries, "text"),
		OnResult: func(r experiment.TaskResult, run int) {
			if cpWriter != nil {
				_ = cpWriter.Append(experiment.CheckpointEntry{
					TaskID:    r.TaskID,
					Condition: "text",
					Run:       run,
					Result:    experiment.ToResultJSON(r),
				})
			}
		},
	}

	runtimeConfig := experiment.ExperimentConfig{
		Tasks:       tasks,
		Provider:    provider,
		Model:       modelName,
		RunsPerTask: runs,
		SkipMap:     experiment.BuildSkipMap(existingEntries, "runtime"),
		OnResult: func(r experiment.TaskResult, run int) {
			if cpWriter != nil {
				_ = cpWriter.Append(experiment.CheckpointEntry{
					TaskID:    r.TaskID,
					Condition: "runtime",
					Run:       run,
					Result:    experiment.ToResultJSON(r),
				})
			}
		},
	}

	fmt.Fprintf(os.Stderr, "Running experiment: %d tasks, %d runs per task...\n", len(tasks), runs)

	// Condition 1: Text
	fmt.Fprintf(os.Stderr, "Condition 1: Text instruction...\n")
	newTextResults, err := experiment.RunTextExperiment(textConfig)
	if err != nil {
		return fmt.Errorf("text experiment: %w", err)
	}
	textResults := append(priorTextResults, newTextResults...)

	// Flush text condition snapshot
	if outputPath != "" {
		flushPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".text.json"
		if data, err := json.MarshalIndent(textResults, "", "  "); err == nil {
			_ = os.WriteFile(flushPath, data, 0644)
			fmt.Fprintf(os.Stderr, "Text results flushed to %s\n", flushPath)
		}
	}

	// Condition 2: Runtime
	fmt.Fprintf(os.Stderr, "Condition 2: Runtime (DAG)...\n")
	newRuntimeResults, err := experiment.RunRuntimeExperiment(runtimeConfig)
	if err != nil {
		return fmt.Errorf("runtime experiment: %w", err)
	}
	runtimeResults := append(priorRuntimeResults, newRuntimeResults...)

	// Flush runtime condition snapshot
	if outputPath != "" {
		flushPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".runtime.json"
		if data, err := json.MarshalIndent(runtimeResults, "", "  "); err == nil {
			_ = os.WriteFile(flushPath, data, 0644)
			fmt.Fprintf(os.Stderr, "Runtime results flushed to %s\n", flushPath)
		}
	}

	// Compare + report
	fmt.Fprintf(os.Stderr, "Comparing results...\n")
	comparison := experiment.Compare(textResults, runtimeResults)

	report := experiment.FormatReport(comparison, experiment.ReportOpts{
		Model: modelName,
	})

	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(report), 0644); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Report written to %s\n", outputPath)
	} else {
		fmt.Print(report)
	}

	return nil
}
```

- [ ] **Step 3: Add missing imports to bench.go**

Ensure these imports are present at the top of `bench.go`:

```go
"encoding/json"
"path/filepath"
"strings"
"github.com/mirandaguillaume/forgent/internal/bench/experiment"
```

(`encoding/json`, `path/filepath`, and `strings` are likely already imported — verify and add only what's missing)

- [ ] **Step 4: Build to verify compilation**

Run: `go build ./cmd/forgent-bench`
Expected: builds without errors

- [ ] **Step 5: Run full test suite**

Run: `go test ./internal/bench/... -v`
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/bench/cmd/bench.go
git commit -m "feat(bench): wire checkpoint persistence and --resume flag"
```

---

### Task 5: Manual smoke test

**Files:** (no code changes)

- [ ] **Step 1: Run experiment with checkpoint (2 runs, fast)**

```bash
./forgent-bench bench . --level experiment --tasks-dir internal/bench/experiment/tasks/ --provider anthropic --model haiku --runs 2 --output smoke-test.md
```

- [ ] **Step 2: Verify checkpoint file was created**

```bash
ls -la smoke-test.checkpoint.jsonl smoke-test.text.json smoke-test.runtime.json
```

Expected: all 3 files exist. The JSONL file has one line per run (2 tasks × 2 runs × 2 conditions = depends on task count).

- [ ] **Step 3: Verify checkpoint content**

```bash
head -3 smoke-test.checkpoint.jsonl | python3 -m json.tool
```

Expected: valid JSON with `task_id`, `condition`, `run`, `result` fields.

- [ ] **Step 4: Test resume (kill and restart)**

```bash
# Run with 5 runs but kill after ~10 seconds
timeout 10 ./forgent-bench bench . --level experiment --tasks-dir internal/bench/experiment/tasks/ --provider anthropic --model haiku --runs 5 --output resume-test.md || true

# Check how many entries were saved
wc -l resume-test.checkpoint.jsonl

# Resume
./forgent-bench bench . --level experiment --tasks-dir internal/bench/experiment/tasks/ --provider anthropic --model haiku --runs 5 --output resume-test.md --resume resume-test.checkpoint.jsonl
```

Expected: second run prints "Resumed from checkpoint: N text results, M runtime results" and skips completed runs.

- [ ] **Step 5: Clean up smoke test files**

```bash
rm -f smoke-test.md smoke-test.checkpoint.jsonl smoke-test.text.json smoke-test.runtime.json
rm -f resume-test.md resume-test.checkpoint.jsonl resume-test.text.json resume-test.runtime.json
```

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "test(experiment): verify checkpoint and resume smoke test"
```

(Only commit if there are any remaining changes — the smoke test files should be cleaned up.)
