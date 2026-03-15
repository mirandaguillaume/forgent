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
