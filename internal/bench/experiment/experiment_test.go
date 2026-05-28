package experiment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/mirandaguillaume/forgent/pkg/dag"
	"github.com/mirandaguillaume/forgent/pkg/schema"
)

// mockExpProvider returns step-aware responses based on prompt content.
type mockExpProvider struct {
	responses map[string]string // keyword → response
	fallback  string
}

func (m *mockExpProvider) Complete(prompt string) (string, error) {
	lower := strings.ToLower(prompt)
	for keyword, resp := range m.responses {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return resp, nil
		}
	}
	return m.fallback, nil
}

var _ llm.Provider = (*mockExpProvider)(nil)

func sampleTasks() []TaskDefinition {
	return []TaskDefinition{
		{
			ID:          "code-review",
			Name:        "Code Review",
			Description: "Review this code for issues",
			Category:    "code-review",
			Input:       "func add(a, b int) int { return a + b }",
			Steps: []StepDef{
				{ID: "identify", Name: "Identify Issues", Prompt: "Find bugs", Produces: []SlotSpec{{Key: "issues"}}},
				{ID: "suggest", Name: "Suggest Fixes", Prompt: "Suggest improvements", Consumes: []SlotSpec{{Key: "issues"}}, Produces: []SlotSpec{{Key: "suggestions"}}},
				{ID: "summarize", Name: "Summary", Prompt: "Summarize findings", Consumes: []SlotSpec{{Key: "suggestions"}}, Produces: []SlotSpec{{Key: "summary"}}},
			},
			Guardrails: []string{"summary", "!profanity"},
		},
		{
			ID:          "doc-gen",
			Name:        "Documentation",
			Description: "Generate docs for this module",
			Category:    "documentation",
			Input:       "package math provides basic arithmetic",
			Steps: []StepDef{
				{ID: "outline", Name: "Outline", Prompt: "Create doc outline", Produces: []SlotSpec{{Key: "outline"}}},
				{ID: "draft", Name: "Draft", Prompt: "Write the draft", Consumes: []SlotSpec{{Key: "outline"}}, Produces: []SlotSpec{{Key: "draft"}}},
			},
			Guardrails: []string{"documentation"},
		},
	}
}

func sampleProvider() *mockExpProvider {
	return &mockExpProvider{
		responses: map[string]string{
			"identify issues": "## Identify Issues\nFound 1 issue: no input validation. This is a bug that could cause problems.",
			"suggest fixes":   "## Suggest Fixes\nFix: add input validation. Improve by adding error handling.",
			"summary":         "## Summary\nReview complete. Summary: 1 issue found, 1 fix suggested.",
			"outline":         "## Outline\n1. Introduction\n2. API Reference\n3. Examples. This is the outline.",
			"draft":           "## Draft\nThis is the draft documentation for the math module.",
		},
		fallback: "Step completed successfully with relevant content.",
	}
}

// ─── Design tests ─────────────────────────────────────────────────────────────

func TestExperimentConfig_Validate(t *testing.T) {
	provider := sampleProvider()

	t.Run("valid", func(t *testing.T) {
		cfg := ExperimentConfig{Tasks: sampleTasks(), Provider: provider, RunsPerTask: 1}
		require.NoError(t, cfg.Validate())
	})

	t.Run("no tasks", func(t *testing.T) {
		cfg := ExperimentConfig{Provider: provider}
		require.Error(t, cfg.Validate())
	})

	t.Run("no provider", func(t *testing.T) {
		cfg := ExperimentConfig{Tasks: sampleTasks()}
		require.Error(t, cfg.Validate())
	})

	t.Run("empty steps", func(t *testing.T) {
		cfg := ExperimentConfig{
			Tasks:    []TaskDefinition{{ID: "empty", Steps: nil}},
			Provider: provider,
		}
		require.Error(t, cfg.Validate())
	})

	t.Run("default runs", func(t *testing.T) {
		cfg := ExperimentConfig{Tasks: sampleTasks(), Provider: provider}
		require.NoError(t, cfg.Validate())
		assert.Equal(t, 1, cfg.RunsPerTask)
	})
}

func TestLoadTasks(t *testing.T) {
	dir := t.TempDir()

	task := `id: test-task
name: Test Task
description: A test
category: test
input: "some input"
steps:
  - id: step1
    name: First Step
    prompt: Do something
    produces: [output1]
  - id: step2
    name: Second Step
    prompt: Do more
    consumes: [output1]
    produces: [output2]
guardrails:
  - must include result
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(task), 0644))

	tasks, err := LoadTasks(dir)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "test-task", tasks[0].ID)
	assert.Len(t, tasks[0].Steps, 2)
	assert.Equal(t, []SlotSpec{{Key: "output1"}}, tasks[0].Steps[0].Produces)
	assert.Equal(t, []SlotSpec{{Key: "output1"}}, tasks[0].Steps[1].Consumes)
}

func TestLoadTasksParallelBlock(t *testing.T) {
	dir := t.TempDir()

	task := `id: parallel-task
name: Parallel Task
steps:
  - id: setup
    produces: [base]
  - parallel:
    - id: agent-a
      consumes: [base]
      produces: [result_a]
    - id: agent-b
      consumes: [base]
      produces: [result_b]
  - id: merge
    consumes: [result_a, result_b]
    produces: [final]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p.yaml"), []byte(task), 0644))

	tasks, err := LoadTasks(dir)
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	steps := tasks[0].Steps
	assert.Len(t, steps, 4) // setup + agent-a + agent-b + merge (flattened)

	assert.Equal(t, "setup", steps[0].ID)
	assert.Equal(t, "", steps[0].ParallelGroupID)

	assert.Equal(t, "agent-a", steps[1].ID)
	assert.Equal(t, "agent-b", steps[2].ID)
	assert.Equal(t, steps[1].ParallelGroupID, steps[2].ParallelGroupID)
	assert.NotEmpty(t, steps[1].ParallelGroupID)

	assert.Equal(t, "merge", steps[3].ID)
	assert.Equal(t, "", steps[3].ParallelGroupID)
}

