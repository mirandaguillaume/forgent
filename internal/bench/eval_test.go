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

func TestLoadEvalTasks_MissingFile(t *testing.T) {
	_, err := LoadEvalTasks("/nonexistent/path.yaml")
	require.Error(t, err)
}

func TestLoadEvalTasks_AllTasksHaveRequiredFields(t *testing.T) {
	tasks, err := LoadEvalTasks("testdata/eval_tasks.yaml")
	require.NoError(t, err)
	for _, task := range tasks {
		assert.NotEmpty(t, task.ID, "task must have an ID")
		assert.NotEmpty(t, task.Input, "task %q must have input", task.ID)
		assert.NotEmpty(t, task.ExpectedCriteria, "task %q must have criteria", task.ID)
	}
}

// --- JSON extraction tests ---

func TestParseJudgeResponse_CleanJSON(t *testing.T) {
	resp, err := parseJudgeResponse(`{"composed_score": 85, "monolithic_score": 70, "reasoning": "better coverage"}`)
	require.NoError(t, err)
	assert.Equal(t, 85, resp.ComposedScore)
	assert.Equal(t, 70, resp.MonolithicScore)
	assert.Equal(t, "better coverage", resp.Reasoning)
}

func TestParseJudgeResponse_MarkdownWrapped(t *testing.T) {
	input := "```json\n{\"composed_score\": 90, \"monolithic_score\": 60, \"reasoning\": \"found more bugs\"}\n```"
	resp, err := parseJudgeResponse(input)
	require.NoError(t, err)
	assert.Equal(t, 90, resp.ComposedScore)
	assert.Equal(t, 60, resp.MonolithicScore)
}

func TestParseJudgeResponse_WithPreamble(t *testing.T) {
	input := "Here is my evaluation:\n{\"composed_score\": 75, \"monolithic_score\": 75, \"reasoning\": \"tie\"}"
	resp, err := parseJudgeResponse(input)
	require.NoError(t, err)
	assert.Equal(t, 75, resp.ComposedScore)
	assert.Equal(t, 75, resp.MonolithicScore)
}

func TestParseJudgeResponse_InvalidJSON(t *testing.T) {
	_, err := parseJudgeResponse("this is not json at all")
	require.Error(t, err)
}

func TestParseJudgeResponse_Empty(t *testing.T) {
	_, err := parseJudgeResponse("")
	require.Error(t, err)
}

func TestParseJudgeResponse_PartialJSON(t *testing.T) {
	_, err := parseJudgeResponse(`{"composed_score": 85, "monolithic_score":`)
	require.Error(t, err)
}

// --- LLM-gated test ---

func TestEval_WithLLM(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	tasks, err := LoadEvalTasks("testdata/eval_tasks.yaml")
	require.NoError(t, err)

	tmpDir := t.TempDir()
	err = buildToDir("../../skills", "../../agents", tmpDir, "claude")
	require.NoError(t, err)

	composedPrompt, err := readAllFiles(tmpDir)
	require.NoError(t, err)

	monolithicPrompt := `You are a code reviewer. Review the code for bugs, security issues,
performance problems, and best practice violations. Be thorough and specific.`

	provider, err := llm.GetProvider("anthropic", apiKey)
	require.NoError(t, err)

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
