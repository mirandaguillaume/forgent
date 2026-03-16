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
	// 16 skills + 1 agent = 17 files
	assert.Equal(t, 17, result.ComposedFiles, "should generate 17 files")
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

func TestTokenOverhead_WordCountIsPositive(t *testing.T) {
	// Lines 66/67: word counting in generated files.
	// If CountWords returns 0 or negative, this test catches it.
	result, err := RunTokenOverhead("../../skills", "../../agents", "claude")
	require.NoError(t, err)
	assert.Greater(t, result.ComposedWords, 0, "composed output must have words")
	assert.Greater(t, result.MonolithicWords, 0, "monolithic output must have words")
	// The composed should have at least as many words as monolithic (due to overhead).
	assert.GreaterOrEqual(t, result.ComposedWords, result.MonolithicWords,
		"composed should have at least as many words due to structural overhead")
}

func TestTokenOverhead_OverheadCalculation(t *testing.T) {
	// Line 67: overhead formula: (composed - monolithic) / monolithic * 100.
	// Mutation: changing + to - or * to / would produce wrong result.
	result, err := RunTokenOverhead("../../skills", "../../agents", "claude")
	require.NoError(t, err)
	// Manually verify the overhead calculation.
	expectedOverhead := float64(result.ComposedWords-result.MonolithicWords) / float64(result.MonolithicWords) * 100
	assert.InDelta(t, expectedOverhead, result.OverheadPct, 0.001,
		"overhead should match manual calculation")
}

func TestTokenOverhead_MonolithicContent(t *testing.T) {
	// Line 129: monolithic content should include all skill/agent semantic content.
	// If countMonolithic is broken, it would return 0.
	result, err := RunTokenOverhead("../../skills", "../../agents", "copilot")
	require.NoError(t, err)
	assert.Greater(t, result.MonolithicWords, 0, "monolithic should have content")
}

func TestTokenOverhead_FileTraversal(t *testing.T) {
	// Lines 147/172: file traversal for skills and agents.
	// Verify that the number of composed files matches expected count.
	result, err := RunTokenOverhead("../../skills", "../../agents", "claude")
	require.NoError(t, err)
	// claude target: 16 skills + 1 agent = 17 files
	assert.Equal(t, 17, result.ComposedFiles)
}

func TestTokenOverhead_CopilotFileCount(t *testing.T) {
	// Copilot target generates additional files (instructions, toolmap).
	result, err := RunTokenOverhead("../../skills", "../../agents", "copilot")
	require.NoError(t, err)
	assert.Greater(t, result.ComposedFiles, 0, "copilot should generate files")
}

func TestTokenOverhead_ZeroMonolithicSafeDivision(t *testing.T) {
	// Line 66: `if monolithicWords > 0` — when monolithic is 0, overhead should be 0.
	tmpSkills := t.TempDir()
	tmpAgents := t.TempDir()
	result, err := RunTokenOverhead(tmpSkills, tmpAgents, "claude")
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.OverheadPct, "zero monolithic → zero overhead (safe division)")
}

func TestTokenOverhead_EmptyDir(t *testing.T) {
	tmpSkills := t.TempDir()
	tmpAgents := t.TempDir()
	result, err := RunTokenOverhead(tmpSkills, tmpAgents, "claude")
	require.NoError(t, err)
	assert.Equal(t, 0, result.ComposedWords, "empty dirs should produce 0 words")
	assert.Equal(t, 0, result.MonolithicWords)
}

func TestTokenOverhead_Compact_Claude(t *testing.T) {
	standard, err := RunTokenOverhead("../../skills", "../../agents", "claude")
	require.NoError(t, err)

	compact, err := RunTokenOverheadCompact("../../skills", "../../agents", "claude")
	require.NoError(t, err)

	assert.Less(t, compact.OverheadPct, standard.OverheadPct,
		"compact should have lower overhead than standard")
	assert.Equal(t, 1, compact.ComposedFiles,
		"compact should produce a single agent file")
	assert.Less(t, compact.ComposedWords, standard.ComposedWords,
		"compact should have fewer words than standard")
	t.Logf("Standard: %d words (%.1f%%) → Compact: %d words (%.1f%%)",
		standard.ComposedWords, standard.OverheadPct,
		compact.ComposedWords, compact.OverheadPct)
}

func TestTokenOverhead_Compact_Copilot(t *testing.T) {
	standard, err := RunTokenOverhead("../../skills", "../../agents", "copilot")
	require.NoError(t, err)

	compact, err := RunTokenOverheadCompact("../../skills", "../../agents", "copilot")
	require.NoError(t, err)

	assert.Less(t, compact.OverheadPct, standard.OverheadPct,
		"compact should have lower overhead than standard")
	t.Logf("Standard: %d words (%.1f%%) → Compact: %d words (%.1f%%)",
		standard.ComposedWords, standard.OverheadPct,
		compact.ComposedWords, compact.OverheadPct)
}