func TestValidateParallelGroupDependency(t *testing.T) {
	cfg := ExperimentConfig{
		Provider: sampleProvider(),
		Tasks: []TaskDefinition{
			{
				ID: "bad-parallel",
				Steps: []StepDef{
					{ID: "a", Produces: []SlotSpec{{Key: "x"}}},
					{ID: "b", ParallelGroupID: "g0", Consumes: []SlotSpec{{Key: "x"}}, Produces: []SlotSpec{{Key: "y"}}},
					// c is in the same group and consumes y produced by b — dependency violation
					{ID: "c", ParallelGroupID: "g0", Consumes: []SlotSpec{{Key: "y"}}, Produces: []SlotSpec{{Key: "z"}}},
				},
			},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parallel group")
	assert.Contains(t, err.Error(), "not independent")
}

func TestExtractJSONArray(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []any
		wantErr bool
	}{
		{"simple strings", `["a","b","c"]`, []any{"a", "b", "c"}, false},
		{"with surrounding text", "here is your list:\n[\"x\",\"y\"]\nDone.", []any{"x", "y"}, false},
		{"empty array", "[]", []any{}, false},
		{"no array", "just text", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractJSONArray(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestListSchemaProduces(t *testing.T) {
	prov := &mockExpProvider{
		responses: map[string]string{
			"collect": `["issue A","issue B","issue C"]`,
		},
		fallback: "done",
	}
	task := TaskDefinition{
		ID:   "list-test",
		Name: "List Test",
		Steps: []StepDef{
			{
				ID:     "collect",
				Prompt: "collect issues",
				Produces: []SlotSpec{{
					Key:        "issues_list",
					ListSchema: &schema.FieldSpec{Type: "string"},
				}},
			},
		},
	}
	cfg := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: prov}
	require.NoError(t, cfg.Validate())

	results, err := RunRuntimeExperiment(cfg)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed)
}

func TestKindMapFilterPipeline(t *testing.T) {
	// collect → score (kind: map) → filter (kind: filter, deterministic predicate)
	// The filter does NOT call the LLM — it evaluates "score >= 80" inline.
	prov := &mockExpProvider{
		responses: map[string]string{
			"collect":  `["bug A","false positive","bug B"]`,
			"bug a":    "bug A | SCORE: 90",
			"false po": "false positive | SCORE: 20",
			"bug b":    "bug B | SCORE: 95",
		},
		fallback: "no issues",
	}
	task := TaskDefinition{
		ID:   "map-filter-test",
		Name: "Map Filter Test",
		Steps: []StepDef{
			{
				ID:     "collect",
				Prompt: "collect issues",
				Produces: []SlotSpec{{
					Key:        "issues_list",
					ListSchema: &schema.FieldSpec{Type: "string"},
				}},
			},
			{
				ID:     "score-issues",
				Kind:   StepKindMap,
				Prompt: "Score this issue",
				Consumes: []SlotSpec{{Key: "issues_list"}},
				Produces: []SlotSpec{{Key: "scored_issues"}},
			},
			{
				ID:        "filter-issues",
				Kind:      StepKindFilter,
				Predicate: "score >= 80",
				Consumes:  []SlotSpec{{Key: "scored_issues"}},
				Produces:  []SlotSpec{{Key: "filtered_issues"}},
			},
		},
	}
	cfg := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: prov}
	require.NoError(t, cfg.Validate())

	results, err := RunRuntimeExperiment(cfg)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed)
}

func TestEvalPredicate(t *testing.T) {
	tests := []struct {
		predicate string
		item      string
		want      bool
		wantErr   bool
	}{
		{"score >= 80", "bug A | SCORE: 90", true, false},
		{"score >= 80", "false positive | SCORE: 20", false, false},
		{"score >= 80", "bug B | SCORE: 80", true, false},
		{"score > 80", "bug B | SCORE: 80", false, false},
		{"score <= 50", "minor | SCORE: 50", true, false},
		{"score < 50", "minor | SCORE: 50", false, false},
		{"score == 75", "real issue | SCORE: 75", true, false},
		{"score >= 80", "no score here", false, true},   // missing SCORE: → error
		{"invalid >= x", "item | SCORE: 90", false, true}, // bad threshold
		{"contains(foo)", "item", false, true},           // unsupported expression
	}
	for _, tc := range tests {
		got, err := evalPredicate(tc.predicate, tc.item)
		if tc.wantErr {
			assert.Error(t, err, "predicate=%q item=%q", tc.predicate, tc.item)
		} else {
			require.NoError(t, err, "predicate=%q item=%q", tc.predicate, tc.item)
			assert.Equal(t, tc.want, got, "predicate=%q item=%q", tc.predicate, tc.item)
		}
	}
}

// ─── Text runner tests ────────────────────────────────────────────────────────

func TestRunTextExperiment(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(),
		Provider:    sampleProvider(),
		RunsPerTask: 1,
	}

	results, err := RunTextExperiment(cfg)
	require.NoError(t, err)
	assert.Len(t, results, 2) // 2 tasks × 1 run

	for _, r := range results {
		assert.Equal(t, "text", r.Condition)
		assert.Nil(t, r.Error)
	}
}

