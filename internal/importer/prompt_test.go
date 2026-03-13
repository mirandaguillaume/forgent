package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildImportPrompt(t *testing.T) {
	source := Source{
		Name:    "review.md",
		Content: "# Review Agent\nReviews PRs",
	}
	fm := AgentFrontmatter{
		Name:        "review",
		Description: "Reviews PRs",
		Tools:       []string{"Read", "Grep"},
	}

	prompt := BuildImportPrompt(source, fm, "# Review Agent\nReviews PRs", []string{"read_file", "grep"})

	assert.Contains(t, prompt, "Forgent skill YAML schema")
	assert.Contains(t, prompt, "review")
	assert.Contains(t, prompt, "Reviews PRs")
	assert.Contains(t, prompt, `"skills"`)
	assert.Contains(t, prompt, "consumes")
	assert.Contains(t, prompt, "produces")
	assert.Contains(t, prompt, "read_file, grep")
}

func TestBuildImportPrompt_NoFrontmatter(t *testing.T) {
	source := Source{Name: "agent.md"}
	fm := AgentFrontmatter{}

	prompt := BuildImportPrompt(source, fm, "# My Agent", nil)

	assert.Contains(t, prompt, "# My Agent")
	assert.NotContains(t, prompt, "Source file:")
}

func TestBuildRetryPrompt(t *testing.T) {
	feedback := []string{
		"skill 'review': missing guardrails.timeout",
		"skill 'review': score 45/100 (below 60 threshold)",
	}
	prompt := BuildRetryPrompt("original prompt", "original response", feedback)

	assert.Contains(t, prompt, "original prompt")
	assert.Contains(t, prompt, "original response")
	assert.Contains(t, prompt, "missing guardrails.timeout")
	assert.Contains(t, prompt, "score 45/100")
	assert.Contains(t, prompt, "Fix these issues")
}
