package experiment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findSlotWriteEntries returns audit entries whose Path is slots/{key}.md AND
// whose action is "slot_write". Used to verify the inline produces validation
// recorded a denial WITHOUT the corresponding successful write.
func findSlotWriteEntries(log []AccessEntry, stepID, key string) []AccessEntry {
	var out []AccessEntry
	target := "slots/" + key + ".md"
	for _, e := range log {
		if e.StepID == stepID && e.Path == target && e.Action == "slot_write" {
			out = append(out, e)
		}
	}
	return out
}

// TestRuntime_ProducesValidation_NoSlotWriteOnFailure — Story 10-4 T5.6 (AC #1).
// A step's produces declares a structural template. The LLM output is missing
// the required heading. The runtime must:
//   - return a validation error from the runFn
//   - NOT write slots/{key}.md to the sandbox
func TestRuntime_ProducesValidation_NoSlotWriteOnFailure(t *testing.T) {
	taskDir := t.TempDir()
	// Structural template: requires a ## Verdict heading with body content.
	schema := "## Verdict\n\nDecision.\n"
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "schema.md"), []byte(schema), 0644))

	provider := &mockGuardrailProvider{
		responses: []string{"some output with no heading at all"},
	}

	task := TaskDefinition{
		ID:      "no-slot-write-on-failure",
		Name:    "No slot write on failure",
		Input:   "do work",
		TaskDir: taskDir,
		Steps: []StepDef{
			{
				ID:     "review",
				Name:   "Review",
				Prompt: "Provide a verdict",
				Produces: []SlotSpec{
					{Key: "verdict", TemplateFile: "schema.md"},
				},
			},
		},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err, "RunRuntimeExperiment must not return a fatal error for a contract violation")
	require.Len(t, results, 1)
	require.Error(t, results[0].Error, "task must surface the contract-violation error")
	assert.Contains(t, results[0].Error.Error(), "quality gate: heading", "error wording must come from the AST structural check")

	sawSlotWrite := false
	for _, e := range results[0].ActionLog {
		if e.StepID == "review" && e.Action == "write" && e.Path == "slots/verdict.md" {
			sawSlotWrite = true
			break
		}
	}
	assert.False(t, sawSlotWrite, "AC #1: slots/verdict.md must NOT be written when the produces contract fails")
}

// TestRuntime_ProducesValidation_AuditEntryOnFailure — Story 10-4 T5.7 (AC #1).
// On contract violation the inline gate records a sandbox audit entry with
// Allowed: false and a Reason that telemetry (story 10-6) can parse.
func TestRuntime_ProducesValidation_AuditEntryOnFailure(t *testing.T) {
	taskDir := t.TempDir()
	// Structural template: requires a ## Verdict heading with body.
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "schema.md"), []byte("## Verdict\n\nDecision.\n"), 0644))

	provider := &mockGuardrailProvider{
		responses: []string{"no heading here at all"},
	}

	task := TaskDefinition{
		ID:      "audit-entry-on-failure",
		Name:    "Audit entry on failure",
		Input:   "do work",
		TaskDir: taskDir,
		Steps: []StepDef{
			{
				ID:       "review",
				Name:     "Review",
				Prompt:   "Provide a verdict",
				Produces: []SlotSpec{{Key: "verdict", TemplateFile: "schema.md"}},
			},
		},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)

	denials := findSlotWriteEntries(results[0].ActionLog, "review", "verdict")
	require.NotEmpty(t, denials, "expected at least one audit entry for the denied slot_write")

	// At least one entry must be a denial with the AST heading-missing prefix.
	sawDenial := false
	for _, e := range denials {
		if !e.Allowed && strings.HasPrefix(e.Reason, "heading missing:") {
			assert.Contains(t, e.Reason, "## verdict", "audit reason must name the missing heading")
			sawDenial = true
		}
	}
	assert.True(t, sawDenial, "expected at least one audit entry with Allowed=false and 'heading missing:' prefix")
}

