package analyzer

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func makeFullSkill(overrides ...func(*model.SkillBehavior)) model.SkillBehavior {
	timeoutGR := model.GuardrailRule{}
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "timeout", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "5min", Tag: "!!str"},
		},
	}
	_ = timeoutGR.UnmarshalYAML(node)

	maxRetriesGR := model.GuardrailRule{}
	strNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "max_retries: 3", Tag: "!!str"}
	_ = maxRetriesGR.UnmarshalYAML(strNode)

	skill := model.SkillBehavior{
		Skill:   "test-skill",
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"input"},
			Produces: []string{"output"},
			Memory:   model.MemoryConversation,
		},
		Strategy: model.StrategyFacet{
			Tools:    []string{"read_file", "grep"},
			Approach: "diff-first",
			Steps:    []string{"step1", "step2"},
		},
		Guardrails: []model.GuardrailRule{timeoutGR, maxRetriesGR},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelDetailed,
			Metrics:    []string{"tokens", "latency"},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadOnly,
			Network:    model.NetworkNone,
			Secrets:    []string{},
		},
	}

	for _, fn := range overrides {
		fn(&skill)
	}
	return skill
}

func makeFullAgent(overrides ...func(*model.AgentComposition)) model.AgentComposition {
	agent := model.AgentComposition{
		Agent:         "test-agent",
		Skills:        []string{"skill-a", "skill-b"},
		Orchestration: model.OrchestrationSequential,
		Description:   "A well-described agent",
	}
	for _, fn := range overrides {
		fn(&agent)
	}
	return agent
}

func TestScoreSkill_HighScore(t *testing.T) {
	result := ScoreSkill(makeFullSkill())
	assert.GreaterOrEqual(t, result.Total, 80)
}

func TestScoreSkill_PenalizeMissingTools(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	noTools := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Strategy = model.StrategyFacet{Tools: []string{}, Approach: "seq", Steps: []string{"s1"}}
	}))
	assert.Less(t, noTools.Total, full.Total)
	assert.Less(t, noTools.Breakdown.Strategy, full.Breakdown.Strategy)
}

func TestScoreSkill_PenalizeMissingGuardrails(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	noGR := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Guardrails = nil
	}))
	assert.Less(t, noGR.Total, full.Total)
	assert.Equal(t, 0, noGR.Breakdown.Guardrails)
}

func TestScoreSkill_PenalizeMissingSteps(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	noSteps := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Strategy = model.StrategyFacet{Tools: []string{"read_file"}, Approach: "seq", Steps: []string{}}
	}))
	assert.Less(t, noSteps.Breakdown.Strategy, full.Breakdown.Strategy)
}

func TestScoreSkill_PenalizeMissingObservability(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	noObs := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Observability = model.ObservabilityFacet{TraceLevel: model.TraceLevelMinimal, Metrics: []string{}}
	}))
	assert.Less(t, noObs.Breakdown.Observability, full.Breakdown.Observability)
}

func TestScoreSkill_RewardRestrictiveSecurity(t *testing.T) {
	readOnly := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Security = model.SecurityFacet{Filesystem: model.AccessReadOnly, Network: model.NetworkNone, Secrets: []string{}}
	}))
	fullAccess := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Security = model.SecurityFacet{Filesystem: model.AccessFull, Network: model.NetworkFull, Secrets: []string{"API_KEY"}}
	}))
	assert.Greater(t, readOnly.Breakdown.Security, fullAccess.Breakdown.Security)
}

func TestScoreSkill_PenalizeEmptyContext(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	emptyCtx := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{}, Memory: model.MemoryShortTerm}
	}))
	assert.Less(t, emptyCtx.Breakdown.Context, full.Breakdown.Context)
}

func TestScoreSkill_BreakdownFields(t *testing.T) {
	result := ScoreSkill(makeFullSkill())
	assert.Greater(t, result.Breakdown.Context, 0)
	assert.Greater(t, result.Breakdown.Strategy, 0)
	assert.Greater(t, result.Breakdown.Guardrails, 0)
	assert.Greater(t, result.Breakdown.Observability, 0)
	assert.Greater(t, result.Breakdown.Security, 0)
}