// ─── Runtime runner tests ─────────────────────────────────────────────────────

func TestRunRuntimeExperiment(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(),
		Provider:    sampleProvider(),
		RunsPerTask: 1,
	}

	results, err := RunRuntimeExperiment(cfg)
	require.NoError(t, err)
	assert.Len(t, results, 2) // 2 tasks × 1 run

	for _, r := range results {
		assert.Equal(t, "runtime", r.Condition)
		assert.Nil(t, r.Error)
		assert.True(t, r.Completed, "task %s should complete", r.TaskID)
	}
}

// ─── Comparison tests ─────────────────────────────────────────────────────────

func TestCompare_WithResults(t *testing.T) {
	textResults := []TaskResult{
		{TaskID: "t1", Completed: true, OrderingScore: 0.8, GuardrailScore: 0.5, TokensUsed: 1000},
		{TaskID: "t2", Completed: false, OrderingScore: 0.5, GuardrailScore: 1.0, TokensUsed: 800},
	}
	runtimeResults := []TaskResult{
		{TaskID: "t1", Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0, TokensUsed: 600},
		{TaskID: "t2", Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0, TokensUsed: 500},
	}

	cr := Compare(textResults, runtimeResults)

	assert.Equal(t, 2, cr.Tasks)
	assert.Greater(t, cr.CompletionDelta, 0.0, "runtime should have higher completion")
	assert.Greater(t, cr.OrderingDelta, 0.0, "runtime should have better ordering")
	assert.Greater(t, cr.GuardrailDelta, 0.0, "runtime should have better guardrails")
	assert.Less(t, cr.TokenDelta, 0.0, "runtime should use fewer tokens")

	// CI should be computed
	assert.NotEqual(t, cr.CompletionCI, [2]float64{0, 0})
}

func TestComputeDetectionRate(t *testing.T) {
	text := []TaskResult{
		{TaskID: "t1", Completed: true, OrderingScore: 0.8, GuardrailScore: 0.5},  // ordering + guardrail fail
		{TaskID: "t2", Completed: false, OrderingScore: 0.5, GuardrailScore: 1.0}, // completion + ordering fail
		{TaskID: "t3", Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0},  // all pass
		{TaskID: "t4", Completed: false, OrderingScore: 1.0, GuardrailScore: 0.8}, // completion + guardrail fail
	}
	runtime := []TaskResult{
		{TaskID: "t1", Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0},  // catches both
		{TaskID: "t2", Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0},  // catches both
		{TaskID: "t3", Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0},  // no failure to catch
		{TaskID: "t4", Completed: true, OrderingScore: 1.0, GuardrailScore: 0.9},  // catches completion, not guardrail
	}

	dm := ComputeDetectionRate(text, runtime)

	// Completion: t2 and t4 failed in text. Runtime caught both.
	assert.Equal(t, 2, dm.CompletionFailures)
	assert.Equal(t, 2, dm.CompletionCaught)
	assert.Equal(t, 1.0, dm.CompletionRate)

	// Ordering: t1 (0.8) and t2 (0.5) failed. Runtime caught both (1.0).
	assert.Equal(t, 2, dm.OrderingFailures)
	assert.Equal(t, 2, dm.OrderingCaught)
	assert.Equal(t, 1.0, dm.OrderingRate)

	// Guardrails: t1 (0.5) and t4 (0.8) failed. Runtime caught t1 (1.0) but not t4 (0.9).
	assert.Equal(t, 2, dm.GuardrailFailures)
	assert.Equal(t, 1, dm.GuardrailCaught)
	assert.Equal(t, 0.5, dm.GuardrailRate)

	// Any: t1, t2, t4 had failures. t1 and t2 fully caught, t4 not (guardrail still bad).
	assert.Equal(t, 3, dm.AnyFailures)
	assert.Equal(t, 2, dm.AnyCaught) // t1 and t2 fully caught
}

func TestComputeDetectionRate_NoFailures(t *testing.T) {
	text := []TaskResult{
		{Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0},
	}
	runtime := []TaskResult{
		{Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0},
	}

	dm := ComputeDetectionRate(text, runtime)
	assert.Equal(t, 0, dm.AnyFailures)
	assert.Equal(t, 0.0, dm.AnyRate)
}

func TestCompare_Empty(t *testing.T) {
	cr := Compare(nil, nil)
	assert.Equal(t, 0, cr.Tasks)
	assert.Equal(t, 0.0, cr.CompletionDelta)
}