// TestRuntime_ProducesValidation_PassesWhenSatisfied — companion to T5.6/T5.7.
// When the LLM output satisfies the structural contract, the slot IS written
// and the audit entry is Allowed: true.
func TestRuntime_ProducesValidation_PassesWhenSatisfied(t *testing.T) {
	taskDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "schema.md"), []byte("## Verdict\n\nDecision.\n"), 0644))

	provider := &mockGuardrailProvider{
		responses: []string{"## Verdict\n\nApproved — no issues found."},
	}

	task := TaskDefinition{
		ID:      "produces-passes",
		Name:    "Produces passes",
		Input:   "do work",
		TaskDir: taskDir,
		Steps: []StepDef{
			{
				ID:       "review",
				Name:     "Review",
				Prompt:   "Provide a verdict",
				Produces: []SlotSpec{{Key: "verdict", TemplateFile: "schema.md"}},
			},
		},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NoError(t, results[0].Error, "task must complete when produces is satisfied")

	// The slot WAS written — audit trail has an Allowed=true write entry.
	sawWrite := false
	for _, e := range results[0].ActionLog {
		if e.StepID == "review" && e.Action == "write" && e.Path == "slots/verdict.md" && e.Allowed {
			sawWrite = true
			break
		}
	}
	assert.True(t, sawWrite, "expected one Allowed write to slots/verdict.md when produces is satisfied")
}

// TestSlotSpec_TemplateFile_AST_RoundTrip — Story 10-4 T5.11 (updated for 10-11).
// End-to-end YAML round-trip: a task YAML referencing a schema file with H2
// headings parses, resolves, and validates correctly through the AST path.
func TestSlotSpec_TemplateFile_AST_RoundTrip(t *testing.T) {
	taskDir := t.TempDir()
	schemaDir := filepath.Join(taskDir, "schemas")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "verdict.md"),
		[]byte("## Verdict\n\nThe ruling.\n"), 0644))

	taskYAML := `id: round-trip-ast
name: Round trip ast
input: review this
steps:
  - id: review
    name: Review
    prompt: Provide a ruling
    produces:
      - key: verdict
        template_file: ./schemas/verdict.md
`
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "task.yaml"), []byte(taskYAML), 0644))

	tasks, err := LoadTasks(taskDir)
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// Output satisfies the structural contract: heading present + non-empty body.
	provider := &mockGuardrailProvider{
		responses: []string{"## Verdict\n\nThe ruling is: ship."},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       tasks,
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NoError(t, results[0].Error, "round-trip with template_file + AST structural check must succeed")

	// The slot must be written with the LLM output content.
	sawWrite := false
	for _, e := range results[0].ActionLog {
		if e.Action == "write" && e.Path == "slots/verdict.md" && e.Allowed {
			sawWrite = true
			break
		}
	}
	assert.True(t, sawWrite, "expected slots/verdict.md to be written end-to-end")
}

// stepKeyedProvider returns different responses based on a substring match
// against the prompt — used to test the consumes-retry path in T5.10 where
// step1's response must change between the first and second invocation.
type stepKeyedProvider struct {
	mu      sync.Mutex
	calls   map[string]int
	respond func(stepKey string, attempt int) string
}

func (p *stepKeyedProvider) Complete(prompt string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var stepKey string
	switch {
	case strings.Contains(prompt, "Step: Producer"):
		stepKey = "producer"
	case strings.Contains(prompt, "Step: Consumer"):
		stepKey = "consumer"
	default:
		return "", fmt.Errorf("stepKeyedProvider: prompt matched no known step key — check that step names contain 'Producer' or 'Consumer'")
	}
	p.calls[stepKey]++
	return p.respond(stepKey, p.calls[stepKey]), nil
}

