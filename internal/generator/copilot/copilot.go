package copilot

import (
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/mirandaguillaume/forgent/pkg/spec"
)

type copilotGenerator struct{}

func (g *copilotGenerator) Target() string           { return "copilot" }
func (g *copilotGenerator) DefaultOutputDir() string { return ".github" }

func (g *copilotGenerator) GenerateSkill(skill model.SkillBehavior) string {
	return GenerateCopilotSkillMd(skill)
}

func (g *copilotGenerator) GenerateAgent(agent model.AgentComposition, skills []model.SkillBehavior, outputDir string) string {
	return GenerateCopilotAgentMd(agent, skills, outputDir)
}

func (g *copilotGenerator) GenerateInstructions(skills []model.SkillBehavior, agents []model.AgentComposition) *string {
	return GenerateCopilotInstructions(skills, agents)
}

func (g *copilotGenerator) SkillPath(name string) string { return "skills/" + name + "/SKILL.md" }
func (g *copilotGenerator) AgentPath(name string) string { return "agents/" + name + ".agent.md" }
func (g *copilotGenerator) ContextDir() string           { return "context" }
func (g *copilotGenerator) InstructionsPath() *string {
	s := "copilot-instructions.md"
	return &s
}

func init() {
	spec.Register("copilot", func() spec.TargetGenerator { return &copilotGenerator{} })
}
