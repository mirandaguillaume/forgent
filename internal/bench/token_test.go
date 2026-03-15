package bench

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenOverhead_CIReviewer(t *testing.T) {
	result, err := RunTokenOverhead("../../skills", "../../agents", "claude")
	require.NoError(t, err)
	assert.Greater(t, result.OverheadPct, 0.0, "overhead should be positive")
	assert.Less(t, result.OverheadPct, 300.0, "overhead should be < 300% (sanity bound)")
	assert.Greater(t, result.ComposedFiles, 0, "should generate at least one file")
	// ci-reviewer has 6 skills + 1 agent = 7 files
	assert.Equal(t, 7, result.ComposedFiles, "ci-reviewer should generate 7 files")
	assert.Greater(t, result.MonolithicWords, 0, "monolithic should have content")
	t.Logf("Composed: %d words (%d files), Monolithic: %d words, Overhead: %.1f%%",
		result.ComposedWords, result.ComposedFiles, result.MonolithicWords, result.OverheadPct)
}

func TestTokenOverhead_Copilot(t *testing.T) {
	result, err := RunTokenOverhead("../../skills", "../../agents", "copilot")
	require.NoError(t, err)
	assert.Greater(t, result.OverheadPct, 0.0, "overhead should be positive")
	assert.Less(t, result.OverheadPct, 300.0, "overhead should be < 300% (sanity bound)")
	t.Logf("Composed: %d words (%d files), Monolithic: %d words, Overhead: %.1f%%",
		result.ComposedWords, result.ComposedFiles, result.MonolithicWords, result.OverheadPct)
}

func TestTokenOverhead_InvalidTarget(t *testing.T) {
	_, err := RunTokenOverhead("../../skills", "../../agents", "nonexistent")
	require.Error(t, err, "should fail for unknown target")
}

func TestTokenOverhead_MissingDir(t *testing.T) {
	_, err := RunTokenOverhead("/nonexistent/skills", "/nonexistent/agents", "claude")
	require.Error(t, err, "should fail for missing directories")
}

func TestTokenOverhead_EmptyDir(t *testing.T) {
	tmpSkills := t.TempDir()
	tmpAgents := t.TempDir()
	result, err := RunTokenOverhead(tmpSkills, tmpAgents, "claude")
	require.NoError(t, err)
	assert.Equal(t, 0, result.ComposedWords, "empty dirs should produce 0 words")
	assert.Equal(t, 0, result.MonolithicWords)
}
