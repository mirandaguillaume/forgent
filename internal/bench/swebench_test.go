package bench

import (
	"os"
	"path/filepath"
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

// --- Mutation-killing tests for extractPatch (line 162 boundary conditions) ---

func TestExtractPatch_EndBeforeStart(t *testing.T) {
	// If end <= start, should fall through to fallback.
	// Here </patch> appears before <patch>.
	response := "</patch>some text<patch>"
	patch := extractPatch(response)
	// The fallback checks for "diff --git" or "---". Neither is present.
	assert.Empty(t, patch, "end before start with no fallback content should return empty")
}

func TestExtractPatch_EndEqualsStart(t *testing.T) {
	// Mutation: `end <= start` → `end < start`. When end == start, should still fail.
	// Construct a case where start and end index are equal (both tags at same position is
	// not possible, but we can test end == start by having </patch> immediately after <patch>
	// starts). Actually, Index finds first occurrence, so we just need end <= start.
	// "</patch><patch>" — end = 0, start = 8, so end < start → fallback.
	response := "</patch><patch>"
	patch := extractPatch(response)
	assert.Empty(t, patch, "end before start should use fallback, no diff content → empty")
}

func TestExtractPatch_FallbackWithTripleDash(t *testing.T) {
	// Fallback: no <patch> tags but contains "---".
	// Mutation might change `||` to `&&` or negate the check.
	response := "Some explanation\n--- a/file.py\n+++ b/file.py\n@@ -1 +1 @@\n-old\n+new"
	patch := extractPatch(response)
	assert.Equal(t, response, patch, "fallback should return full response when --- is present")
}

func TestExtractPatch_FallbackWithDiffGit(t *testing.T) {
	response := "diff --git a/file.py b/file.py\n@@ -1 +1 @@\n-old\n+new"
	patch := extractPatch(response)
	assert.Equal(t, response, patch, "fallback should return full response when diff --git is present")
}

func TestExtractPatch_NoTagsNoDiff(t *testing.T) {
	// Neither tags nor diff markers → empty.
	response := "I cannot generate a fix for this issue."
	patch := extractPatch(response)
	assert.Empty(t, patch)
}

func TestExtractPatch_TagsWithEmptyContent(t *testing.T) {
	response := "<patch></patch>"
	patch := extractPatch(response)
	assert.Empty(t, patch, "empty patch between tags should produce empty string after TrimSpace")
}

func TestExtractPatch_TagsWithWhitespaceOnly(t *testing.T) {
	response := "<patch>   \n  \n  </patch>"
	patch := extractPatch(response)
	assert.Empty(t, patch, "whitespace-only patch should be trimmed to empty")
}

func TestExtractPatch_ValidTagsExtractsContent(t *testing.T) {
	response := "Here: <patch>--- a/f.py\n+++ b/f.py\n-old\n+new</patch> done"
	patch := extractPatch(response)
	assert.Equal(t, "--- a/f.py\n+++ b/f.py\n-old\n+new", patch)
}

func TestExtractPatch_StartEqualsZero(t *testing.T) {
	// <patch> at the very beginning.
	response := "<patch>content</patch>"
	patch := extractPatch(response)
	assert.Equal(t, "content", patch)
}

func TestExtractPatch_OnlyOpenTag(t *testing.T) {
	// start >= 0 but end < 0 → fallback.
	response := "<patch>some content but no closing tag"
	patch := extractPatch(response)
	assert.Empty(t, patch, "no closing tag and no diff markers → empty")
}

func TestExtractPatch_OnlyCloseTag(t *testing.T) {
	// start < 0 → fallback.
	response := "some content</patch>"
	patch := extractPatch(response)
	assert.Empty(t, patch, "no opening tag and no diff markers → empty")
}

func TestLoadSWEBenchTasks_SkipsEmptyLines(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "tasks.jsonl")
	content := `{"instance_id":"t1","problem_statement":"fix","repo":"o/r","base_commit":"abc"}

{"instance_id":"t2","problem_statement":"fix2","repo":"o/r2","base_commit":"def"}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))
	tasks, err := LoadSWEBenchTasks(tmpFile)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestLoadSWEBenchTasks_InvalidJSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "tasks.jsonl")
	require.NoError(t, os.WriteFile(tmpFile, []byte("not json\n"), 0644))
	_, err := LoadSWEBenchTasks(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse task")
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
