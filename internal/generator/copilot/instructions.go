package copilot

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/pkg/model"
)

// GenerateCopilotInstructions generates a copilot-instructions.md from skills and agents.
// Returns nil if there are no skills or agents.
func GenerateCopilotInstructions(skills []model.SkillBehavior, agents []model.AgentComposition) *string {
	if len(skills) == 0 && len(agents) == 0 {
		return nil
	}

	var lines []string
	lines = append(lines, "# Project Instructions")
	lines = append(lines, "")

	if len(skills) > 0 {
		lines = append(lines, "## Available Skills")
		lines = append(lines, "")
		for _, skill := range skills {
			desc := generator.BuildSkillDescription(skill)
			lines = append(lines, "- **"+generator.ToTitle(skill.Skill)+"**: "+desc)
		}
		lines = append(lines, "")
	}

	if len(agents) > 0 {
		lines = append(lines, "## Available Agents")
		lines = append(lines, "")
		for _, agent := range agents {
			desc := agent.Description
			if desc == "" {
				desc = string(agent.Orchestration) + " agent with " + fmt.Sprintf("%d", len(agent.Skills)) + " skills"
			}
			lines = append(lines, "- **"+generator.ToTitle(agent.Agent)+"**: "+desc)
		}
		lines = append(lines, "")
	}

	// Global guardrails
	var allGuardrails []model.GuardrailRule
	for _, s := range skills {
		allGuardrails = append(allGuardrails, s.Guardrails...)
	}
	if len(allGuardrails) > 0 {
		lines = append(lines, "## Global Guardrails")
		lines = append(lines, "")
		for _, g := range allGuardrails {
			lines = append(lines, generator.FormatGuardrail(g))
		}
		lines = append(lines, "")
	}

	result := strings.Join(lines, "\n")
	return &result
}
