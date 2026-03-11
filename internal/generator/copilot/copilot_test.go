package copilot_test

import (
	"testing"

	_ "github.com/mirandaguillaume/forgent/internal/generator/copilot"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/mirandaguillaume/forgent/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopilotGenerator_Registration(t *testing.T) {
	gen, err := spec.Get("copilot")
	require.NoError(t, err)
	assert.NotNil(t, gen)
}

func TestCopilotGenerator_Target(t *testing.T) {
	gen, _ := spec.Get("copilot")
	assert.Equal(t, "copilot", gen.Target())
}

func TestCopilotGenerator_DefaultOutputDir(t *testing.T) {
	gen, _ := spec.Get("copilot")
	assert.Equal(t, ".github", gen.DefaultOutputDir())
}

func TestCopilotGenerator_SkillPath(t *testing.T) {
	gen, _ := spec.Get("copilot")
	assert.Equal(t, "skills/code-review/SKILL.md", gen.SkillPath("code-review"))
}

func TestCopilotGenerator_AgentPath(t *testing.T) {
	gen, _ := spec.Get("copilot")
	assert.Equal(t, "agents/code-reviewer.agent.md", gen.AgentPath("code-reviewer"))
}

func TestCopilotGenerator_InstructionsPath(t *testing.T) {
	gen, _ := spec.Get("copilot")
	path := gen.InstructionsPath()
	require.NotNil(t, path)
	assert.Equal(t, "copilot-instructions.md", *path)
}

func TestCopilotGenerator_GenerateInstructions_NotNil(t *testing.T) {
	gen, _ := spec.Get("copilot")
	skills := []model.SkillBehavior{
		{
			Skill: "test-skill",
			Strategy: model.StrategyFacet{
				Approach: "analytical",
			},
			Context: model.ContextFacet{
				Memory: model.MemoryShortTerm,
			},
			Security: model.SecurityFacet{
				Filesystem: model.AccessReadOnly,
				Network:    model.NetworkNone,
			},
		},
	}
	result := gen.GenerateInstructions(skills, nil)
	require.NotNil(t, result)
	assert.Contains(t, *result, "# Project Instructions")
}

func TestCopilotGenerator_GenerateInstructions_NilForEmpty(t *testing.T) {
	gen, _ := spec.Get("copilot")
	result := gen.GenerateInstructions(nil, nil)
	assert.Nil(t, result)
}

func TestCopilotGenerator_GenerateSkill(t *testing.T) {
	gen, _ := spec.Get("copilot")
	skill := model.SkillBehavior{
		Skill: "test-skill",
		Strategy: model.StrategyFacet{
			Approach: "analytical",
		},
		Context: model.ContextFacet{
			Memory: model.MemoryShortTerm,
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadOnly,
			Network:    model.NetworkNone,
		},
	}
	md := gen.GenerateSkill(skill)
	assert.Contains(t, md, "# Test Skill")
	assert.Contains(t, md, "name: test-skill")
}

func TestCopilotGenerator_GenerateAgent(t *testing.T) {
	gen, _ := spec.Get("copilot")
	agent := model.AgentComposition{
		Agent:         "my-agent",
		Orchestration: model.OrchestrationSequential,
		Skills:        []string{"skill-a"},
	}
	md := gen.GenerateAgent(agent, nil, ".github")
	assert.Contains(t, md, "name: my-agent")
}
