package experiment

import (
	"net/http"
	"testing"
	"time"

	pkgsandbox "github.com/mirandaguillaume/forgent/pkg/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// mockNetworkRequestProvider makes a real HTTP request via the env-var proxy
// during Complete so the proxy's audit callback fires. The HTTP error is
// intentionally swallowed; only the LLM response matters to the runtime.
type mockNetworkRequestProvider struct {
	url      string
	response string
}

func (m *mockNetworkRequestProvider) Complete(_ string) (string, error) {
	// Proxy: http.ProxyFromEnvironment is required — &http.Transport{} with nil
	// Proxy does not fall back to env vars (unlike http.DefaultTransport which
	// explicitly sets Proxy: ProxyFromEnvironment). This ensures the transport
	// honours the HTTP_PROXY env var set by the runtime's proxy lifecycle block.
	resp, _ := (&http.Client{
		Timeout:   5 * time.Second, // P3: prevent CI hang if proxy is slow
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
	}).Get(m.url)
	if resp != nil {
		resp.Body.Close()
	}
	return m.response, nil
}

// taskWithStep builds a minimal TaskDefinition with one LLM step so tests
// can focus on what they're actually testing.
func taskWithStep(step StepDef) TaskDefinition {
	step.Prompt = "do something"
	if len(step.Produces) == 0 {
		step.Produces = []SlotSpec{{Key: "out"}}
	}
	return TaskDefinition{
		ID:    "t1",
		Name:  "Test Task",
		Input: "input",
		Steps: []StepDef{step},
	}
}

// ─── T5.1 ─────────────────────────────────────────────────────────────────────

func TestNodeViolation_GuardrailViolationOnFirstAttempt(t *testing.T) {
	// Step guardrail "!secret" fails on attempt 1, passes on attempt 2.
	provider := &mockGuardrailProvider{
		responses: []string{
			"this response has secret info",  // attempt 1: violates "!secret"
			"this response is completely clean", // attempt 2: passes
		},
	}
	task := taskWithStep(StepDef{
		ID:         "s1",
		Guardrails: []string{"!secret"},
	})

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	r := results[0]

	require.Len(t, r.NodeViolations, 1)
	v := r.NodeViolations[0]
	assert.Equal(t, "s1", v.NodeID)
	assert.Equal(t, ViolationTypeGuardrail, v.ViolationType)
	assert.True(t, v.Attempted)
	assert.False(t, v.Prevented, "guardrail violations are observed post-hoc, not blocked before effect")
	assert.Equal(t, 1, v.AttemptNumber)
	assert.Contains(t, v.Reason, "secret")
	assert.InDelta(t, 0.0, r.ContractComplianceRate, 0.001)
}

// ─── T5.2 ─────────────────────────────────────────────────────────────────────

func TestNodeViolation_FilePermissionViolation(t *testing.T) {
	// Step is read-only; LLM emits a file_write tool call which is blocked at
	// layer 1 (checkFileAccess). The step then succeeds on the follow-up call.
	provider := &mockGuardrailProvider{
		responses: []string{
			`<tool name="file_write" path="src/main.go" content="x"/>`, // tool → blocked
			"analysis complete", // follow-up after block report → final response
		},
	}
	task := taskWithStep(StepDef{
		ID:         "s1",
		FileAccess: FileAccessReadOnly,
		Tools:      []string{"file_write"},
	})

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	r := results[0]

	require.Len(t, r.NodeViolations, 1)
	v := r.NodeViolations[0]
	assert.Equal(t, ViolationTypeFilePermission, v.ViolationType)
	assert.True(t, v.Prevented, "layer-1 file access denial happens before the write takes effect")
	assert.Equal(t, 1, v.AttemptNumber)
	assert.Equal(t, "src/main.go", v.Target)
	assert.InDelta(t, 0.0, r.ContractComplianceRate, 0.001)
}

// ─── T5.3 ─────────────────────────────────────────────────────────────────────

func TestNodeViolation_OutputContractViolation(t *testing.T) {
	// Step produces a slot with a structural template requiring a ## Verdict heading.
	// First invocation returns text without the heading → slot_write blocked.
	// DAG retries (MaxRetries=2); second invocation returns valid structured text.
	provider := &mockGuardrailProvider{
		responses: []string{
			"analysis done without heading",        // attempt 1: no ## Verdict → contract violation
			"## Verdict\n\nguilty as charged.",     // DAG retry attempt: has heading + body → passes
		},
	}
	task := taskWithStep(StepDef{
		ID: "s1",
		Produces: []SlotSpec{{
			Key:      "analysis",
			Template: "## Verdict\n\nSome analysis.\n",
		}},
	})

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	r := results[0]

	require.Len(t, r.NodeViolations, 1)
	v := r.NodeViolations[0]
	assert.Equal(t, "s1", v.NodeID)
	assert.Equal(t, ViolationTypeOutputContract, v.ViolationType)
	assert.True(t, v.Prevented, "slot_write blocked before the slot file is written")
	assert.Equal(t, 1, v.AttemptNumber)
	assert.Equal(t, "slots/analysis.md", v.Target)
	assert.InDelta(t, 0.0, r.ContractComplianceRate, 0.001)
}

// ─── T5.4 ─────────────────────────────────────────────────────────────────────

func TestNodeViolation_NetworkViolation(t *testing.T) {
	// Step has ForbiddenDomains: ["api.stripe.com"].
	// The provider makes an HTTP request to http://api.stripe.com/ during
	// Complete. The enforcement proxy blocks it and records a "network" entry
	// with Allowed=false. The actual request never reaches stripe.com (blocked
	// at proxy level before any upstream connection).
	provider := &mockNetworkRequestProvider{
		url:      "http://api.stripe.com/",
		response: "analysis complete",
	}
	task := taskWithStep(StepDef{
		ID: "s1",
		NetworkPolicy: &pkgsandbox.NetworkPolicy{
			ForbiddenDomains: []string{"api.stripe.com"},
		},
	})

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	r := results[0]

	require.Len(t, r.NodeViolations, 1)
	v := r.NodeViolations[0]
	assert.Equal(t, "s1", v.NodeID)
	assert.Equal(t, ViolationTypeNetwork, v.ViolationType)
	assert.True(t, v.Prevented, "proxy blocks the request before it reaches the host")
	assert.Equal(t, 1, v.AttemptNumber)
	assert.Equal(t, "api.stripe.com", v.Target)
	assert.InDelta(t, 0.0, r.ContractComplianceRate, 0.001)
}

// ─── T5.5 ─────────────────────────────────────────────────────────────────────

func TestNodeViolation_NoViolation_EmptySlice(t *testing.T) {
	provider := &mockGuardrailProvider{responses: []string{"clean response"}}
	task := taskWithStep(StepDef{ID: "s1"})

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	r := results[0]

	assert.Empty(t, r.NodeViolations)
	assert.InDelta(t, 1.0, r.ContractComplianceRate, 0.001)
}

// ─── T5.6 ─────────────────────────────────────────────────────────────────────

func TestNodeViolation_MultipleAttemptsRecorded(t *testing.T) {
	// Guardrail "!secret" fails on attempts 1 and 2, passes on attempt 3.
	provider := &mockGuardrailProvider{
		responses: []string{
			"this has secret content",  // attempt 1: violates
			"still has secret here",   // attempt 2: violates
			"completely clean output", // attempt 3: passes (maxAttempts=3)
		},
	}
	task := taskWithStep(StepDef{
		ID:         "s1",
		Guardrails: []string{"!secret"},
	})

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	r := results[0]

	require.Len(t, r.NodeViolations, 2)
	assert.Equal(t, 1, r.NodeViolations[0].AttemptNumber)
	assert.Equal(t, 2, r.NodeViolations[1].AttemptNumber)
	assert.InDelta(t, 0.0, r.ContractComplianceRate, 0.001, "attempt-1 violation = non-compliant regardless of subsequent retries")
}

// ─── T5.7 ─────────────────────────────────────────────────────────────────────

func TestNodeViolation_MultipleViolationTypesSameAttempt(t *testing.T) {
	// Step is read-only with a guardrail "!secret".
	// Attempt 1: LLM emits file_write tool (blocked) → follow-up response violates "!secret".
	// Attempt 2: clean response.
	// Expected: 2 NodeViolations (file_permission + guardrail), both AttemptNumber=1.
	// ContractComplianceRate = 0.0 (step has violations on first attempt).
	provider := &mockGuardrailProvider{
		responses: []string{
			`<tool name="file_write" path="src/main.go" content="x"/>`, // iter 0: tool call
			"this has secret and is bad", // iter 1: follow-up → no tool calls → lastResponse
			"totally clean answer",       // attempt 2: passes guardrail
		},
	}
	task := taskWithStep(StepDef{
		ID:         "s1",
		FileAccess: FileAccessReadOnly,
		Tools:      []string{"file_write"},
		Guardrails: []string{"!secret"},
	})

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	r := results[0]

	require.Len(t, r.NodeViolations, 2)

	types := map[string]bool{}
	for _, v := range r.NodeViolations {
		assert.Equal(t, 1, v.AttemptNumber)
		types[v.ViolationType] = true
	}
	assert.True(t, types[ViolationTypeFilePermission])
	assert.True(t, types[ViolationTypeGuardrail])

	assert.InDelta(t, 0.0, r.ContractComplianceRate, 0.001,
		"multiple violations on same attempt count as ONE non-compliant step")
}

// ─── T5.8 ─────────────────────────────────────────────────────────────────────

func TestContractComplianceRate_MixedSteps(t *testing.T) {
	// 4 steps: step2 and step4 have first-attempt violations, step1 and step3 are clean.
	violations := []NodeViolation{
		{NodeID: "step2", ViolationType: ViolationTypeGuardrail, AttemptNumber: 1},
		{NodeID: "step4", ViolationType: ViolationTypeFilePermission, AttemptNumber: 1},
	}
	rate := computeContractComplianceRate(4, violations)
	assert.InDelta(t, 0.5, rate, 0.001)
}

// ─── T5.9 ─────────────────────────────────────────────────────────────────────

func TestNodeViolation_ChronologicalOrder(t *testing.T) {
	// Violations recorded out of step order; aggregateViolations must sort by
	// (step declaration index, attempt number).
	task := TaskDefinition{
		ID: "t1",
		Steps: []StepDef{
			{ID: "s1"}, // index 0
			{ID: "s2"}, // index 1
		},
	}
	sb := NewSandboxFS(nil, nil, SandboxPolicy{})

	// Record in inverse order: s2 attempt 1, then s1 attempt 2, then s1 attempt 1.
	sb.SetCurrentAttempt("s2", 1)
	sb.record("s2", "guardrail", "rule-A", false, "violated")
	sb.SetCurrentAttempt("s1", 2)
	sb.record("s1", "network", "api.evil.com", false, "forbidden")
	sb.SetCurrentAttempt("s1", 1)
	sb.record("s1", "write", "src/main.go", false, "read-only")

	violations := aggregateViolations(task, sb)

	require.Len(t, violations, 3)
	// s1 attempt 1 (write) comes first
	assert.Equal(t, "s1", violations[0].NodeID)
	assert.Equal(t, 1, violations[0].AttemptNumber)
	assert.Equal(t, ViolationTypeFilePermission, violations[0].ViolationType)
	// s1 attempt 2 (network)
	assert.Equal(t, "s1", violations[1].NodeID)
	assert.Equal(t, 2, violations[1].AttemptNumber)
	assert.Equal(t, ViolationTypeNetwork, violations[1].ViolationType)
	// s2 attempt 1 (guardrail)
	assert.Equal(t, "s2", violations[2].NodeID)
	assert.Equal(t, 1, violations[2].AttemptNumber)
	assert.Equal(t, ViolationTypeGuardrail, violations[2].ViolationType)
}

// ─── T5.10 ────────────────────────────────────────────────────────────────────

func TestContractComplianceRate_ZeroSteps_DefensiveOne(t *testing.T) {
	// Validate() rejects zero-step tasks, but the helper must not divide by zero.
	rate := computeContractComplianceRate(0, nil)
	assert.Equal(t, 1.0, rate)
}
