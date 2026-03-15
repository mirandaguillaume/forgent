package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func TestIsReachable_ExactMatch(t *testing.T) {
	structure := []scanner.DirEntry{
		{Path: "src/controllers", Files: []string{"auth.go"}},
	}
	assert.True(t, isReachable("src/controllers/auth.go", structure))
}

func TestIsReachable_PrefixMatch(t *testing.T) {
	structure := []scanner.DirEntry{
		{Path: "apps/bo", Files: []string{"components/", "hooks/"}},
	}
	// File is deeper than entry — entry is prefix of file's dir.
	assert.True(t, isReachable("apps/bo/hooks/useAuth.ts", structure))
}

func TestIsReachable_ReversePrefix(t *testing.T) {
	structure := []scanner.DirEntry{
		{Path: "apps/bo/common/hooks", Files: []string{"useAuth.ts"}},
	}
	// File's dir is shallower than entry — but entry starts with file's parent.
	assert.True(t, isReachable("apps/bo/common/utils.ts", structure))
}

func TestIsReachable_NoMatch(t *testing.T) {
	structure := []scanner.DirEntry{
		{Path: "src/controllers", Files: []string{"auth.go"}},
	}
	assert.False(t, isReachable("lib/utils/helper.go", structure))
}

func TestRunProxy_SmallProject(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "main.go"), "package main\n")
	writeFile(t, filepath.Join(root, "cmd/root.go"), "package cmd\n")
	writeFile(t, filepath.Join(root, "pkg/model/skill.go"), "package model\n")
	writeFile(t, filepath.Join(root, "pkg/model/agent.go"), "package model\n")

	result, err := RunProxy(root, 100, 42)
	require.NoError(t, err)

	assert.Equal(t, 4, result.TotalSourceFiles)
	assert.Equal(t, 4, result.SampledFiles) // less than 100, returns all
	assert.Equal(t, 100.0, result.Reachability, "all files should be reachable")
	assert.Greater(t, result.IndexEntries, 0)
	assert.Greater(t, result.IndexBytes, 0)
}

func TestRunProxy_Deterministic(t *testing.T) {
	root := t.TempDir()

	// Create enough files to trigger sampling.
	for i := 0; i < 50; i++ {
		writeFile(t, filepath.Join(root, "src", "pkg"+string(rune('a'+i%26)), "main.go"), "package main\n")
	}

	r1, err := RunProxy(root, 10, 42)
	require.NoError(t, err)

	r2, err := RunProxy(root, 10, 42)
	require.NoError(t, err)

	assert.Equal(t, r1.ReachableFiles, r2.ReachableFiles, "same seed should give same result")
	assert.Equal(t, r1.SampledFiles, r2.SampledFiles)
}

func TestCollectSourceFiles(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "main.go"), "package main\n")
	writeFile(t, filepath.Join(root, "README.md"), "# Hello\n")
	writeFile(t, filepath.Join(root, ".git/config"), "gitconfig\n")
	writeFile(t, filepath.Join(root, "node_modules/pkg/index.js"), "module.exports = {}\n")
	writeFile(t, filepath.Join(root, "src/app.ts"), "export {}\n")

	files, err := collectSourceFiles(root)
	require.NoError(t, err)

	assert.Len(t, files, 2) // main.go + src/app.ts
	assert.NotContains(t, files, "README.md")
	assert.NotContains(t, files, ".git/config")
}

func TestSampleFiles(t *testing.T) {
	files := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

	sampled := sampleFiles(files, 3, 42)
	assert.Len(t, sampled, 3)

	// All returned should be unique (no duplicates).
	seen := map[string]bool{}
	for _, f := range sampled {
		assert.False(t, seen[f], "duplicate in sample")
		seen[f] = true
	}
}

// --- Mutation-killing tests for sampleFiles boundary (line 99: len(files) <= n) ---

func TestSampleFiles_ExactlyN(t *testing.T) {
	// Line 99: `len(files) <= n` — mutation to `< n` would trigger shuffle when len == n.
	// With len == n, should return all files unshuffled.
	files := []string{"a", "b", "c"}
	original := make([]string, len(files))
	copy(original, files)
	sampled := sampleFiles(files, 3, 42)
	assert.Len(t, sampled, 3)
	// When len == n, we return the original slice, so order should be preserved.
	assert.Equal(t, original, sampled, "len(files) == n should return original slice without shuffle")
}

func TestSampleFiles_LessThanN(t *testing.T) {
	files := []string{"a", "b"}
	sampled := sampleFiles(files, 5, 42)
	assert.Len(t, sampled, 2, "should return all files when len < n")
}

func TestSampleFiles_NPlus1(t *testing.T) {
	// len(files) == n+1 should trigger shuffle and return n items.
	files := []string{"a", "b", "c", "d"}
	sampled := sampleFiles(files, 3, 42)
	assert.Len(t, sampled, 3)
}

func TestSampleFiles_Empty(t *testing.T) {
	var files []string
	sampled := sampleFiles(files, 5, 42)
	assert.Empty(t, sampled)
}

// --- Mutation-killing test for safePercent ---

func TestSafePercent_ZeroDenominator(t *testing.T) {
	assert.Equal(t, 0.0, safePercent(0, 0))
	assert.Equal(t, 0.0, safePercent(5, 0))
}

