package bench

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMartianTasks_LoadsAllPRs(t *testing.T) {
	tasks, err := MartianTasks("testdata/martian")
	require.NoError(t, err)
	assert.Equal(t, 50, len(tasks), "should load 50 PRs")
}

func TestMartianTasks_AllHaveCriteria(t *testing.T) {
	tasks, err := MartianTasks("testdata/martian")
	require.NoError(t, err)
	for _, task := range tasks {
		assert.NotEmpty(t, task.Criteria, "task %s should have criteria", task.ID)
		assert.NotEmpty(t, task.Category, "task %s should have category", task.ID)
		assert.NotEmpty(t, task.Title, "task %s should have title", task.ID)
		assert.NotEmpty(t, task.URL, "task %s should have URL", task.ID)
	}
}

func TestMartianTasks_Categories(t *testing.T) {
	tasks, err := MartianTasks("testdata/martian")
	require.NoError(t, err)
	categories := make(map[string]int)
	for _, task := range tasks {
		categories[task.Category]++
	}
	assert.Equal(t, 10, categories["grafana"])
	assert.Equal(t, 10, categories["sentry"])
	assert.Equal(t, 10, categories["cal_dot_com"])
	assert.Equal(t, 10, categories["discourse"])
	assert.Equal(t, 10, categories["keycloak"])
}

func TestMartianTasks_UniqueIDs(t *testing.T) {
	tasks, err := MartianTasks("testdata/martian")
	require.NoError(t, err)
	seen := make(map[string]bool)
	for _, task := range tasks {
		assert.False(t, seen[task.ID], "duplicate ID: %s", task.ID)
		seen[task.ID] = true
	}
}

func TestMartianTasks_CriteriaCount(t *testing.T) {
	tasks, err := MartianTasks("testdata/martian")
	require.NoError(t, err)
	total := 0
	for _, task := range tasks {
		total += len(task.Criteria)
	}
	assert.Equal(t, 137, total, "should have 137 total criteria")
}

func TestMartianTasks_SeveritiesPropagated(t *testing.T) {
	tasks, err := MartianTasks("testdata/martian")
	require.NoError(t, err)

	validSeverities := map[string]bool{"Critical": true, "High": true, "Medium": true, "Low": true}
	for _, task := range tasks {
		assert.Equal(t, len(task.Criteria), len(task.Severities),
			"task %s: Severities length should match Criteria length", task.ID)
		for _, sev := range task.Severities {
			assert.True(t, validSeverities[sev],
				"task %s: unknown severity %q", task.ID, sev)
		}
	}
}

func TestMartianTasks_MissingDir(t *testing.T) {
	_, err := MartianTasks("/nonexistent")
	require.Error(t, err)
}

func TestParsePRURL_Standard(t *testing.T) {
	owner, repo, number, err := ParsePRURL("https://github.com/grafana/grafana/pull/79265")
	require.NoError(t, err)
	assert.Equal(t, "grafana", owner)
	assert.Equal(t, "grafana", repo)
	assert.Equal(t, 79265, number)
}

func TestParsePRURL_Forked(t *testing.T) {
	owner, repo, number, err := ParsePRURL("https://github.com/ai-code-review-evaluation/sentry-greptile/pull/1")
	require.NoError(t, err)
	assert.Equal(t, "ai-code-review-evaluation", owner)
	assert.Equal(t, "sentry-greptile", repo)
	assert.Equal(t, 1, number)
}

func TestParsePRURL_TrailingSlash(t *testing.T) {
	owner, repo, number, err := ParsePRURL("https://github.com/calcom/cal.com/pull/8087/")
	require.NoError(t, err)
	assert.Equal(t, "calcom", owner)
	assert.Equal(t, "cal.com", repo)
	assert.Equal(t, 8087, number)
}

func TestParsePRURL_Invalid(t *testing.T) {
	_, _, _, err := ParsePRURL("not-a-url")
	require.Error(t, err)
}

func TestParsePRURL_NoPullSegment(t *testing.T) {
	_, _, _, err := ParsePRURL("https://github.com/grafana/grafana/issues/79265")
	require.Error(t, err)
}

func TestBuildReviewPrompt_WithCode(t *testing.T) {
	task := H2HTask{Code: "func foo() {}", Title: "test"}
	prompt := BuildReviewPrompt("You review code.", task)
	assert.Contains(t, prompt, "func foo() {}")
	assert.Contains(t, prompt, "You review code.")
}

func TestBuildReviewPrompt_WithURL(t *testing.T) {
	task := H2HTask{URL: "https://github.com/grafana/grafana/pull/79265", Title: "Add device limit"}
	prompt := BuildReviewPrompt("You review code.", task)
	assert.Contains(t, prompt, "You review code.")
	// If gh succeeds, prompt contains the diff; if it fails, prompt contains the title as fallback
	assert.True(t, strings.Contains(prompt, "diff --git") || strings.Contains(prompt, "Add device limit"),
		"prompt should contain either the fetched diff or the fallback title")
}

func TestBuildReviewPrompt_PreferCode(t *testing.T) {
	task := H2HTask{Code: "inline code", URL: "https://github.com/x/y/pull/1", Title: "test"}
	prompt := BuildReviewPrompt("System prompt.", task)
	assert.Contains(t, prompt, "inline code")
}
