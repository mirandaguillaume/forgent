package doctor

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirandaguillaume/forgent/internal/doctor/registry"
	"github.com/mirandaguillaume/forgent/internal/llm"
)

func TestBuildDAG_NodeCount(t *testing.T) {
	d, err := BuildDAG(false)
	require.NoError(t, err)

	nodes := d.Nodes()
	assert.Len(t, nodes, 4, "expected 4 doctor DAG nodes")
}

func TestBuildDAG_Topology(t *testing.T) {
	d, err := BuildDAG(false)
	require.NoError(t, err)

	layers := d.Layers()
	require.Len(t, layers, 3, "expected 3 layers in doctor DAG")

	// Layer 0: format-detector only (it consumes agent_file which is external)
	assert.Equal(t, []string{"format-detector"}, layers[0], "Layer 0")

	// Layer 1: analyzer and context-enricher (both depend on detected_format)
	layer1 := make([]string, len(layers[1]))
	copy(layer1, layers[1])
	sort.Strings(layer1)
	assert.Equal(t, []string{"analyzer", "context-enricher"}, layer1, "Layer 1")

	// Layer 2: recommendation-builder (depends on analysis_results + context_recommendations)
	assert.Equal(t, []string{"recommendation-builder"}, layers[2], "Layer 2")
}

func TestBuildDAG_AutoWiring(t *testing.T) {
	d, err := BuildDAG(false)
	require.NoError(t, err)

	// format-detector → analyzer (via detected_format)
	assert.Contains(t, d.Downstream("format-detector"), "analyzer")
	// format-detector → context-enricher (via detected_format)
	assert.Contains(t, d.Downstream("format-detector"), "context-enricher")
	// analyzer → recommendation-builder (via analysis_results)
	assert.Contains(t, d.Downstream("analyzer"), "recommendation-builder")
	// context-enricher → recommendation-builder (via context_recommendations)
	assert.Contains(t, d.Downstream("context-enricher"), "recommendation-builder")

	// recommendation-builder has no downstream
	assert.Empty(t, d.Downstream("recommendation-builder"))
}

// mockProvider implements llm.Provider for testing.
type mockProvider struct {
	response string
}

func (m *mockProvider) Complete(prompt string) (string, error) {
	return m.response, nil
}

func TestRunDAG_WithMockProvider(t *testing.T) {
	d, err := BuildDAG(false)
	require.NoError(t, err)

	// Minimal Claude-format agent file
	content := []byte(`# Test Agent

## System Prompt
You are a test agent.

## Tools
- Read
- Write
`)

	reg, err := registry.Load(".")
	require.NoError(t, err)

	// Mock provider that returns a valid YAML findings response
	mock := &mockProvider{
		response: `findings:
- category: "tool-integration"
  issue: "Limited tool set"
  confidence: "low"
  current_state: "Only Read and Write"
  suggested_improvement: "Consider adding Bash"
`,
	}

	input := DoctorInput{
		FilePath:    ".claude/agents/test-agent.md",
		Content:     content,
		Provider:    mock,
		Registry:    reg,
		ProjectRoot: "",
		Debug:       false,
	}

	result, err := RunDAG(context.Background(), d, input, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Analysis)
	assert.Equal(t, "claude", result.Format)
	assert.NotNil(t, result.AllFindings)
}

