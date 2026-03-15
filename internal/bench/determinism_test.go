package bench

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeterminism_ThreeRuns(t *testing.T) {
	result, err := RunDeterminism("../../skills", "../../agents", "claude", 3)
	require.NoError(t, err)
	assert.True(t, result.Identical, "3 runs should produce identical output")
	assert.Equal(t, 0, result.DiffCount)
	t.Logf("Runs: %d, Identical: %v, Diffs: %d", result.Runs, result.Identical, result.DiffCount)
	for _, f := range result.DiffFiles {
		t.Logf("  Diff: %s", f)
	}
}

func TestDeterminism_Copilot(t *testing.T) {
	result, err := RunDeterminism("../../skills", "../../agents", "copilot", 3)
	require.NoError(t, err)
	assert.True(t, result.Identical, "3 runs should produce identical output")
	assert.Equal(t, 0, result.DiffCount)
}
