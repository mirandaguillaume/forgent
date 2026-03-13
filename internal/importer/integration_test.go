package importer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_SimpleReviewer(t *testing.T) {
	// Read golden input
	input, err := os.ReadFile("testdata/input/simple-reviewer.md")
	require.NoError(t, err)

	// Mock provider returns a pre-built valid response
	provider := &testProvider{
		response: `{"skills": [{"yaml": "skill: code-reviewer\nversion: \"0.1.0\"\ncontext:\n  consumes: [git_diff, source_code]\n  produces: [review_comments]\n  memory: short-term\nstrategy:\n  tools: [read_file, grep, bash]\n  approach: sequential\n  steps:\n    - Read the git diff\n    - Check for security, performance, and style issues\n    - Write review comments with specific suggestions\nguardrails:\n  - timeout: 120s\n  - Be constructive and focus on bugs over style\nobservability:\n  trace_level: standard\n  metrics: [comments_count, issues_found]\nsecurity:\n  filesystem: read-only\n  network: none\n  secrets: []\nnegotiation:\n  file_conflicts: yield\n  priority: 0"}], "agent": null}`,
	}

	// Write input to temp file
	dir := t.TempDir()
	inputPath := dir + "/code-reviewer.md"
	os.WriteFile(inputPath, input, 0644)

	result := RunImport(ImportOptions{
		Source:   inputPath,
		Provider: provider,
		MinScore: 0,
		OutputDir: dir,
	})

	require.True(t, result.Success, "import failed: %s", result.Error)
	assert.Len(t, result.Skills, 1)

	skill := result.Skills[0].Skill
	assert.Equal(t, "code-reviewer", skill.Skill)
	assert.Contains(t, skill.Context.Consumes, "git_diff")
	assert.Contains(t, skill.Context.Produces, "review_comments")
	assert.Contains(t, skill.Strategy.Tools, "read_file")
	assert.Nil(t, result.Agent)

	// Score should be reasonable
	assert.Greater(t, result.Skills[0].Score.Total, 40)

	// Write and verify output
	written, err := WriteImportResult(result, dir)
	require.NoError(t, err)
	assert.Len(t, written, 1)

	// Verify written file is valid YAML
	content, err := os.ReadFile(written[0])
	require.NoError(t, err)
	assert.Contains(t, string(content), "code-reviewer")
}
