package model_test

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
)

func validSkill() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "test-skill",
		Version: "1.0",
		Context: model.ContextFacet{
			Consumes: []string{"input"},
			Produces: []string{"output"},
			Memory:   model.MemoryShortTerm,
		},
		Strategy: model.StrategyFacet{
			Tools:    []string{"tool-a"},
			Approach: "do things",
		},
		Guardrails: []model.GuardrailRule{},
		DependsOn:  []model.Dependency{},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelMinimal,
			Metrics:    []string{"latency"},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessNone,
			Network:    model.NetworkNone,
			Secrets:    []string{},
		},
		Negotiation: model.NegotiationFacet{
			FileConflicts: model.NegotiationYield,
			Priority:      1,
		},
	}
}

func TestValidateSkillValid(t *testing.T) {
	errs := model.ValidateSkill(validSkill())
	assert.Empty(t, errs)
}

func TestValidateSkillMissingName(t *testing.T) {
	s := validSkill()
	s.Skill = ""
	errs := model.ValidateSkill(s)
	assert.Contains(t, errs, "skill name is required")
}

func TestValidateSkillMissingVersion(t *testing.T) {
	s := validSkill()
	s.Version = ""
	errs := model.ValidateSkill(s)
	assert.Contains(t, errs, "version is required")
}

func TestValidateSkillMissingApproach(t *testing.T) {
	s := validSkill()
	s.Strategy.Approach = ""
	errs := model.ValidateSkill(s)
	assert.Contains(t, errs, "strategy.approach is required")
}

func TestValidateSkillInvalidMemory(t *testing.T) {
	s := validSkill()
	s.Context.Memory = "invalid"
	errs := model.ValidateSkill(s)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid memory type")
}

func TestValidateSkillInvalidTraceLevel(t *testing.T) {
	s := validSkill()
	s.Observability.TraceLevel = "bogus"
	errs := model.ValidateSkill(s)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid trace level")
}

func TestValidateSkillInvalidAccessLevel(t *testing.T) {
	s := validSkill()
	s.Security.Filesystem = "root"
	errs := model.ValidateSkill(s)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid filesystem access level")
}

func TestValidateSkillInvalidNetworkAccess(t *testing.T) {
	s := validSkill()
	s.Security.Network = "everywhere"
	errs := model.ValidateSkill(s)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid network access")
}

func TestValidateSkillInvalidNegotiationStrategy(t *testing.T) {
	s := validSkill()
	s.Negotiation.FileConflicts = "fight"
	errs := model.ValidateSkill(s)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid negotiation strategy")
}

func TestValidateSkillMultipleErrors(t *testing.T) {
	s := model.SkillBehavior{} // everything empty/zero
	errs := model.ValidateSkill(s)
	assert.True(t, len(errs) >= 3, "expected multiple errors, got %d", len(errs))
}

// --- Agent validation ---

func validAgent() model.AgentComposition {
	return model.AgentComposition{
		Agent:         "test-agent",
		Skills:        []string{"skill-a", "skill-b"},
		Orchestration: model.OrchestrationSequential,
	}
}

func TestValidateAgentValid(t *testing.T) {
	errs := model.ValidateAgent(validAgent())
	assert.Empty(t, errs)
}

func TestValidateAgentMissingName(t *testing.T) {
	a := validAgent()
	a.Agent = ""
	errs := model.ValidateAgent(a)
	assert.Contains(t, errs, "agent name is required")
}

func TestValidateAgentMissingSkills(t *testing.T) {
	a := validAgent()
	a.Skills = nil
	errs := model.ValidateAgent(a)
	assert.Contains(t, errs, "at least one skill is required")
}

func TestValidateAgentEmptySkills(t *testing.T) {
	a := validAgent()
	a.Skills = []string{}
	errs := model.ValidateAgent(a)
	assert.Contains(t, errs, "at least one skill is required")
}

func TestValidateAgentInvalidOrchestration(t *testing.T) {
	a := validAgent()
	a.Orchestration = "chaos"
	errs := model.ValidateAgent(a)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid orchestration strategy")
}

func TestValidateAgentMultipleErrors(t *testing.T) {
	a := model.AgentComposition{}
	errs := model.ValidateAgent(a)
	assert.True(t, len(errs) >= 2, "expected multiple errors, got %d", len(errs))
}
