package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEvalTasks(t *testing.T) {
	tasks, err := LoadEvalTasks("testdata/eval_tasks.yaml")
	require.NoError(t, err)
	assert.Len(t, tasks, 5)
	assert.Equal(t, "missing-error-handling", tasks[0].ID)
	assert.Contains(t, tasks[0].ExpectedCriteria, "error handling")
}

func TestEval_WithLLM(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	tasks, err := LoadEvalTasks("testdata/eval_tasks.yaml")
	require.NoError(t, err)

	// Build composed prompt from actual generated artifacts
	tmpDir := t.TempDir()
	err = buildToDir("../../skills", "../../agents", tmpDir, "claude")
	require.NoError(t, err)

	composedPrompt, err := readAllFiles(tmpDir)
	require.NoError(t, err)

	monolithicPrompt := `You are a code reviewer. Review the code for bugs, security issues,
performance problems, and best practice violations. Be thorough and specific.`

	provider, err := llm.GetProvider("anthropic", apiKey)
	require.NoError(t, err)

	// Run with first 2 tasks only (to keep API costs down)
	result, err := RunEval(tasks[:2], composedPrompt, monolithicPrompt, provider)
	require.NoError(t, err)

	assert.Equal(t, 2, result.Tasks)
	for _, r := range result.Results {
		assert.GreaterOrEqual(t, r.ComposedScore, 0)
		assert.LessOrEqual(t, r.ComposedScore, 100)
		assert.GreaterOrEqual(t, r.MonolithicScore, 0)
		assert.LessOrEqual(t, r.MonolithicScore, 100)
		t.Logf("Task %s: Composed=%d, Monolithic=%d, Wins=%v",
			r.TaskID, r.ComposedScore, r.MonolithicScore, r.ComposedWins)
	}
	t.Logf("Win rate: %.1f%% (%d/%d wins, %d ties)",
		result.WinRate, result.ComposedWins, result.Tasks, result.Ties)
}

// readAllFiles concatenates all files in a directory tree.
func readAllFiles(dir string) (string, error) {
	var parts []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		parts = append(parts, string(data))
		return nil
	})
	if err != nil {
		return "", err
	}
	return joinStrings(parts, "\n\n---\n\n"), nil
}

func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
