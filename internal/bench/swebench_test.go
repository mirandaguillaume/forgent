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

// --- extractAndConvertPatch tests ---

func TestExtractPatch_UnifiedDiffInPatchTags(t *testing.T) {
	response := `Here is the fix:
<patch>
--- a/foo.py
+++ b/foo.py
@@ -1,3 +1,3 @@
-old line
+new line
</patch>
Done.`
	patch := extractAndConvertPatch(response, t.TempDir())
	assert.Contains(t, patch, "--- a/foo.py")
	assert.Contains(t, patch, "+new line")
}

func TestExtractPatch_NoPatch(t *testing.T) {
	response := "I don't know how to fix this."
	patch := extractAndConvertPatch(response, t.TempDir())
	assert.Empty(t, patch)
}

func TestExtractPatch_RawDiff(t *testing.T) {
	response := `diff --git a/foo.py b/foo.py
--- a/foo.py
+++ b/foo.py
@@ -1 +1 @@
-old
+new`
	patch := extractAndConvertPatch(response, t.TempDir())
	assert.NotEmpty(t, patch)
}

func TestExtractPatch_RawTripleDash(t *testing.T) {
	response := "Some explanation\n--- a/file.py\n+++ b/file.py\n@@ -1 +1 @@\n-old\n+new"
	patch := extractAndConvertPatch(response, t.TempDir())
	assert.NotEmpty(t, patch)
	assert.Contains(t, patch, "--- a/file.py")
}

func TestExtractPatch_NoTagsNoDiff(t *testing.T) {
	response := "I cannot generate a fix for this issue."
	patch := extractAndConvertPatch(response, t.TempDir())
	assert.Empty(t, patch)
}

func TestExtractPatch_EmptyPatchTags(t *testing.T) {
	response := "<patch></patch>"
	patch := extractAndConvertPatch(response, t.TempDir())
	assert.Empty(t, patch)
}

// --- extractTagContent tests ---

func TestExtractTagContent(t *testing.T) {
	assert.Equal(t, "content", extractTagContent("<tag>content</tag>", "tag"))
	assert.Equal(t, "", extractTagContent("no tags here", "tag"))
	assert.Equal(t, "", extractTagContent("</tag><tag>", "tag"))
	assert.Equal(t, "spaced", extractTagContent("<x>  spaced  </x>", "x"))
}

// --- extractFilesFromPatch tests ---

func TestExtractFilesFromPatch(t *testing.T) {
	patch := `diff --git a/foo.py b/foo.py
--- a/foo.py
+++ b/foo.py
@@ -1 +1 @@
-old
+new
diff --git a/bar.py b/bar.py
--- a/bar.py
+++ b/bar.py
@@ -5 +5 @@
-x
+y`
	files := extractFilesFromPatch(patch)
	assert.Equal(t, []string{"foo.py", "bar.py"}, files)
}

func TestExtractFilesFromPatch_NoPatch(t *testing.T) {
	files := extractFilesFromPatch("")
	assert.Empty(t, files)
}

func TestExtractFilesFromPatch_DevNull(t *testing.T) {
	patch := "--- /dev/null\n+++ b/new_file.py"
	files := extractFilesFromPatch(patch)
	assert.Equal(t, []string{"new_file.py"}, files)
}

// --- buildSWEPrompt tests ---

func TestBuildSWEPrompt_WithPlaceholders(t *testing.T) {
	template := "Fix this:\n{problem_statement}\n\nCode:\n{content}"
	result := buildSWEPrompt(template, "the bug", "the code", "owner/repo")
	assert.Contains(t, result, "the bug")
	assert.Contains(t, result, "the code")
	assert.NotContains(t, result, "{problem_statement}")
	assert.NotContains(t, result, "{content}")
}

func TestBuildSWEPrompt_NoPlaceholders(t *testing.T) {
	template := "You are a helpful assistant."
	result := buildSWEPrompt(template, "the bug", "the code", "owner/repo")
	assert.Contains(t, result, "You are a helpful assistant.")
	assert.Contains(t, result, "the bug")
	assert.Contains(t, result, "the code")
	assert.Contains(t, result, "owner/repo")
}