func TestScoreSkill_TotalBetween0And100(t *testing.T) {
	result := ScoreSkill(makeFullSkill())
	assert.GreaterOrEqual(t, result.Total, 0)
	assert.LessOrEqual(t, result.Total, 100)
}

func TestScoreSkill_WorstCase(t *testing.T) {
	worst := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{}, Memory: model.MemoryShortTerm}
		s.Strategy = model.StrategyFacet{Tools: []string{}, Approach: "", Steps: []string{}}
		s.Guardrails = nil
		s.Observability = model.ObservabilityFacet{TraceLevel: model.TraceLevelMinimal, Metrics: []string{}}
		s.Security = model.SecurityFacet{Filesystem: model.AccessFull, Network: model.NetworkFull, Secrets: []string{"SECRET"}}
	}))
	assert.Less(t, worst.Total, 30)
}

func TestScoreAgent_HighScore(t *testing.T) {
	skills := []model.SkillBehavior{
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-a"
			s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{"data"}, Memory: model.MemoryShortTerm}
		}),
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-b"
			s.Context = model.ContextFacet{Consumes: []string{"data"}, Produces: []string{"result"}, Memory: model.MemoryShortTerm}
		}),
	}
	result := ScoreAgent(makeFullAgent(), skills)
	assert.GreaterOrEqual(t, result.Total, 80)
}

func TestScoreAgent_PenalizeMissingDescription(t *testing.T) {
	withDesc := ScoreAgent(makeFullAgent(), nil)
	noDesc := ScoreAgent(makeFullAgent(func(a *model.AgentComposition) {
		a.Description = ""
	}), nil)
	assert.Less(t, noDesc.Total, withDesc.Total)
}

func TestScoreAgent_PenalizeBrokenDataFlow(t *testing.T) {
	goodFlow := []model.SkillBehavior{
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-a"
			s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{"data"}, Memory: model.MemoryShortTerm}
		}),
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-b"
			s.Context = model.ContextFacet{Consumes: []string{"data"}, Produces: []string{"result"}, Memory: model.MemoryShortTerm}
		}),
	}
	brokenFlow := []model.SkillBehavior{
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-a"
			s.Context = model.ContextFacet{Consumes: []string{"data"}, Produces: []string{"result"}, Memory: model.MemoryShortTerm}
		}),
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-b"
			s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{"data"}, Memory: model.MemoryShortTerm}
		}),
	}
	good := ScoreAgent(makeFullAgent(), goodFlow)
	broken := ScoreAgent(makeFullAgent(), brokenFlow)
	assert.Less(t, broken.Breakdown.DataFlow, good.Breakdown.DataFlow)
}

func TestScoreAgent_EnvironmentInputsNotPenalized(t *testing.T) {
	skills := []model.SkillBehavior{
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-a"
			s.Context = model.ContextFacet{Consumes: []string{"file_tree", "source_code"}, Produces: []string{"lint_results"}, Memory: model.MemoryShortTerm}
		}),
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-b"
			s.Context = model.ContextFacet{Consumes: []string{"file_tree", "lint_results"}, Produces: []string{"report"}, Memory: model.MemoryShortTerm}
		}),
	}
	result := ScoreAgent(makeFullAgent(), skills)
	// lint_results is the only inter-skill dependency and it's correctly ordered
	assert.Equal(t, 35, result.Breakdown.DataFlow)
}

func TestScoreAgent_PenalizeSingleSkill(t *testing.T) {
	multi := ScoreAgent(makeFullAgent(func(a *model.AgentComposition) {
		a.Skills = []string{"a", "b"}
	}), nil)
	single := ScoreAgent(makeFullAgent(func(a *model.AgentComposition) {
		a.Skills = []string{"a"}
	}), nil)
	assert.Less(t, single.Breakdown.Composition, multi.Breakdown.Composition)
}

func TestScoreAgent_TotalBetween0And100(t *testing.T) {
	result := ScoreAgent(makeFullAgent(), nil)
	assert.GreaterOrEqual(t, result.Total, 0)
	assert.LessOrEqual(t, result.Total, 100)
}
