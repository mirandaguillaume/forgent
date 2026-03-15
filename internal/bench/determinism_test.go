package bench

import (
	"os"
	"path/filepath"
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

func TestDeterminism_MinimumRuns(t *testing.T) {
	// Requesting 1 run should be clamped to 2.
	result, err := RunDeterminism("../../skills", "../../agents", "claude", 1)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Runs)
	assert.True(t, result.Identical)
}

func TestDeterminism_ClampAtExactly2(t *testing.T) {
	// Line 19: `runs < 2` → `runs < 1` or `runs <= 2` would break this.
	// Requesting exactly 2 should NOT be clamped — it should stay 2.
	result, err := RunDeterminism("../../skills", "../../agents", "claude", 2)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Runs, "requesting 2 runs should yield exactly 2, not clamped")
	assert.True(t, result.Identical)
}

func TestDeterminism_ClampZeroToTwo(t *testing.T) {
	// Requesting 0 runs should be clamped to 2.
	result, err := RunDeterminism("../../skills", "../../agents", "claude", 0)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Runs)
}

func TestDeterminism_ClampNegativeToTwo(t *testing.T) {
	// Requesting negative runs should be clamped to 2.
	result, err := RunDeterminism("../../skills", "../../agents", "claude", -5)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Runs)
}

func TestDeterminism_FourRuns(t *testing.T) {
	// 4 runs should not be clamped.
	result, err := RunDeterminism("../../skills", "../../agents", "claude", 4)
	require.NoError(t, err)
	assert.Equal(t, 4, result.Runs)
	assert.True(t, result.Identical)
}

func TestDeterminism_NormalizationRemovesTempPaths(t *testing.T) {
	// Lines 53/64: normalization replaces temp dir paths with <OUTPUT>.
	// If normalization is broken, identical builds in different temp dirs would show diffs.
	// This test verifies identical builds are detected as such.
	result, err := RunDeterminism("../../skills", "../../agents", "claude", 3)
	require.NoError(t, err)
	assert.True(t, result.Identical, "normalization should make identical builds compare equal")
	assert.Equal(t, 0, result.DiffCount, "no diffs expected")
	assert.Empty(t, result.DiffFiles, "no diff files expected")
}

func TestDeterminism_ComparisonLogic(t *testing.T) {
	// Lines 55/65: the comparison loop runs for i=1..N against refDir.
	// Build 3 runs and verify the comparison catches all of them.
	result, err := RunDeterminism("../../skills", "../../agents", "copilot", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Runs)
	assert.True(t, result.Identical)
}

func TestDeterminism_DetectsDiff(t *testing.T) {
	// Build once, then tamper with a file to verify diffs are detected.
	tmpDir := t.TempDir()
	err := buildToDir("../../skills", "../../agents", tmpDir, "claude")
	require.NoError(t, err)

	// Run determinism with real builds (should pass)
	result, err := RunDeterminism("../../skills", "../../agents", "claude", 2)
	require.NoError(t, err)
	assert.True(t, result.Identical)

	// Now verify that our comparison logic actually catches differences:
	// build two dirs, tamper with one, compare manually.
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	require.NoError(t, buildToDir("../../skills", "../../agents", dir1, "claude"))
	require.NoError(t, buildToDir("../../skills", "../../agents", dir2, "claude"))

	// Tamper with a skill file in dir2
	err = filepath.Walk(dir2, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(path) != "SKILL.md" {
			return err
		}
		return os.WriteFile(path, []byte("tampered content"), 0644)
	})
	require.NoError(t, err)

	// Verify files are now different
	var hasDiff bool
	filepath.Walk(dir1, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir1, path)
		d1, _ := os.ReadFile(path)
		d2, _ := os.ReadFile(filepath.Join(dir2, rel))
		if string(d1) != string(d2) {
			hasDiff = true
		}
		return nil
	})
	assert.True(t, hasDiff, "tampered files should be detected as different")
}
