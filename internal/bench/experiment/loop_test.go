package experiment

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ─── T2 helper tests ──────────────────────────────────────────────────────────

func TestEffectiveLoopPolicy_NilReturnsDefault(t *testing.T) {
	step := StepDef{ID: "s", Loop: nil}
	p := effectiveLoopPolicy(step)
	assert.Equal(t, DefaultMaxToolIterations, p.MaxIterations)
	assert.Empty(t, p.ExitWhen.OutputMarker)
	assert.Empty(t, p.ExitWhen.ToolCalled)
}

func TestEffectiveLoopPolicy_NonNilReturnedAsIs(t *testing.T) {
	step := StepDef{
		ID: "s",
		Loop: &LoopPolicy{
			MaxIterations: 10,
			ExitWhen:      ExitCriteria{OutputMarker: "<!-- DONE -->"},
		},
	}
	p := effectiveLoopPolicy(step)
	assert.Equal(t, 10, p.MaxIterations)
	assert.Equal(t, "<!-- DONE -->", p.ExitWhen.OutputMarker)
}

// ─── T1 Validate tests ────────────────────────────────────────────────────────

func TestValidate_RejectsZeroOrNegativeMaxIterations(t *testing.T) {
	for _, maxIter := range []int{0, -1} {
		task := taskWithStep(StepDef{
			ID:       "s1",
			Produces: []SlotSpec{{Key: "out"}},
			Loop:     &LoopPolicy{MaxIterations: maxIter},
		})
		cfg := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: &funcProvider{fn: func(s string) (string, error) { return "ok", nil }}}
		err := cfg.Validate()
		require.Error(t, err, "max_iterations=%d should be rejected", maxIter)
		assert.Contains(t, err.Error(), "max_iterations must be a positive integer")
	}
}

func TestValidate_RejectsUnknownLoopField(t *testing.T) {
	// Simulate what happens when YAML contains max_attempts (typo for max_iterations).
	// We can't load from YAML in unit tests, so we directly inject unknownKeys.
	lp := &LoopPolicy{MaxIterations: 3}
	lp.unknownKeys = []string{"max_attempts"}

	task := taskWithStep(StepDef{
		ID:       "s1",
		Produces: []SlotSpec{{Key: "out"}},
		Loop:     lp,
	})
	cfg := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: &funcProvider{fn: func(s string) (string, error) { return "ok", nil }}}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_attempts")
	assert.Contains(t, err.Error(), "max_iterations")
	assert.Contains(t, err.Error(), "exit_when")
}

func TestValidate_RejectsUnknownExitWhenField(t *testing.T) {
	lp := &LoopPolicy{MaxIterations: 3}
	lp.ExitWhen.unknownKeys = []string{"marker"} // typo for output_marker

	task := taskWithStep(StepDef{
		ID:       "s1",
		Produces: []SlotSpec{{Key: "out"}},
		Loop:     lp,
	})
	cfg := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: &funcProvider{fn: func(s string) (string, error) { return "ok", nil }}}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marker")
	assert.Contains(t, err.Error(), "exit_when.output_marker")
}

func TestValidate_LoopWithMaxIterationsOnly_Valid(t *testing.T) {
	// Pure safety backstop — no exit criteria — is valid.
	task := taskWithStep(StepDef{
		ID:       "s1",
		Produces: []SlotSpec{{Key: "out"}},
		Loop:     &LoopPolicy{MaxIterations: 3},
	})
	cfg := ExperimentConfig{Tasks: []TaskDefinition{task}, Provider: &funcProvider{fn: func(s string) (string, error) { return "ok", nil }}}
	assert.NoError(t, cfg.Validate())
}

// ─── T1.5 YAML round-trip ─────────────────────────────────────────────────────