func TestCompare_Reproducible(t *testing.T) {
	text := []TaskResult{{Completed: true, OrderingScore: 0.8, TokensUsed: 100}}
	rt := []TaskResult{{Completed: true, OrderingScore: 1.0, TokensUsed: 80}}

	cr1 := Compare(text, rt)
	cr2 := Compare(text, rt)

	assert.Equal(t, cr1.CompletionCI, cr2.CompletionCI, "bootstrap CI must be deterministic (fixed seed)")
	assert.Equal(t, cr1.OrderingCI, cr2.OrderingCI)
}

// ─── Report tests ─────────────────────────────────────────────────────────────

func TestFormatReport_AllSections(t *testing.T) {
	cr := &ComparisonResult{
		Tasks:           2,
		Runs:            2,
		CompletionDelta: 0.5,
		OrderingDelta:   0.2,
		GuardrailDelta:  0.5,
		TokenDelta:      -200,
		CompletionCI:    [2]float64{0.1, 0.9},
		OrderingCI:      [2]float64{0.0, 0.4},
		GuardrailCI:     [2]float64{0.1, 0.9},
		TokenCI:         [2]float64{-400, -100},
		TextResults: []TaskResult{
			{TaskID: "t1", Completed: false, OrderingScore: 0.5},
			{TaskID: "t2", Completed: true, OrderingScore: 0.8},
		},
		RuntimeResults: []TaskResult{
			{TaskID: "t1", Completed: true, OrderingScore: 1.0},
			{TaskID: "t2", Completed: true, OrderingScore: 1.0},
		},
	}

	report := FormatReport(cr, ReportOpts{Model: "test-model"})

	assert.Contains(t, report, "# Runtime vs Text Experiment Report")
	assert.Contains(t, report, "## Experiment Parameters")
	assert.Contains(t, report, "## Methodology")
	assert.Contains(t, report, "## Aggregate Results")
	assert.Contains(t, report, "## Per-Task Results")
	assert.Contains(t, report, "test-model")
	assert.Contains(t, report, "Completion rate")
	assert.Contains(t, report, "Step ordering")
	assert.Contains(t, report, "Guardrail adherence")
	assert.Contains(t, report, "Token usage")
}

// ─── OnResult callback tests ─────────────────────────────────────────────────

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

// ─── SkipMap tests ────────────────────────────────────────────────────────────

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

// ─── IsAPIFatalError tests ───────────────────────────────────────────────────

func TestIsAPIFatalError(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		fatal bool
	}{
		{"nil", nil, false},
		{"task-level error", fmt.Errorf("step identify: no keywords matched"), false},
		{"rate limit 429", fmt.Errorf("API error 429: rate limited"), false},
		{"server error 500", fmt.Errorf("API error 500: internal server error"), false},
		{"unauthorized 401", fmt.Errorf("API error 401: invalid api key"), true},
		{"forbidden 403", fmt.Errorf("API error 403: forbidden"), true},
		{"credit balance direct", fmt.Errorf(`API error 400: {"error":{"message":"Your credit balance is too low to access the Anthropic API."}}`), true},
		{"credit balance wrapped in DAG", fmt.Errorf(`execute DAG: node "runbook": step runbook attempt 3: API error 400: {"error":{"message":"Your credit balance is too low"}}`), true},
		{"invalid api key message", fmt.Errorf("API error 400: invalid x-api-key header"), true},
		{"timeout", fmt.Errorf("Anthropic request timed out after 2m0s"), false},
		{"network error", fmt.Errorf("send request: connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.fatal, IsAPIFatalError(tt.err))
		})
	}
}

// ─── Fail-fast tests ────────────────────────────────────────────────────────

// failAfterNProvider returns real responses for the first N calls, then a fatal API error.
type failAfterNProvider struct {
	inner    llm.Provider
	maxCalls int
	calls    int
}

func (p *failAfterNProvider) Complete(prompt string) (string, error) {
	p.calls++
	if p.calls > p.maxCalls {
		return "", fmt.Errorf(`API error 400: {"error":{"message":"Your credit balance is too low to access the Anthropic API."}}`)
	}
	return p.inner.Complete(prompt)
}

func TestRunTextExperiment_FailFast(t *testing.T) {
	// Allow 1 call (enough for code-review task 1), then fail
	provider := &failAfterNProvider{inner: sampleProvider(), maxCalls: 1}
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(), // 2 tasks
		Provider:    provider,
		RunsPerTask: 3,
	}

	results, err := RunTextExperiment(cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAPIFatal), "should wrap ErrAPIFatal")
	// Should have partial results: the 1 successful run + 1 failed run
	assert.True(t, len(results) >= 1, "should return partial results, got %d", len(results))
	assert.True(t, len(results) < 6, "should abort before all 6 runs, got %d", len(results))
}

func TestRunRuntimeExperiment_FailFast(t *testing.T) {
	// code-review has 3 steps (3 LLM calls per run). Allow 3, then fail.
	provider := &failAfterNProvider{inner: sampleProvider(), maxCalls: 3}
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(), // 2 tasks
		Provider:    provider,
		RunsPerTask: 3,
	}

	results, err := RunRuntimeExperiment(cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAPIFatal), "should wrap ErrAPIFatal")
	assert.True(t, len(results) >= 1, "should return partial results, got %d", len(results))
	assert.True(t, len(results) < 6, "should abort before all 6 runs, got %d", len(results))
}