func TestSafePercent_NonZero(t *testing.T) {
	assert.InDelta(t, 50.0, safePercent(1, 2), 0.001)
	assert.InDelta(t, 100.0, safePercent(3, 3), 0.001)
}

func TestSafePercent_ZeroNumerator(t *testing.T) {
	assert.Equal(t, 0.0, safePercent(0, 10))
}

func TestAutoGenerateTasks(t *testing.T) {
	ctx := &scanner.CodebaseContext{
		Structure: []scanner.DirEntry{
			{Path: "src/controllers", Files: []string{"AuthController.ts", "UserController.ts"}},
			{Path: "src/services", Files: []string{"AuthService.ts"}},
			{Path: "config", Files: []string{"database.yaml"}},
		},
	}

	tasks := AutoGenerateTasks(ctx)

	assert.Greater(t, len(tasks), 0, "should generate tasks")
	for _, task := range tasks {
		assert.NotEmpty(t, task.Query)
		assert.NotEmpty(t, task.ExpectedPaths)
	}
}

// --- Mutation-killing tests for AutoGenerateTasks boundary (line 170: len(tasks) >= 20) ---

func TestAutoGenerateTasks_CapsAt20(t *testing.T) {
	// Line 170: `len(tasks) >= 20` — mutation to `> 20` would allow 21 tasks.
	// Build a context with more than 20 source files.
	var entries []scanner.DirEntry
	for i := 0; i < 30; i++ {
		entries = append(entries, scanner.DirEntry{
			Path:  "src/pkg" + string(rune('a'+i%26)),
			Files: []string{"main.go"},
		})
	}
	ctx := &scanner.CodebaseContext{Structure: entries}
	tasks := AutoGenerateTasks(ctx)
	assert.Equal(t, 20, len(tasks), "should cap at exactly 20 tasks, not 21")
}

func TestAutoGenerateTasks_Exactly20(t *testing.T) {
	// If we have exactly 20 source files, should get exactly 20 tasks.
	var entries []scanner.DirEntry
	for i := 0; i < 20; i++ {
		entries = append(entries, scanner.DirEntry{
			Path:  "src/pkg" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Files: []string{"file.go"},
		})
	}
	ctx := &scanner.CodebaseContext{Structure: entries}
	tasks := AutoGenerateTasks(ctx)
	assert.Equal(t, 20, len(tasks))
}

func TestAutoGenerateTasks_Exactly19(t *testing.T) {
	// 19 source files should give exactly 19 tasks (below cap).
	var entries []scanner.DirEntry
	for i := 0; i < 19; i++ {
		entries = append(entries, scanner.DirEntry{
			Path:  "src/pkg" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Files: []string{"file.go"},
		})
	}
	ctx := &scanner.CodebaseContext{Structure: entries}
	tasks := AutoGenerateTasks(ctx)
	assert.Equal(t, 19, len(tasks))
}

func TestAutoGenerateTasks_SkipsConfigFiles(t *testing.T) {
	ctx := &scanner.CodebaseContext{
		Structure: []scanner.DirEntry{
			{Path: "config", Files: []string{"database.yaml", "settings.json"}},
		},
	}
	tasks := AutoGenerateTasks(ctx)
	assert.Empty(t, tasks, "config files should be skipped")
}

func TestAutoGenerateTasks_SkipsDirHints(t *testing.T) {
	ctx := &scanner.CodebaseContext{
		Structure: []scanner.DirEntry{
			{Path: "src", Files: []string{"components/", "main.go"}},
		},
	}
	tasks := AutoGenerateTasks(ctx)
	// Only main.go should generate a task (components/ is a dir hint).
	assert.Equal(t, 1, len(tasks))
}

// --- Mutation-killing tests for matchesExpected ---

func TestMatchesExpected_ExactMatch(t *testing.T) {
	assert.True(t, matchesExpected("src/main.go", []string{"src/main.go"}))
}

func TestMatchesExpected_NoMatch(t *testing.T) {
	assert.False(t, matchesExpected("src/main.go", []string{"lib/util.go"}))
}

func TestMatchesExpected_ParentDirMatch(t *testing.T) {
	assert.True(t, matchesExpected("src/controllers", []string{"src/controllers/auth.go"}))
}

func TestMatchesExpected_FileDirMatch(t *testing.T) {
	assert.True(t, matchesExpected("src/controllers/auth.go", []string{"src/controllers"}))
}

func TestMatchesExpected_ReversePrefix(t *testing.T) {
	assert.True(t, matchesExpected("scripts/build.sh", []string{"scripts"}))
}

func TestMatchesExpected_MultiLineResponse(t *testing.T) {
	response := "src/main.go\nlib/util.go"
	assert.True(t, matchesExpected(response, []string{"lib/util.go"}))
}

// --- Mutation-killing tests for extractCandidates ---

func TestExtractCandidates_StripsBackticks(t *testing.T) {
	candidates := extractCandidates("`src/main.go`")
	assert.Contains(t, candidates, "src/main.go")
}

func TestExtractCandidates_StripsDotSlash(t *testing.T) {
	candidates := extractCandidates("./src/main.go")
	assert.Contains(t, candidates, "src/main.go")
}

func TestExtractCandidates_EmptyInput(t *testing.T) {
	candidates := extractCandidates("")
	assert.Empty(t, candidates)
}

func TestExtractCandidates_DotOnly(t *testing.T) {
	candidates := extractCandidates(".")
	assert.Empty(t, candidates, "single dot should be filtered")
}
