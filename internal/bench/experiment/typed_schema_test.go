package experiment

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── T4.4: provider dispatch ──────────────────────────────────────────────────

// mockStructuredProvider implements llm.StructuredProvider.
type mockStructuredProvider struct {
	called     bool
	returnData map[string]any
	returnErr  error
}

func (m *mockStructuredProvider) Complete(prompt string) (string, error) {
	return "", nil
}

func (m *mockStructuredProvider) CompleteStructured(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	m.called = true
	return m.returnData, m.returnErr
}

func TestTypedSchema_ProviderDispatch_StructuredProvider_Called(t *testing.T) {
	sp := &mockStructuredProvider{returnData: map[string]any{"ruling": "approved"}}

	task := TaskDefinition{
		ID:    "dispatch-test",
		Name:  "Dispatch Test",
		Input: "input",
		Steps: []StepDef{{
			ID:     "step1",
			Name:   "Step1",
			Prompt: "decide",
			Produces: []SlotSpec{{
				Key: "verdict",
				Schema: schema.FieldSchema{
					"ruling": {Type: "string", Enum: []string{"approved", "rejected"}},
				},
			}},
		}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    sp,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed, "task must complete via StructuredProvider path")
	assert.True(t, sp.called, "CompleteStructured must have been called")
}

func TestTypedSchema_ProviderDispatch_NoStructuredProvider_UsesFormatInstruction(t *testing.T) {
	var capturedPrompt string
	provider := &capturePromptProvider{
		response: `{"ruling": "approved"}`,
		capture:  func(p string) { capturedPrompt = p },
	}

	task := TaskDefinition{
		ID:    "fallback-test",
		Name:  "Fallback Test",
		Input: "input",
		Steps: []StepDef{{
			ID:     "step1",
			Name:   "Step1",
			Prompt: "decide",
			Produces: []SlotSpec{{
				Key: "verdict",
				Schema: schema.FieldSchema{
					"ruling": {Type: "string", Enum: []string{"approved", "rejected"}},
				},
			}},
		}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed)
	assert.Contains(t, capturedPrompt, "Output ONLY a valid JSON object",
		"fallback path must inject format instruction into prompt")
	assert.Contains(t, capturedPrompt, `"ruling"`)
}

// capturePromptProvider records the last prompt and returns a fixed response.
type capturePromptProvider struct {
	response string
	capture  func(string)
}

func (p *capturePromptProvider) Complete(prompt string) (string, error) {
	if p.capture != nil {
		p.capture(prompt)
	}
	return p.response, nil
}

// ─── T7.1: end-to-end produce → consume ──────────────────────────────────────

func TestTypedSchema_ProducesAndConsumes_EndToEnd(t *testing.T) {
	sp := &mockStructuredProvider{returnData: map[string]any{"verdict": "approved"}}

	task := TaskDefinition{
		ID:    "end-to-end",
		Name:  "End to End",
		Input: "input",
		Steps: []StepDef{
			{
				ID:     "producer",
				Name:   "Producer",
				Prompt: "produce a verdict",
				Produces: []SlotSpec{{
					Key: "result",
					Schema: schema.FieldSchema{
						"verdict": {Type: "string", Enum: []string{"approved"}},
					},
				}},
			},
			{
				ID:       "consumer",
				Name:     "Consumer",
				Prompt:   "use the verdict",
				Consumes: []SlotSpec{{Key: "result"}},
				Produces: []SlotSpec{{Key: "summary"}},
			},
		},
	}

	provider := &mockStepProvider{
		// consumer step returns plain text for the summary
		responses: map[string]string{
			"consumer": "Summary: approved.",
		},
	}
	provider.structured = sp

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed)
	assert.Nil(t, results[0].Error)
}

// mockStepProvider: structured for the first step, then falls back to a plain response.
type mockStepProvider struct {
	responses  map[string]string
	structured *mockStructuredProvider
	callCount  int
}

func (m *mockStepProvider) Complete(prompt string) (string, error) {
	m.callCount++
	// Return the "consumer" response for any non-structured call.
	for _, v := range m.responses {
		return v, nil
	}
	return `{"verdict": "approved"}`, nil
}

func (m *mockStepProvider) CompleteStructured(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	if m.structured != nil {
		return m.structured.CompleteStructured(context.Background(), "", nil)
	}
	return map[string]any{"verdict": "approved"}, nil
}

// ─── T7.2: validation failure triggers DAG retry ─────────────────────────────

func TestTypedSchema_ValidationFailure_TriggersRetry(t *testing.T) {
	// First call: enum violation. Second: valid.
	responses := []string{`{"ruling": "pending"}`, `{"ruling": "approved"}`}
	idx := 0
	provider := &singleFuncProvider{fn: func(prompt string) (string, error) {
		r := responses[idx]
		if idx < len(responses)-1 {
			idx++
		}
		return r, nil
	}}

	task := TaskDefinition{
		ID:    "retry-test",
		Name:  "Retry Test",
		Input: "input",
		Steps: []StepDef{{
			ID:     "step1",
			Name:   "Step1",
			Prompt: "decide",
			Produces: []SlotSpec{{
				Key: "verdict",
				Schema: schema.FieldSchema{
					"ruling": {Type: "string", Enum: []string{"approved", "rejected"}},
				},
			}},
		}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed, "task must complete after retry: %v", results[0].Error)
}

// ─── T7.3: JSON extraction failure triggers retry ────────────────────────────

func TestTypedSchema_JSONExtractionFailure_TriggersRetry(t *testing.T) {
	responses := []string{"plain prose with no JSON at all", `{"ruling": "approved"}`}
	idx := 0
	provider := &singleFuncProvider{fn: func(prompt string) (string, error) {
		r := responses[idx]
		if idx < len(responses)-1 {
			idx++
		}
		return r, nil
	}}

	task := TaskDefinition{
		ID:    "extract-retry",
		Name:  "Extract Retry",
		Input: "input",
		Steps: []StepDef{{
			ID:     "step1",
			Name:   "Step1",
			Prompt: "decide",
			Produces: []SlotSpec{{
				Key: "verdict",
				Schema: schema.FieldSchema{
					"ruling": {Type: "string", Enum: []string{"approved", "rejected"}},
				},
			}},
		}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed, "task must complete after extraction retry: %v", results[0].Error)
}

// ─── T7.4: backward compat — no schema → .md path ────────────────────────────

func TestTypedSchema_NoSchema_BackwardCompat(t *testing.T) {
	provider := &mockGuardrailProvider{responses: []string{"## Verdict\n\nApproved.\n"}}

	task := TaskDefinition{
		ID:    "compat",
		Name:  "Compat",
		Input: "input",
		Steps: []StepDef{{
			ID:     "step1",
			Name:   "Step1",
			Prompt: "produce verdict",
			// No schema — plain string slot, backward compat
			Produces: []SlotSpec{{Key: "verdict"}},
		}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed)

	// Slot file must be .md, not .json
	sawMD := false
	for _, e := range results[0].ActionLog {
		if e.StepID == "step1" && e.Action == "write" && e.Path == "slots/verdict.md" {
			sawMD = true
		}
		assert.NotEqual(t, "slots/verdict.json", e.Path, "non-typed step must not write .json slot")
	}
	assert.True(t, sawMD, "non-typed step must write slots/verdict.md")
}

// ─── T7.5: audit denial on typed schema violation ────────────────────────────

func TestTypedSchema_AuditDenial_OnValidationFailure(t *testing.T) {
	// Always return enum violation — no retry succeeds (MaxRetries exhausted → DAG error).
	provider := &mockGuardrailProvider{
		responses: []string{`{"ruling": "invalid"}`, `{"ruling": "invalid"}`, `{"ruling": "invalid"}`},
	}

	task := TaskDefinition{
		ID:    "audit-deny",
		Name:  "Audit Deny",
		Input: "input",
		Steps: []StepDef{{
			ID:     "step1",
			Name:   "Step1",
			Prompt: "decide",
			Produces: []SlotSpec{{
				Key: "verdict",
				Schema: schema.FieldSchema{
					"ruling": {Type: "string", Enum: []string{"approved", "rejected"}},
				},
			}},
		}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err) // RunRuntimeExperiment doesn't fail for task errors
	require.Len(t, results, 1)
	require.Error(t, results[0].Error)

	// Audit trail must contain a slot_write denial with typed schema prefix
	sawDenial := false
	for _, e := range results[0].ActionLog {
		if e.StepID == "step1" && e.Action == "slot_write" && !e.Allowed {
			sawDenial = true
			assert.True(t, strings.HasPrefix(e.Reason, "typed schema violation:"),
				"audit reason must have 'typed schema violation:' prefix, got: %s", e.Reason)
		}
	}
	assert.True(t, sawDenial, "must record a slot_write denial for schema violation")
}

// ─── T5.4: pin tests for new classifyValidationReason prefixes ───────────────

func TestClassifyValidationReason_JSONExtraction(t *testing.T) {
	err := fmt.Errorf("json extraction: no valid JSON object found")
	assert.Equal(t, "json extraction failed", classifyValidationReason(err))
}

func TestClassifyValidationReason_TypedSchemaField(t *testing.T) {
	err := fmt.Errorf("typed schema: field %q missing from output", "ruling")
	assert.Equal(t, `typed schema violation: ruling`, classifyValidationReason(err))
}

func TestClassifyValidationReason_TypedSchemaEnum(t *testing.T) {
	err := fmt.Errorf(`typed schema: field "ruling" value "pending" not in enum [approved rejected]`)
	assert.Equal(t, `typed schema violation: ruling`, classifyValidationReason(err))
}

// singleFuncProvider calls fn for every Complete call.
type singleFuncProvider struct {
	fn func(string) (string, error)
}

func (f *singleFuncProvider) Complete(prompt string) (string, error) {
	return f.fn(prompt)
}
