package analyzer

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
)

func makeOrderingSkill(name string, consumes, produces []string) model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   name,
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: consumes,
			Produces: produces,
			Memory:   model.MemoryShortTerm,
		},
	}
}

func makeOrderingAgent(skills []string, orchestration model.OrchestrationStrategy, description string) model.AgentComposition {
	return model.AgentComposition{
		Agent:         "test-agent",
		Skills:        skills,
		Orchestration: orchestration,
		Description:   description,
	}
}

// Description order tests

func TestCheckSkillOrdering_DescriptionReversed(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"code-review", "test-runner"},
		model.OrchestrationSequential,
		"First use test-runner, then apply code-review",
	)
	skillMap := map[string]model.SkillBehavior{
		"code-review": makeOrderingSkill("code-review", nil, nil),
		"test-runner": makeOrderingSkill("test-runner", nil, nil),
	}

	issues := CheckSkillOrdering(agent, skillMap)

	hasDescMismatch := false
	for _, issue := range issues {
		if issue.Type == OrderDescriptionMismatch {
			hasDescMismatch = true
			assert.Equal(t, "warning", issue.Severity)
		}
	}
	assert.True(t, hasDescMismatch, "expected description-order-mismatch issue")
}

func TestCheckSkillOrdering_DescriptionMatchesOrder(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"test-runner", "code-review"},
		model.OrchestrationSequential,
		"First run tests, then review the code",
	)
	skillMap := map[string]model.SkillBehavior{
		"test-runner": makeOrderingSkill("test-runner", nil, nil),
		"code-review": makeOrderingSkill("code-review", nil, nil),
	}

	issues := CheckSkillOrdering(agent, skillMap)

	for _, issue := range issues {
		assert.NotEqual(t, OrderDescriptionMismatch, issue.Type)
	}
}

func TestCheckSkillOrdering_ParallelNoIssues(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"code-review", "test-runner"},
		model.OrchestrationParallel,
		"First running tests, then analyzing the diff",
	)

	issues := CheckSkillOrdering(agent, nil)
	assert.Empty(t, issues)
}

func TestCheckSkillOrdering_NoDescription(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"code-review", "test-runner"},
		model.OrchestrationSequential,
		"",
	)

	issues := CheckSkillOrdering(agent, map[string]model.SkillBehavior{})
	assert.Empty(t, issues)
}

func TestCheckSkillOrdering_DescriptionMentionsOneSkill(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"code-review", "test-runner"},
		model.OrchestrationSequential,
		"Focuses on reviewing code",
	)
	skillMap := map[string]model.SkillBehavior{
		"code-review": makeOrderingSkill("code-review", nil, nil),
		"test-runner": makeOrderingSkill("test-runner", nil, nil),
	}

	issues := CheckSkillOrdering(agent, skillMap)

	for _, issue := range issues {
		assert.NotEqual(t, OrderDescriptionMismatch, issue.Type)
	}
}

func TestCheckSkillOrdering_HyphensAsSpaces(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"code-review", "test-runner"},
		model.OrchestrationSequential,
		"Run the test runner first, then do code review",
	)
	skillMap := map[string]model.SkillBehavior{
		"code-review": makeOrderingSkill("code-review", nil, nil),
		"test-runner": makeOrderingSkill("test-runner", nil, nil),
	}

	issues := CheckSkillOrdering(agent, skillMap)

	hasDescMismatch := false
	for _, issue := range issues {
		if issue.Type == OrderDescriptionMismatch {
			hasDescMismatch = true
		}
	}
	assert.True(t, hasDescMismatch, "expected description-order-mismatch with hyphen-to-space matching")
}

// Data flow order tests

func TestCheckSkillOrdering_DataFlowConsumerBeforeProducer(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"code-review", "test-runner"},
		model.OrchestrationSequential,
		"",
	)
	skillMap := map[string]model.SkillBehavior{
		"code-review": makeOrderingSkill("code-review", []string{"test_results"}, []string{"review_comments"}),
		"test-runner": makeOrderingSkill("test-runner", nil, []string{"test_results"}),
	}

	issues := CheckSkillOrdering(agent, skillMap)

	hasDataFlowMismatch := false
	for _, issue := range issues {
		if issue.Type == OrderDataFlowMismatch {
			hasDataFlowMismatch = true
			assert.Contains(t, issue.Message, "test_results")
		}
	}
	assert.True(t, hasDataFlowMismatch, "expected data-flow-order-mismatch issue")
}

func TestCheckSkillOrdering_DataFlowCorrectOrder(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"test-runner", "code-review"},
		model.OrchestrationSequential,
		"",
	)
	skillMap := map[string]model.SkillBehavior{
		"test-runner": makeOrderingSkill("test-runner", nil, []string{"test_results"}),
		"code-review": makeOrderingSkill("code-review", []string{"test_results"}, []string{"review_comments"}),
	}

	issues := CheckSkillOrdering(agent, skillMap)

	for _, issue := range issues {
		assert.NotEqual(t, OrderDataFlowMismatch, issue.Type)
	}
}

func TestCheckSkillOrdering_DataFlowParallelNoIssues(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"code-review", "test-runner"},
		model.OrchestrationParallel,
		"",
	)
	skillMap := map[string]model.SkillBehavior{
		"code-review": makeOrderingSkill("code-review", []string{"test_results"}, nil),
		"test-runner": makeOrderingSkill("test-runner", nil, []string{"test_results"}),
	}

	issues := CheckSkillOrdering(agent, skillMap)
	assert.Empty(t, issues)
}

func TestCheckSkillOrdering_DataFlowMissingSkills(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"nonexistent-a", "nonexistent-b"},
		model.OrchestrationSequential,
		"",
	)

	issues := CheckSkillOrdering(agent, map[string]model.SkillBehavior{})

	for _, issue := range issues {
		assert.NotEqual(t, OrderDataFlowMismatch, issue.Type)
	}
}

func TestCheckSkillOrdering_MultiStepChain(t *testing.T) {
	agent := makeOrderingAgent(
		[]string{"a", "b", "c"},
		model.OrchestrationSequential,
		"",
	)
	skillMap := map[string]model.SkillBehavior{
		"a": makeOrderingSkill("a", []string{"x"}, []string{"y"}),
		"b": makeOrderingSkill("b", []string{"y"}, []string{"z"}),
		"c": makeOrderingSkill("c", nil, []string{"x"}),
	}

	issues := CheckSkillOrdering(agent, skillMap)

	hasDataFlowMismatch := false
	for _, issue := range issues {
		if issue.Type == OrderDataFlowMismatch {
			hasDataFlowMismatch = true
		}
	}
	assert.True(t, hasDataFlowMismatch, "expected data-flow-order-mismatch for multi-step chain")
}