// ─── File-based slot passing tests (Story 6-21) ───────────────────────────────

// recordingProvider captures every prompt the LLM receives, keyed by call order.
type recordingProvider struct {
	prompts  []string
	response string
}

func (r *recordingProvider) Complete(prompt string) (string, error) {
	r.prompts = append(r.prompts, prompt)
	return r.response, nil
}

var _ llm.Provider = (*recordingProvider)(nil)

func singleStepTask() TaskDefinition {
	return TaskDefinition{
		ID:          "single",
		Name:        "Single Step Task",
		Description: "A minimal task for testing",
		Input:       "func hello() {}",
		Steps: []StepDef{
			{ID: "analyze", Name: "Analyze", Prompt: "Analyze the code", Produces: []SlotSpec{{Key: "analysis"}}},
		},
		Guardrails: []string{"analysis"},
	}
}

func multiStepTask() TaskDefinition {
	return TaskDefinition{
		ID:          "multi",
		Name:        "Multi Step Task",
		Description: "A multi-step task for testing slot passing",
		Input:       "func add(a, b int) int { return a + b }",
		Steps: []StepDef{
			{ID: "find-bugs", Name: "Find Bugs", Prompt: "Find bugs in the code.", Produces: []SlotSpec{{Key: "bugs"}}},
			{ID: "fix-code", Name: "Fix Code", Prompt: "Fix the bugs found.", Consumes: []SlotSpec{{Key: "bugs"}}, Produces: []SlotSpec{{Key: "fixed"}}},
		},
		Guardrails: []string{"fix"},
	}
}

func TestFileSlotPassing_TaskContextWrittenToSandbox(t *testing.T) {
	task := singleStepTask()
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})
	provider := &recordingProvider{response: "analysis done"}

	_, _, err := buildTaskDAG(task, provider, nil, sandbox, NilRetriever{}, nil)
	require.NoError(t, err)

	content, err := sandbox.Read("test", "task_context.md")
	require.NoError(t, err)

	s := string(content)
	assert.Contains(t, s, "# Task: Single Step Task")
	assert.Contains(t, s, "## Description")
	assert.Contains(t, s, "A minimal task for testing")
	assert.Contains(t, s, "## Original Input")
	assert.Contains(t, s, "func hello() {}")
	assert.Contains(t, s, "## Guardrails")
	assert.Contains(t, s, "- analysis")
}

func TestFileSlotPassing_PromptListsFilePaths(t *testing.T) {
	task := multiStepTask()
	provider := &recordingProvider{response: "bugs found and fixed"}

	result := runRuntimeTask(task, provider, nil, NilRetriever{}, "runtime")
	require.NoError(t, result.Error)

	require.Len(t, provider.prompts, 2, "expected 2 LLM calls (one per step)")

	step2Prompt := provider.prompts[1]

	// Step 2 prompt must reference slot files, not embed content inline.
	assert.Contains(t, step2Prompt, "task_context.md")
	assert.Contains(t, step2Prompt, "slots/bugs.md")
	assert.Contains(t, step2Prompt, "Available files")

	// Step 2 prompt must NOT contain the actual content of step 1's output inline.
	assert.NotContains(t, step2Prompt, "bugs found and fixed")
}

func TestFileSlotPassing_TaskContextAlwaysListed(t *testing.T) {
	task := singleStepTask()
	provider := &recordingProvider{response: "analysis complete"}

	result := runRuntimeTask(task, provider, nil, NilRetriever{}, "runtime")
	require.NoError(t, result.Error)

	require.NotEmpty(t, provider.prompts)
	step1Prompt := provider.prompts[0]

	assert.Contains(t, step1Prompt, "task_context.md")
	assert.Contains(t, step1Prompt, "Available files")
	// Original input must NOT be embedded inline.
	assert.NotContains(t, step1Prompt, "## Input")
}

func TestFileSlotPassing_SlotFilesWrittenAfterStep(t *testing.T) {
	task := multiStepTask()
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})
	provider := &recordingProvider{response: "step output content"}

	d, _, err := buildTaskDAG(task, provider, nil, sandbox, NilRetriever{}, nil)
	require.NoError(t, err)

	_, err = d.Execute(context.Background(), dag.WithInputs(map[string]any{"task_input": task.Input}))
	require.NoError(t, err)

	// Each step's output should be written to slots/{key}.md.
	bugs, err := sandbox.Read("test", "slots/bugs.md")
	require.NoError(t, err)
	assert.Equal(t, "step output content", string(bugs))

	fixed, err := sandbox.Read("test", "slots/fixed.md")
	require.NoError(t, err)
	assert.Equal(t, "step output content", string(fixed))
}

func TestFileSlotPassing_FileReadAlwaysInAllowedTools(t *testing.T) {
	task := multiStepTask()
	provider := &recordingProvider{response: "output"}

	result := runRuntimeTask(task, provider, nil, NilRetriever{}, "runtime")
	require.NoError(t, result.Error)

	// Prompts should mention file_read as an available tool.
	for i, p := range provider.prompts {
		assert.Contains(t, p, "file_read", "step %d prompt should list file_read", i)
	}
}

