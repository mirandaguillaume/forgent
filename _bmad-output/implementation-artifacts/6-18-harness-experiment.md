# Story 6-18: Harness Experiment — Wire Middleware Guards into Runtime Runner

**Status:** review
**Epic:** 6 — Runtime Harness
**Sprint:** 7

## Goal

Wire the implemented middleware guards (QualityGateMiddleware, BudgetMiddleware) into the experiment runtime runner so that running `--harness` attaches programmatic enforcement to DAG nodes. This enables a controlled comparison: baseline (no middlewares) vs harness (with middlewares) on identical tasks.

## Changes

### 1. `internal/bench/experiment/design.go`
- Added `HarnessEnabled bool` field to `ExperimentConfig`

### 2. `internal/bench/experiment/runtime_runner.go`
- Added imports for `pkg/budget` and `pkg/qualitygate`
- Threaded `harnessEnabled bool` through `RunRuntimeExperiment` -> `runRuntimeTask` -> `buildTaskDAG`
- In `buildTaskDAG`, when `harnessEnabled` is true and step is an LLM step:
  - Attaches `QualityGateMiddleware` using the step's `ExpectedKeywords`
  - Attaches `BudgetMiddleware` with default policy (MaxTokens=8192, Timeout=60s)
  - Sets `MaxRetries=2` on the node so quality gate can trigger retries
- Static and assert steps are NOT affected (no middlewares attached)

### 3. `internal/bench/cmd/bench.go`
- Reads the existing `--harness` flag in `runExperimentBench`
- Passes `HarnessEnabled` to the runtime `ExperimentConfig` (text config is unmodified)
- Logs harness state in the experiment start message

## Usage

```bash
# Baseline (no middlewares)
forgent-bench bench . --level experiment --tasks-dir internal/bench/experiment/tasks --model haiku --runs 3 --output experiment-no-harness.md

# With harness (QualityGate + Budget middlewares on runtime nodes)
forgent-bench bench . --level experiment --tasks-dir internal/bench/experiment/tasks --model haiku --runs 3 --output experiment-with-harness.md --harness
```

## Acceptance Criteria

- [x] `go build ./cmd/forgent-bench` compiles
- [x] `go test ./... -count=1 -short` passes
- [x] `--harness` flag wires middlewares only to runtime condition LLM nodes
- [x] Text runner is unmodified
- [x] `compare.go` and `report.go` are unmodified
- [x] Without `--harness`, behavior is identical to before (no middlewares attached)