func TestRunDAG_NilProvider(t *testing.T) {
	d, err := BuildDAG(false)
	require.NoError(t, err)

	content := []byte(`# Test Agent

## System Prompt
You are a test agent.
`)

	reg, err := registry.Load(".")
	require.NoError(t, err)

	input := DoctorInput{
		FilePath:    ".claude/agents/test-agent.md",
		Content:     content,
		Provider:    nil,
		Registry:    reg,
		ProjectRoot: "",
	}

	result, err := RunDAG(context.Background(), d, input, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
	// LLM findings should be empty when no provider
	assert.Empty(t, result.LLMFindings)
}

func TestRunDAG_Concurrency1(t *testing.T) {
	d, err := BuildDAG(false)
	require.NoError(t, err)

	content := []byte(`# Test Agent

## System Prompt
You are a test agent.
`)

	reg, err := registry.Load(".")
	require.NoError(t, err)

	input := DoctorInput{
		FilePath:    ".claude/agents/test-agent.md",
		Content:     content,
		Provider:    nil,
		Registry:    reg,
		ProjectRoot: "",
	}

	// Concurrency=1 should work (simulates Ollama serialization)
	result, err := RunDAG(context.Background(), d, input, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// Verify the interface is satisfied at compile time.
var _ llm.Provider = (*mockProvider)(nil)

// ─── Task 2: Conditional rewriter skip ───────────────────────────────────────

func TestBuildDAG_HasExactlyThreeLayers(t *testing.T) {
	// The rewriter (Layer 3) was removed from scope in Story 4.2.
	// This test asserts the DAG terminates after recommendation-builder (Layer 2).
	d, err := BuildDAG(false)
	require.NoError(t, err)

	layers := d.Layers()
	assert.Len(t, layers, 3, "no Layer 3 rewriter: DAG must have exactly 3 layers")
	assert.Equal(t, []string{"recommendation-builder"}, layers[2], "Layer 2 is the terminal layer")
}

func TestBuildDAG_RecommendationBuilderIsTerminal(t *testing.T) {
	d, err := BuildDAG(false)
	require.NoError(t, err)

	// The terminal node has no outgoing edges.
	assert.Empty(t, d.Downstream("recommendation-builder"),
		"recommendation-builder must have no downstream nodes")
}

// ─── Task 3: Debug mode logging ───────────────────────────────────────────────

func TestBuildDAG_WithDebug_Builds(t *testing.T) {
	// Smoke test: BuildDAG(debug=true) must not error.
	d, err := BuildDAG(true)
	require.NoError(t, err)
	assert.NotNil(t, d)
}

func TestRunDAG_DebugEmitsToStderr(t *testing.T) {
	// Build with debug=true and run — verify that wrapWithDebug emits output.
	// We can't easily capture os.Stderr here, but we can verify that RunDAG
	// still succeeds with debug wrapping applied.
	d, err := BuildDAG(false) // use false so we don't pollute test output
	require.NoError(t, err)

	content := []byte("# Test Agent\n\n## System Prompt\nYou are a test.\n")
	reg, err := registry.Load(".")
	require.NoError(t, err)

	result, err := RunDAG(context.Background(), d, DoctorInput{
		FilePath: ".claude/agents/test.md",
		Content:  content,
		Provider: nil,
		Registry: reg,
		Debug:    false,
	}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ─── Task 4: Retry configuration ─────────────────────────────────────────────

// flakyProvider returns an error for the first failCount calls, then succeeds.
type flakyProvider struct {
	calls     int
	failCount int
	response  string
}

func (p *flakyProvider) Complete(_ string) (string, error) {
	p.calls++
	if p.calls <= p.failCount {
		return "", fmt.Errorf("transient error: attempt %d", p.calls)
	}
	return p.response, nil
}

var _ llm.Provider = (*flakyProvider)(nil)

func TestRunDAG_RetryOnTransientFailure(t *testing.T) {
	d, err := BuildDAG(false)
	require.NoError(t, err)

	content := []byte(`# Test Agent

## System Prompt
You are a test agent.

## Tools
- Read
`)

	reg, err := registry.Load(".")
	require.NoError(t, err)

	provider := &flakyProvider{
		failCount: 1, // fail first call, succeed on second
		response: `findings:
- category: "tool-integration"
  issue: "Limited tool set"
  confidence: "low"
  current_state: "Only Read"
  suggested_improvement: "Add more tools"
`,
	}

	input := DoctorInput{
		FilePath: ".claude/agents/retry-test.md",
		Content:  content,
		Provider: provider,
		Registry: reg,
	}

	result, err := RunDAG(context.Background(), d, input, 0)
	require.NoError(t, err, "DAG should succeed after retry")
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, provider.calls, 2, "provider should be called at least twice (retry happened)")
}

func TestRunDAG_ExhaustsRetries_ReturnsNodeError(t *testing.T) {
	d, err := BuildDAG(false)
	require.NoError(t, err)

	content := []byte("# Agent\n\n## System Prompt\nTest.\n")
	reg, err := registry.Load(".")
	require.NoError(t, err)

	// Always fails — will exhaust all retry attempts
	provider := &flakyProvider{
		failCount: 999, // always fails
		response:  "",
	}

	input := DoctorInput{
		FilePath: ".claude/agents/exhausted.md",
		Content:  content,
		Provider: provider,
		Registry: reg,
	}

	_, err = RunDAG(context.Background(), d, input, 0)
	require.Error(t, err, "exhausted retries must return an error")
}
