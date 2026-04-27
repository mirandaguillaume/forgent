package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirandaguillaume/forgent/internal/doctor/registry"
	"github.com/mirandaguillaume/forgent/internal/llm"
)

// mockBenchProvider returns a fixed findings YAML for any prompt.
type mockBenchProvider struct {
	response string
}

func (m *mockBenchProvider) Complete(_ string) (string, error) {
	return m.response, nil
}

var _ llm.Provider = (*mockBenchProvider)(nil)

// createMockCorpus writes 5 synthetic agent files to a temp directory.
func createMockCorpus(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	claudeDir := filepath.Join(dir, "claude")
	copilotDir := filepath.Join(dir, "copilot")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	require.NoError(t, os.MkdirAll(copilotDir, 0755))

	claudeFiles := []struct {
		name    string
		content string
	}{
		{"agent-a.md", "# Agent A\n\n## System Prompt\nYou are Agent A.\n\n## Tools\n- Read\n- Write\n"},
		{"agent-b.md", "# Agent B\n\n## System Prompt\nYou are Agent B.\n\n## Tools\n- Read\n- Grep\n- Bash\n"},
		{"agent-c.md", "# Agent C\n\n## System Prompt\nYou are Agent C.\n"},
	}

	copilotFiles := []struct {
		name    string
		content string
	}{
		{"helper-x.md", "---\nname: helper-x\ntools: read\n---\n\n## Instructions\nHelp users.\n"},
		{"helper-y.md", "---\nname: helper-y\ntools: search\n---\n\n## Instructions\nSearch docs.\n"},
	}

	for _, f := range claudeFiles {
		path := filepath.Join(claudeDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte(f.content), 0644))
	}
	for _, f := range copilotFiles {
		path := filepath.Join(copilotDir, f.name)
		require.NoError(t, os.WriteFile(path, []byte(f.content), 0644))
	}

	return dir
}

func TestRunInstructionBench_MockCorpus(t *testing.T) {
	corpusDir := createMockCorpus(t)
	reg, err := registry.Load("")
	require.NoError(t, err)

	provider := &mockBenchProvider{
		response: `findings:
- category: "tool-integration"
  issue: "Limited tool set"
  confidence: "moderate"
  current_state: "Few tools"
  suggested_improvement: "Add more tools"
- category: "context-management"
  issue: "No context boundaries"
  confidence: "low"
  current_state: "Missing"
  suggested_improvement: "Define boundaries"
`,
	}

	result, err := RunInstructionBench(InstructionBenchOpts{
		CorpusDir: corpusDir,
		Provider:  provider,
		Registry:  reg,
		Model:     "test-model",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 5, result.TotalFiles)
	assert.Equal(t, 10, result.TotalFindings) // 2 findings per file × 5 files
	assert.Contains(t, result.ByCategory, "tool-integration")
	assert.Contains(t, result.ByCategory, "context-management")
	assert.Equal(t, 5, result.ByCategory["tool-integration"].Count)
	assert.Equal(t, 5, result.ByCategory["context-management"].Count)

	// Confidence distribution: 5 moderate + 5 low
	assert.Equal(t, 5, result.ConfidenceDistribution["moderate"])
	assert.Equal(t, 5, result.ConfidenceDistribution["low"])

	// Overall score: all 5 files have findings → 100%
	assert.Equal(t, 100.0, result.OverallScore)
}

func TestRunInstructionBench_NilProvider(t *testing.T) {
	corpusDir := createMockCorpus(t)
	reg, err := registry.Load("")
	require.NoError(t, err)

	result, err := RunInstructionBench(InstructionBenchOpts{
		CorpusDir: corpusDir,
		Provider:  nil, // static-only mode
		Registry:  reg,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalFiles)
	assert.Equal(t, 0, result.TotalFindings, "no findings without LLM provider")
}

func TestRunInstructionBench_EmptyCorpus(t *testing.T) {
	dir := t.TempDir()
	_, err := RunInstructionBench(InstructionBenchOpts{
		CorpusDir: dir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no .md files")
}

func TestRunInstructionBench_MissingDir(t *testing.T) {
	_, err := RunInstructionBench(InstructionBenchOpts{
		CorpusDir: "/nonexistent/path",
	})
	require.Error(t, err)
}

func TestRunInstructionBench_Reproducible(t *testing.T) {
	corpusDir := createMockCorpus(t)
	reg, err := registry.Load("")
	require.NoError(t, err)

	provider := &mockBenchProvider{
		response: `findings:
- category: "guardrails"
  issue: "No guardrails"
  confidence: "high"
  current_state: "Missing"
  suggested_improvement: "Add guardrails"
`,
	}

	opts := InstructionBenchOpts{
		CorpusDir: corpusDir,
		Provider:  provider,
		Registry:  reg,
		Model:     "test",
	}

	result1, err := RunInstructionBench(opts)
	require.NoError(t, err)

	result2, err := RunInstructionBench(opts)
	require.NoError(t, err)

	// Deterministic: same corpus + same config → same measurements
	assert.Equal(t, result1.TotalFiles, result2.TotalFiles)
	assert.Equal(t, result1.TotalFindings, result2.TotalFindings)
	assert.Equal(t, result1.OverallScore, result2.OverallScore)
	assert.Equal(t, len(result1.ByCategory), len(result2.ByCategory))
}

func TestRunInstructionBench_GoldenSet(t *testing.T) {
	corpusDir := createMockCorpus(t)
	reg, err := registry.Load("")
	require.NoError(t, err)

	// Write a golden set that expects only "tool-integration" for claude files
	goldenPath := filepath.Join(t.TempDir(), "golden.yaml")
	goldenContent := `files:
  claude/agent-a.md:
    expected_categories: [tool-integration]
  claude/agent-b.md:
    expected_categories: [tool-integration]
  claude/agent-c.md:
    expected_categories: [tool-integration]
`
	require.NoError(t, os.WriteFile(goldenPath, []byte(goldenContent), 0644))

	provider := &mockBenchProvider{
		response: `findings:
- category: "tool-integration"
  issue: "Test"
  confidence: "moderate"
  current_state: "x"
  suggested_improvement: "y"
- category: "context-management"
  issue: "Unexpected"
  confidence: "low"
  current_state: "x"
  suggested_improvement: "y"
`,
	}

	result, err := RunInstructionBench(InstructionBenchOpts{
		CorpusDir:  corpusDir,
		Provider:   provider,
		GoldenPath: goldenPath,
		Registry:   reg,
	})
	require.NoError(t, err)

	// context-management findings on golden-annotated files are false positives
	// (3 claude files are annotated, each gets 1 FP for "context-management")
	cmCat := result.ByCategory["context-management"]
	require.NotNil(t, cmCat)
	assert.Equal(t, 3, cmCat.FalsePositives, "3 golden-annotated files × 1 unexpected category each")
}
