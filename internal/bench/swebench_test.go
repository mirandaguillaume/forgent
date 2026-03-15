package bench

import (
	"os"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSWEBenchTasks(t *testing.T) {
	tasks, err := LoadSWEBenchTasks("testdata/swebench_sample.jsonl")
	require.NoError(t, err)
	assert.Len(t, tasks, 10)
	assert.Equal(t, "astropy__astropy-12907", tasks[0].InstanceID)
	assert.Equal(t, "astropy/astropy", tasks[0].Repo)
	assert.NotEmpty(t, tasks[0].ProblemStatement)
	assert.NotEmpty(t, tasks[0].BaseCommit)
}

func TestLoadSWEBenchTasks_MissingFile(t *testing.T) {
	_, err := LoadSWEBenchTasks("/nonexistent/tasks.jsonl")
	require.Error(t, err)
}

func TestExtractPatch_WithTags(t *testing.T) {
	response := `Here is the fix:
<patch>
--- a/foo.py
+++ b/foo.py
@@ -1,3 +1,3 @@
-old line
+new line
</patch>
Done.`
	patch := extractPatch(response)
	assert.Contains(t, patch, "--- a/foo.py")
	assert.Contains(t, patch, "+new line")
}

func TestExtractPatch_NoPatch(t *testing.T) {
	response := "I don't know how to fix this."
	patch := extractPatch(response)
	assert.Empty(t, patch)
}

func TestExtractPatch_RawDiff(t *testing.T) {
	response := `diff --git a/foo.py b/foo.py
--- a/foo.py
+++ b/foo.py
@@ -1 +1 @@
-old
+new`
	patch := extractPatch(response)
	assert.NotEmpty(t, patch)
}

func TestSWEBench_WithLLM(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	if testing.Short() {
		t.Skip("skipping SWE-bench in short mode")
	}

	provider, err := llm.GetProvider("anthropic", apiKey)
	require.NoError(t, err)

	// Build composed prompt
	tmpDir := t.TempDir()
	err = buildToDir("../../skills", "../../agents", tmpDir, "claude")
	require.NoError(t, err)

	composedPrompt, err := readAllFiles(tmpDir)
	require.NoError(t, err)

	// Run with only first task to keep costs down
	result, err := RunSWEBench("testdata/swebench_sample.jsonl", composedPrompt, provider)
	require.NoError(t, err)

	assert.Equal(t, 10, result.Tasks)
	for _, d := range result.Details {
		t.Logf("Task %s: Applied=%v, Error=%s", d.InstanceID, d.Applied, d.Error)
	}
	t.Logf("Resolved: %d/%d (%.1f%%)", result.Resolved, result.Tasks, result.Rate)
}
