package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteMiniSWEOverride_WithPrompt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "override.yaml")
	err := writeMiniSWEOverride(path, "You are a helpful bug fixer.", 5.0)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "system_template")
	assert.Contains(t, content, "bug fixer")
	assert.Contains(t, content, "cost_limit")
}

func TestWriteMiniSWEOverride_EmptyPrompt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "override.yaml")
	err := writeMiniSWEOverride(path, "", 3.0)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	// Empty prompt means no system_template override
	assert.NotContains(t, string(data), "system_template")
}

func TestParseMiniSWEResults(t *testing.T) {
	dir := t.TempDir()

	// Write a successful trajectory
	traj1 := map[string]interface{}{
		"info": map[string]interface{}{
			"exit_status": "submitted",
			"submission":  "diff --git a/foo.py b/foo.py\n-old\n+new",
			"model_stats": map[string]interface{}{
				"instance_cost": 0.45,
			},
		},
	}
	data1, _ := json.Marshal(traj1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance1.json"), data1, 0644))

	// Write a failed trajectory (limits exceeded)
	traj2 := map[string]interface{}{
		"info": map[string]interface{}{
			"exit_status": "LimitsExceeded",
			"submission":  "",
			"model_stats": map[string]interface{}{
				"instance_cost": 3.0,
			},
		},
	}
	data2, _ := json.Marshal(traj2)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance2.json"), data2, 0644))

	// Write a successful trajectory with empty submission
	traj3 := map[string]interface{}{
		"info": map[string]interface{}{
			"exit_status": "submitted",
			"submission":  "",
			"model_stats": map[string]interface{}{
				"instance_cost": 0.30,
			},
		},
	}
	data3, _ := json.Marshal(traj3)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance3.json"), data3, 0644))

	instances, resolved, totalCost := parseMiniSWEResults(dir)
	assert.Equal(t, 3, instances)
	assert.Equal(t, 1, resolved) // only traj1 has non-empty submission and no LimitsExceeded
	assert.InDelta(t, 3.75, totalCost, 0.01)
}

func TestParseMiniSWEResults_EmptyDir(t *testing.T) {
	instances, resolved, cost := parseMiniSWEResults(t.TempDir())
	assert.Equal(t, 0, instances)
	assert.Equal(t, 0, resolved)
	assert.Equal(t, 0.0, cost)
}

func TestDefaultMiniSWEConfig(t *testing.T) {
	cfg := DefaultMiniSWEConfig()
	assert.Equal(t, "/tmp/miniswe-env/bin/python", cfg.PythonBin)
	assert.NotEmpty(t, cfg.Model)
	assert.Greater(t, cfg.Workers, 0)
	assert.NotEmpty(t, cfg.Subset)
}

func TestCheckMiniSWEInstalled(t *testing.T) {
	// Should fail with a non-existent python
	err := CheckMiniSWEInstalled("/nonexistent/python")
	assert.Error(t, err)
}