// --- SEARCH/REPLACE conversion tests ---

func TestSearchReplaceToUDiff(t *testing.T) {
	// Create a temp repo with a file
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "foo.py"), []byte("line1\nline2\nline3\n"), 0644))

	content := `### foo.py
<<<<<<< SEARCH
line2
=======
line2_fixed
>>>>>>> REPLACE`

	udiff := searchReplaceToUDiff(content, repoDir)
	assert.NotEmpty(t, udiff)
	assert.Contains(t, udiff, "--- a/foo.py")
	assert.Contains(t, udiff, "+++ b/foo.py")
	assert.Contains(t, udiff, "-line2")
	assert.Contains(t, udiff, "+line2_fixed")
}

func TestSearchReplaceToUDiff_NoSearchReplace(t *testing.T) {
	udiff := searchReplaceToUDiff("no search replace here", t.TempDir())
	assert.Empty(t, udiff)
}

func TestSearchReplaceToUDiff_BareFilePath(t *testing.T) {
	repoDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "src", "main.py"), []byte("old\n"), 0644))

	content := `src/main.py
<<<<<<< SEARCH
old
=======
new
>>>>>>> REPLACE`

	udiff := searchReplaceToUDiff(content, repoDir)
	assert.NotEmpty(t, udiff)
	assert.Contains(t, udiff, "--- a/src/main.py")
}

// --- ACR format conversion tests ---

func TestACRToUDiff(t *testing.T) {
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "foo.py"), []byte("line1\nline2\nline3\n"), 0644))

	content := `# modification 1
<file>foo.py</file>
<original>line2</original>
<patched>line2_fixed</patched>`

	udiff := acrToUDiff(content, repoDir)
	assert.NotEmpty(t, udiff)
	assert.Contains(t, udiff, "--- a/foo.py")
	assert.Contains(t, udiff, "-line2")
	assert.Contains(t, udiff, "+line2_fixed")
}

func TestACRToUDiff_NoACRFormat(t *testing.T) {
	udiff := acrToUDiff("no acr format here", t.TempDir())
	assert.Empty(t, udiff)
}

// --- buildCodeContext tests ---

func TestBuildCodeContext(t *testing.T) {
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "foo.py"), []byte("def foo():\n    pass\n"), 0644))

	goldPatch := "--- a/foo.py\n+++ b/foo.py\n@@ -1 +1 @@\n-old\n+new"
	ctx := buildCodeContext(goldPatch, repoDir)
	assert.Contains(t, ctx, "### foo.py")
	assert.Contains(t, ctx, "def foo():")
}

func TestBuildCodeContext_EmptyPatch(t *testing.T) {
	ctx := buildCodeContext("", t.TempDir())
	assert.Empty(t, ctx)
}

// --- isUnifiedDiff tests ---

func TestIsUnifiedDiff(t *testing.T) {
	assert.True(t, isUnifiedDiff("diff --git a/f b/f"))
	assert.True(t, isUnifiedDiff("--- a/f\n+++ b/f"))
	assert.False(t, isUnifiedDiff("no diff here"))
}

// --- Skill chain tests ---

func TestLoadSkillChain(t *testing.T) {
	dir := t.TempDir()

	// Create skills
	for _, name := range []string{"analyze-issue", "locate-code", "generate-patch"} {
		skillDir := filepath.Join(dir, "skills", name)
		require.NoError(t, os.MkdirAll(skillDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"),
			[]byte("You are the "+name+" skill."), 0644))
	}

	// Create agent with skill order
	agentsDir := filepath.Join(dir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))
	agentMd := "## Execution\n### Step 1: Analyze Issue\n### Step 2: Locate Code\n### Step 3: Generate Patch\n"
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "fixer.md"), []byte(agentMd), 0644))

	chain, totalWords, err := loadSkillChain(dir)
	require.NoError(t, err)
	assert.Len(t, chain, 3)
	assert.Equal(t, "analyze-issue", chain[0].Name)
	assert.Equal(t, "locate-code", chain[1].Name)
	assert.Equal(t, "generate-patch", chain[2].Name)
	assert.Greater(t, totalWords, 0)
}

