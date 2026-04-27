package experiment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ─── T3.1: StepDef YAML round-trip with guardrails field ──────────────────────

func TestStepDef_Guardrails_YAMLRoundTrip(t *testing.T) {
	raw := `
id: review
name: Code Review
prompt: Review the code
guardrails:
  - Bearer
  - "!DROP TABLE"
consumes:
  - key: code
produces:
  - key: verdict
`
	var step StepDef
	err := yaml.Unmarshal([]byte(raw), &step)
	require.NoError(t, err)
	assert.Equal(t, []string{"Bearer", "!DROP TABLE"}, step.Guardrails)
	assert.Equal(t, "review", step.ID)

	// Round-trip: marshal back and verify guardrails survive
	out, err := yaml.Marshal(step)
	require.NoError(t, err)
	var step2 StepDef
	require.NoError(t, yaml.Unmarshal(out, &step2))
	assert.Equal(t, step.Guardrails, step2.Guardrails)
}

func TestStepDef_NoGuardrails_OmitEmpty(t *testing.T) {
	// Verify that omitempty suppresses the field when marshaling
	step := StepDef{ID: "x", Name: "X", Prompt: "do stuff"}
	out, err := yaml.Marshal(step)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "guardrails", "omitempty should suppress empty slice")

	// Verify that unmarshaling YAML without the guardrails key leaves the field nil
	raw := "id: x\nname: X\nprompt: do stuff\n"
	var step2 StepDef
	require.NoError(t, yaml.Unmarshal([]byte(raw), &step2))
	assert.Nil(t, step2.Guardrails, "absent guardrails key should unmarshal to nil, not empty slice")

	// Verify that disable_task_guardrails also omits when false
	assert.NotContains(t, string(out), "disable_task_guardrails")
}

// ─── T3.2: step-level guardrails override task-level ──────────────────────────

// mockGuardrailProvider tracks how many times Complete is called and returns a
// configurable sequence of responses.
type mockGuardrailProvider struct {
	responses []string
	callIdx   int
}

func (m *mockGuardrailProvider) Complete(prompt string) (string, error) {
	if m.callIdx < len(m.responses) {
		r := m.responses[m.callIdx]
		m.callIdx++
		return r, nil
	}
	return "fallback response", nil
}

// effectiveGuardrails is the helper we expect to extract from runtime_runner.go.
// We test it in isolation here.

func TestEffectiveGuardrails_StepOverridesTask(t *testing.T) {
	task := TaskDefinition{
		Guardrails: []string{"task-guardrail"},
	}
	step := StepDef{
		Guardrails: []string{"step-guardrail"},
	}
	got := effectiveGuardrails(step, task)
	assert.Equal(t, []string{"step-guardrail"}, got)
}

func TestEffectiveGuardrails_FallbackToTask(t *testing.T) {
	task := TaskDefinition{
		Guardrails: []string{"task-guardrail"},
	}
	step := StepDef{
		// no step-level guardrails
	}
	got := effectiveGuardrails(step, task)
	assert.Equal(t, []string{"task-guardrail"}, got)
}

func TestEffectiveGuardrails_NoneOnEither(t *testing.T) {
	task := TaskDefinition{}
	step := StepDef{}
	got := effectiveGuardrails(step, task)
	assert.Empty(t, got)
}

// ─── T3.3: runtime uses step-level guardrails when both exist ─────────────────