func TestLoopPolicy_YAMLRoundTrip(t *testing.T) {
	var td TaskDefinition
	require.NoError(t, yaml.Unmarshal([]byte(`
id: rt
name: test
input: input
steps:
  - id: s1
    prompt: do it
    produces:
      - key: out
    loop:
      max_iterations: 7
      exit_when:
        output_marker: "<!-- DONE -->"
        tool_called: finish
`), &td))

	require.Len(t, td.Steps, 1)
	lp := td.Steps[0].Loop
	require.NotNil(t, lp)
	assert.Equal(t, 7, lp.MaxIterations)
	assert.Equal(t, "<!-- DONE -->", lp.ExitWhen.OutputMarker)
	assert.Equal(t, "finish", lp.ExitWhen.ToolCalled)
	assert.Empty(t, lp.unknownKeys)
	assert.Empty(t, lp.ExitWhen.unknownKeys)
}

// ─── T6: declarative loop behaviour integration tests ────────────────────────

// T6.1: OutputMarker fires early exit.
func TestStepLoop_OutputMarkerExits(t *testing.T) {
	callCount := 0
	p := &funcProvider{fn: func(prompt string) (string, error) {
		callCount++
		if callCount == 1 {
			return "... working ...", nil // tool call would come here but no tools
		}
		return "<!-- APPROVED --> final answer", nil
	}}

	// No tools declared — the tool loop doesn't fire, each attempt is one call.
	task := taskWithStep(StepDef{
		ID:       "s1",
		Produces: []SlotSpec{{Key: "out"}},
		Loop: &LoopPolicy{
			MaxIterations: 5,
			ExitWhen:      ExitCriteria{OutputMarker: "<!-- APPROVED -->"},
		},
	})

	result := runRuntimeTask(task, p, nil, nil, "runtime")
	assert.NoError(t, result.Error)
	assert.True(t, result.Completed)
	// The loop exits on the second call (attempt 2 of the guardrail+loop combined).
	// Exact call count depends on wiring — assert at most MaxIterations+1.
	assert.LessOrEqual(t, callCount, 6)
}

// T6.2: ToolCalled exit criterion.
func TestStepLoop_ToolCalledExits(t *testing.T) {
	callCount := 0
	p := &funcProvider{fn: func(prompt string) (string, error) {
		callCount++
		// Return a tool call for "submit" on the first try.
		return `<tool name="submit" verdict="approved"/>`, nil
	}}

	task := taskWithStep(StepDef{
		ID:       "s1",
		Tools:    []string{"submit"},
		Produces: []SlotSpec{{Key: "out"}},
		Loop: &LoopPolicy{
			MaxIterations: 5,
			ExitWhen:      ExitCriteria{ToolCalled: "submit"},
		},
	})
	task.Workspace = &WorkspaceDef{
		Files: []WorkspaceFile{{Path: "task_context.md", Content: "ctx"}},
	}

	result := runRuntimeTask(task, p, nil, nil, "runtime")
	assert.NoError(t, result.Error)
	assert.True(t, result.Completed)
}

// T6.3: Both criteria declared — either one fires exit.
func TestStepLoop_BothCriteriaOR(t *testing.T) {
	// Sub-case A: marker-only response exits.
	t.Run("marker", func(t *testing.T) {
		p := &funcProvider{fn: func(prompt string) (string, error) {
			return "<!-- APPROVED --> yes", nil
		}}
		task := taskWithStep(StepDef{
			ID:       "s1",
			Produces: []SlotSpec{{Key: "out"}},
			Loop: &LoopPolicy{
				MaxIterations: 3,
				ExitWhen:      ExitCriteria{OutputMarker: "<!-- APPROVED -->", ToolCalled: "submit"},
			},
		})
		result := runRuntimeTask(task, p, nil, nil, "runtime")
		assert.NoError(t, result.Error)
		assert.True(t, result.Completed)
	})

	// Sub-case B: tool-only response exits.
	t.Run("tool", func(t *testing.T) {
		p := &funcProvider{fn: func(prompt string) (string, error) {
			return `<tool name="submit" verdict="ok"/>`, nil
		}}
		task := taskWithStep(StepDef{
			ID:       "s1",
			Tools:    []string{"submit"},
			Produces: []SlotSpec{{Key: "out"}},
			Loop: &LoopPolicy{
				MaxIterations: 3,
				ExitWhen:      ExitCriteria{OutputMarker: "<!-- APPROVED -->", ToolCalled: "submit"},
			},
		})
		task.Workspace = &WorkspaceDef{
			Files: []WorkspaceFile{{Path: "task_context.md", Content: "ctx"}},
		}
		result := runRuntimeTask(task, p, nil, nil, "runtime")
		assert.NoError(t, result.Error)
		assert.True(t, result.Completed)
	})
}