// TestRuntime_ConsumesTemplate_RetryClearsUpstreamSlot — Story 10-4 T5.10
// (AC #5 + Task 5.4 from 10-0 deferred).
//
// Setup: 2-step task. Producer's first response is bad (lacks the "## Verdict"
// heading the consumer's consumes_template requires). Consumer's middleware
// pre-check fails. The wrapper must:
//   - clear slots/verdict.md from sandbox (sandbox.Delete + audit "slot_clear")
//   - re-invoke producer.runFn — producer's SECOND response IS valid
//   - retry consumer with refreshed inputs — consumer's middleware now passes
//
// Asserts:
//   - producer's provider.Complete called twice (per-step counter)
//   - consumer's provider.Complete called at least once (and ultimately succeeds)
//   - audit trail has a slot_clear entry for slots/verdict.md
//   - task completes (no surfaced error)
func TestRuntime_ConsumesTemplate_RetryClearsUpstreamSlot(t *testing.T) {
	taskDir := t.TempDir()
	// Consumer's consumes_template requires the "## Verdict" heading.
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "consumer-input.md"),
		[]byte("## Verdict\n- decision: ...\n"), 0644))

	provider := &stepKeyedProvider{
		calls: make(map[string]int),
		respond: func(stepKey string, attempt int) string {
			switch stepKey {
			case "producer":
				if attempt == 1 {
					// Bad: no heading — consumer's consumes-template fails.
					return "approved"
				}
				return "## Verdict\n- decision: approved"
			case "consumer":
				return "## Result\nlooks good"
			}
			return ""
		},
	}

	task := TaskDefinition{
		ID:      "consumes-retry",
		Name:    "Consumes retry",
		Input:   "review this",
		TaskDir: taskDir,
		Steps: []StepDef{
			{
				ID:       "producer",
				Name:     "Producer",
				Prompt:   "Produce a verdict",
				Produces: []SlotSpec{{Key: "verdict"}},
			},
			{
				ID:       "consumer",
				Name:     "Consumer",
				Prompt:   "Consume the verdict",
				Consumes: []SlotSpec{{Key: "verdict", TemplateFile: "consumer-input.md"}},
				Produces: []SlotSpec{{Key: "summary"}},
			},
		},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NoError(t, results[0].Error, "task must recover and complete: %v", results[0].Error)

	provider.mu.Lock()
	prodCalls := provider.calls["producer"]
	consCalls := provider.calls["consumer"]
	provider.mu.Unlock()
	assert.Equal(t, 2, prodCalls, "producer must be called twice (1 initial + 1 recovery re-invocation)")
	assert.GreaterOrEqual(t, consCalls, 1, "consumer must be called at least once after recovery")

	// Audit trail has a slot_clear entry for slots/verdict.md initiated by the
	// consumer step (the wrapper records the action under the consumer's stepID).
	sawClear := false
	for _, e := range results[0].ActionLog {
		if e.Action == "slot_clear" && e.Path == "slots/verdict.md" && e.StepID == "consumer" {
			sawClear = true
			break
		}
	}
	assert.True(t, sawClear, "expected a slot_clear audit entry for slots/verdict.md by consumer")
}

// ─── classifyValidationReason pin tests (Story 6-18 review patch P3) ─────────

// classifyValidationReason maps qualitygate error strings to stable audit-trail
// Reason prefixes consumed by story 10-6 telemetry. Any rewording of either
// side must break these tests — the breakage is intentional.
func TestClassifyValidationReason_UnknownError_PassesThrough(t *testing.T) {
	err := fmt.Errorf("some unexpected error message")
	got := classifyValidationReason(err)
	assert.Equal(t, "some unexpected error message", got,
		"unknown error messages pass through verbatim to preserve audit trail informativeness")
}

// ─── classifyValidationReason pin tests — Story 10-11 AST prefixes (T5.2) ────

// TestClassifyValidationReason_HeadingMissing pins the "heading missing:" prefix
// produced by AST validation when an expected heading is absent from the output.
func TestClassifyValidationReason_HeadingMissing(t *testing.T) {
	err := fmt.Errorf(`quality gate: heading "## verdict" not found`)
	got := classifyValidationReason(err)
	assert.Equal(t, "heading missing: ## verdict", got,
		"stable prefix for 10-6 telemetry: heading missing: <heading>")
}

// TestClassifyValidationReason_SectionEmpty pins the "section structural violation:"
// prefix for an AST-detected empty section.
func TestClassifyValidationReason_SectionEmpty(t *testing.T) {
	err := fmt.Errorf(`quality gate: section "## verdict" is empty`)
	got := classifyValidationReason(err)
	assert.Equal(t, "section structural violation: ## verdict", got,
		"stable prefix for 10-6 telemetry: section structural violation: <heading>")
}

// TestClassifyValidationReason_SectionMissingList pins the same
// "section structural violation:" prefix for a missing-list failure.
func TestClassifyValidationReason_SectionMissingList(t *testing.T) {
	err := fmt.Errorf(`quality gate: section "## issues" requires a list`)
	got := classifyValidationReason(err)
	assert.Equal(t, "section structural violation: ## issues", got,
		"stable prefix for 10-6 telemetry: section structural violation: <heading>")
}
