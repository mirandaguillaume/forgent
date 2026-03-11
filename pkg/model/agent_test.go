package model_test

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAgentCompositionYAMLParsing(t *testing.T) {
	input := `
agent: review-bot
skills:
  - code-review
  - lint
orchestration: parallel-then-merge
description: Reviews PRs with linting and code analysis
`
	var ac model.AgentComposition
	err := yaml.Unmarshal([]byte(input), &ac)
	require.NoError(t, err)

	assert.Equal(t, "review-bot", ac.Agent)
	assert.Equal(t, []string{"code-review", "lint"}, ac.Skills)
	assert.Equal(t, model.OrchestrationParallelThenMerge, ac.Orchestration)
	assert.Equal(t, "Reviews PRs with linting and code analysis", ac.Description)
}

func TestAgentCompositionOptionalDescription(t *testing.T) {
	input := `
agent: simple-bot
skills:
  - greet
orchestration: sequential
`
	var ac model.AgentComposition
	err := yaml.Unmarshal([]byte(input), &ac)
	require.NoError(t, err)

	assert.Equal(t, "simple-bot", ac.Agent)
	assert.Equal(t, []string{"greet"}, ac.Skills)
	assert.Equal(t, model.OrchestrationSequential, ac.Orchestration)
	assert.Empty(t, ac.Description)
}

func TestOrchestrationStrategyConstants(t *testing.T) {
	assert.Equal(t, model.OrchestrationStrategy("sequential"), model.OrchestrationSequential)
	assert.Equal(t, model.OrchestrationStrategy("parallel"), model.OrchestrationParallel)
	assert.Equal(t, model.OrchestrationStrategy("parallel-then-merge"), model.OrchestrationParallelThenMerge)
	assert.Equal(t, model.OrchestrationStrategy("adaptive"), model.OrchestrationAdaptive)
}
