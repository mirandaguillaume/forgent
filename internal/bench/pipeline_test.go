package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPipelineProvider returns a valid import response with 1 skill + 1 agent.
type mockPipelineProvider struct{}

func (m *mockPipelineProvider) Complete(prompt string) (string, error) {
	return `{"skills": [{"yaml": "skill: test-skill\nversion: 1.0.0\ncontext:\n  consumes: [input]\n  produces: [output]\n  memory: short-term\nstrategy:\n  tools: [read_file]\n  approach: analyze input\n  steps:\n    - Read the input data\n    - Produce the output\nguardrails:\n  - timeout: 60\nobservability:\n  trace_level: minimal\nsecurity:\n  filesystem: read-only\n  network: none\nnegotiation:\n  file_conflicts: yield\n  priority: 1"}], "agent": {"yaml": "agent: test-agent\nskills: [test-skill]\norchestration: sequential\ndescription: test agent\nconsumes: [input]\nproduces: [output]"}, "contracts": null}`, nil
}

// mockFailProvider returns invalid JSON.
type mockFailProvider struct{}

func (m *mockFailProvider) Complete(prompt string) (string, error) {
	return "not valid json at all", nil
}

func TestRunPipeline_MockProvider(t *testing.T) {
	// Create a temp agent .md file
	tmpDir := t.TempDir()
	agentPath := filepath.Join(tmpDir, "test-agent.md")
	err := os.WriteFile(agentPath, []byte("# Test Agent\n\nYou are a test agent that processes input."), 0644)
	require.NoError(t, err)

	result, err := RunPipeline(agentPath, &mockPipelineProvider{})
	require.NoError(t, err)

	assert.Equal(t, "test-agent", result.Source)
	assert.Equal(t, "test-agent", result.HandWritten.Name)
	assert.True(t, result.HandWritten.Words > 0)

	assert.Equal(t, "test-agent/forgent-standard", result.Standard.Name)
	assert.True(t, result.Standard.Words > 0)
	assert.NotEmpty(t, result.Standard.Prompt)

	assert.Equal(t, "test-agent/forgent-compact", result.Compact.Name)
	assert.True(t, result.Compact.Words > 0)
	assert.NotEmpty(t, result.Compact.Prompt)

	assert.True(t, result.ImportTime > 0)
	assert.True(t, result.BuildTime > 0)
}

func TestRunPipeline_ImportFailure(t *testing.T) {
	tmpDir := t.TempDir()
	agentPath := filepath.Join(tmpDir, "bad-agent.md")
	err := os.WriteFile(agentPath, []byte("# Bad Agent"), 0644)
	require.NoError(t, err)

	_, err = RunPipeline(agentPath, &mockFailProvider{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "import failed")
}

func TestRunPipeline_MissingFile(t *testing.T) {
	_, err := RunPipeline("/nonexistent/agent.md", &mockPipelineProvider{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read agent file")
}

func TestRunH2HWithTasks_EmptyList(t *testing.T) {
	result, err := RunH2HWithTasks(nil, nil, nil, 1, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Contestants)
}
