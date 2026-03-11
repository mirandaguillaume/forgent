package claude_test

import (
	"testing"

	_ "github.com/mirandaguillaume/forgent/internal/generator/claude"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/mirandaguillaume/forgent/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeGenerator_Registration(t *testing.T) {
	gen, err := spec.Get("claude")
	require.NoError(t, err)
	assert.NotNil(t, gen)
}

func TestClaudeGenerator_Target(t *testing.T) {
	gen, _ := spec.Get("claude")
	assert.Equal(t, "claude", gen.Target())
}

func TestClaudeGenerator_DefaultOutputDir(t *testing.T) {
	gen, _ := spec.Get("claude")
	assert.Equal(t, ".claude", gen.DefaultOutputDir())
}

func TestClaudeGenerator_SkillPath(t *testing.T) {
	gen, _ := spec.Get("claude")
	assert.Equal(t, "skills/code-review/SKILL.md", gen.SkillPath("code-review"))
}

func TestClaudeGenerator_AgentPath(t *testing.T) {
	gen, _ := spec.Get("claude")
	assert.Equal(t, "agents/code-reviewer.md", gen.AgentPath("code-reviewer"))
}

func TestClaudeGenerator_InstructionsPath(t *testing.T) {
	gen, _ := spec.Get("claude")
	assert.Nil(t, gen.InstructionsPath())
}

func TestClaudeGenerator_GenerateInstructions(t *testing.T) {
	gen, _ := spec.Get("claude")
	result := gen.GenerateInstructions(nil, nil)
	assert.Nil(t, result)
}

func TestClaudeGenerator_GenerateSkill(t *testing.T) {
	gen, _ := spec.Get("claude")
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

func TestClaudeGenerator_GenerateAgent(t *testing.T) {
	gen, _ := spec.Get("claude")
	agent := model.AgentComposition{
		Agent:         "my-agent",
		Orchestration: model.OrchestrationSequential,
		Skills:        []string{"skill-a"},
	}
	md := gen.GenerateAgent(agent, nil, ".claude")
	assert.Contains(t, md, "name: my-agent")
}
