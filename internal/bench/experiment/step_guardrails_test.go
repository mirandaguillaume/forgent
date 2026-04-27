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

// TestRuntime_StepGuardrails_3RetryExhaustion — story 10-0 Task 4.3.
// All three attempts violate the step-level guardrail; the runtime should
// stop retrying after the third attempt and return the task with the
// guardrail violation surfaced (Completed reflects whether the slot was
// produced; the violation is captured in GuardrailScore).
func TestRuntime_StepGuardrails_3RetryExhaustion(t *testing.T) {
	callCount := 0
	provider := &callCountingMock{
		fn: func(prompt string) (string, error) {
			callCount++
			// Always return output that violates the guardrail
			return "The verdict is unclear and undetermined.", nil
		},
	}

	task := TaskDefinition{
		ID:    "guardrail-3retry-exhaustion",
		Name:  "Three Retry Exhaustion",
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

	// Provider called 3 times = max attempts
	assert.Equal(t, 3, callCount, "should call provider 3 times (max attempts) before giving up")
	// GuardrailScore = 0 since "APPROVED" never present
	assert.Equal(t, 0.0, results[0].GuardrailScore, "all 3 attempts violate guardrail → score 0")
	// The step still produces output text on every attempt — Completed reflects
	// slot production, not guardrail satisfaction. Pin both so a future refactor
	// that errors out at attempt 3 instead of accepting the bad output would be
	// detected.
	assert.True(t, results[0].Completed, "step produces output on every attempt — Completed = true even when GuardrailScore is 0")
	assert.NoError(t, results[0].Error, "guardrail violation should not be surfaced as a hard task error in the current contract")
}

// TestRuntime_StepGuardrails_NegativePrefix — story 10-0 Task 4.4.
// "!token" semantics: output must NOT contain "token". Pre-existing logic
// in checkGuardrail; this test pins the behaviour at the step level so a
// future refactor of the guardrail evaluation path cannot silently break it.
func TestRuntime_StepGuardrails_NegativePrefix(t *testing.T) {
	provider := &mockGuardrailProvider{
		responses: []string{
			// Step output: contains "DROP TABLE" → !DROP TABLE guardrail violated
			"DROP TABLE users; -- approved",
			// Retry: clean output
			"Approved without destructive SQL",
		},
	}

	task := TaskDefinition{
		ID:    "negative-step-guardrail",
		Name:  "Negative Step Guardrail",
		Input: "review this",
		Steps: []StepDef{
			{
				ID:         "review",
				Name:       "Review",
				Prompt:     "Review the code",
				Guardrails: []string{"!DROP TABLE"},
				Produces:   []SlotSpec{{Key: "out"}},
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

	// 2 calls (first retry succeeds because !DROP TABLE is now respected).
	assert.Equal(t, 2, provider.callIdx)
	assert.True(t, results[0].Completed)
	assert.Equal(t, 1.0, results[0].GuardrailScore, "negative guardrail respected on second attempt")
}

// TestEffectiveGuardrails_ReturnsDefensiveCopy — story 10-0 Task 4.2.
// Mutating the returned slice must not affect the source TaskDefinition or
// StepDef. Required for concurrent-safe use under DAG execution.
func TestEffectiveGuardrails_ReturnsDefensiveCopy(t *testing.T) {
	task := TaskDefinition{Guardrails: []string{"task-rule"}}
	step := StepDef{ID: "x"}
	out := effectiveGuardrails(step, task)
	require.Equal(t, []string{"task-rule"}, out)

	out[0] = "MUTATED"
	assert.Equal(t, "task-rule", task.Guardrails[0],
		"mutating the returned slice must not affect the source TaskDefinition")

	step2 := StepDef{ID: "y", Guardrails: []string{"step-rule"}}
	out2 := effectiveGuardrails(step2, task)
	out2[0] = "MUTATED"
	assert.Equal(t, "step-rule", step2.Guardrails[0],
		"mutating the returned slice must not affect the source StepDef")

	// Disabled-with-step-guardrails case: the function returns nil, not an
	// empty mutable slice. Pin the contract so a future caller assuming
	// "always non-nil writable" is forced to nil-check first.
	step3 := StepDef{ID: "z", DisableTaskGuardrails: true, Guardrails: []string{"x"}}
	out3 := effectiveGuardrails(step3, task)
	assert.Nil(t, out3, "DisableTaskGuardrails: true must return nil regardless of step.Guardrails")
}

// TestTaskContextContent_IncludesStepLevelGuardrails — story 10-0 Task 4.1.
// task_context.md must list step-level guardrails when steps override the
// task-level set, so an agent reading the audit file sees the full picture.
func TestTaskContextContent_IncludesStepLevelGuardrails(t *testing.T) {
	task := TaskDefinition{
		ID:         "ctx-test",
		Name:       "Ctx Test",
		Input:      "code",
		Guardrails: []string{"task-level-rule"},
		Steps: []StepDef{
			{ID: "s1", Name: "Step One", Guardrails: []string{"step-rule-A", "!secret"}},
			{ID: "s2", Name: "Step Two", DisableTaskGuardrails: true},
			{ID: "step-three"}, // no overrides — should not appear in step section
		},
	}
	got := string(taskContextContent(task))
	assert.Contains(t, got, "task-level-rule", "task-level guardrails always rendered")
	assert.Contains(t, got, "Step One", "step with overrides rendered by name")
	assert.Contains(t, got, "step-rule-A", "step-level guardrail listed")
	assert.Contains(t, got, "!secret", "step-level negative guardrail listed verbatim")
	assert.Contains(t, got, "Step Two", "step that disables task guardrails rendered")
	assert.Contains(t, got, "task-level disabled", "disable_task_guardrails flag annotated")
	assert.NotContains(t, got, "step-three", "step with no overrides not listed in step section")
}

// TestTaskContextContent_DisableSuppressesStepGuardrails — story 10-0 Task 4.1
// + post-review patch P1. When DisableTaskGuardrails is true, effectiveGuardrails
// returns nil regardless of step.Guardrails. The audit file must reflect that
// — listing step.Guardrails as enforced when the runtime ignores them would
// lie to the agent reading task_context.md.
func TestTaskContextContent_DisableSuppressesStepGuardrails(t *testing.T) {
	task := TaskDefinition{
		ID:         "ctx-disable",
		Name:       "Disable Test",
		Input:      "code",
		Guardrails: []string{"task-rule"},
		Steps: []StepDef{
			{
				ID:                    "s1",
				Name:                  "Disabled Step",
				DisableTaskGuardrails: true,
				Guardrails:            []string{"would-be-step-rule"},
			},
		},
	}
	got := string(taskContextContent(task))
	assert.Contains(t, got, "Disabled Step", "step rendered in audit")
	assert.Contains(t, got, "step-level suppressed", "first-line annotation distinguishes disabled-with-step-guardrails from disabled-without")
	assert.Contains(t, got, "suppressed by disable_task_guardrails",
		"step-level guardrails declared with disable flag must be marked suppressed, not enforced")
	// Stronger than `- would-be-step-rule` — a regression that re-renders the
	// suppressed rule with a different bullet style (`  ~ would-be-step-rule`,
	// `  • would-be-step-rule`, etc.) would still leak the rule string. The
	// rule name must not appear at all.
	assert.NotContains(t, got, "would-be-step-rule",
		"step.Guardrails string must NOT appear anywhere when DisableTaskGuardrails is set, regardless of bullet style")
}
