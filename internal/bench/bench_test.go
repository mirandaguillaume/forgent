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