// T6.4: MaxIterations exceeded → ErrLoopExhausted, Completed=false.
func TestStepLoop_MaxIterationsExceeded(t *testing.T) {
	callCount := 0
	p := &funcProvider{fn: func(prompt string) (string, error) {
		callCount++
		return "plain text no marker no tool call", nil
	}}

	task := taskWithStep(StepDef{
		ID:       "s1",
		Produces: []SlotSpec{{Key: "out"}},
		Loop: &LoopPolicy{
			MaxIterations: 2,
			ExitWhen:      ExitCriteria{OutputMarker: "<!-- APPROVED -->"},
		},
	})
	task.Workspace = &WorkspaceDef{
		Files: []WorkspaceFile{{Path: "task_context.md", Content: "ctx"}},
	}

	result := runRuntimeTask(task, p, nil, nil, "runtime")
	assert.True(t, errors.Is(result.Error, ErrLoopExhausted), "expected ErrLoopExhausted, got: %v", result.Error)
	assert.False(t, result.Completed)
	// MaxIterations=2 loop iterations + 1 final synthesize call = 3.
	// Wrapped in the guardrail retry (up to 3 guardrail attempts) — the loop
	// exhaustion error propagates out on the first guardrail attempt.
	assert.GreaterOrEqual(t, callCount, 3, "need MaxIterations+1 calls for the synthesize attempt")
}

// T6.5: Loop=nil → implicit default (backward compat, AC#5).
func TestStepLoop_NoLoopBlock_DefaultsToImplicit(t *testing.T) {
	// When Loop is nil, effectiveLoopPolicy returns DefaultMaxToolIterations.
	// We don't call the LLM MaxIterations times — just verify it doesn't error.
	p := &funcProvider{fn: func(prompt string) (string, error) {
		return "clean answer", nil
	}}
	task := taskWithStep(StepDef{
		ID:       "s1",
		Produces: []SlotSpec{{Key: "out"}},
		Loop:     nil,
	})
	result := runRuntimeTask(task, p, nil, nil, "runtime")
	assert.NoError(t, result.Error)
}

// T6.6: Declarative MaxIterations overrides implicit default.
func TestStepLoop_DeclarativeOverridesImplicit(t *testing.T) {
	callCount := 0
	p := &funcProvider{fn: func(prompt string) (string, error) {
		callCount++
		return "plain text no marker", nil
	}}

	task := taskWithStep(StepDef{
		ID:       "s1",
		Produces: []SlotSpec{{Key: "out"}},
		Loop: &LoopPolicy{
			MaxIterations: 2, // less than DefaultMaxToolIterations=5
			ExitWhen:      ExitCriteria{OutputMarker: "<!-- DONE -->"},
		},
	})
	task.Workspace = &WorkspaceDef{
		Files: []WorkspaceFile{{Path: "task_context.md", Content: "ctx"}},
	}

	result := runRuntimeTask(task, p, nil, nil, "runtime")
	assert.True(t, errors.Is(result.Error, ErrLoopExhausted))
	// MaxIterations=2 → 3 calls per runFn (2 loop + 1 synthesize).
	// DAG MaxRetries=2 causes runFn to be invoked up to 3 times → 9 max.
	// DefaultMaxToolIterations=5 would give 18 — assert we're well below that.
	assert.LessOrEqual(t, callCount, 9, "loop must terminate at MaxIterations=2, not DefaultMaxToolIterations=5")
}
