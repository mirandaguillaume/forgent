package analyzer

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
)

func makeSkill(name string, consumes, produces []string) model.SkillBehavior {
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

func makeAgent(name string, skills []string, consumes, produces []string) model.AgentComposition {
	return model.AgentComposition{
		Agent:         name,
		Skills:        skills,
		Orchestration: model.OrchestrationSequential,
		Consumes:      consumes,
		Produces:      produces,
	}
}

func TestCheckDependencies_NoIssues(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("skill-a", nil, []string{"data"}),
		makeSkill("skill-b", []string{"data"}, []string{"result"}),
	}
	agent := makeAgent("test-agent", []string{"skill-a", "skill-b"}, nil, []string{"result"})

	issues := CheckDependencies(agent, skills)
	assert.Empty(t, issues)
}

func TestCheckDependencies_MissingSkill(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("skill-a", nil, []string{"data"}),
	}
	agent := makeAgent("test-agent", []string{"skill-a", "nonexistent"}, nil, nil)

	issues := CheckMissingDependencies(agent, skills)
	assert.Len(t, issues, 1)
	assert.Equal(t, IssueMissing, issues[0].Type)
	assert.Equal(t, "nonexistent", issues[0].Skill)
}

func TestCheckDependencies_CircularDependency(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("skill-a", []string{"data"}, []string{"result"}),
		makeSkill("skill-b", []string{"result"}, []string{"data"}),
	}
	agent := makeAgent("test-agent", []string{"skill-a", "skill-b"}, nil, nil)

	issues := CheckCircularDependencies(agent, skills)

	hasCircular := false
	for _, issue := range issues {
		if issue.Type == IssueCircular {
			hasCircular = true
			assert.Contains(t, issue.Message, "Circular dependency detected")
		}
	}
	assert.True(t, hasCircular, "expected at least one circular dependency issue")
}

func TestCheckDependencies_NoCycle(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("skill-a", nil, []string{"data"}),
		makeSkill("skill-b", []string{"data"}, []string{"result"}),
	}
	agent := makeAgent("test-agent", []string{"skill-a", "skill-b"}, nil, nil)

	issues := CheckCircularDependencies(agent, skills)
	assert.Empty(t, issues)
}

func TestCheckUnmetContext_Unmet(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("skill-a", []string{"missing-input"}, []string{"data"}),
	}
	agent := makeAgent("test-agent", []string{"skill-a"}, nil, nil)

	issues := CheckUnmetContext(agent, skills)
	assert.Len(t, issues, 1)
	assert.Equal(t, IssueUnmetContext, issues[0].Type)
	assert.Contains(t, issues[0].Message, "missing-input")
}

func TestCheckUnmetContext_SatisfiedByAgentConsumes(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("skill-a", []string{"external-input"}, []string{"data"}),
	}
	agent := makeAgent("test-agent", []string{"skill-a"}, []string{"external-input"}, nil)

	issues := CheckUnmetContext(agent, skills)
	assert.Empty(t, issues)
}

func TestCheckUnmetContext_SatisfiedByOtherSkill(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("skill-a", nil, []string{"data"}),
		makeSkill("skill-b", []string{"data"}, []string{"result"}),
	}
	agent := makeAgent("test-agent", []string{"skill-a", "skill-b"}, nil, nil)

	issues := CheckUnmetContext(agent, skills)
	assert.Empty(t, issues)
}

func TestCheckAgentProduces_Valid(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("skill-a", nil, []string{"data"}),
	}
	agent := makeAgent("test-agent", []string{"skill-a"}, nil, []string{"data"})

	issues := CheckAgentProduces(agent, skills)
	assert.Empty(t, issues)
}

func TestCheckAgentProduces_Invalid(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("skill-a", nil, []string{"data"}),
	}
	agent := makeAgent("test-agent", []string{"skill-a"}, nil, []string{"nonexistent"})

	issues := CheckAgentProduces(agent, skills)
	assert.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, "nonexistent")
}
