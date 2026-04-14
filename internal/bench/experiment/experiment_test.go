package experiment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirandaguillaume/forgent/internal/llm"
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
				{ID: "identify", Name: "Identify Issues", Prompt: "Find bugs", Produces: []string{"issues"}, ExpectedKeywords: []string{"issue", "bug"}},
				{ID: "suggest", Name: "Suggest Fixes", Prompt: "Suggest improvements", Consumes: []string{"issues"}, Produces: []string{"suggestions"}, ExpectedKeywords: []string{"fix", "improve"}},
				{ID: "summarize", Name: "Summary", Prompt: "Summarize findings", Consumes: []string{"suggestions"}, Produces: []string{"summary"}, ExpectedKeywords: []string{"summary"}},
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
				{ID: "outline", Name: "Outline", Prompt: "Create doc outline", Produces: []string{"outline"}, ExpectedKeywords: []string{"outline"}},
				{ID: "draft", Name: "Draft", Prompt: "Write the draft", Consumes: []string{"outline"}, Produces: []string{"draft"}, ExpectedKeywords: []string{"draft"}},
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
	assert.Equal(t, []string{"output1"}, tasks[0].Steps[0].Produces)
	assert.Equal(t, []string{"output1"}, tasks[0].Steps[1].Consumes)
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
