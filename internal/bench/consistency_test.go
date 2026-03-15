package bench

import (
	"os"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsistency_Normalization(t *testing.T) {
	// Unit test for normalization logic — no LLM needed.
	responses := []string{
		"Hello  World\n",
		"hello world",
		"HELLO   WORLD  ",
	}
	assert.Equal(t, 1, countUnique(responses), "should normalize to same string")
}

func TestConsistency_DifferentResponses(t *testing.T) {
	responses := []string{
		"The code has a bug in line 5",
		"There is a security issue",
		"No issues found",
	}
	assert.Equal(t, 3, countUnique(responses))
}

func TestConsistency_WithLLM(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	tasks, err := LoadEvalTasks("testdata/eval_tasks.yaml")
	require.NoError(t, err)
	require.Greater(t, len(tasks), 0)

	provider, err := llm.GetProvider("anthropic", apiKey)
	require.NoError(t, err)

	prompt := "You are a code reviewer. Review the code for bugs and security issues."

	result, err := RunConsistency(tasks[0], prompt, provider, 3)
	require.NoError(t, err)

	assert.Equal(t, 3, result.Runs)
	assert.GreaterOrEqual(t, result.ConsistencyRate, 0.0)
	assert.LessOrEqual(t, result.ConsistencyRate, 1.0)
	t.Logf("Task: %s, Unique: %d/%d, Consistency: %.2f, AvgLen: %d",
		result.Task, result.UniqueResponses, result.Runs, result.ConsistencyRate, result.AvgResponseLen)
}