func TestRuntime_StepGuardrails_UsedNotTaskGuardrails(t *testing.T) {
	// Step guardrail: output must contain "Bearer"
	// Task guardrail: output must contain "task-token" (should NOT be checked for this step)
	// Provider first returns a response that satisfies step guardrail but NOT task guardrail.
	// If runtime uses step guardrails, it passes. If runtime uses task guardrails, it fails and retries.

	provider := &mockGuardrailProvider{
		responses: []string{
			// First attempt: contains "Bearer" (satisfies step guardrail), but NOT "task-token"
			"This is the output with Bearer token present.",
		},
	}

	task := TaskDefinition{
		ID:   "test-task",
		Name: "Test",
		Input: "some input",
		Guardrails: []string{"task-token"}, // task-level: must contain "task-token"
		Steps: []StepDef{
			{
				ID:         "step1",
				Name:       "Step 1",
				Prompt:     "Do something",
				Guardrails: []string{"Bearer"}, // step-level: must contain "Bearer"
				Produces:   []SlotSpec{{Key: "output1"}},
			},
		},
	}

	config := ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	}

	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// The step should have completed (Bearer is present → step guardrail satisfied)
	// and the runtime should have called provider only ONCE (no retry needed)
	assert.Equal(t, 1, provider.callIdx, "step-level guardrail should pass on first attempt — no retry")
	assert.True(t, results[0].Completed, "task should complete when step guardrail is satisfied")
	// Verify the output contains "Bearer" (step guardrail enforced) and NOT "task-token" (task guardrail NOT enforced)
	require.Len(t, results[0].StepResults, 1)
	assert.Contains(t, results[0].StepResults[0].Output, "Bearer", "step output should contain step guardrail token")
	assert.NotContains(t, results[0].StepResults[0].Output, "task-token", "task guardrail should not have been enforced")
}

// ─── T3.4: retry limited to the violating step ────────────────────────────────

func TestRuntime_StepGuardrails_RetryOnViolation(t *testing.T) {
	// Step guardrail: output must contain "APPROVED"
	// First attempt returns output without "APPROVED" → violates guardrail → retry
	// Second attempt returns output with "APPROVED" → passes

	callCount := 0
	provider := &callCountingMock{
		fn: func(prompt string) (string, error) {
			callCount++
			// "[Previous attempt" only appears in retry prompts (attempt > 0)
			if strings.Contains(prompt, "[Previous attempt") {
				return "The verdict is APPROVED after careful review.", nil
			}
			return "The verdict is unclear.", nil
		},
	}

	task := TaskDefinition{
		ID:    "guardrail-step-test",
		Name:  "Code Review Task",
		Input: "some code",
		Steps: []StepDef{
			{
				ID:         "review",
				Name:       "Review",
				Prompt:     "Review the code",
				Guardrails: []string{"APPROVED"},
				Produces:   []SlotSpec{{Key: "verdict"}},
			},
		},
	}

	config := ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	}

	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, 2, callCount, "should call provider twice: first fail, second pass")
	assert.True(t, results[0].Completed)
}

// callCountingMock is a minimal provider that delegates to a function.
type callCountingMock struct {
	fn func(prompt string) (string, error)
}

func (m *callCountingMock) Complete(prompt string) (string, error) {
	return m.fn(prompt)
}

// ─── P5: GuardrailScore per-step ──────────────────────────────────────────────

func TestGuardrailScore_PerStep(t *testing.T) {
	// Step 1 has step-level guardrail "VERDICT" — output includes it → respected
	// Step 2 has no step-level guardrail → falls back to task-level "doc" → output includes it → respected
	// Expected: GuardrailScore = 1.0 (2/2 guardrails respected)
	provider := &mockGuardrailProvider{
		responses: []string{
			"Here is the VERDICT: approved.",   // step1 output
			"Full documentation provided here.", // step2 output
		},
	}
	task := TaskDefinition{
		ID:         "per-step-score",
		Name:       "Per-step Score Test",
		Input:      "input",
		Guardrails: []string{"doc"},
		Steps: []StepDef{
			{ID: "s1", Name: "S1", Prompt: "step1", Guardrails: []string{"VERDICT"}, Produces: []SlotSpec{{Key: "out1"}}},
			{ID: "s2", Name: "S2", Prompt: "step2", Consumes: []SlotSpec{{Key: "out1"}}, Produces: []SlotSpec{{Key: "out2"}}},
		},
	}
	config := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1}
	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 1.0, results[0].GuardrailScore, "both step guardrails respected → score 1.0")
}