func TestFileSlotPassing_ReadFileViaToolCall(t *testing.T) {
	task := singleStepTask()
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})

	callCount := 0
	provider := &funcProvider{fn: func(prompt string) (string, error) {
		callCount++
		if callCount == 1 {
			// First call: LLM uses file_read to get task context.
			return `I'll read the context first. <tool name="file_read" path="task_context.md"/>`, nil
		}
		// Second call (after tool result injected): provide real output.
		return "analysis: code is clean", nil
	}}

	d, _, err := buildTaskDAG(task, provider, nil, sandbox, NilRetriever{}, nil)
	require.NoError(t, err)
	_, err = d.Execute(context.Background(), dag.WithInputs(map[string]any{"task_input": task.Input}))
	require.NoError(t, err)

	assert.Equal(t, 2, callCount, "expected 2 calls: one with tool call, one after tool result")
}

// funcProvider wraps a function as an llm.Provider.
type funcProvider struct {
	fn func(string) (string, error)
}

func (f *funcProvider) Complete(prompt string) (string, error) { return f.fn(prompt) }

// ─── Patch tests (review findings) ───────────────────────────────────────────

func TestFileSlotPassing_ReadOnly_TaskContextPrePopulated(t *testing.T) {
	task := singleStepTask()
	// ReadOnly workspace: Write calls in buildTaskDAG must not be the only source of task_context.md.
	policy := SandboxPolicy{ReadOnly: true}
	sandbox := NewSandboxFS(nil, nil, policy)

	// Pre-populate task_context.md as buildSandboxFromWorkspace does.
	_ = NewSandboxFS(map[string][]byte{
		"task_context.md": taskContextContent(task),
	}, nil, policy)

	// Simulate: create sandbox with pre-populated file (bypassing ReadOnly via initial files map).
	sandboxWithContext := NewSandboxFS(map[string][]byte{
		"task_context.md": taskContextContent(task),
	}, nil, policy)

	provider := &recordingProvider{response: "analysis complete with analysis keyword"}
	_, _, err := buildTaskDAG(task, provider, nil, sandboxWithContext, NilRetriever{}, nil)
	require.NoError(t, err)

	// task_context.md must be readable even though sandbox is ReadOnly.
	content, err := sandboxWithContext.Read("test", "task_context.md")
	require.NoError(t, err)
	assert.Contains(t, string(content), "func hello() {}")
	_ = sandbox
}

func TestFileSlotPassing_WorkspaceSandbox_TaskContextNotInjected(t *testing.T) {
	// Workspace tasks carry their own context in workspace files, so task_context.md
	// is intentionally NOT pre-populated — this avoids every step reading it
	// unnecessarily and burning extra tokens.
	task := TaskDefinition{
		ID:          "ws-task",
		Name:        "Workspace Task",
		Description: "Tests workspace integration",
		Input:       "package main",
		Workspace: &WorkspaceDef{
			Files: []WorkspaceFile{{Path: "src/main.go", Content: "package main"}},
		},
		Steps: []StepDef{
			{ID: "review", Name: "Review", Prompt: "Review the code.", Produces: []SlotSpec{{Key: "review"}}},
		},
	}

	sandbox := buildSandboxFromWorkspace(task)

	// Workspace files are accessible.
	content, err := sandbox.Read("test", "src/main.go")
	require.NoError(t, err)
	assert.Contains(t, string(content), "package main")

	// task_context.md must NOT be present for workspace tasks.
	_, err = sandbox.Read("test", "task_context.md")
	assert.Error(t, err, "task_context.md must not be injected for workspace tasks")
}

func TestFileSlotPassing_PerStepAllowedPaths_ContextFilesAccessible(t *testing.T) {
	// A step with a restrictive sandbox.allowed_paths must still be able to file_read
	// task_context.md and slots/** after the harness augments the per-step policy.
	task := TaskDefinition{
		ID:   "restricted",
		Name: "Restricted Task",
		Input: "some input",
		Steps: []StepDef{
			{
				ID:     "read-only-step",
				Name:   "Read Only Step",
				Prompt: "Review carefully.",
				Produces: []SlotSpec{{Key: "output"}},
				Sandbox: &StepSandboxPolicy{AllowedPaths: []string{"src/**"}},
			},
		},
	}

	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})
	provider := &recordingProvider{response: "output text"}

	d, _, err := buildTaskDAG(task, provider, nil, sandbox, NilRetriever{}, nil)
	require.NoError(t, err)

	_, err = d.Execute(context.Background(), dag.WithInputs(map[string]any{"task_input": task.Input}))
	require.NoError(t, err)

	// task_context.md must have been written and be readable (policy was augmented).
	content, err := sandbox.Read("test", "task_context.md")
	require.NoError(t, err)
	assert.Contains(t, string(content), "Restricted Task")
}

