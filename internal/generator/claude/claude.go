package claude

import (
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/mirandaguillaume/forgent/pkg/spec"
)

type claudeGenerator struct {
	compact bool
}

func (g *claudeGenerator) Target() string           { return "claude" }
func (g *claudeGenerator) DefaultOutputDir() string { return ".claude" }

func (g *claudeGenerator) GenerateSkill(skill model.SkillBehavior) string {
	return GenerateSkillMd(skill)
}

func (g *claudeGenerator) SetOptions(opts spec.GeneratorOptions) {
	g.compact = opts.Compact
}

func (g *claudeGenerator) GenerateAgent(agent model.AgentComposition, skills []model.SkillBehavior, outputDir string) string {
	if g.compact {
		return GenerateCompactAgentMd(agent, skills)
	}
	return GenerateAgentMd(agent, skills, outputDir)
}

func (g *claudeGenerator) SkillPath(name string) string { return "skills/" + name + "/SKILL.md" }
func (g *claudeGenerator) AgentPath(name string) string { return "agents/" + name + ".md" }
func (g *claudeGenerator) ContextDir() string            { return "context" }

func init() {
	spec.Register("claude", func() spec.Generator { return &claudeGenerator{} })
}