func TestLoadSkillChain_FallbackAlphabetical(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")

	for _, name := range []string{"alpha", "beta"} {
		d := filepath.Join(skillsDir, name)
		require.NoError(t, os.MkdirAll(d, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(d, "SKILL.md"), []byte("skill "+name), 0644))
	}

	chain, _, err := loadSkillChain(dir)
	require.NoError(t, err)
	assert.Len(t, chain, 2)
	assert.Equal(t, "alpha", chain[0].Name)
	assert.Equal(t, "beta", chain[1].Name)
}

// --- Integration / existing tests ---

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

func TestLoadSWEBenchTasks_WithGoldPatch(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "tasks.jsonl")
	content := `{"instance_id":"t1","problem_statement":"fix","repo":"o/r","base_commit":"abc","patch":"--- a/f.py\n+++ b/f.py\n-old\n+new","hints_text":"check the validator"}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))
	tasks, err := LoadSWEBenchTasks(tmpFile)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Contains(t, tasks[0].GoldPatch, "--- a/f.py")
	assert.Equal(t, "check the validator", tasks[0].HintsText)
}

func TestLoadSWEContestants(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-a.md"), []byte("You are a bug fixer."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-b.md"), []byte("Fix bugs carefully."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0644))

	contestants, err := LoadSWEContestants(dir)
	require.NoError(t, err)
	assert.Len(t, contestants, 2)
	assert.Equal(t, "agent-a", contestants[0].Name)
	assert.Equal(t, "agent-b", contestants[1].Name)
	assert.Greater(t, contestants[0].Words, 0)
}

func TestLoadSWEContestants_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	contestants, err := LoadSWEContestants(dir)
	require.NoError(t, err)
	assert.Empty(t, contestants)
}

func TestLoadSWEContestants_MissingDir(t *testing.T) {
	_, err := LoadSWEContestants("/nonexistent/dir")
	require.Error(t, err)
}

func TestLoadSWEContestants_RealFixtures(t *testing.T) {
	dir := "fixtures/swebench-agents"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("swebench-agents fixtures not found")
	}
	contestants, err := LoadSWEContestants(dir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(contestants), 4, "should have at least 4 SWE-bench agents")
	for _, c := range contestants {
		assert.Greater(t, c.Words, 0, "contestant %s should have words", c.Name)
		t.Logf("  %s: %d words", c.Name, c.Words)
	}
}

func TestLoadSWEForgentContestants(t *testing.T) {
	dir := t.TempDir()

	// Standard variant: skill-chain layout (skills/ + agents/ subdirs)
	stdDir := filepath.Join(dir, "agent-a", "output-standard")
	stdSkillDir := filepath.Join(stdDir, "skills", "analyze-issue")
	stdAgentsDir := filepath.Join(stdDir, "agents")
	require.NoError(t, os.MkdirAll(stdSkillDir, 0755))
	require.NoError(t, os.MkdirAll(stdAgentsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(stdSkillDir, "SKILL.md"), []byte("Analyze the issue carefully."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(stdAgentsDir, "agent.md"), []byte("### Step 1: Analyze Issue\nRead skill."), 0644))

	// Compact variant: single file
	compDir := filepath.Join(dir, "agent-a", "output-compact")
	require.NoError(t, os.MkdirAll(compDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(compDir, "agent.md"), []byte("compact"), 0644))

	contestants, err := LoadSWEForgentContestants(dir)
	require.NoError(t, err)
	assert.Len(t, contestants, 2)
	assert.Equal(t, "agent-a/forgent-standard", contestants[0].Name)
	assert.NotNil(t, contestants[0].SkillChain)
	assert.Len(t, contestants[0].SkillChain, 1)
	assert.Equal(t, "analyze-issue", contestants[0].SkillChain[0].Name)
	assert.Equal(t, "agent-a/forgent-compact", contestants[1].Name)
	assert.Nil(t, contestants[1].SkillChain)
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