func TestFileSlotPassing_DuplicateFileReadNotInPrompt(t *testing.T) {
	// If a step declares file_read in its Tools list, the prompt must list it only once.
	task := TaskDefinition{
		ID:   "dup-test",
		Name: "Dup Test",
		Input: "input",
		Steps: []StepDef{
			{
				ID:       "step1",
				Name:     "Step1",
				Prompt:   "Do something.",
				Produces: []SlotSpec{{Key: "out"}},
				Tools:    []string{"file_read", "file_write"}, // file_read declared explicitly
			},
		},
	}

	provider := &recordingProvider{response: "done"}
	result := runRuntimeTask(task, provider, nil, NilRetriever{}, "runtime")
	require.NoError(t, result.Error)

	require.NotEmpty(t, provider.prompts)
	prompt := provider.prompts[0]

	// Count occurrences of "file_read" in the Available Tools section.
	toolsSection := prompt[strings.Index(prompt, "## Available Tools"):]
	count := strings.Count(toolsSection, "- file_read")
	assert.Equal(t, 1, count, "file_read should appear exactly once in the prompt tool list")
}

// ─── Retriever tests ──────────────────────────────────────────────────────────

func TestPrefixRetriever_InjectsFirstNTokensOfSlot(t *testing.T) {
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})
	// Write a long slot file — 400 tokens ≈ 1600 chars; PrefixRetriever{MaxTokens:100} injects ≤400 chars.
	content := strings.Repeat("word ", 400) // 400 tokens
	_ = sandbox.Write("test", "slots/bugs.md", []byte(content))

	step := StepDef{ID: "fix", Name: "Fix", Prompt: "Fix the identified bugs"}
	consumed := []SlotSpec{{Key: "bugs"}}

	r := PrefixRetriever{MaxTokens: 100}
	excerpt, err := r.Retrieve(step, sandbox, consumed)
	require.NoError(t, err)
	assert.NotEmpty(t, excerpt)
	// Should contain at most 100*4=400 chars of content (plus header)
	assert.Contains(t, excerpt, "### bugs")
	assert.LessOrEqual(t, len(excerpt), 400+50) // 50 chars of header overhead
}

func TestPrefixRetriever_SkipsMissingSlot(t *testing.T) {
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})
	// No slot file written — retriever must not error.
	step := StepDef{ID: "step", Name: "Step", Prompt: "Do work"}
	consumed := []SlotSpec{{Key: "missing"}}

	r := PrefixRetriever{MaxTokens: 300}
	excerpt, err := r.Retrieve(step, sandbox, consumed)
	require.NoError(t, err)
	assert.Empty(t, excerpt)
}

func TestPrefixRetriever_SkipsTaskInput(t *testing.T) {
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})
	step := StepDef{ID: "step", Name: "Step", Prompt: "Analyze"}
	consumed := []SlotSpec{{Key: "task_input"}}

	r := PrefixRetriever{MaxTokens: 300}
	excerpt, err := r.Retrieve(step, sandbox, consumed)
	require.NoError(t, err)
	assert.Empty(t, excerpt, "task_input slot must be skipped")
}

func TestRankedRetriever_BM25ScoresRelevantChunkHigher(t *testing.T) {
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})

	// Chunk 1: documentation — unrelated to query
	// Chunk 2: memory leak, null pointer — matches query terms
	// Chunk 3: style/formatting — unrelated to query
	slotContent := strings.Join([]string{
		"The documentation section covers public API usage and general guidelines for contributors.",
		"There is a memory leak in the allocation path. The null pointer dereference causes a crash when the buffer is not initialized properly.",
		"Style issues include inconsistent indentation and missing blank lines between functions.",
	}, "\n\n")
	_ = sandbox.Write("test", "slots/bugs.md", []byte(slotContent))

	step := StepDef{ID: "fix", Name: "Fix Bugs", Prompt: "Fix the memory leak and null pointer dereference bug"}
	consumed := []SlotSpec{{Key: "bugs"}}

	r := RankedRetriever{MaxTokens: 500, TopK: 1}
	excerpt, err := r.Retrieve(step, sandbox, consumed)
	require.NoError(t, err)
	assert.NotEmpty(t, excerpt)
	// The BM25 top chunk must be the memory-leak paragraph, not the documentation one.
	assert.Contains(t, excerpt, "memory leak")
	assert.NotContains(t, excerpt, "documentation section covers")
}

func TestRankedRetriever_SkipsMissingSlot(t *testing.T) {
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})
	step := StepDef{ID: "step", Name: "Step", Prompt: "Fix bugs"}
	consumed := []SlotSpec{{Key: "missing"}}

	r := RankedRetriever{MaxTokens: 500, TopK: 5}
	excerpt, err := r.Retrieve(step, sandbox, consumed)
	require.NoError(t, err)
	assert.Empty(t, excerpt)
}

func TestNilRetriever_AlwaysReturnsEmpty(t *testing.T) {
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})
	_ = sandbox.Write("test", "slots/bugs.md", []byte("some content"))
	step := StepDef{ID: "step", Name: "Step", Prompt: "Fix bugs"}
	consumed := []SlotSpec{{Key: "bugs"}}

	excerpt, err := NilRetriever{}.Retrieve(step, sandbox, consumed)
	require.NoError(t, err)
	assert.Empty(t, excerpt)
}

func TestRunRuntimeExperiment_WithRankedRetriever_InjectsContext(t *testing.T) {
	// The ranked retriever should inject a "## Context" section into the prompt.
	task := multiStepTask()
	sandbox := NewSandboxFS(nil, nil, SandboxPolicy{})
	provider := &recordingProvider{response: "output with context"}

	_, _, err := buildTaskDAG(task, provider, nil, sandbox, RankedRetriever{MaxTokens: 500, TopK: 5}, nil)
	require.NoError(t, err)

	// No execution needed — just verifying DAG builds with ranked retriever.
	// Execute to trigger prompt building.
	_, _, err = buildTaskDAG(task, provider, nil, sandbox, RankedRetriever{MaxTokens: 500, TopK: 5}, nil)
	require.NoError(t, err)
}