func TestGuardrailScore_PerStep_PartialFailure(t *testing.T) {
	// Step 1 has step-level guardrail "VERDICT" — output does NOT include it → violated
	// Step 2 has no guardrail → falls back to task-level "" (none) → skipped
	// Expected: GuardrailScore < 1.0 (0/1 respected)
	provider := &mockGuardrailProvider{
		responses: []string{
			"Analysis complete — no clear result.",
			"Documentation done.",
		},
	}
	task := TaskDefinition{
		ID:    "per-step-fail",
		Name:  "Partial Fail",
		Input: "input",
		Steps: []StepDef{
			{ID: "s1", Name: "S1", Prompt: "step1", Guardrails: []string{"VERDICT"}, Produces: []SlotSpec{{Key: "out1"}}},
			{ID: "s2", Name: "S2", Prompt: "step2", Consumes: []SlotSpec{{Key: "out1"}}, Produces: []SlotSpec{{Key: "out2"}}},
		},
	}
	config := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1}
	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)
	// Step 1 guardrail violated (VERDICT not in output), step 2 has no guardrails → 0/1
	assert.Equal(t, 0.0, results[0].GuardrailScore, "step guardrail violated → score 0.0")
}

// ─── P6: DisableTaskGuardrails ────────────────────────────────────────────────

func TestEffectiveGuardrails_DisableTaskGuardrails(t *testing.T) {
	task := TaskDefinition{Guardrails: []string{"task-guardrail"}}
	step := StepDef{DisableTaskGuardrails: true}
	got := effectiveGuardrails(step, task)
	assert.Nil(t, got, "DisableTaskGuardrails=true should return nil even with task guardrails")
}

func TestEffectiveGuardrails_DisableTaskGuardrails_WithStepGuardrails(t *testing.T) {
	// DisableTaskGuardrails wins even when step also has guardrails declared
	task := TaskDefinition{Guardrails: []string{"task-guardrail"}}
	step := StepDef{DisableTaskGuardrails: true, Guardrails: []string{"step-guardrail"}}
	got := effectiveGuardrails(step, task)
	assert.Nil(t, got, "DisableTaskGuardrails=true should return nil regardless of step.Guardrails")
}

func TestRuntime_DisableTaskGuardrails_NoRetry(t *testing.T) {
	// Task has guardrail "REQUIRED" but step disables task guardrails → no retry even if output lacks it
	callCount := 0
	provider := &callCountingMock{
		fn: func(prompt string) (string, error) {
			callCount++
			return "output without the required token", nil
		},
	}
	task := TaskDefinition{
		ID:         "disable-test",
		Name:       "Disable Test",
		Input:      "input",
		Guardrails: []string{"REQUIRED"},
		Steps: []StepDef{
			{
				ID:                    "step1",
				Name:                  "Step 1",
				Prompt:                "do work",
				DisableTaskGuardrails: true,
				Produces:              []SlotSpec{{Key: "out1"}},
			},
		},
	}
	config := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1}
	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 1, callCount, "no retry — task guardrails disabled for this step")
	assert.True(t, results[0].Completed)
}

// ─── T3.3b: backward compat — no step guardrails uses task guardrails ──────────

func TestRuntime_BackwardCompat_TaskGuardrailsWhenNoStepGuardrails(t *testing.T) {
	// Task guardrail: output must contain "documentation"
	// Step has NO step-level guardrails → runtime should use task guardrails

	provider := &mockGuardrailProvider{
		responses: []string{
			// First attempt: does NOT contain "documentation"
			"This is some output without the required word.",
			// Second attempt: contains "documentation"
			"This documentation is complete and well-structured.",
		},
	}

	task := TaskDefinition{
		ID:         "compat-test",
		Name:       "Compat Test",
		Input:      "some input",
		Guardrails: []string{"documentation"},
		Steps: []StepDef{
			{
				ID:       "step1",
				Name:     "Step 1",
				Prompt:   "Generate docs",
				// No step-level guardrails → should use task.Guardrails
				Produces: []SlotSpec{{Key: "docs"}},
			},
		},
	}

	config := ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	}

	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, 2, provider.callIdx, "task guardrail should cause retry when not satisfied")
	assert.True(t, results[0].Completed)
	// Verify the final output contains "documentation" — the task guardrail that was enforced
	require.Len(t, results[0].StepResults, 1)
	assert.Contains(t, results[0].StepResults[0].Output, "documentation", "output should contain the enforced task guardrail token")
}
