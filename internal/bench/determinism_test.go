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