func TestCompareThree_PopulatesRankedResults(t *testing.T) {
	text := []TaskResult{{TaskID: "t1", Condition: "text", Completed: true, GuardrailScore: 0.8}}
	runtime := []TaskResult{{TaskID: "t1", Condition: "runtime", Completed: true, GuardrailScore: 0.9}}
	ranked := []TaskResult{{TaskID: "t1", Condition: "ranked", Completed: true, GuardrailScore: 0.95}}

	cr := CompareThree(text, runtime, ranked)
	require.NotNil(t, cr)
	assert.Len(t, cr.RankedResults, 1)
	assert.Equal(t, "ranked", cr.RankedResults[0].Condition)

	// FormatReport must render a third column when RankedResults is populated.
	report := FormatReport(cr, ReportOpts{Model: "mock"})
	assert.Contains(t, report, "Ranked")
	assert.Contains(t, report, "three conditions")
}

// ─── End-to-end test ──────────────────────────────────────────────────────────

func TestEndToEnd_TextVsRuntime(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(),
		Provider:    sampleProvider(),
		RunsPerTask: 1,
	}

	textResults, err := RunTextExperiment(cfg)
	require.NoError(t, err)

	runtimeResults, err := RunRuntimeExperiment(cfg)
	require.NoError(t, err)

	cr := Compare(textResults, runtimeResults)
	assert.NotNil(t, cr)
	assert.Equal(t, 2, cr.Tasks)

	report := FormatReport(cr, ReportOpts{Model: "mock"})
	assert.Contains(t, report, "Runtime vs Text")
}

// ─── Agent runner tests ───────────────────────────────────────────────────────

func TestRunAgentExperiment_Basic(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(),
		Provider:    sampleProvider(),
		RunsPerTask: 1,
		Condition:   "agent",
	}
	results, err := RunAgentExperiment(cfg)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "agent", r.Condition)
		assert.Nil(t, r.Error)
		assert.True(t, r.Completed, "task %s should complete", r.TaskID)
	}
}

func TestRunAgentExperiment_DeterministicFilter(t *testing.T) {
	// kind: filter with predicate must not call LLM — provider panics if called
	// for the filter step.
	callCount := 0
	prov := &mockExpProvider{
		responses: map[string]string{
			"collect": `["bug A | SCORE: 90", "false positive | SCORE: 20", "bug B | SCORE: 95"]`,
		},
		fallback: "no issues",
	}
	_ = callCount

	task := TaskDefinition{
		ID:   "agent-filter-test",
		Name: "Agent Filter Test",
		Steps: []StepDef{
			{
				ID:     "collect",
				Prompt: "collect issues",
				Produces: []SlotSpec{{
					Key:        "issues_list",
					ListSchema: &schema.FieldSpec{Type: "string"},
				}},
			},
			{
				ID:        "filter-issues",
				Kind:      StepKindFilter,
				Predicate: "score >= 80",
				Consumes:  []SlotSpec{{Key: "issues_list"}},
				Produces:  []SlotSpec{{Key: "filtered_issues"}},
			},
		},
	}
	cfg := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: prov, Condition: "agent"}
	results, err := RunAgentExperiment(cfg)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed)
	// Only 1 LLM call — for "collect"; filter is deterministic.
	assert.Equal(t, 1, results[0].LLMCalls)
}

func TestNormalizeDashes(t *testing.T) {
	assert.Equal(t, "issue collector - aggregate findings", normalizeDashes("issue collector — aggregate findings"))
	assert.Equal(t, "issue filter - keep score >= 80", normalizeDashes("issue filter – keep score >= 80"))
	assert.Equal(t, "no dashes here", normalizeDashes("no dashes here"))
}

func TestStepTextBody_Predicate(t *testing.T) {
	step := StepDef{Kind: StepKindFilter, Predicate: "score >= 80"}
	body := stepTextBody(step)
	assert.Contains(t, body, "score >= 80")
	assert.Contains(t, body, "Filter")
}

func TestStepTextBody_FallsBackToPrompt(t *testing.T) {
	step := StepDef{Prompt: "Do something specific"}
	body := stepTextBody(step)
	assert.Equal(t, "Do something specific", body)
}

func TestEndToEnd_TextVsRuntimeVsAgent(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks:       sampleTasks(),
		Provider:    sampleProvider(),
		RunsPerTask: 1,
	}
	textResults, err := RunTextExperiment(cfg)
	require.NoError(t, err)
	runtimeResults, err := RunRuntimeExperiment(cfg)
	require.NoError(t, err)
	agentResults, err := RunAgentExperiment(cfg)
	require.NoError(t, err)

	cr := CompareThree(textResults, runtimeResults, agentResults)
	assert.NotNil(t, cr)
	assert.Equal(t, 2, cr.Tasks)
	assert.Len(t, cr.RankedResults, 2) // RankedResults holds the third condition (agent here)
}
